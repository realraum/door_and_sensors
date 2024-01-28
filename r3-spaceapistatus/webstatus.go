// (c) Bernhard Tittelbach, 2013..2019

package main

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/realraum/door_and_sensors/r3-spaceapistatus/spaceapi"

	r3events "github.com/realraum/door_and_sensors/r3events"
)

type spaceState struct {
	present           bool
	buttonpress_until int64
	space1present     bool
	space2present     bool
}

var (
	spaceapidata        spaceapi.SpaceInfo = spaceapi.NewSpaceInfo("realraum", "https://realraum.at", "https://realraum.at/logo-red_250x250.png").SetSpaceState(spaceapi.SpaceState{LastChange: time.Now().Unix(), Icon: &spaceapi.SpaceStateIcon{OpenIconURI: "https://realraum.at/logo-re_open_100x100.png", CloseIconURI: "https://realraum.at/logo-re_empty_100x100.png"}}).SetSpaceLocation(spaceapi.SpaceLocation{Address: "Brockmanngasse 15, 8010 Graz, Austria", Lat: 47.065554, Lon: 15.450435}).AddBaseExt("ext_ccc", "chaostreff")
	statusstate         *spaceState        = new(spaceState)
	re_querystresc_     *regexp.Regexp     = regexp.MustCompile("[^\x30-\x39\x41-\x7E]")
	spaceapijsonbytes   []byte
	spaceapijsonRWMutex sync.RWMutex
)

func init() {
	spaceapidata.AddSpaceFeed("calendar", "https://www.realraum.at/shmcache/grical_realraum_only.ical", "ical")
	spaceapidata.AddSpaceFeed("blog", "https://wp.realraum.at/feed/", "rss")
	spaceapidata.AddSpaceFeed("wiki", "https://realraum.at/wiki/feed.php", "rss")
	spaceapidata.SetSpaceContactIRC("irc://irc.oftc.net/#realraum", false).SetSpaceContactMailinglist("realraum@realraum.at", false).SetSpaceContactEmail("vorstand@realraum.at", true)
	spaceapidata.SetSpaceContactIssueMail("vorstand@realraum.at", true).SetSpaceContactTwitter("@realraum@chaos.social", false)
	spaceapidata.SetSpaceContactPhone("+49221596191003", false).SetSpaceContactJabber("realraum@realraum.at", false)
	spaceapidata.AddProjectsURLs([]string{"https://chaos.social/@realraum", "https://git.github.com/realraum", "https://wiki.realraum.at/wiki/doku.php?id=projekte", "https://wp.realraum.at/", "https://git.realraum.at"})
	if len(os.Getenv("R3_TOTAL_MEMBERCOUNT")) > 0 {
		total_member_count, err := strconv.Atoi(os.Getenv("R3_TOTAL_MEMBERCOUNT"))
		if err == nil {
			spaceapidata.MergeInSensor(spaceapi.MakeMemberCountSensor("Member Count", "realraum", int64(total_member_count)))
		}
	}
}

func roundToDecimalPlaces(value float64, decimals uint) float64 {
	value *= 1*math.Pow(10,float64(decimals))
	value = math.Round(value)
	value /= 1*math.Pow(10,float64(decimals))
	return value
}

func updateStatusString() {
	var spacestatus string
	if statusstate.present {
		if statusstate.buttonpress_until > time.Now().Unix() {
			spacestatus = "People Having Fun!"
		} else if true == statusstate.space1present {
			spacestatus = "Leute Anwesend"
		} else if false == statusstate.space1present && true == statusstate.space2present {
			spacestatus = "Nur Hinterwhg"
		}
	} else {
		spacestatus = "Keiner Da"
	}
	spaceapidata.UpdateSpaceStatus(statusstate.present, spacestatus)
}

func publishStateNotKnown() {
	spaceapidata.UpdateSpaceStatusNotKnown("Status Unknown")
	publishStateToWeb()
}

func publishStateToWeb() {
	updateStatusString()
	jsondata_b, err := spaceapidata.MakeJSON()
	if err != nil {
		Syslog_.Println("Error:", err)
		return
	}

	go func() { //update (and block) in background
		spaceapijsonRWMutex.Lock()
		spaceapijsonbytes = jsondata_b
		spaceapijsonRWMutex.Unlock()
	}()

	session := getWebStatusSSHSession()
	if session == nil {
		return
	}

	defer session.Close()

	timeout_tmr := time.NewTimer(4 * time.Second)
	done_chan := make(chan bool, 2)

	go func() {
		defer func() { done_chan <- true }()
		stdinp, err := session.StdinPipe()
		if err != nil {
			Syslog_.Println("Error: Failed to create ssh stdin pipe:", err.Error())
			return
		}
		defer stdinp.Close()

		if err := session.Start("set"); err != nil {
			Syslog_.Println("Error: Failed to run ssh command:", err.Error())
			return
		}
		written, err := stdinp.Write(jsondata_b)
		if err != nil {
			Syslog_.Println("Error: Failed to publish status info", err.Error())
			return
		}
		Syslog_.Printf("updated status.json (sent %d bytes)", written)
	}()

	select {
	case <-done_chan:
	case <-timeout_tmr.C:
		Syslog_.Printf("Error: publishStateToWeb timed out")
	}
}

func EventToWeb(events chan *r3events.R3MQTTMsg) {
	ticker := time.NewTicker(time.Duration(6) * time.Minute)
	for {
		select {

		case <-ticker.C:
			spaceapidata.CleanOutdatedSpaceEvents()
			spaceapidata.CleanOutdatedSensorData(15 * time.Minute)
			publishStateToWeb()

		case r3msg := <-events:
			Debug_.Printf("EventToWeb: %T %+v", r3msg, r3msg)
			eventinterface := r3msg.Event
			switch event := eventinterface.(type) {
			case r3events.PresenceUpdate:
				statusstate.present = event.Present
				statusstate.space1present = event.InSpace1
				statusstate.space2present = event.InSpace2
				spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("Space1Empty", "AllW1", "Niemand in Whg1", !statusstate.space1present))
				spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("Space2Empty", "AllW2", "Niemand in Whg2", !statusstate.space2present))
				publishStateToWeb()
			case r3events.DoorAjarUpdate:
				if len(r3msg.Topic) < 3 {
					continue
				}
				switch r3msg.Topic[1] {
				case r3events.CLIENTID_FRONTDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorAjarSensor("AjarW1Torwaechter", "W1 Frontdoor", "Türkontakt Whg1 Eingangstür", event.Shut))
				case r3events.CLIENTID_BACKDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorAjarSensor("AjarBackdoorBlue", "W1 BackdoorBlue", "Türkontakt Whg1 Hintertür", event.Shut))
				case r3events.CLIENTID_W2FRONTDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorAjarSensor("AjarW2Door", "W2 Frontdoor", "Türkontakt Whg2 Eingangstür", event.Shut))
				default:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorAjarSensor("AjarUnknown", "unknown", "Unbekannter Türkontakt", event.Shut))
				}
				publishStateToWeb()
			case r3events.ZigbeeAjarSensor:
				spaceapidata.MergeInSensor(spaceapi.MakeDoorAjarSensor("AjarWindow"+event.Location, "Window "+event.Location, "Fensterkontakt "+event.Location, event.Contact))
				spaceapidata.MergeInSensor(spaceapi.MakeVoltageSensor(fmt.Sprintf("Voltage@AjarWindow%s", event.Location), event.Location, "V", float64(event.Millivolt)/1000.0, event.Ts))
				publishStateToWeb()
			case r3events.DoorLockUpdate:
				if len(r3msg.Topic) < 3 {
					continue
				}
				switch r3msg.Topic[1] {
				case r3events.CLIENTID_FRONTDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("LockW1Torwaechter", "W1 Frontdoor", "Türschloß Whg1 Eingangstür", event.Locked))
				case r3events.CLIENTID_BACKDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("LockBackdoorBlue", "W1 BackdoorBlue", "Türschloß Whg1 Hintertür", event.Locked))
				case r3events.CLIENTID_W2FRONTDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("LockW2Door", "W2 Frontdoor", "Türschloß Whg2 Eingangstür", event.Locked))
				default:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("LockUnknown", "unknown", "Unbekanntes Türschloß", event.Locked))
				}
				publishStateToWeb()
			case r3events.BoreDoomButtonPressEvent:
				statusstate.buttonpress_until = event.Ts + 3600
				spaceapidata.AddSpaceEvent("BoreDOOMButton", "check-in", "The button has been pressed", event.Ts, 4*time.Hour)
				publishStateToWeb()
			case r3events.TempSensorUpdate:
				spaceapidata.MergeInSensor(spaceapi.MakeTempCSensor(fmt.Sprintf("Temp@%s", event.Location), event.Location, roundToDecimalPlaces(event.Value,2), event.Ts))
			case r3events.IlluminationSensorUpdate:
				spaceapidata.MergeInSensor(spaceapi.MakeIlluminationSensor("Photoresistor", event.Location, "/2^10", event.Value, event.Ts))
			case r3events.RelativeHumiditySensorUpdate:
				spaceapidata.MergeInSensor(spaceapi.MakeHumiditySensor(fmt.Sprintf("Humidity@%s", event.Location), event.Location, "%", roundToDecimalPlaces(event.Percent, 2), event.Ts))
			case r3events.Voltage:
				spaceapidata.MergeInSensor(spaceapi.MakeVoltageSensor(fmt.Sprintf("Voltage@%s", event.Location), event.Location, "V", roundToDecimalPlaces(event.Value, 2), event.Ts))
				if event.Min != event.Max {
					spaceapidata.MergeInSensor(spaceapi.MakeBatteryChargeSensor(fmt.Sprintf("BatteryCharge@%s", event.Location), event.Location, "%", roundToDecimalPlaces(event.Percent, 2), event.Ts))
				}
			case r3events.BarometerUpdate:
				spaceapidata.MergeInSensor(spaceapi.MakeBarometerSensor(fmt.Sprintf("Barometer@%s", event.Location), event.Location, "hPa", roundToDecimalPlaces(event.HPa, 2), event.Ts))
			case r3events.GasLeakAlert:
				spaceapidata.AddSpaceEvent("GasLeak", "alert", "GasLeak Alert has been triggered", event.Ts, 24*time.Hour)
				publishStateToWeb()
			case r3events.TempOverThreshold:
				spaceapidata.AddSpaceEvent("TemperatureLimitExceeded", "alert", fmt.Sprintf("Temperature %s has exceeded limit at %f °C", event.Location, event.Value), event.Ts, 24*time.Hour)
				publishStateToWeb()
			case r3events.UPSPowerUpdate:
				if event.PercentBattery < 100 {
					if event.OnBattery {
						spaceapidata.AddSpaceEvent("PowerLoss", "alert", fmt.Sprintf("UPS reports power loss. Battery at %d%%.", event.PercentBattery), event.Ts, 24*time.Hour)
					} else {
						spaceapidata.AddSpaceEvent("PowerLoss", "alert", fmt.Sprintf("UPS reports power resumed. Battery at %d%%.", event.PercentBattery), event.Ts, 24*time.Hour)
					}
				}
				publishStateToWeb()
			case r3events.LaserCutter:
				spaceapidata.MergeInSensor(spaceapi.MakeLasercutterHotSensor("LasercutterHot", "M500", event.IsHot, time.Now().Unix()))
				publishStateToWeb()
			case r3events.ThreeDimensionalPrinterProgress:
				spaceapidata.MergeInSensor(spaceapi.Make3DPrinterSensor(event.Printer, event.Job, roundToDecimalPlaces(event.Progress_percent, 1), event.Elapsed_time_s, time.Now().Unix()))
				publishStateToWeb()
			case r3events.FoodOrderETA:
				//TODO: remember food orders TSofInvite and overwrite with new ETA if the same or add additonal if new
				unixeta := time.Unix(event.ETA, 0)
				timestr := unixeta.Format("15:04")
				spaceapidata.AddSpaceEvent("Food Order ETA: "+timestr, "food", fmt.Sprintf("Food will arrive at %s", timestr), event.Ts, unixeta.Sub(time.Now()))
				publishStateToWeb()
			}
		}
	}
}

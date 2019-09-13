// (c) Bernhard Tittelbach, 2013..2019

package main

import (
	"fmt"
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
}

var (
	spaceapidata        spaceapi.SpaceInfo = spaceapi.NewSpaceInfo("realraum", "https://realraum.at", "https://realraum.at/logo-red_250x250.png").SetSpaceState(spaceapi.SpaceState{LastChange: time.Now().Unix(), Icon: &spaceapi.SpaceStateIcon{OpenIconURI: "https://realraum.at/logo-re_open_100x100.png", CloseIconURI: "https://realraum.at/logo-re_empty_100x100.png"}}).SetSpaceLocation(spaceapi.SpaceLocation{Address: "Brockmanngasse 15, 8010 Graz, Austria", Lat: 47.065554, Lon: 15.450435}).AddBaseExt("ext_ccc", "chaostreff")
	statusstate         *spaceState        = new(spaceState)
	re_querystresc_     *regexp.Regexp     = regexp.MustCompile("[^\x30-\x39\x41-\x7E]")
	spaceapijsonbytes   []byte
	spaceapijsonRWMutex sync.RWMutex
)

func init() {
	spaceapidata.AddSpaceFeed("calendar", "https://www.realraum.at/shmcache/grical_realraum.ical", "ical")
	spaceapidata.AddSpaceFeed("blog", "https://wp.realraum.at/feed/", "rss")
	spaceapidata.AddSpaceFeed("wiki", "https://realraum.at/wiki/feed.php", "rss")
	spaceapidata.SetSpaceContactIRC("irc://irc.oftc.net/#realraum", false).SetSpaceContactMailinglist("realraum@realraum.at", false).SetSpaceContactEmail("vorstand@realraum.at", true)
	spaceapidata.SetSpaceContactIssueMail("vorstand@realraum.at", true).SetSpaceContactTwitter("@realraum", false)
	spaceapidata.SetSpaceContactJabber("realraum@realraum.at", false)
	spaceapidata.AddProjectsURLs([]string{"https://git.github.com/realraum", "https://wiki.realraum.at/wiki/doku.php?id=projekte", "https://wp.realraum.at/", "https://synbiota.com/projects/openbiolabgraz/summary", "https://git.realraum.at"})
	if len(os.Getenv("R3_TOTAL_MEMBERCOUNT")) > 0 {
		total_member_count, err := strconv.Atoi(os.Getenv("R3_TOTAL_MEMBERCOUNT"))
		if err == nil {
			spaceapidata.MergeInSensor(spaceapi.MakeMemberCountSensor("Member Count", "realraum", int64(total_member_count)))
		}
	}
}

func updateStatusString() {
	var spacestatus string
	if statusstate.present {
		if statusstate.buttonpress_until > time.Now().Unix() {
			spacestatus = "Panic! Present&Bored"
		} else {
			spacestatus = "Leute Anwesend"
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
				publishStateToWeb()
			case r3events.DoorAjarUpdate:
				if len(r3msg.Topic) < 3 {
					continue
				}
				switch r3msg.Topic[1] {
				case r3events.CLIENTID_FRONTDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("TorwaechterAjarSensor", "Türkontakt", event.Shut))
				case r3events.CLIENTID_BACKDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("HintertürAjarSensor", "Hintertürkontakt", event.Shut))
				case r3events.CLIENTID_W2FRONTDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("Wohnung2AjarSensor", "Zweitwohnungstürkontakt", event.Shut))
				default:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("UnknownAjarSensor", "Unbekannter Türkontakt", event.Shut))
				}
				publishStateToWeb()
			case r3events.DoorLockUpdate:
				if len(r3msg.Topic) < 3 {
					continue
				}
				switch r3msg.Topic[1] {
				case r3events.CLIENTID_FRONTDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("TorwaechterLock", "Haupttürschloß", event.Locked))
				case r3events.CLIENTID_W2FRONTDOOR:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("Wohnung2Lock", "Zweitwohungtürschloß", event.Locked))
				default:
					spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("UnknownLock", "Unbekanntes Türschloß", event.Locked))
				}
				publishStateToWeb()
			case r3events.BoreDoomButtonPressEvent:
				statusstate.buttonpress_until = event.Ts + 3600
				spaceapidata.AddSpaceEvent("BoreDOOMButton", "check-in", "The button has been pressed", event.Ts, 4*time.Hour)
				publishStateToWeb()
			case r3events.TempSensorUpdate:
				spaceapidata.MergeInSensor(spaceapi.MakeTempCSensor(fmt.Sprintf("Temp@%s", event.Location), event.Location, event.Value, event.Ts))
			case r3events.IlluminationSensorUpdate:
				spaceapidata.MergeInSensor(spaceapi.MakeIlluminationSensor("Photoresistor", event.Location, "/2^10", event.Value, event.Ts))
			case r3events.RelativeHumiditySensorUpdate:
				spaceapidata.MergeInSensor(spaceapi.MakeHumiditySensor(fmt.Sprintf("Humidity@%s", event.Location), event.Location, "%", event.Percent, event.Ts))
			case r3events.Voltage:
				spaceapidata.MergeInSensor(spaceapi.MakeVoltageSensor(fmt.Sprintf("Voltage@%s", event.Location), event.Location, "V", event.Value, event.Ts))
				if event.Min != event.Max {
					spaceapidata.MergeInSensor(spaceapi.MakeBatteryChargeSensor(fmt.Sprintf("BatteryCharge@%s", event.Location), event.Location, "%", event.Percent, event.Ts))
				}
			case r3events.BarometerUpdate:
				spaceapidata.MergeInSensor(spaceapi.MakeBarometerSensor(fmt.Sprintf("Barometer@%s", event.Location), event.Location, "hPa", event.HPa, event.Ts))
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
				spaceapidata.MergeInSensor(spaceapi.MakeLasercutterHotSensor("LasercutterHot", "M500", event.IsHot))
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

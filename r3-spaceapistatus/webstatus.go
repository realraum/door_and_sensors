// (c) Bernhard Tittelbach, 2013

package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"./spaceapi"
	//r3events "github.com/realraum/door_and_sensors/r3events"
	r3events "../r3events"
)

type spaceState struct {
	present           bool
	buttonpress_until int64
}

var (
	spaceapidata    spaceapi.SpaceInfo = spaceapi.NewSpaceInfo("realraum", "http://realraum.at", "http://realraum.at/logo-red_250x250.png", "http://realraum.at/logo-re_open_100x100.png", "http://realraum.at/logo-re_empty_100x100.png", 47.065554, 15.450435).AddSpaceAddress("Brockmanngasse 15, 8010 Graz, Austria")
	statusstate     *spaceState        = new(spaceState)
	re_querystresc_ *regexp.Regexp     = regexp.MustCompile("[^\x30-\x39\x41-\x7E]")
)

func init() {
	spaceapidata.AddSpaceFeed("calendar", "http://grical.realraum.at/s/?query=!realraum&view=rss")
	spaceapidata.AddSpaceFeed("blog", "https://wp.realraum.at/feed/")
	spaceapidata.AddSpaceFeed("wiki", "http://realraum.at/wiki")
	spaceapidata.SetSpaceContactIRC("irc://irc.oftc.net/#realraum", false).SetSpaceContactMailinglist("realraum@realraum.at", false).SetSpaceContactEmail("vorstand@realraum.at", true)
	spaceapidata.SetSpaceContactIssueMail("vorstand@realraum.at", true).SetSpaceContactTwitter("@realraum", false)
	spaceapidata.SetSpaceContactGooglePlus("https://plus.google.com/+RealraumAt/").SetSpaceContactJabber("realraum@realraum.at", false)
	spaceapidata.AddProjectsURLs([]string{"https://git.github.com/realraum", "http://wiki.realraum.at/wiki/doku.php?id=projekte", "https://wp.realraum.at/", "https://synbiota.com/projects/openbiolabgraz/summary", "https://plus.google.com/+RealraumAt", "http://git.realraum.at"})
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
	spaceapidata.SetStatus(statusstate.present, spacestatus)
}

func publishStateToWeb() {
	updateStatusString()
	jsondata_b, err := spaceapidata.MakeJSON()
	if err != nil {
		Syslog_.Println("Error:", err)
		return
	}

	session := getWebStatusSSHSession()
	if session == nil {
		return
	}
	defer session.Close()

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
}

func EventToWeb(events chan interface{}) {
	for eventinterface := range events {
		//Debug_.Printf("EventToWeb: %s" , eventinterface)
		switch event := eventinterface.(type) {
		case r3events.TimeTick:
			publishStateToWeb()
		case r3events.PresenceUpdate:
			statusstate.present = event.Present
			publishStateToWeb()
		case r3events.BackdoorAjarUpdate:
			spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("HintertorwaechterAjarSensor", "Hintertürkontakt", event.Shut))
			publishStateToWeb()
		case r3events.DoorAjarUpdate:
			spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("TorwaechterAjarSensor", "Türkontakt", event.Shut))
			publishStateToWeb()
		case r3events.DoorLockUpdate:
			spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("TorwaechterLock", "Türschloß", event.Locked))
			publishStateToWeb()
		case r3events.BoreDoomButtonPressEvent:
			statusstate.buttonpress_until = event.Ts + 3600
			spaceapidata.AddSpaceEvent("BoreDOOMButton", "check-in", "The button has been pressed")
			publishStateToWeb()
		case r3events.TempSensorUpdate:
			spaceapidata.MergeInSensor(spaceapi.MakeTempCSensor(fmt.Sprintf("Temp%s", event.Location), event.Location, event.Value))
		case r3events.IlluminationSensorUpdate:
			spaceapidata.MergeInSensor(spaceapi.MakeIlluminationSensor("Photoresistor", event.Location, "/2^10", event.Value))
		case r3events.RelativeHumiditySensorUpdate:
			spaceapidata.MergeInSensor(spaceapi.MakeHumiditySensor("DHT11Humidity", event.Location, "%", event.Percent))
		case r3events.GasLeakAlert:
			spaceapidata.AddSpaceEvent("GasLeak", "alert", "GasLeak Alert has been triggered")
			publishStateToWeb()
		case r3events.TempOverThreshold:
			spaceapidata.AddSpaceEvent("TemperatureLimitExceeded", "alert", fmt.Sprintf("Temperature %s has exceeded limit at %f °C", event.Location, event.Value))
			publishStateToWeb()
		case r3events.UPSPowerLoss:
			spaceapidata.AddSpaceEvent("PowerLoss", "alert", fmt.Sprintf("UPS reports power loss. Battery at %d%%.", event.PercentBattery))
			publishStateToWeb()
		case r3events.LaserCutter:
			spaceapidata.MergeInSensor(spaceapi.MakeLasercutterHotSensor("LasercutterHot", "M500", event.IsHot))
			publishStateToWeb()
		}
	}
}

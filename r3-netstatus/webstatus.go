// (c) Bernhard Tittelbach, 2013

package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"time"

	"golang.org/x/crypto/ssh"

	"./spaceapi"
	r3events "github.com/realraum/door_and_sensors/r3events"
	pubsub "github.com/tuxychandru/pubsub"
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
	spaceapidata.AddSpaceFeed("blog", "https://plus.google.com/113737596421797426873")
	spaceapidata.AddSpaceFeed("wiki", "http://realraum.at/wiki")
	spaceapidata.AddSpaceContactInfo("+43780700888524", "irc://irc.oftc.net/#realraum", "realraum@realraum.at", "realraum@realraum.at", "realraum@realraum.at", "vorstand@realraum.at")
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

	privateBytes, err := ioutil.ReadFile(environOrDefault("SSH_ID_FILE", "/flash/tuer/id_rsa"))
	if err != nil {
		Syslog_.Println("Error: Failed to load ssh private key:", err.Error())
		return
	}
	signer, err := ssh.ParsePrivateKey([]byte(privateBytes))
	if err != nil {
		Syslog_.Println("Error: Failed to parse ssh private key:", err.Error())
		return
	}

	config := &ssh.ClientConfig{
		User: environOrDefault("SSH_USER", "www-data"),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}
	client, err := ssh.Dial("tcp", environOrDefault("SSH_HOST_PORT", "vex.realraum.at:2342"), config)
	if err != nil {
		Syslog_.Println("Error: Failed to connect to ssh host:", err.Error())
		return
	}
	defer client.Conn.Close()

	session, err := client.NewSession()
	if err != nil {
		Syslog_.Println("Error: Failed to create ssh session:", err.Error())
		return
	}
	defer session.Close()

	stdinp, err := session.StdinPipe()
	if err != nil {
		Syslog_.Println("Error: Failed to publish status info", err.Error())
		return
	}
	defer stdinp.Close()

	if err := session.Run("set"); err != nil {
		Syslog_.Println("Error: Failed to run ssh command:", err.Error())
		return
	}
	stdinp.Write(jsondata_b)
}

func EventToWeb(ps *pubsub.PubSub) {
	events := ps.Sub("presence", "door", "sensors", "buttons", "updateinterval")

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
			var tempsensorlocation string
			switch event.Sensorindex {
			case 0:
				tempsensorlocation = "LoTHR"
			case 1:
				tempsensorlocation = "CX"
			default:
				tempsensorlocation = "Sonstwo"
			}
			spaceapidata.MergeInSensor(spaceapi.MakeTempCSensor(fmt.Sprintf("Temp%d", event.Sensorindex), tempsensorlocation, event.Value))
		case r3events.IlluminationSensorUpdate:
			spaceapidata.MergeInSensor(spaceapi.MakeIlluminationSensor("Photodiode", "LoTHR", "1024V/5V", event.Value))
		case r3events.GasLeakAlert:
			spaceapidata.AddSpaceEvent("GasLeak", "alert", "GasLeak Alert has been triggered")
			publishStateToWeb()
		}
	}
}

// (c) Bernhard Tittelbach, 2013

package main

import (
    pubsub "github.com/tuxychandru/pubsub"
    "./spaceapi"
    "regexp"
	"net/http"
	"net/url"
    "log"
    "time"
    r3events "svn.spreadspace.org/realraum/go.svn/r3-eventbroker_zmq/r3events"
    )


type spaceState struct {
	present           bool
	buttonpress_until int64
}

var (
	spaceapidata    spaceapi.SpaceInfo = spaceapi.NewSpaceInfo("realraum", "http://realraum.at", "http://realraum.at/logo-red_250x250.png", "http://realraum.at/logo-re_open_100x100.png", "http://realraum.at/logo-re_empty_100x100.png",47.065554, 15.450435).AddSpaceAddress("Brockmanngasse 15, 8010 Graz, Austria")
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
		log.Println("Error:", err)
		return
	}
	//jsondata_b := re_querystresc_.ReplaceAllFunc(jsondata_b, func(in []byte) []byte {
	//	out := make([]byte, 4)
	//	out[0] = '%'
	//	copy(out[1:], []byte(strconv.FormatInt(int64(in[0]), 16)))
	//	return out
	//})
	jsondata := url.QueryEscape(string(jsondata_b))
	resp, err := http.Get("http://www.realraum.at/cgi/status.cgi?pass=jako16&set=" + jsondata)
	if err != nil {
		log.Println("Error publishing realraum info", err)
		return
	}
	resp.Body.Close()
}

func EventToWeb(ps *pubsub.PubSub) {
    events := ps.Sub("presence","door","sensors","buttons","updateinterval")

    for eventinterface := range(events) {
        //log.Printf("EventToWeb: %s" , eventinterface)
        switch event := eventinterface.(type) {
            case r3events.TimeTick:
                publishStateToWeb()
            case r3events.PresenceUpdate:
                statusstate.present = event.Present
                publishStateToWeb()
            case r3events.DoorAjarUpdate:
                spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("TorwaechterAjarSensor", "Türkontakt", event.Shut))
                publishStateToWeb()
            case r3events.DoorLockUpdate:
                spaceapidata.MergeInSensor(spaceapi.MakeDoorLockSensor("TorwaechterLock", "Türschloß", event.Locked))
                publishStateToWeb()
            case r3events.ButtonPressUpdate:
                statusstate.buttonpress_until = event.Ts + 3600
                spaceapidata.AddSpaceEvent("PanicButton", "check-in", "The button has been pressed")
                publishStateToWeb()
            case r3events.TempSensorUpdate:
                spaceapidata.MergeInSensor(spaceapi.MakeTempCSensor("Temp0","Decke", event.Value))
            case r3events.IlluminationSensorUpdate:
                spaceapidata.MergeInSensor(spaceapi.MakeIlluminationSensor("Photodiode","Decke","1024V/5V", event.Value))
        }
	}
}


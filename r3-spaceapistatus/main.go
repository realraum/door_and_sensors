// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	//r3events "github.com/realraum/door_and_sensors/r3events"
	r3events "../r3events"
)

type SpaceState struct {
	present           bool
	buttonpress_until int64
	door_locked       bool
	door_shut         bool
}

const (
	DEFAULT_TUER_STATUSPUSH_SSH_ID_FILE   string = "/home/realraum/.ssh/id_rsa"
	DEFAULT_TUER_STATUSPUSH_SSH_USER      string = "www-data"
	DEFAULT_TUER_STATUSPUSH_SSH_HOST_PORT string = "vex.realraum.at:2342"
	DEFAULT_R3_MQTT_BROKER                string = "tcp://mqtt.realraum.at:1883"
)

var (
	enable_syslog_ bool
	enable_debug_  bool
)

func init() {
	flag.BoolVar(&enable_syslog_, "syslog", false, "enable logging to syslog")
	flag.BoolVar(&enable_debug_, "debug", false, "enable debug output")
	flag.Parse()
}

//-------

func main() {
	if enable_syslog_ {
		LogEnableSyslog()
	}
	if enable_debug_ {
		LogEnableDebuglog()
	}
	Syslog_.Print("started")
	defer Syslog_.Print("exiting")

	mqttc := ConnectMQTTBroker(EnvironOrDefault("R3_MQTT_BROKER", DEFAULT_R3_MQTT_BROKER), "r3-spaceapistatus")
	if mqttc != nil {
		defer mqttc.Disconnect(20)
	} else {
		//MQTT can't be reached....
		//upload a fallback Status JSON
		//then exit
		publishStateNotKnown()
		return
	}

	events_to_status_chan := make(chan interface{}, 80)
	go EventToWeb(events_to_status_chan)

	// --- receive and distribute events ---
	ticker := time.NewTicker(time.Duration(6) * time.Minute)
	incoming_message_chan := SubscribeMultipleAndForwardToChannel(mqttc, []string{
		r3events.TOPIC_META_PRESENCE,
		"realraum/+/" + r3events.TYPE_TEMP,
		r3events.TOPIC_XBEE_TEMP,
		r3events.TOPIC_XBEE_VOLTAGE,
		"realraum/+/" + r3events.TYPE_ILLUMINATION,
		"realraum/+/" + r3events.TYPE_RELHUMIDITY,
		r3events.TOPIC_XBEE_RELHUMIDITY,
		"realraum/+/" + r3events.TYPE_LOCK,
		"realraum/+/" + r3events.TYPE_AJAR,
		"realraum/+/" + r3events.TYPE_TEMPOVER,
		"realraum/+/" + r3events.TYPE_DOOMBUTTON,
		"realraum/+/" + r3events.TYPE_GASALERT,
		"realraum/+/" + r3events.TYPE_POWERLOSS,
		r3events.TOPIC_LASER_CARD,
		r3events.TOPIC_IRCBOT_FOODETA})
	for {
		if len(events_to_status_chan) > 72 {
			Syslog_.Println("events_to_status_chan is nearly full, dropping events, cleaning it out")
		CLEANFOR:
			for {
				select {
				case <-events_to_status_chan:
				default:
					break CLEANFOR
				}
			}
		}
		select {
		case msg := <-incoming_message_chan:
			evnt, err := r3events.UnmarshalTopicByte2Event(msg.(mqtt.Message).Topic(), msg.(mqtt.Message).Payload())
			if err == nil {
				events_to_status_chan <- evnt
			} else {
				Syslog_.Printf("Error Unmarshalling Event", err)
				Syslog_.Printf(msg.(mqtt.Message).Topic(), msg.(mqtt.Message).Payload())
			}

		case ts := <-ticker.C:
			events_to_status_chan <- r3events.TimeTick{ts.Unix()}
		}
	}
}

// (c) Bernhard Tittelbach, 2013

package main

import (
	"encoding/json"
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
	DEFAULT_SPACEAPI_HTTP_INTERFACE       string = ":8080"
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

func TranslateSonOffSensor(msg mqtt.Message, events_to_status_chan chan<- *r3events.R3MQTTMsg) bool {
	ts := time.Now().Unix()
	switch msg.Topic() {
	case r3events.TOPIC_COUCHRED_SENSOR:
		var newevent r3events.SonOffSensor
		err := json.Unmarshal(msg.Payload(), &newevent)
		if err == nil {
			events_to_status_chan <- &r3events.R3MQTTMsg{Event: r3events.BarometerUpdate{Location: r3events.CLIENTID_COUCHRED, HPa: newevent.BMP280.Pressure, Ts: ts}, Msg: msg, Topic: r3events.SplitTopic(msg.Topic())}
			events_to_status_chan <- &r3events.R3MQTTMsg{Event: r3events.TempSensorUpdate{Location: r3events.CLIENTID_COUCHRED, Value: newevent.BMP280.Temperature, Ts: ts}, Msg: msg, Topic: r3events.SplitTopic(msg.Topic())}
		}
	default:
		return false
	}
	return true
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

	events_to_status_chan := make(chan *r3events.R3MQTTMsg, 80)
	go EventToWeb(events_to_status_chan)
	go goRunWebserver()

	// --- receive and distribute events ---
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
		"realraum/+/" + r3events.TYPE_BAROMETER,
		r3events.TOPIC_COUCHRED_SENSOR,
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
		for msg := range incoming_message_chan {
			if TranslateSonOffSensor(msg, events_to_status_chan) {
				continue
			}
			evnt, err := r3events.R3ifyMQTTMsg(msg.(mqtt.Message))
			if err == nil {
				events_to_status_chan <- evnt
			} else {
				Syslog_.Printf("Error Unmarshalling Event", err)
				Syslog_.Printf(msg.(mqtt.Message).Topic(), msg.(mqtt.Message).Payload())
			}
		}
	}
}

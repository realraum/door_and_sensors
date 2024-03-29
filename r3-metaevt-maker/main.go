// (c) Bernhard Tittelbach, 2013-2021

package main

import (
	"flag"
	"time"
	//~ "time"
	// r3events "../r3events"
	pubsub "github.com/btittelbach/pubsub"
	r3events "github.com/realraum/door_and_sensors/r3events"
)

//~ func StringArrayToByteArray(ss []string) [][]byte {
//~ bb := make([][]byte, len(ss))
//~ for index, s := range(ss) {
//~ bb[index] = []byte(s)
//~ }
//~ return bb
//~ }

// ---------- Main Code -------------

var (
	use_syslog_      bool
	enable_debuglog_ bool
)

//-------
// available Config Environment Variables
// TUER_ZMQDOORCMD_ADDR
// TUER_ZMQDOOREVTS_ADDR
// TUER_R3EVENTS_ZMQBROKERINPUT_ADDR
// TUER_R3EVENTS_ZMQBROKERINPUT_LISTEN_ADDR
// TUER_R3EVENTS_ZMQBROKER_LISTEN_ADDR
// TUER_R3EVENTS_ZMQBRAIN_LISTEN_ADDR
// TUER_ZMQKEYNAMELOOKUP_ADDR

const (
	DEFAULT_R3_MQTT_BROKER string = "tcp://mqtt.realraum.at:1883"
	PS_R3EVENTS            string = "r3events"
)

func init() {
	flag.BoolVar(&use_syslog_, "syslog", false, "log to syslog local2 facility")
	flag.BoolVar(&enable_debuglog_, "debug", false, "enable debug logging")
	flag.Parse()
}

var topics_monitor_if_sensors_disappear = []string{
	r3events.TOPIC_PILLAR_RELHUMIDITY,
	r3events.TOPIC_PILLAR_TEMP,
	r3events.TOPIC_SMALLKIOSK_TEMP,
	//r3events.TOPIC_OLGAFREEZER_TEMP, //already sent by olga_freezer_sensordata_forwarder.py
	r3events.TOPIC_BACKDOOR_TEMP,
	r3events.TOPIC_GW_STATS,
	r3events.TOPIC_XBEE_TEMP,
	r3events.TOPIC_XBEE_RELHUMIDITY,
}

var topics_needed_for_presenceevent = []string{
	r3events.TOPIC_META_REALMOVE,
	"realraum/+/movement",
	"realraum/" + r3events.CLIENTID_FRONTDOOR + "/+",
	"realraum/+/lock",
	"realraum/+/ajar",
	"realraum/+/boredoombuttonpressed",
	r3events.ZB_AJARWINDOW_MASHA,
	r3events.ZB_AJARWINDOW_R2W2left,
	r3events.ZB_AJARWINDOW_R2W2right,
	r3events.ZB_AJARWINDOW_TESLA,
	r3events.ZB_AJARWINDOW_Kitchen,
	r3events.ZB_AJARWINDOW_OLGA,
}

func main() {
	if enable_debuglog_ {
		LogEnableDebuglog()
	}
	if use_syslog_ {
		LogEnableSyslog()
		Syslog_.Print("started")
		defer Syslog_.Print("exiting")
	}

	mqttc := ConnectMQTTBroker(EnvironOrDefault("R3_MQTT_BROKER", DEFAULT_R3_MQTT_BROKER), r3events.CLIENTID_META)

	ps := pubsub.New[any](100)
	defer ps.Shutdown() // ps.Shutdown should be called before zmq_ctx.Close(), since it will cause goroutines to shutdown and close zqm_sockets which is needed for zmq_ctx.Close() to return

	go MetaEventRoutine_Movement(ps, mqttc, 10, 20, 10)
	go MetaEventRoutine_MovementSum(ps, mqttc, time.Second*30)
	go MetaEventRoutine_Presence(ps, mqttc, 21, 200, 40)
	go MetaEventRoutine_SensorLost(ps, mqttc, topics_monitor_if_sensors_disappear)
	go MetaEventRoutine_DuskDawnEventGenerator(mqttc)
	go MetaEventRoutine_TimeToClean(mqttc)

	mqtt_subscription_filters := make([]string, len(topics_needed_for_presenceevent)+len(topics_monitor_if_sensors_disappear))
	copy(mqtt_subscription_filters[0:len(topics_needed_for_presenceevent)], topics_needed_for_presenceevent)
	copy(mqtt_subscription_filters[len(topics_needed_for_presenceevent):cap(mqtt_subscription_filters)], topics_monitor_if_sensors_disappear)
	Debug_.Println(mqtt_subscription_filters)
	got_events_chan := SubscribeMultipleAndForwardToChannel(mqttc, mqtt_subscription_filters)
	for msg := range got_events_chan {
		evnt, err := r3events.R3ifyMQTTMsg(msg)
		if err == nil {
			ps.Pub(evnt, PS_R3EVENTS)
			ps.Pub(msg.Topic(), "seentopics")
		}
	}
}

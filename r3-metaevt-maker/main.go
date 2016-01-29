// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	//~ "time"
	r3events "../r3events"
	//r3events "github.com/realraum/door_and_sensors/r3events"
	pubsub "github.com/btittelbach/pubsub"
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
)

func init() {
	flag.BoolVar(&use_syslog_, "syslog", false, "log to syslog local2 facility")
	flag.BoolVar(&enable_debuglog_, "debug", false, "enable debug logging")
	flag.Parse()
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

	ps := pubsub.New(100)
	defer ps.Shutdown() // ps.Shutdown should be called before zmq_ctx.Close(), since it will cause goroutines to shutdown and close zqm_sockets which is needed for zmq_ctx.Close() to return

	go MetaEventRoutine_Movement(ps, mqttc, 10, 20, 10)
	go MetaEventRoutine_Presence(ps, mqttc, 21, 200)
	go MetaEventRoutine_SensorLost(ps, mqttc, []string{
		r3events.TOPIC_PILLAR_RELHUMIDITY,
		r3events.TOPIC_PILLAR_TEMP,
		r3events.TOPIC_OLGAFREEZER_TEMP,
		r3events.TOPIC_BACKDOOR_TEMP,
		r3events.TOPIC_GW_STATS})

	mqtt_subscription_filters := []string{
		r3events.TOPIC_META_REALMOVE,
		"realraum/+/movement",
		"realraum/frontdoor/+",
		"realraum/+/ajar",
		"realraum/+/boredoombuttonpressed",
	}

	got_events_chan := SubscribeMultipleAndForwardToChannel(mqttc, mqtt_subscription_filters)
	for msg := range got_events_chan {
		evnt, err := r3events.UnmarshalTopicByte2Event(msg.Topic(), msg.Payload())
		if err == nil {
			ps.Pub(evnt, "r3events")
			ps.Pub(msg.Topic(), "seentopics")
		}
	}
}

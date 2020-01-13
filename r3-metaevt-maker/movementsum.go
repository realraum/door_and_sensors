// (c) Bernhard Tittelbach, 2019

package main

import (
	"strings"
	"time"

	// r3events "../r3events"
	pubsub "github.com/btittelbach/pubsub"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	r3events "github.com/realraum/door_and_sensors/r3events"
)

func MetaEventRoutine_MovementSum(ps *pubsub.PubSub, mqttc mqtt.Client, interval time.Duration) {
	events_chan := ps.Sub(PS_R3EVENTS)
	defer ps.Unsub(events_chan, PS_R3EVENTS)
	myticker := time.NewTicker(interval)
	topic_ctr_map := make(map[string]*r3events.MovementSum, 4)

	for {
		select {
		case r3mqttmsg := <-events_chan:
			switch event := r3mqttmsg.(*r3events.R3MQTTMsg).Event.(type) {
			case r3events.MovementSensorUpdate:
				topic := strings.Join(r3mqttmsg.(*r3events.R3MQTTMsg).Topic[0:len(r3mqttmsg.(*r3events.R3MQTTMsg).Topic)-1], "/") //cut away last "/movement"
				msum, inmap := topic_ctr_map[topic]
				if !inmap {
					msum = &r3events.MovementSum{Sensorindex: event.Sensorindex, NumEvents: 0, IntervalSeconds: int(interval.Seconds()), Ts: 0}
					topic_ctr_map[topic] = msum
				}
				msum.NumEvents++
			default:
			}
		case gots := <-myticker.C:
			ts := gots.Unix()
			for origtopic, mctr := range topic_ctr_map {
				mctr.Ts = ts
				//publish to almost original topic
				//if original was "realraum/abcd/movement" then publish to "realraum/abcd/movementsum"
				mqttc.Publish(origtopic+"/"+r3events.TYPE_MOVEMENTSUM, MQTT_QOS_NOCONFIRMATION, false, r3events.MarshalEvent2ByteOrPanic(*mctr))
				mctr.NumEvents = 0
				mctr.Ts = 0
			}
		}
	}
}

// (c) Bernhard Tittelbach, 2013

package main

import (
	"container/ring"
	"time"

	r3events "../r3events"
	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	//r3events "github.com/realraum/door_and_sensors/r3events"
	pubsub "github.com/tuxychandru/pubsub"
)

/// Movement Meta Event Injector:
///     threshold number of movements within gran_duration*granularity seconds -> SomethingReallyIsMoving{True}
///     No movement within 3 hours but movement within the last 6 hours -> SomethingReallyIsMoving{False}
///
/// Thus SomethingReallyIsMoving{True} fires regularly, but at most every gran_duration seconds
/// While SomethingReallyIsMoving{False} fires only once to assure us that everybody might really be gone

func MetaEventRoutine_Movement(ps *pubsub.PubSub, mqttc *mqtt.Client, granularity, gran_duration int, threshold uint32) {
	var last_movement, last_movement1, last_movement2, last_movement3 int64
	var confidence uint8
	movement_window := ring.New(granularity + 1)
	events_chan := ps.Sub("r3events")
	defer ps.Unsub(events_chan, "r3events")
	myticker := time.NewTicker(time.Duration(gran_duration) * time.Second)

	for {
		select {
		case event := <-events_chan:
			switch event.(type) {
			case r3events.MovementSensorUpdate:
				if movement_window.Value == nil {
					movement_window.Value = uint32(1)
				} else {
					movement_window.Value = uint32(movement_window.Value.(uint32) + 1)
				}
			default:
			}
		case gots := <-myticker.C:
			ts := gots.Unix()
			movement_window.Prev().Value = (uint32)(0)
			movement_window = movement_window.Next()
			var movsum uint32 = 0
			movement_window.Do(func(v interface{}) {
				if v != nil {
					movsum += v.(uint32)
				}
			})
			if movsum > threshold {
				confidence = uint8(movsum)
				last_movement = ts
				last_movement1 = ts
				last_movement2 = ts
				last_movement3 = ts
				mqttc.Publish(r3events.TOPIC_META_REALMOVE, MQTT_QOS_NOCONFIRMATION, false, r3events.MarshalEvent2ByteOrPanic(r3events.SomethingReallyIsMoving{true, confidence, ts}))
			}
			// this sucks.....
			if last_movement > 0 && ts-last_movement < 3600*6 {
				if ts-last_movement > 3600*3 {
					last_movement = 0
					mqttc.Publish(r3events.TOPIC_META_REALMOVE, MQTT_QOS_NOCONFIRMATION, false, r3events.MarshalEvent2ByteOrPanic(r3events.SomethingReallyIsMoving{true, 99, ts}))
				} else if ts-last_movement > 3600 && last_movement3 > 0 {
					last_movement3 = 0
					mqttc.Publish(r3events.TOPIC_META_REALMOVE, MQTT_QOS_NOCONFIRMATION, false, r3events.MarshalEvent2ByteOrPanic(r3events.SomethingReallyIsMoving{true, 50, ts}))
				} else if ts-last_movement > 1800 && last_movement2 > 0 {
					last_movement2 = 0
					mqttc.Publish(r3events.TOPIC_META_REALMOVE, MQTT_QOS_NOCONFIRMATION, false, r3events.MarshalEvent2ByteOrPanic(r3events.SomethingReallyIsMoving{true, 20, ts}))
				} else if ts-last_movement > 120 && last_movement1 > 0 {
					last_movement1 = 0
					mqttc.Publish(r3events.TOPIC_META_REALMOVE, MQTT_QOS_NOCONFIRMATION, false, r3events.MarshalEvent2ByteOrPanic(r3events.SomethingReallyIsMoving{true, 5, ts}))
				}
			}
		}
	}
}

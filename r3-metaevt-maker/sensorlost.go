// (c) Bernhard Tittelbach, 2013

package main

import (
	"time"
	//~ "./brain"
	r3events "../r3events"
	//r3events "github.com/realraum/door_and_sensors/r3events"
	pubsub "github.com/btittelbach/pubsub"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type TopicSeen struct {
	last_seen    int64
	num_seen     int64
	avg_interval float64
	varianz      float64
	max_interval int64
}

func MetaEventRoutine_SensorLost(ps *pubsub.PubSub, mqttc mqtt.Client, topics_to_monitor []string) {
	topicData := make(map[string]TopicSeen, 10)

	events_chan := ps.Sub("seentopics")
	defer ps.Unsub(events_chan, "seentopics")

	ticker := time.NewTicker(30 * time.Second)

	for _, topic := range topics_to_monitor {
		topicData[topic] = TopicSeen{0, 0, 0, 0, -1}
	}

	for {
		select {
		case ts := <-ticker.C:
			unixts := ts.Unix()
			for topic, topicseen := range topicData {
				if topicseen.num_seen < 10 {
					continue
				}
				if unixts-topicseen.last_seen > 10*topicseen.max_interval {
					topicseen.num_seen = 0
					Syslog_.Printf("Sensor Lost: %s, %s", topic, topicseen)
					mqttc.Publish(r3events.TOPIC_META_SENSORLOST, MQTT_QOS_4STPHANDSHAKE, false, r3events.MarshalEvent2ByteOrPanic(r3events.SensorLost{topic, topicseen.last_seen, int64(topicseen.avg_interval), unixts}))
				}
			}
		case topic_i := <-events_chan:
			topic := topic_i.(string)
			if tdata, inmap := topicData[topic]; inmap {
				now := time.Now().Unix()
				interv := now - tdata.last_seen
				tdata.last_seen = now
				if tdata.num_seen == 0 {
					tdata.avg_interval = float64(interv)
				} else if tdata.num_seen == 1 {
					tdata.avg_interval += float64(interv)
					tdata.avg_interval /= 2
					tdata.max_interval = interv
				} else if tdata.num_seen == 2 {
					tdata.varianz = (tdata.avg_interval - float64(interv)) * (tdata.avg_interval - float64(interv))
					tdata.avg_interval = 0.1*float64(interv) + 0.9*tdata.avg_interval
					if interv > tdata.max_interval {
						tdata.max_interval = interv
					}
				} else {
					tdata.varianz = 0.9*tdata.varianz + 0.1*((tdata.avg_interval-float64(interv))*(tdata.avg_interval-float64(interv)))
					tdata.avg_interval = 0.1*float64(interv) + 0.9*tdata.avg_interval
					if interv > tdata.max_interval && interv < 10*tdata.max_interval {
						tdata.max_interval = interv
					}
				}
				tdata.num_seen++
			}
		}
	}
}

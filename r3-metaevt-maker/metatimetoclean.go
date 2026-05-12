// (c) Bernhard Tittelbach, 2019

package main

import (
	"time"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	r3events "github.com/realraum/door_and_sensors/r3events"
)


func MetaEventRoutine_TimeToClean(mqttc mqtt.Client) {
	for {
		the_hour_to_clean := 20

	    now := time.Now()
	    yyyy, mm, dd := now.Date()

	    if now.Hour() < the_hour_to_clean {
	    	dd = dd
	    } else {
	    	dd = dd +1
	    }

	    new20h := time.Date(yyyy, mm, dd, the_hour_to_clean, 0, 0, 0, now.Location())

		time.Sleep(time.Until(new20h))

		mqttc.Publish(r3events.TOPIC_META_TIMETOCLEAN, MQTT_QOS_4STPHANDSHAKE, false, r3events.TimeToClean{})
		time.Sleep(1 * time.Minute)

	}
}

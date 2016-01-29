// (c) Bernhard Tittelbach, 2015

package main

import (
	"time"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	//r3events "github.com/realraum/door_and_sensors/r3events"
	pubsub "github.com/btittelbach/pubsub"
	"github.com/cpucycle/astrotime"
)

const LATITUDE = float64(38.8895)
const LONGITUDE = float64(77.0352)

func MetaEventRoutine_DuskDawnEventGenerator(ps *pubsub.PubSub, mqttc *mqtt.Client) {
	for {
		now := time.Now()
		tsunrise := astrotime.NextSunrise(now, LATITUDE, LONGITUDE)
		tsunset := astrotime.NextSunset(now, LATITUDE, LONGITUDE)
		event := "Sunrise"
		nexttrigger := tsunrise
		if tsunset.Before(tsunrise) {
			nexttrigger = tsunset
			event = "Sunset"
		}
		<-time.NewTimer(nexttrigger.Sub(now)).C
		mqttc.Publish(r3events.TOPIC_META_DUSKORDAWN, MQTT_QOS_4STPHANDSHAKE, false, r3events.MarshalEvent2ByteOrPanic(r3events.DuskOrDawn{Event: event, CivilSunlight: event == "Sunrise", ts: now.Unix()}))
	}
}

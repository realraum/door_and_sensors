// (c) Bernhard Tittelbach, 2015
package main

import (
	"log"
	"time"

	"github.com/realraum/door_and_sensors/r3events"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
)

const DEFAULT_R3_MQTT_BROKER string = "tcp://mqtt.realraum.at:1883"

func main() {
	options := mqtt.NewClientOptions().AddBroker(DEFAULT_R3_MQTT_BROKER).SetAutoReconnect(true).SetProtocolVersion(4).SetCleanSession(true)
	c := mqtt.NewClient(options)
	ctk := c.Connect()
	ctk.Wait()
	log.Print("connect:", ctk.Error())
	payload, err := r3events.MarshalEvent2Byte(r3events.PresenceUpdate{Present: true, Ts: time.Now().Unix()})
	if err == nil {
		tk := c.Publish("realraum/metaevt/presence", 1, false, payload)
		tk.Wait()
		log.Print("error:", tk.Error())

	} else {
		log.Print("marshall error:", err)
	}
	c.Disconnect(1000)
}

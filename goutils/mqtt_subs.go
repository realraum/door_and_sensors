// (c) Bernhard Tittelbach, 2015
package main

import (
	"regexp"
	"time"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/btittelbach/pubsub"
)

const MQTT_QOS_NOCONFIRMATION byte = 0
const MQTT_QOS_REQCONFIRMATION byte = 1
const MQTT_QOS_4STPHANDSHAKE byte = 2

var re_cardid_ *regexp.Regexp = regexp.MustCompile("card\\(([a-fA-F0-9]+)\\)")

func ConnectMQTTBroker(brocker_addr, clientid string) *mqtt.Client {
	options := mqtt.NewClientOptions().AddBroker(brocker_addr).SetAutoReconnect(true).SetKeepAlive(30 * time.Second).SetMaxReconnectInterval(2 * time.Minute)
	options = options.SetClientID(clientid).SetConnectionLostHandler(func(c *mqtt.Client, err error) { Syslog_.Print("ERROR MQTT connection lost:", err) })
	c := mqtt.NewClient(options)
	tk := c.Connect()
	tk.Wait()
	if tk.Error() != nil {
		Syslog_.Fatal("Error connecting to mqtt broker", tk.Error())
	}
	return c
}

func SubscribeAndForwardToChannel(mqttc *mqtt.Client, filter string) (channel chan mqtt.Message) {
	channel = make(chan mqtt.Message, 100)
	tk := mqttc.Subscribe(filter, 0, func(mqttc *mqtt.Client, msg mqtt.Message) { channel <- msg })
	tk.Wait()
	if tk.Error() != nil {
		Syslog_.Fatalf("Error subscribing to %s:%s", filter, tk.Error())
	}
	return
}

func SubscribeMultipleAndForwardToChannel(mqttc *mqtt.Client, filters []string) (channel chan mqtt.Message) {
	channel = make(chan mqtt.Message, 100)
	filtermap := make(map[string]byte, len(filters))
	for _, topicfilter := range filters {
		filtermap[topicfilter] = 0 //qos == 0
	}
	tk := mqttc.SubscribeMultiple(filtermap, func(mqttc *mqtt.Client, msg mqtt.Message) {
		Debug_.Println("forwarding mqtt message to channel", msg)
		channel <- msg
	})
	tk.Wait()
	if tk.Error() != nil {
		Syslog_.Fatalf("Error subscribing to %s:%s", filters, tk.Error())
	}
	return
}

func SubscribeAndPublishToPubSub(mqttc *mqtt.Client, ps *pubsub.PubSub, filter string, pstopics ...string) {
	tk := mqttc.Subscribe(filter, 0, func(mqttc *mqtt.Client, msg mqtt.Message) { ps.Pub(msg, pstopics...) })
	tk.Wait()
	if tk.Error() != nil {
		Syslog_.Fatalf("Error subscribing to %s:%s", filter, tk.Error())
	}
	return
}

func SubscribeMultipleAndPublishToPubSub(mqttc *mqtt.Client, ps *pubsub.PubSub, filters []string, pstopics ...string) {
	filtermap := make(map[string]byte, len(filters))
	for _, topicfilter := range filters {
		filtermap[topicfilter] = 0 //qos == 0
	}
	tk := mqttc.SubscribeMultiple(filtermap, func(mqttc *mqtt.Client, msg mqtt.Message) {
		Debug_.Println("forwarding mqtt message to pubsub", msg)
		ps.Pub(msg, pstopics...)
	})
	tk.Wait()
	if tk.Error() != nil {
		Syslog_.Fatalf("Error subscribing to %s:%s", filters, tk.Error())
	}
	return
}

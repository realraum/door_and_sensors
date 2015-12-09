// (c) Bernhard Tittelbach, 2015
package main

import (
	"regexp"
	"sync"
	"time"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/btittelbach/pubsub"
)

const MQTT_QOS_NOCONFIRMATION byte = 0
const MQTT_QOS_REQCONFIRMATION byte = 1
const MQTT_QOS_4STPHANDSHAKE byte = 2

var re_cardid_ *regexp.Regexp = regexp.MustCompile("card\\(([a-fA-F0-9]+)\\)")

var mqtt_topics_we_subscribed_ map[string]byte
var mqtt_topics_we_subscribed_lock_ sync.RWMutex

func init() {
	mqtt_topics_we_subscribed_ = make(map[string]byte, 1)
}

func addSubscribedTopics(subresult map[string]byte) {
	mqtt_topics_we_subscribed_lock_.Lock()
	defer mqtt_topics_we_subscribed_lock_.Unlock()
	for topic, qos := range subresult {
		if qos < 0 || qos > 2 {
			Syslog_.Printf("addSubscribedTopics: not remembering topic since we didn't subscribe it successfully: %s (qos: %d)", topic, qos)
			continue
		}
		Syslog_.Printf("addSubscribedTopics: remembering subscribed topic: %s (qos: %d)", topic, qos)
		mqtt_topics_we_subscribed_[topic] = qos
	}
}

func removeSubscribedTopic(topic string) {
	mqtt_topics_we_subscribed_lock_.Lock()
	defer mqtt_topics_we_subscribed_lock_.Unlock()
	delete(mqtt_topics_we_subscribed_, topic)
	Syslog_.Printf("removeSubscribedTopics: %s ", topic)
}

func mqttOnConnectionHandler(mqttc *mqtt.Client) {
	Syslog_.Print("MQTT connection to broker established. (re)subscribing topics")
	mqtt_topics_we_subscribed_lock_.RLock()
	defer mqtt_topics_we_subscribed_lock_.RUnlock()
	if len(mqtt_topics_we_subscribed_) > 0 {
		tk := mqttc.SubscribeMultiple(mqtt_topics_we_subscribed_, nil)
		tk.Wait()
		if tk.Error() != nil {
			Syslog_.Fatalf("Error resubscribing on connect", tk.Error())
		}
	}
}

func ConnectMQTTBroker(brocker_addr, clientid string) *mqtt.Client {
	options := mqtt.NewClientOptions().AddBroker(brocker_addr).SetAutoReconnect(true).SetKeepAlive(30 * time.Second).SetMaxReconnectInterval(2 * time.Minute)
	options = options.SetClientID(clientid).SetConnectionLostHandler(func(c *mqtt.Client, err error) { Syslog_.Print("ERROR MQTT connection lost:", err) })
	options = options.SetOnConnectHandler(mqttOnConnectionHandler)
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
	} else {
		Syslog_.Printf("SubscribeAndForwardToChannel successfull")
		addSubscribedTopics(tk.(*mqtt.SubscribeToken).Result())
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
	} else {
		Syslog_.Printf("SubscribeMultipleAndForwardToChannel successfull")
		addSubscribedTopics(tk.(*mqtt.SubscribeToken).Result())
	}
	return
}

func SubscribeAndPublishToPubSub(mqttc *mqtt.Client, ps *pubsub.PubSub, filter string, pstopics ...string) {
	tk := mqttc.Subscribe(filter, 0, func(mqttc *mqtt.Client, msg mqtt.Message) { ps.Pub(msg, pstopics...) })
	tk.Wait()
	if tk.Error() != nil {
		Syslog_.Fatalf("Error subscribing to %s:%s", filter, tk.Error())
	} else {
		Syslog_.Printf("SubscribeAndPublishToPubSub successfull")
		addSubscribedTopics(tk.(*mqtt.SubscribeToken).Result())
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
	} else {
		Syslog_.Printf("SubscribeMultipleAndPublishToPubSub successfull")
		addSubscribedTopics(tk.(*mqtt.SubscribeToken).Result())
	}
	return
}

func UnsubscribeMultiple(mqttc *mqtt.Client, topics ...string) {
	mqttc.Unsubscribe(topics...)
	for _, topic := range topics {
		removeSubscribedTopic(topic)
	}
}

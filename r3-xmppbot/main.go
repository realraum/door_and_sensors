// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	"time"

	"../r3events"
	"./r3xmppbot"
	"github.com/eclipse/paho.mqtt.golang"
)

type SpaceState struct {
	present           bool
	buttonpress_until int64
	door_locked       bool
	door_shut         bool
}

var (
	xmpp_login_ struct {
		jid  string
		pass string
	}
	button_press_timeout_         int64 = 3600
	enable_syslog_, enable_debug_ bool
)

type EventToXMPPStartupFinished struct{}

//-------
// available Config Environment Variables
// TUER_XMPP_STATE_SAVEDIR
// TUER_XMPP_JID
// TUER_XMPP_PASS
// TUER_XMPP_CHATAUTHSTRING
// R3_MQTT_BROKER

const (
	XMPP_PING_TIMEOUT               time.Duration = 1500 * time.Millisecond
	DEFAULT_TUER_XMPP_STATE_SAVEDIR string        = "/var/lib/r3-xmppbot/"
	DEFAULT_TUER_XMPP_JID           string        = "realrauminfo@realraum.at/Tuer"
	DEFAULT_R3_MQTT_BROKER          string        = "tcp://mqtt.realraum.at:1883"
)

func init() {
	flag.BoolVar(&enable_syslog_, "syslog", false, "enable logging to syslog")
	flag.BoolVar(&enable_debug_, "debug", false, "enable debug output")
	flag.Parse()
	xmpp_login_.jid = EnvironOrDefault("TUER_XMPP_JID", DEFAULT_TUER_XMPP_JID)
	xmpp_login_.pass = EnvironOrDefault("TUER_XMPP_PASS", "")
}

//-------

func RunXMPPBotForever(psevents chan *r3events.R3MQTTMsg, incoming_message_chan chan mqtt.Message, mqttc mqtt.Client, mqtt_subscription_topics []string, watchdog_timeout time.Duration) {
	var xmpperr error
	var bot *r3xmppbot.XmppBot
	var xmpp_presence_events_chan chan interface{}
	for {
		bot, xmpp_presence_events_chan, xmpperr = r3xmppbot.NewStartedBot(xmpp_login_.jid, xmpp_login_.pass, EnvironOrDefault("TUER_XMPP_CHATAUTHSTRING", ""), EnvironOrDefault("TUER_XMPP_STATE_SAVEDIR", DEFAULT_TUER_XMPP_STATE_SAVEDIR), true)

		if xmpperr == nil {
			Syslog_.Printf("Successfully (re)started XMPP Bot")
			//this are the converted messages we get and publish
			//this are the raw messages we subscribe to NOW, so that we get the latest persistent messages from the broker
			SubscribeMultipleAndForwardToGivenChannel(mqttc, mqtt_subscription_topics, incoming_message_chan)
			//wait till inital messages are queued in channel, then tell EventToXMPP to go into normal mode
			go func() {
				time.Sleep(300 * time.Millisecond)
				psevents <- &r3events.R3MQTTMsg{Event: EventToXMPPStartupFinished{}}
			}()
			//enter and stay in BotMainRoutine: receive r3Events and send XMPP functions
			EventToXMPP(bot, psevents, xmpp_presence_events_chan, watchdog_timeout)
			// unsubscribe mqtt events
			UnsubscribeMultiple(mqttc, mqtt_subscription_topics...)
			Syslog_.Printf("Stopping XMPP Bot, waiting for 20s")
			bot.StopBot()
		} else {
			Syslog_.Printf("Error starting XMPP Bot: %s", xmpperr.Error())
		}
		time.Sleep(20 * time.Second)
	}
}

func main() {
	if enable_syslog_ {
		LogEnableSyslog()
		r3xmppbot.LogEnableSyslog()
	}
	if enable_debug_ {
		LogEnableDebuglog()
		r3xmppbot.LogEnableDebuglog()
	}
	Syslog_.Print("started")
	defer Syslog_.Print("exiting")

	mqttc := ConnectMQTTBroker(EnvironOrDefault("R3_MQTT_BROKER", DEFAULT_R3_MQTT_BROKER), "r3xmppbot")
	defer mqttc.Disconnect(20)

	mqtt_subscription_filters := []string{
		"realraum/+/" + r3events.TYPE_TEMP,
		r3events.TOPIC_XBEE_TEMP,
		"realraum/+/" + r3events.TYPE_ILLUMINATION,
		"realraum/metaevt/#",
		"realraum/+/" + r3events.TYPE_LOCK,
		"realraum/+/" + r3events.TYPE_AJAR,
		"realraum/+/" + r3events.TYPE_MANUALLOCK,
		r3events.TOPIC_FRONTDOOR_CMDEVT,
		r3events.TOPIC_FRONTDOOR_PROBLEM,
		"realraum/+/" + r3events.TYPE_TEMPOVER,
		"realraum/+/" + r3events.TYPE_DOOMBUTTON,
		"realraum/+/" + r3events.TYPE_GASALERT,
		"realraum/+/" + r3events.TYPE_SENSORLOST,
		"realraum/+/" + r3events.TYPE_POWERLOSS,
		"realraum/" + r3events.CLIENTID_IRCBOT + "/#"}
	incoming_message_chan := make(chan mqtt.Message, 100)
	psevents := make(chan *r3events.R3MQTTMsg, 100)
	go RunXMPPBotForever(psevents, incoming_message_chan, mqttc, mqtt_subscription_filters, time.Duration(7)*time.Minute)

	// --- receive and distribute events ---
	for msg := range incoming_message_chan {
		evnt, err := r3events.R3ifyMQTTMsg(msg)
		if err == nil {
			psevents <- evnt
		} else {
			Syslog_.Printf("Error Unmarshalling Event", err)
			Syslog_.Printf(msg.Topic(), msg.Payload())
		}
	}
}

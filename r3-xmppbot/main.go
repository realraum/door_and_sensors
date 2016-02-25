// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	"time"

	"./r3xmppbot"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	pubsub "github.com/btittelbach/pubsub"
	r3events "github.com/realraum/door_and_sensors/r3events"
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

func RunXMPPBotForever(ps *pubsub.PubSub, mqttc *mqtt.Client, mqtt_subscription_topics []string) {
	var xmpperr error
	var bot *r3xmppbot.XmppBot
	var xmpp_presence_events_chan chan interface{}
	for {
		bot, xmpp_presence_events_chan, xmpperr = r3xmppbot.NewStartedBot(xmpp_login_.jid, xmpp_login_.pass, EnvironOrDefault("TUER_XMPP_CHATAUTHSTRING", ""), EnvironOrDefault("TUER_XMPP_STATE_SAVEDIR", DEFAULT_TUER_XMPP_STATE_SAVEDIR), true)

		if xmpperr == nil {
			Syslog_.Printf("Successfully (re)started XMPP Bot")
			//this are the converted messages we get and publish
			psevents := ps.Sub("r3events", "updateinterval")
			//this are the raw messages we subscribe to NOW, so that we get the latest persistent messages from the broker
			SubscribeMultipleAndPublishToPubSub(mqttc, ps, mqtt_subscription_topics, "mqttrawmessages")
			//wait till inital messages are queued in channel, then tell EventToXMPP to go into normal mode
			go func() {
				time.Sleep(300 * time.Millisecond)
				psevents <- EventToXMPPStartupFinished{}
			}()
			//enter and stay in BotMainRoutine: receive r3Events and send XMPP functions
			EventToXMPP(bot, psevents, xmpp_presence_events_chan)
			// unsubscribe right away, since we don't known when reconnect will succeed and we don't want to block PubSub
			ps.Unsub(psevents, "r3events", "updateinterval")
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

	ps := pubsub.NewNonBlocking(50)
	defer ps.Shutdown()

	mqtt_subscription_filters := []string{
		"realraum/+/" + r3events.TYPE_TEMP,
		"realraum/+/illumination",
		"realraum/metaevt/#",
		"realraum/+/" + r3events.TYPE_LOCK,
		"realraum/+/" + r3events.TYPE_AJAR,
		"realraum/+/" + r3events.TYPE_MANUALLOCK,
		r3events.TOPIC_FRONTDOOR_CMDEVT,
		r3events.TOPIC_FRONTDOOR_PROBLEM,
		"realraum/+/overtemp",
		"realraum/+/" + r3events.TYPE_DOOMBUTTON,
		"realraum/+/gasalert",
		"realraum/+/sensorlost",
		"realraum/+/powerloss",
		"realraum/" + r3events.CLIENTID_IRCBOT + "/#"}
	incoming_message_chan := ps.Sub("mqttrawmessages")
	go RunXMPPBotForever(ps, mqttc, mqtt_subscription_filters)

	// --- receive and distribute events ---
	ticker := time.NewTicker(time.Duration(6) * time.Minute)
	for {
		select {
		case msg := <-incoming_message_chan:
			evnt, err := r3events.UnmarshalTopicByte2Event(msg.(mqtt.Message).Topic(), msg.(mqtt.Message).Payload())
			if err == nil {
				ps.Pub(evnt, "r3events")
			} else {
				Syslog_.Printf("Error Unmarshalling Event", err)
				Syslog_.Printf(msg.(mqtt.Message).Topic(), msg.(mqtt.Message).Payload())
			}
		case <-ticker.C:
			ps.Pub(r3events.TimeTick{time.Now().Unix()}, "updateinterval")
		}
	}
}

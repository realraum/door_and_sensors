// (c) Bernhard Tittelbach, 2015
package main

import (
	"regexp"
	"strings"
	"time"

	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/realraum/door_and_sensors/r3events"
)

const MQTT_QOS_NOCONFIRMATION byte = 0
const MQTT_QOS_REQCONFIRMATION byte = 1
const MQTT_QOS_4STPHANDSHAKE byte = 2

var re_cardid_ *regexp.Regexp = regexp.MustCompile("card\\(([a-fA-F0-9]+)\\)")

func parseSocketInputLine_State(lines []string, mqttc *mqtt.Client, ts int64) {
	switch lines[0] {
	case "reset", "closed":
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_LOCK, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorLockUpdate{true, ts}))
	case "opened":
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_LOCK, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorLockUpdate{false, ts}))
	case "manual", "manual_movement": //movement
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_MANUALLOCK, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorManualMovementEvent{ts}))
	case "error":
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_PROBLEM, MQTT_QOS_REQCONFIRMATION, false, r3events.MarshalEvent2ByteOrPanic(r3events.DoorProblemEvent{100, strings.Join(lines, " "), ts}))
	case "timeout_after_open":
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_PROBLEM, MQTT_QOS_REQCONFIRMATION, false, r3events.MarshalEvent2ByteOrPanic(r3events.DoorProblemEvent{10, strings.Join(lines, " "), ts}))
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_LOCK, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorLockUpdate{true, ts}))
	case "timeout_after_close":
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_PROBLEM, MQTT_QOS_REQCONFIRMATION, false, r3events.MarshalEvent2ByteOrPanic(r3events.DoorProblemEvent{20, strings.Join(lines, " "), ts}))
		// can't say for sure that door is locked if we ran into timeout while closing
		//~ ps.Pub(r3events.DoorLockUpdate{true, ts}, "door")
	case "opening":
	case "closing":
	default:
		Syslog_.Print("parseSocketInputLine_State: Unexpected State:", lines)
	}
}

func ParseSocketInputLineAndPublish(lines []string, mqttc *mqtt.Client, keynickstore *KeyNickStore) {
	ts := time.Now().Unix()
	if len(lines) < 1 {
		return
	}
	Debug_.Printf("ParseSocketInputLineAndPublish: %s", lines)
	switch lines[0] {
	case "State:":
		if len(lines) < 2 {
			return
		}
		parseSocketInputLine_State(lines[1:], mqttc, ts)
	case "Status:":
		if len(lines) < 3 {
			return
		}
		if len(lines[1]) < 4 {
			return
		}
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_LOCK, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorLockUpdate{lines[1][0:4] != "open", ts}))
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_AJAR, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorAjarUpdate{lines[len(lines)-1] == "shut", ts}))
	case "Info(card):":
		if len(lines) < 3 {
			return
		}
		if lines[2] != "found" {
			return
		}
		match_cardid := re_cardid_.FindStringSubmatch(lines[1])
		if len(match_cardid) > 1 {
			// PreCondition: same thread/goroutinge as created keylookup_socket !!!!
			nick, err := keynickstore.LookupHexKeyNick(match_cardid[1])
			if err != nil {
				Syslog_.Print("CardID Lookup Error", err)
				nick = "Unresolvable KeyID"
			}
			// new event: toggle by user nick using card
			mqttc.Publish(r3events.TOPIC_FRONTDOOR_CMDEVT, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorCommandEvent{"toggle", "Card", nick, ts}))
		}
	case "Info(ajar):":
		if len(lines) < 5 {
			return
		}
		mqttc.Publish(r3events.TOPIC_FRONTDOOR_AJAR, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorAjarUpdate{lines[4] == "shut", ts}))
	case "open", "close", "toggle", "reset", "openfrominside", "closefrominside", "togglefrominside":
		switch len(lines) {
		case 2:
			mqttc.Publish(r3events.TOPIC_FRONTDOOR_CMDEVT, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorCommandEvent{Command: lines[0], Using: lines[1], Ts: ts}))
		case 3:
			mqttc.Publish(r3events.TOPIC_FRONTDOOR_CMDEVT, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorCommandEvent{Command: lines[0], Using: lines[1], Who: lines[2], Ts: ts}))
		default:
			return
		}
	default:
	}
}

func ConnectChannelToMQTT(publish_chan chan SerialLine, brocker_addr string, keystore *KeyNickStore) {
	options := mqtt.NewClientOptions().AddBroker(brocker_addr).SetAutoReconnect(true).SetClientID(r3events.CLIENTID_FRONTDOOR)
	c := mqtt.NewClient(options)
	c.Connect()
	for sl := range publish_chan {
		c.Publish(r3events.TOPIC_FRONTDOOR_RAWFWLINES, MQTT_QOS_REQCONFIRMATION, false, strings.Join(sl, " "))
		ParseSocketInputLineAndPublish(sl, c, keystore)
	}
}

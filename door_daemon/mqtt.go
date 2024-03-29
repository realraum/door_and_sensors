// (c) Bernhard Tittelbach, 2015
package main

import (
	"regexp"
	"strings"
	"time"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/realraum/door_and_sensors/r3events"
)

const MQTT_QOS_NOCONFIRMATION byte = 0
const MQTT_QOS_REQCONFIRMATION byte = 1
const MQTT_QOS_4STPHANDSHAKE byte = 2

var re_cardid_ *regexp.Regexp = regexp.MustCompile("card\\(([a-fA-F0-9]+)\\)")

var door_online_topic_ string = r3events.TOPIC_R3 + r3events.CLIENTID_FRONTDOOR + "/" + r3events.TYPE_ONLINEJSON
var door_online_ip_ string = ""

func init() {
	door_online_ip_, _ = os.Hostname()
}


func parseSocketInputLine_State(lines []string, mqttc mqtt.Client, ts int64) {
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
		//// can't say for sure that door is locked so commented out. Might always have been true for older version of door (?)
		// mqttc.Publish(r3events.TOPIC_FRONTDOOR_LOCK, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.DoorLockUpdate{true, ts}))
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

func ParseSocketInputLineAndPublish(lines []string, mqttc mqtt.Client, keynickstore *KeyNickStore) {
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

func mqttOnConnectionHandler(mqttc mqtt.Client) {
	Syslog_.Print("MQTT connection to broker established. (re)subscribing topics")
	tk := mqttc.Subscribe(r3events.ACT_RESEND_STATUS_TRIGGER, 0, nil) //re-subscribe after disconnect
	tk.Wait()
	if tk.Error() != nil {
		Syslog_.Fatalf("Error (re)subscribing to %s:%s", r3events.ACT_RESEND_STATUS_TRIGGER, tk.Error())
	}
	mqttc.Publish(door_online_topic_, MQTT_QOS_REQCONFIRMATION, true, r3events.MarshalEvent2ByteOrPanic(r3events.Online{Ip: door_online_ip_, Online: true}))
}

func ConnectChannelToMQTT(publish_chan chan SerialLine, brocker_addr string, keystore *KeyNickStore, runOnMQTTConnect func(mqtt.Client)) {
	options := mqtt.NewClientOptions().AddBroker(brocker_addr).SetAutoReconnect(true).SetClientID(r3events.CLIENTID_FRONTDOOR).SetKeepAlive(49 * time.Second).SetMaxReconnectInterval(2 * time.Minute)
	options = options.SetConnectionLostHandler(func(c mqtt.Client, err error) { Debug_.Print("ERROR MQTT connection lost:", err) })
	options = options.SetOnConnectHandler(mqttOnConnectionHandler)
	options = options.SetWill(door_online_topic_, string(r3events.MarshalEvent2ByteOrPanic(r3events.Online{Ip: door_online_ip_, Online: false})), MQTT_QOS_REQCONFIRMATION, true)
	c := mqtt.NewClient(options)
	//gooble up all publish_chan stuff for as long as mqtt is not connected
	shutdown_gobbler_c := make(chan bool, 1)
	go func() {
		for {
			select {
			case <-publish_chan:
			case <-shutdown_gobbler_c:
				runOnMQTTConnect(c)
				return
			}
		}
	}()
	for {
		tk := c.Connect()
		tk.Wait() //may wait indefinately, or return immediately if mqttbroker not reachable
		if tk.Error() == nil {
			Debug_.Println("Connected to mqtt broker!!")
			break
		}
		Debug_.Println("ERROR connecting to mqtt broker", tk.Error())
		time.Sleep(5 * time.Minute)
	}
	shutdown_gobbler_c <- true
	for sl := range publish_chan {
		c.Publish(r3events.TOPIC_FRONTDOOR_RAWFWLINES, MQTT_QOS_REQCONFIRMATION, false, strings.Join(sl, " "))
		ParseSocketInputLineAndPublish(sl, c, keystore)
	}
}

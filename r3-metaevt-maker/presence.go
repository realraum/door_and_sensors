// (c) Bernhard Tittelbach, 2013

package main

import (
	"time"
	//~ "./brain"
	// r3events "../r3events"
	pubsub "github.com/btittelbach/pubsub"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	r3events "github.com/realraum/door_and_sensors/r3events"
)

const w1frontdoor_key = r3events.CLIENTID_FRONTDOOR
const w1backdoor_key = r3events.CLIENTID_BACKDOOR
const w2frontdoor_key = r3events.CLIENTID_W2FRONTDOOR
const PS_PRESENCE_W1 = "presenceW1"
const PS_PRESENCE_W2 = "presenceW2"

func allTrue(x map[string]bool) bool {
	for _, v := range x {
		if v == false {
			return false
		}
	}
	return true
}

func anyTrue(x map[string]bool) bool {
	for _, v := range x {
		if v == true {
			return true
		}
	}
	return false
}

func MetaEventRoutine_Presence(ps *pubsub.PubSub, mqttc mqtt.Client, movement_timeout, button_timeout, between_spaces_timeout int64) {
	var last_door_cmd *r3events.DoorCommandEvent
	var last_event_indicating_presence, last_manual_lockhandling int64
	locked := map[string]bool{w1frontdoor_key: true, w2frontdoor_key: true}
	lock_use_ts := map[string]int64{w1frontdoor_key: 0, w2frontdoor_key: 0}
	shut := map[string]bool{w1frontdoor_key: true, w1backdoor_key: true}
	presences_last := map[string]bool{PS_PRESENCE_W1: false, PS_PRESENCE_W2: false}

	anyPresenceTrue := func() bool { return anyTrue(presences_last) }
	allTrue := func(x map[string]bool) bool {
		for _, v := range x {
			if v == false {
				return false
			}
		}
		return true
	}

	// anyDoorAjar := func() bool { return allTrue(shut) == false }
	areAllDoorsLocked := func() bool { return allTrue(locked) }

	manualInsideButtonUsed := func(ldc *r3events.DoorCommandEvent) bool {
		return ldc.Using == "Button" || (len(ldc.Command) > 10 && ldc.Command[len(ldc.Command)-10:] == "frominside")
	}

	events_chan := ps.Sub(PS_R3EVENTS)
	defer ps.Unsub(events_chan, PS_R3EVENTS)

PRESFORLOOP:
	for r3eventi := range events_chan {
		r3event := r3eventi.(*r3events.R3MQTTMsg)
		presences := map[string]bool{PS_PRESENCE_W1: presences_last[PS_PRESENCE_W1], PS_PRESENCE_W2: presences_last[PS_PRESENCE_W2]}
		ts := time.Now().Unix()
		evnt_type := r3events.NameOfStruct(r3event.Event)
		switch evnt := r3event.Event.(type) {
		case r3events.SomethingReallyIsMoving:
			if evnt.Movement {
				//ignore movements that happened just after locking door
				if (evnt.Ts - last_event_indicating_presence) > movement_timeout {
					if false == presences[PS_PRESENCE_W1] {
						Syslog_.Println("PresenceW1 true, due to movement")
					}
					presences[PS_PRESENCE_W1] = true
				}
				last_event_indicating_presence = evnt.Ts
			} else {
				if presences_last[PS_PRESENCE_W1] {
					Syslog_.Printf("Presence: Mhh, SomethingReallyIsMoving{%+v} received but presence still true. Quite still a bunch we have here.", evnt)
				}
				if areAllDoorsLocked() && shut[w1frontdoor_key] && shut[w1backdoor_key] && evnt.Confidence >= 90 && last_event_indicating_presence > 1800 && (last_door_cmd == nil || (!manualInsideButtonUsed(last_door_cmd) && last_door_cmd.Ts >= last_manual_lockhandling)) {
					presences[PS_PRESENCE_W1] = false
				}
			}
		case r3events.BoreDoomButtonPressEvent:
			presences[PS_PRESENCE_W1] = true
			Syslog_.Println("PresenceW1 true, due to BoreDoomButtonPress")
			last_event_indicating_presence = evnt.Ts
		case r3events.DoorCommandEvent:
			last_door_cmd = &evnt
		case r3events.DoorManualMovementEvent:
			last_manual_lockhandling = evnt.Ts
			last_event_indicating_presence = evnt.Ts
		case r3events.DoorLockUpdate:
			if len(r3event.Topic) < 3 {
				continue //ignore this update
			}
			key := r3event.Topic[1]
			if locked[key] != evnt.Locked {
				//check if changed, in case that some locks send periodic status updates, which would NOT indicate presence
				last_event_indicating_presence = evnt.Ts
			}
			locked[key] = evnt.Locked
			lock_use_ts[key] = evnt.Ts
		case r3events.DoorAjarUpdate:
			if len(r3event.Topic) < 3 {
				continue //ignore this update
			}
			key := r3event.Topic[1]
			last_event_indicating_presence = evnt.Ts
			//check if we ignore this update
			if key == w1frontdoor_key && shut[w1frontdoor_key] == false && evnt.Shut && locked[w1frontdoor_key] && evnt.Ts-lock_use_ts[w1frontdoor_key] > 2 {
				Syslog_.Print("Presence: ignoring frontdoor ajar event, since obviously someone is fooling around with the microswitch while the door is still open")
				continue
			}
			shut[key] = evnt.Shut
		default:
			continue PRESFORLOOP
		}

		change := false

		///////////////////////////////
		// decide on Space 1 Presence
		if presences[PS_PRESENCE_W1] != presences_last[PS_PRESENCE_W1] {
			//... skip state check .. we had a definite presence event
		} else if shut[w1backdoor_key] == false || shut[w1frontdoor_key] == false || locked[w1frontdoor_key] == false {
			presences[PS_PRESENCE_W1] = true
		} else if last_door_cmd != nil && (manualInsideButtonUsed(last_door_cmd) || last_door_cmd.Ts < last_manual_lockhandling) {
			// if last_door_cmd is set then: if either door was closed using Button
			//or if time of manual lock movement is greater (newer) than timestamp of last_door_cmd
			//or if was closed/opened/toggled with -frominside addition to simulate close/open/toggle from inside
			presences[PS_PRESENCE_W1] = true
		} else if evnt_type == "DoorCommandEvent" {
			//don't set presence to false for just a door command, wait until we receive a LockUpdate
			//(fixes "door manually locked -> door opened with card from outside -> nobody present status sent since door is locked and we havent received door unloked event yet"-problem)
			continue
		} else {
			presences[PS_PRESENCE_W1] = false
		}
		if presences_last[PS_PRESENCE_W1] != presences[PS_PRESENCE_W1] {
			presences_last[PS_PRESENCE_W1] = presences[PS_PRESENCE_W1]
			change = true
		}
		///////////////////////////////
		// decide on Space 2 Presence
		if presences[PS_PRESENCE_W2] != presences_last[PS_PRESENCE_W2] {
			//... skip state check .. we had a definite presence event
		} else {
			presences[PS_PRESENCE_W2] = !locked[w2frontdoor_key]
		}
		if presences_last[PS_PRESENCE_W2] != presences[PS_PRESENCE_W2] {
			presences_last[PS_PRESENCE_W2] = presences[PS_PRESENCE_W2]
			change = true
		}

		presence_overall := anyPresenceTrue()
		// if !presence_overall && (time.Now().Unix()-lock_use_ts[w1frontdoor_key] < between_spaces_timeout || time.Now().Unix()-lock_use_ts[w2frontdoor_key] < between_spaces_timeout) {
		// }
		if change {
			mqttc.Publish(r3events.TOPIC_META_PRESENCE, MQTT_QOS_4STPHANDSHAKE, true, r3events.MarshalEvent2ByteOrPanic(r3events.PresenceUpdate{Present: presence_overall, InSpace1: presences_last[PS_PRESENCE_W1], InSpace2: presences_last[PS_PRESENCE_W2], Ts: ts}))
			Syslog_.Printf("Presence: %t (%+v)", presence_overall, presences_last)
		}
	}
}

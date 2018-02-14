// (c) Bernhard Tittelbach, 2013

package main

import (
	"time"
	//~ "./brain"
	r3events "../r3events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	//r3events "github.com/realraum/door_and_sensors/r3events"
	pubsub "github.com/btittelbach/pubsub"
)

func MetaEventRoutine_Presence(ps *pubsub.PubSub, mqttc mqtt.Client, movement_timeout, button_timeout int64) {
	var last_door_cmd *r3events.DoorCommandEvent
	var last_presence bool
	var last_event_indicating_presence, last_frontlock_use, last_manual_lockhandling int64
	const w1frontdoor_key = r3events.CLIENTID_FRONTDOOR
	const w1backdoor_key = r3events.CLIENTID_BACKDOOR
	const w2frontdoor_key = r3events.CLIENTID_W2FRONTDOOR
	locked := map[string]bool{w1frontdoor_key: true, w2frontdoor_key: true}
	shut := map[string]bool{w1frontdoor_key: true, w1backdoor_key: true}

	allTrue := func(x map[string]bool) bool {
		for _, v := range x {
			if v == false {
				return false
			}
		}
		return true
	}

	anyDoorAjar := func() bool { return allTrue(shut) == false }
	areAllDoorsLocked := func() bool { return allTrue(locked) }

	manualInsideButtonUsed := func(ldc *r3events.DoorCommandEvent) bool {
		return ldc.Using == "Button" || (len(ldc.Command) > 10 && ldc.Command[len(ldc.Command)-10:] == "frominside")
	}

	events_chan := ps.Sub(PS_R3EVENTS)
	defer ps.Unsub(events_chan, PS_R3EVENTS)

	for r3eventi := range events_chan {
		r3event := r3eventi.(r3events.R3MQTTMsg)
		Debug_.Printf("Presence prior: %t : %T %+v", last_presence, r3event.Event, r3event.Event)
		new_presence := last_presence
		ts := time.Now().Unix()
		evnt_type := r3events.NameOfStruct(r3event.Event)
		switch evnt := r3event.Event.(type) {
		case r3events.SomethingReallyIsMoving:
			if evnt.Movement {
				//ignore movements that happened just after locking door
				if (evnt.Ts - last_event_indicating_presence) > movement_timeout {
					new_presence = true
				}
				last_event_indicating_presence = evnt.Ts
			} else {
				if last_presence {
					Syslog_.Printf("Presence: Mhh, SomethingReallyIsMoving{%+v} received but presence still true. Quite still a bunch we have here.", evnt)
				}
				if areAllDoorsLocked() && shut[w1frontdoor_key] && shut[w1backdoor_key] && evnt.Confidence >= 90 && last_event_indicating_presence > 1800 && (last_door_cmd == nil || (!manualInsideButtonUsed(last_door_cmd) && last_door_cmd.Ts >= last_manual_lockhandling)) {
					new_presence = false
				}
			}
		case r3events.BoreDoomButtonPressEvent:
			new_presence = true
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
			if key == w1frontdoor_key {
				last_frontlock_use = evnt.Ts
			}
		case r3events.DoorAjarUpdate:
			if len(r3event.Topic) < 3 {
				continue //ignore this update
			}
			key := r3event.Topic[1]
			last_event_indicating_presence = evnt.Ts
			//check if we ignore this update
			if key == w1frontdoor_key && shut[w1frontdoor_key] == false && evnt.Shut && locked[w1frontdoor_key] && evnt.Ts-last_frontlock_use > 2 {
				Syslog_.Print("Presence: ignoring frontdoor ajar event, since obviously someone is fooling around with the microswitch while the door is still open")
				continue
			}
			shut[key] = evnt.Shut
		default:
			continue
		}

		if new_presence != last_presence {
			//... skip state check .. we had a definite presence event
		} else if !areAllDoorsLocked() || anyDoorAjar() {
			new_presence = true
		} else if last_door_cmd != nil && (manualInsideButtonUsed(last_door_cmd) || last_door_cmd.Ts < last_manual_lockhandling) {
			// if last_door_cmd is set then: if either door was closed using Button
			//or if time of manual lock movement is greater (newer) than timestamp of last_door_cmd
			//or if was closed/opened/toggled with -frominside addition to simulate close/open/toggle from inside
			new_presence = true
		} else if evnt_type == "DoorCommandEvent" {
			//don't set presence to false for just a door command, wait until we receive a LockUpdate
			//(fixes "door manually locked -> door opened with card from outside -> nobody present status sent since door is locked and we havent received door unloked event yet"-problem)
			continue
		} else {
			new_presence = false
		}
		//~ Debug_.Printf("Presence: new: %s , last:%s", new_presence, last_presence)
		if new_presence != last_presence {
			last_presence = new_presence
			mqttc.Publish(r3events.TOPIC_META_PRESENCE, MQTT_QOS_4STPHANDSHAKE, true, r3events.MarshalEvent2ByteOrPanic(r3events.PresenceUpdate{new_presence, ts}))
			Syslog_.Printf("Presence: %t", new_presence)
		}
	}
}

package main

import (
	"time"

	pubsub "github.com/btittelbach/pubsub"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	r3events "github.com/realraum/door_and_sensors/r3events"
)

func MetaEventRoutine_ContactsensorAggregation(ps *pubsub.PubSub[any], mqttc mqtt.Client, interval time.Duration) {
	locked := map[string]bool{w1frontdoor_key: true, w1backdoor_key: true, w2frontdoor_key: true}
	shut := map[string]bool{w1frontdoor_key: true, w1backdoor_key: true, w2frontdoor_key: true}
	zigbeecontact_shut := make(map[string]bool) //TODO future: presence==true falls ein Fenster offen ist

	areAllDoorsShut := func() bool { return allTrue(shut) }
	areAllDoorsLocked := func() bool { return allTrue(locked) }
	areAllWindowsShut := func() bool { return allTrue(zigbeecontact_shut) }
	last_areAllDoorsShut := true
	last_areAllDoorsLocked := true
	last_areAllWindowsShut := true

	statuschanged := func() bool {
		rv := areAllDoorsShut() != last_areAllDoorsShut || areAllDoorsLocked() != last_areAllDoorsLocked || areAllWindowsShut() != last_areAllWindowsShut
		last_areAllDoorsShut = areAllDoorsShut()
		last_areAllDoorsLocked = areAllDoorsLocked()
		last_areAllWindowsShut = areAllWindowsShut()
		return rv
	}

	events_chan := ps.Sub(PS_R3EVENTS)
	defer ps.Unsub(events_chan, PS_R3EVENTS)

PRESFORLOOP:
	for r3eventi := range events_chan {
		r3event := r3eventi.(*r3events.R3MQTTMsg)
		ts := time.Now().Unix()
		// evnt_type := r3events.NameOfStruct(r3event.Event)
		switch evnt := r3event.Event.(type) {
		case r3events.DoorLockUpdate:
			if len(r3event.Topic) < 3 {
				continue //ignore this update
			}
			key := r3event.Topic[1]
			locked[key] = evnt.Locked
		case r3events.DoorAjarUpdate:
			if len(r3event.Topic) < 3 {
				continue //ignore this update
			}
			key := r3event.Topic[1]
			shut[key] = evnt.Shut
		case r3events.ZigbeeAjarSensor:
			if len(r3event.Topic) < 3 {
				continue
			}
			zigbeecontact_shut[r3event.Topic[1]+r3event.Topic[2]] = evnt.Contact
		default:
			continue PRESFORLOOP
		}

		if statuschanged() {
			mqttc.Publish(r3events.TOPIC_META_AGGR_CONTACT_S, MQTT_QOS_4STPHANDSHAKE, true, r3events.MarshalEvent2ByteOrPanic(r3events.AggregateContactsensor{AllDoorsShut: last_areAllDoorsShut, AllWindowsShut: last_areAllWindowsShut, AllDoorsLocked: last_areAllDoorsLocked, Ts: ts}))
		}
	}
}

package r3events

import (
	"testing"
	"time"
)

func TestUnMarshallingAddTime(t *testing.T) {

	topic := "lala/blabla/" + TYPE_LOCK
	payload := []byte("{\"Locked\":true}")
	t_before := time.Now().Unix()
	evt_i, err := UnmarshalTopicByte2Event(topic, payload)
	if err != nil {
		t.Error(err)
	}
	t_after := time.Now().Unix()
	evt, ok := evt_i.(DoorLockUpdate)
	if !ok {
		t.Error("Unmarshalling used wrong Type")
	}
	if evt.Ts >= t_before && evt.Ts <= t_after {
		//time was correctly added
	} else {
		t.Errorf("Time was not added correctly: Was %d but should be around %d", evt.Ts, t_after)
	}
}

func TestUnMarshallingDontAddTime(t *testing.T) {

	topic := "lala/blabla/" + TYPE_ONLINE
	payload := []byte("ONLINE")
	evt_i, err := UnmarshalTopicByte2Event(topic, payload)
	if err != nil {
		t.Error(err)
	}
	_, ok := evt_i.(Online)
	if !ok {
		t.Error("Unmarshalling used wrong Type")
	}
}

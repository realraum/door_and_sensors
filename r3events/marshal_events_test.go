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

	topic := "lala/blabla/" + TYPE_ONLINESTR
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


func TestConvertEsphomeSensor(t *testing.T) {

	topic := TOPIC_ESPHOME_R2W2_TEMPERATURE
	payload := []byte("19.0")
	t_before := time.Now().Unix()
	evt_i, err := UnmarshalTopicByte2Event(topic, payload)
	if err != nil {
		t.Error(err)
	}
	t_after := time.Now().Unix()
	evt, ok := evt_i.(TempSensorUpdate)
	if !ok {
		t.Error("Unmarshalling used wrong Type.")
	}
	if evt.Ts >= t_before && evt.Ts <= t_after {
		//time was correctly added
	} else {
		t.Errorf("Time was not added correctly: Was %d but should be around %d", evt.Ts, t_after)
	}
	if evt.Value != 19.0 {
		t.Errorf("Value was not converted correctly, is %f should be 19.0", evt.Value)
	}
	if evt.Location != "R2W2" {
		t.Errorf("Location was not set correctly. Is %s instead of R2W2", evt.Location)
	}
}
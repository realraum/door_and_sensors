// (c) Bernhard Tittelbach, 2013

package r3events

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type R3MQTTMsg struct {
	Msg   mqtt.Message
	Topic []string
	Event interface{}
}

func SplitTopic(s string) []string {
	return strings.Split(s, "/")
}

func R3ifyMQTTMsg(msg mqtt.Message) (*R3MQTTMsg, error) {
	r3evnt, err := UnmarshalTopicByte2Event(msg.Topic(), msg.Payload())
	r3msg := &R3MQTTMsg{
		Msg:   msg,
		Topic: SplitTopic(msg.Topic()),
		Event: r3evnt,
	}
	return r3msg, err
}

func NameOfStruct(evi interface{}) (name string) {
	etype := fmt.Sprintf("%T", evi)
	etype_lastsep := strings.LastIndex(etype, ".")
	return etype[etype_lastsep+1:] //works in all cases for etype_lastsep in range -1 to len(etype)-1
}

func MarshalEvent2Byte(event_interface interface{}) (data []byte, err error) {
	var msg []byte
	//~ fmt.Printf("%T%+v\n", event_interface, event_interface)
	msg, err = json.Marshal(event_interface)
	if err != nil {
		return
	}
	data = msg
	return
}

func MarshalEvent2ByteOrPanic(event_interface interface{}) (data []byte) {
	var err error
	data, err = json.Marshal(event_interface)
	if err != nil {
		panic(err)
	}
	return
}

func UnmarshalTopicByte2Event(topic string, data []byte) (event interface{}, err error) {
	fixTs := func(event interface{}) interface{} {
		//fill Ts field with current timestamp if present and unset
		if reflect.TypeOf(event).Kind() == reflect.Ptr {
			v := reflect.Indirect(reflect.ValueOf(event))
			if reflect.TypeOf(v).Kind() == reflect.Struct {
				ts_field := v.FieldByName("Ts")
				// tskind, err := reflect.TypeOf(v).FieldByName("Ts")
				// for f := 0; f < v.NumField(); f++ {
				// ff := v.Field(f)
				// fmt.Printf("field %d: %+v kind:%s canaddr:%+v canset:%+v\n", f, ff, ff.Kind(), ff.CanAddr(), ff.CanSet())
				// }
				// fmt.Printf("tskind: %+v err: %+v\n", tskind, err)
				// fmt.Printf("%T  canset:%+v\n", ts_field, ts_field.CanSet())
				if ts_field.IsValid() && ts_field.Kind() == reflect.Int64 && ts_field.Int() <= 0 && ts_field.CanSet() {
					ts_field.SetInt(time.Now().Unix())
				}
			}
			return v.Interface()
		}
		return event
	}
	topics := strings.Split(topic, "/")
	toplvltopic := topics[len(topics)-1]
	switch topic {
	case TOPIC_FRONTDOOR_CMDEVT:
		typed_event := DoorCommandEvent{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_FRONTDOOR_PROBLEM:
		typed_event := DoorProblemEvent{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_FRONTDOOR_MANUALLOCK:
		typed_event := DoorManualMovementEvent{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_META_PRESENCE:
		typed_event := PresenceUpdate{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_META_REALMOVE:
		typed_event := SomethingReallyIsMoving{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_META_TEMPSPIKE:
		typed_event := TempSensorSpike{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_META_HUMIDITYSPIKE:
		typed_event := HumiditySensorSpike{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_META_DUSTSPIKE:
		typed_event := DustSensorSpike{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_META_DUSKORDAWN:
		typed_event := DuskOrDawn{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_GW_DHCPACK:
		typed_event := NetDHCPACK{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_GW_STATS:
		typed_event := NetGWStatUpdate{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_LASER_CARD:
		typed_event := LaserCutter{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case ACT_YAMAHA_SEND:
		typed_event := YamahaIRCmd{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case ACT_RF433_SEND:
		typed_event := SendRF433Code{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_BACKDOOR_POWERLOSS:
		typed_event := UPSPowerUpdate{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_OLGAFREEZER_SENSORLOST, TOPIC_META_SENSORLOST:
		typed_event := SensorLost{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case ACT_RF433_SETDELAY:
		typed_event := SetRF433Delay{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_IRCBOT_FOODREQUEST:
		typed_event := FoodOrderRequest{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_IRCBOT_FOODINVITE:
		typed_event := FoodOrderInvite{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	case TOPIC_IRCBOT_FOODETA:
		typed_event := FoodOrderETA{}
		err = json.Unmarshal(data, &typed_event)
		event = fixTs(&typed_event)
	default:
		event = nil
		err = errors.New("cannot unmarshal unknown topic") // we'll never see this error, it only tells the next if-check that we want to give the next switch a try
	}

	//no special topic matched, let's match generic types
	if event == nil {
		switch toplvltopic {
		case TYPE_LOCK:
			typed_event := DoorLockUpdate{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_AJAR:
			typed_event := DoorAjarUpdate{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_DOOMBUTTON:
			typed_event := BoreDoomButtonPressEvent{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_TEMPOVER:
			typed_event := TempOverThreshold{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_TEMP:
			typed_event := TempSensorUpdate{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_ILLUMINATION:
			typed_event := IlluminationSensorUpdate{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_DUST:
			typed_event := DustSensorUpdate{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_RELHUMIDITY:
			typed_event := RelativeHumiditySensorUpdate{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_GASALERT:
			typed_event := GasLeakAlert{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_MOVEMENTPIR:
			typed_event := MovementSensorUpdate{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_POWERLOSS:
			typed_event := UPSPowerUpdate{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_SENSORLOST:
			typed_event := SensorLost{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_VOLTAGE:
			typed_event := Voltage{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_BAROMETER:
			typed_event := BarometerUpdate{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_VENTILATIONSTATE:
			typed_event := VentilationState{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_SONOFFSENSOR:
			typed_event := SonOffSensor{}
			err = json.Unmarshal(data, &typed_event)
			event = fixTs(&typed_event)
		case TYPE_ONLINEJSON:
			typed_event := Online{}
			err = json.Unmarshal(data, &typed_event)
			event = typed_event
		case TYPE_ONLINESTR:
			event = Online{Online: string(data) == "ONLINE"}
		default:
			event = nil
			err = errors.New("cannot unmarshal unknown type or topic")
		}
		//if err was set before and now we have an event, we need to unset err
		if event != nil {
			err = nil
		}
	}

	return
}

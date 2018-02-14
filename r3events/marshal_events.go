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
	topics := strings.Split(topic, "/")
	toplvltopic := topics[len(topics)-1]
	switch topic {
	case TOPIC_FRONTDOOR_CMDEVT:
		event = DoorCommandEvent{}
	case TOPIC_FRONTDOOR_PROBLEM:
		event = DoorProblemEvent{}
	case TOPIC_FRONTDOOR_MANUALLOCK:
		event = DoorManualMovementEvent{}
	case TOPIC_META_PRESENCE:
		event = PresenceUpdate{}
	case TOPIC_META_REALMOVE:
		event = SomethingReallyIsMoving{}
	case TOPIC_META_TEMPSPIKE:
		event = TempSensorSpike{}
	case TOPIC_META_HUMIDITYSPIKE:
		event = HumiditySensorSpike{}
	case TOPIC_META_DUSTSPIKE:
		event = DustSensorSpike{}
	case TOPIC_META_DUSKORDAWN:
		event = DuskOrDawn{}
	case TOPIC_GW_DHCPACK:
		event = NetDHCPACK{}
	case TOPIC_GW_STATS:
		event = NetGWStatUpdate{}
	case TOPIC_LASER_CARD:
		event = LaserCutter{}
	case ACT_YAMAHA_SEND:
		event = YamahaIRCmd{}
	case ACT_RF433_SEND:
		event = SendRF433Code{}
	case TOPIC_BACKDOOR_POWERLOSS:
		event = UPSPowerUpdate{}
	case TOPIC_OLGAFREEZER_SENSORLOST, TOPIC_META_SENSORLOST:
		event = SensorLost{}
	case ACT_RF433_SETDELAY:
		event = SetRF433Delay{}
	case TOPIC_IRCBOT_FOODREQUEST:
		event = FoodOrderRequest{}
	case TOPIC_IRCBOT_FOODINVITE:
		event = FoodOrderInvite{}
	case TOPIC_IRCBOT_FOODETA:
		event = FoodOrderETA{}
	default:
		event = nil
		err = errors.New("cannot unmarshal unknown topic") // we'll never see this error, it only tells the next if-check that we want to give the next switch a try
	}

	//no special topic matched, let's match generic types
	if event == nil {
		switch toplvltopic {
		case TYPE_LOCK:
			event = DoorLockUpdate{}
		case TYPE_AJAR:
			event = DoorAjarUpdate{}
		case TYPE_DOOMBUTTON:
			event = BoreDoomButtonPressEvent{}
		case TYPE_TEMPOVER:
			event = TempOverThreshold{}
		case TYPE_TEMP:
			event = TempSensorUpdate{}
		case TYPE_ILLUMINATION:
			event = IlluminationSensorUpdate{}
		case TYPE_DUST:
			event = DustSensorUpdate{}
		case TYPE_RELHUMIDITY:
			event = RelativeHumiditySensorUpdate{}
		case TYPE_GASALERT:
			event = GasLeakAlert{}
		case TYPE_MOVEMENTPIR:
			event = MovementSensorUpdate{}
		case TYPE_POWERLOSS:
			event = UPSPowerUpdate{}
		case TYPE_SENSORLOST:
			event = SensorLost{}
		case TYPE_VOLTAGE:
			event = Voltage{}
		case TYPE_BAROMETER:
			event = BarometerUpdate{}
		case TYPE_VENTILATIONSTATE:
			event = VentilationState{}
		case TYPE_SONOFFSENSOR:
			event = SonOffSensor{}
		case TYPE_ONLINE:
			event = Online{Online: string(data) == "ONLINE"}
			return event, nil
		default:
			event = nil
			err = errors.New("cannot unmarshal unknown type or topic")
		}
	}

	//if we have found a type and not returned before: parse the json
	if event != nil {
		err = json.Unmarshal(data, &event)
	}

	//fill Ts field with current timestamp if present and unset
	if reflect.TypeOf(&event).Kind() == reflect.Struct {
		ts_field := reflect.ValueOf(&event).Elem().FieldByName("Ts")
		if ts_field.IsValid() && ts_field.Kind() == reflect.Int64 && ts_field.Int() <= 0 {
			ts_field.SetInt(time.Now().Unix())
		}
	}

	return
}

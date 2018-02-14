// (c) Bernhard Tittelbach, 2013

package r3events

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
	case TOPIC_FRONTDOOR_LOCK:
		newevent := new(DoorLockUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TOPIC_FRONTDOOR_AJAR:
		newevent := new(DoorAjarUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_BACKDOOR_AJAR:
		newevent := new(BackdoorAjarUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_FRONTDOOR_CMDEVT:
		newevent := new(DoorCommandEvent)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_FRONTDOOR_PROBLEM:
		newevent := new(DoorProblemEvent)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_FRONTDOOR_MANUALLOCK:
		newevent := new(DoorManualMovementEvent)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case "realraum/backdoorcx/timetick":
		newevent := new(TimeTick)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_META_PRESENCE:
		newevent := new(PresenceUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_META_REALMOVE:
		newevent := new(SomethingReallyIsMoving)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_META_TEMPSPIKE:
		newevent := new(TempSensorSpike)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_META_HUMIDITYSPIKE:
		newevent := new(HumiditySensorSpike)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_META_DUSTSPIKE:
		newevent := new(DustSensorSpike)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_META_DUSKORDAWN:
		newevent := new(DuskOrDawn)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_GW_DHCPACK:
		newevent := new(NetDHCPACK)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_GW_STATS:
		newevent := new(NetGWStatUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_LASER_CARD:
		newevent := new(LaserCutter)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case ACT_YAMAHA_SEND:
		newevent := new(YamahaIRCmd)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case ACT_RF433_SEND:
		newevent := new(SendRF433Code)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_BACKDOOR_POWERLOSS:
		newevent := new(UPSPowerUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_OLGAFREEZER_SENSORLOST, TOPIC_META_SENSORLOST:
		newevent := new(SensorLost)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case ACT_RF433_SETDELAY:
		newevent := new(SetRF433Delay)
		err = json.Unmarshal(data, newevent)
		event = *newevent

	case TOPIC_IRCBOT_FOODREQUEST:
		newevent := new(FoodOrderRequest)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_IRCBOT_FOODINVITE:
		newevent := new(FoodOrderInvite)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	case TOPIC_IRCBOT_FOODETA:
		newevent := new(FoodOrderETA)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent

	default:
		event = nil
		err = errors.New("cannot unmarshal unknown topic") // we'll never see this error, it only tells the next if-check that we want to give the next switch a try
	}
	if event != nil && err == nil {
		return
	}

	switch toplvltopic {
	case TYPE_DOOMBUTTON:
		newevent := new(BoreDoomButtonPressEvent)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_TEMPOVER:
		newevent := new(TempOverThreshold)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_TEMP:
		newevent := new(TempSensorUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_ILLUMINATION:
		newevent := new(IlluminationSensorUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_DUST:
		newevent := new(DustSensorUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_RELHUMIDITY:
		newevent := new(RelativeHumiditySensorUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_GASALERT:
		newevent := new(GasLeakAlert)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_MOVEMENTPIR:
		newevent := new(MovementSensorUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_POWERLOSS:
		newevent := new(UPSPowerUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_SENSORLOST:
		newevent := new(SensorLost)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_VOLTAGE:
		newevent := new(Voltage)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_BAROMETER:
		newevent := new(BarometerUpdate)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_VENTILATIONSTATE:
		newevent := new(VentilationState)
		err = json.Unmarshal(data, newevent)
		if newevent.Ts <= 0 {
			newevent.Ts = time.Now().Unix()
		}
		event = *newevent
	case TYPE_SONOFFSENSOR:
		newevent := new(SonOffSensor)
		err = json.Unmarshal(data, newevent)
		event = *newevent
	case TYPE_ONLINE:
		event = Online{Online: string(data) == "ONLINE"}
	default:
		event = nil
		err = errors.New("cannot unmarshal unknown type or topic")
	}
	return
}

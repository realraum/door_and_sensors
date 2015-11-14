// (c) Bernhard Tittelbach, 2013

package r3events

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func UnmarshalByteByte2Event(topic string, data []byte) (event interface{}, category string, err error) {
	switch topic {
	case TOPIC_FRONTDOOR_LOCK:
		newevent := new(DoorLockUpdate)
		err = json.Unmarshal(data, newevent)
		category = "door"
		event = *newevent
	case TOPIC_FRONTDOOR_AJAR:
		newevent := new(DoorAjarUpdate)
		err = json.Unmarshal(data, newevent)
		category = "door"
		event = *newevent
	case TOPIC_BACKDOOR_LOCK:
		newevent := new(BackdoorAjarUpdate)
		err = json.Unmarshal(data, newevent)
		category = "door"
		event = *newevent
	case TOPIC_FRONTDOOR_CMDEVT:
		newevent := new(DoorCommandEvent)
		err = json.Unmarshal(data, newevent)
		category = "door"
		event = *newevent
	case TOPIC_FRONTDOOR_PROBLEM:
		newevent := new(DoorProblemEvent)
		err = json.Unmarshal(data, newevent)
		category = "door"
		event = *newevent
	case TOPIC_FRONTDOOR_MANUALLOCK:
		newevent := new(DoorManualMovementEvent)
		err = json.Unmarshal(data, newevent)
		category = "door"
		event = *newevent
	case TOPIC_PILLAR_DOOMBUTTON:
		newevent := new(BoreDoomButtonPressEvent)
		err = json.Unmarshal(data, newevent)
		category = "buttons"
		event = *newevent
	case TOPIC_OLGAFREEZER_TEMPOVER:
		newevent := new(TempOverThreshold)
		err = json.Unmarshal(data, newevent)
		category = "sensors"
		event = *newevent
	case TOPIC_PILLAR_TEMP:
	case TOPIC_BACKDOOR_TEMP:
	case TOPIC_OLGAFREEZER_TEMP:
		newevent := new(TempSensorUpdate)
		err = json.Unmarshal(data, newevent)
		category = "sensors"
		event = *newevent
	case TOPIC_PILLAR_ILLUMINATION:
		newevent := new(IlluminationSensorUpdate)
		err = json.Unmarshal(data, newevent)
		category = "sensors"
		event = *newevent
	case TOPIC_PILLAR_DUST:
		newevent := new(DustSensorUpdate)
		err = json.Unmarshal(data, newevent)
		category = "sensors"
		event = *newevent
	case TOPIC_PILLAR_RELHUMIDITY:
		newevent := new(RelativeHumiditySensorUpdate)
		err = json.Unmarshal(data, newevent)
		category = "sensors"
		event = *newevent
	case "realraum/backdoorcx/timetick":
		newevent := new(TimeTick)
		err = json.Unmarshal(data, newevent)
		category = "time"
		event = *newevent
	case TOPIC_BACKDOOR_GASALERT:
		newevent := new(GasLeakAlert)
		err = json.Unmarshal(data, newevent)
		category = "sensors"
		event = *newevent
	case TOPIC_PILLAR_MOVEMENTPIR:
	case TOPIC_BACKDOOR_MOVEMENTPIR:
		newevent := new(MovementSensorUpdate)
		err = json.Unmarshal(data, newevent)
		category = "movement"
		event = *newevent
	case TOPIC_META_PRESENCE:
		newevent := new(PresenceUpdate)
		err = json.Unmarshal(data, newevent)
		category = "presence"
		event = *newevent
	case TOPIC_META_REALMOVE:
		newevent := new(SomethingReallyIsMoving)
		err = json.Unmarshal(data, newevent)
		category = "movement"
		event = *newevent
	case TOPIC_META_TEMPSPIKE:
		newevent := new(TempSensorSpike)
		err = json.Unmarshal(data, newevent)
		category = "sensors"
		event = *newevent
	case TOPIC_META_DUSTSPIKE:
		newevent := new(DustSensorSpike)
		err = json.Unmarshal(data, newevent)
		category = "sensors"
		event = *newevent
	case TOPIC_GW_DHCPACK:
		newevent := new(NetDHCPACK)
		err = json.Unmarshal(data, newevent)
		category = "network"
		event = *newevent
	case TOPIC_GW_STATS:
		newevent := new(NetGWStatUpdate)
		err = json.Unmarshal(data, newevent)
		category = "network"
		event = *newevent
	case TOPIC_LASER_CARD:
		newevent := new(LaserCutter)
		err = json.Unmarshal(data, newevent)
		category = "sensors"
		event = *newevent
	case ACT_YAMAHA_SEND:
		newevent := new(LaserCutter)
		err = json.Unmarshal(data, newevent)
		category = "actions"
		event = *newevent
	case ACT_RF433_SEND:
		newevent := new(LaserCutter)
		err = json.Unmarshal(data, newevent)
		category = "actions"
		event = *newevent
	default:
		event = nil
		err = errors.New("cannot unmarshal unknown type")
		category = ""
	}
	return
}

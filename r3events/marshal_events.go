// (c) Bernhard Tittelbach, 2013

package r3events

import (
    "encoding/json"
    "fmt"
    "errors"
    "strings"
    )

func NameOfStruct(evi interface{}) (name string) {
    etype := fmt.Sprintf("%T", evi)
    etype_lastsep := strings.LastIndex(etype,".")
    return etype[etype_lastsep+1:] //works in all cases for etype_lastsep in range -1 to len(etype)-1
}

func MarshalEvent2ByteByte(event_interface interface{}) (data [][]byte, err error) {
    var msg []byte
    //~ fmt.Printf("%T%+v\n", event_interface, event_interface)
	msg, err = json.Marshal(event_interface)
	if err != nil {
		return
	}
    data = [][]byte{[]byte(NameOfStruct(event_interface)), msg} //works in all cases for etype_lastsep in range -1 to len(etype)-1
    return
}

func UnmarshalByteByte2Event(data [][]byte) (event interface{}, category string, err error) {
    if len(data) != 2 {
        return nil, "", errors.New("not a r3event message")
    }
    switch string(data[0]) {
        case "DoorLockUpdate":
            newevent := new(DoorLockUpdate)
            err = json.Unmarshal(data[1],newevent)
            category = "door"
            event = *newevent
        case "DoorAjarUpdate":
            newevent := new(DoorAjarUpdate)
            err = json.Unmarshal(data[1], newevent)
            category = "door"
            event = *newevent
        case "BackdoorAjarUpdate":
            newevent := new(BackdoorAjarUpdate)
            err = json.Unmarshal(data[1], newevent)
            category = "door"
            event = *newevent
        case "DoorCommandEvent":
            newevent := new(DoorCommandEvent)
            err = json.Unmarshal(data[1], newevent)
            category = "door"
            event = *newevent
        case "DoorProblemEvent":
            newevent := new(DoorProblemEvent)
            err = json.Unmarshal(data[1], newevent)
            category = "door"
            event = *newevent
        case "BoreDoomButtonPressEvent":
            newevent := new(BoreDoomButtonPressEvent)
            err = json.Unmarshal(data[1], newevent)
            category = "buttons"
            event = *newevent
        case "TempSensorUpdate":
            newevent := new(TempSensorUpdate)
            err = json.Unmarshal(data[1], newevent)
            category = "sensors"
            event = *newevent
        case "IlluminationSensorUpdate":
            newevent := new(IlluminationSensorUpdate)
            err = json.Unmarshal(data[1], newevent)
            category = "sensors"
            event = *newevent
        case "DustSensorUpdate":
            newevent := new(DustSensorUpdate)
            err = json.Unmarshal(data[1], newevent)
            category = "sensors"
            event = *newevent
        case "RelativeHumiditySensorUpdate":
            newevent := new(RelativeHumiditySensorUpdate)
            err = json.Unmarshal(data[1], newevent)
            category = "sensors"
            event = *newevent
        case "TimeTick":
            newevent := new(TimeTick)
            err = json.Unmarshal(data[1], newevent)
            category = "time"
            event = *newevent
        case "MovementSensorUpdate":
            newevent := new(MovementSensorUpdate)
            err = json.Unmarshal(data[1], newevent)
            category = "movement"
            event = *newevent
        case "PresenceUpdate":
            newevent := new(PresenceUpdate)
            err = json.Unmarshal(data[1], newevent)
            category = "presence"
            event = *newevent
        case "SomethingReallyIsMoving":
            newevent := new(SomethingReallyIsMoving)
            err = json.Unmarshal(data[1], newevent)
            category = "movement"
            event = *newevent
        case "NetDHCPACK":
            newevent := new(NetDHCPACK)
            err = json.Unmarshal(data[1], newevent)
            category = "network"
            event = *newevent
        case "NetGWStatUpdate":
            newevent := new(NetGWStatUpdate)
            err = json.Unmarshal(data[1], newevent)
            category = "network"
            event = *newevent
        default:
            event = nil
            err = errors.New("cannot unmarshal unknown type")
            category = ""
    }
    return
}

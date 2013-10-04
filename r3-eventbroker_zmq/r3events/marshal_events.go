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

func UnmarshalByteByte2Event(data [][]byte) (event interface{}, err error) {
    if len(data) != 2 {
        return nil, errors.New("not a r3event message")
    }
    switch string(data[0]) {
        case "DoorLockUpdate":
            event = new(DoorLockUpdate)
            err = json.Unmarshal(data[1],event)
        case "DoorAjarUpdate":
            event := new(DoorAjarUpdate)
            err = json.Unmarshal(data[1], event)
        case "BackdoorAjarUpdate":
            event := new(DoorAjarUpdate)
            err = json.Unmarshal(data[1], event)
        case "DoorCommandEvent":
            event := new(DoorCommandEvent)
            err = json.Unmarshal(data[1], event)
        case "BoreDoomButtonPressEvent":
            event := new(BoreDoomButtonPressEvent)
            err = json.Unmarshal(data[1], event)
        case "TempSensorUpdate":
            event := new(TempSensorUpdate)
            err = json.Unmarshal(data[1], event)
        case "IlluminationSensorUpdate":
            event := new(IlluminationSensorUpdate)
            err = json.Unmarshal(data[1], event)
        case "TimeTick":
            event := new(TimeTick)
            err = json.Unmarshal(data[1], event)
        case "MovementSensorUpdate":
            event := new(MovementSensorUpdate)
            err = json.Unmarshal(data[1], event)
        case "PresenceUpdate":
            event := new(PresenceUpdate)
            err = json.Unmarshal(data[1], event)
        case "SomethingReallyIsMoving":
            event := new(SomethingReallyIsMoving)
            err = json.Unmarshal(data[1], event)
        default:
            event = nil
            err = errors.New("cannot unmarshal unknown type")
    }
    return
}

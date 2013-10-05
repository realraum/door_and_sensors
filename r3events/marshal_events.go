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
            event = DoorLockUpdate{}
            err = json.Unmarshal(data[1],&event)
            category = "door"
        case "DoorAjarUpdate":
            event := DoorAjarUpdate{}
            err = json.Unmarshal(data[1], &event)
            category = "door"
        case "BackdoorAjarUpdate":
            event := DoorAjarUpdate{}
            err = json.Unmarshal(data[1], &event)
            category = "door"
        case "DoorCommandEvent":
            event := DoorCommandEvent{}
            err = json.Unmarshal(data[1], &event)
            category = "door"
        case "DoorProblemEvent":
            event := DoorProblemEvent{}
            err = json.Unmarshal(data[1], &event)
            category = "door"
        case "BoreDoomButtonPressEvent":
            event := BoreDoomButtonPressEvent{}
            err = json.Unmarshal(data[1], &event)
            category = "buttons"
        case "TempSensorUpdate":
            event := TempSensorUpdate{}
            err = json.Unmarshal(data[1], &event)
            category = "sensors"
        case "IlluminationSensorUpdate":
            event := IlluminationSensorUpdate{}
            err = json.Unmarshal(data[1], &event)
            category = "sensors"
        case "DustSensorUpdate":
            event := DustSensorUpdate{}
            err = json.Unmarshal(data[1], &event)
            category = "sensors"
        case "RelativeHumiditySensorUpdate":
            event := RelativeHumiditySensorUpdate{}
            err = json.Unmarshal(data[1], &event)
            category = "sensors"
        case "TimeTick":
            event := TimeTick{}
            err = json.Unmarshal(data[1], &event)
            category = "time"
        case "MovementSensorUpdate":
            event := MovementSensorUpdate{}
            err = json.Unmarshal(data[1], &event)
            category = "movement"
        case "PresenceUpdate":
            event := PresenceUpdate{}
            err = json.Unmarshal(data[1], &event)
            category = "presence"
        case "SomethingReallyIsMoving":
            event := SomethingReallyIsMoving{}
            err = json.Unmarshal(data[1], &event)
            category = "movement"
        case "NetDHCPACK":
            event := NetDHCPACK{}
            err = json.Unmarshal(data[1], &event)
            category = "network"
        case "NetGWStatUpdate":
            event := NetGWStatUpdate{}
            err = json.Unmarshal(data[1], &event)
            category = "network"
        default:
            event = nil
            err = errors.New("cannot unmarshal unknown type")
            category = ""
    }
    return
}

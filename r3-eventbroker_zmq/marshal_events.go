// (c) Bernhard Tittelbach, 2013

package main

import (
    "encoding/json"
    "fmt"
    "errors"
    )


func MarshalEvent(event_interface interface{}) (data [][]byte, err error) {
    var msg []byte
    fmt.Printf("%T%+v\n", event_interface, event_interface)
	msg, err = json.Marshal(event_interface)
	if err != nil {
		return
	}
    etype := []byte(fmt.Sprintf("%T", event_interface)[5:])
    data = [][]byte{etype, msg}
    return
}

func UnmarshalEvent(data [][]byte) (event interface{}, err error) {
    switch string(data[0]) {
        case "DoorLockUpdate":
            typedevent := new(DoorLockUpdate)
            err = json.Unmarshal(data[1], typedevent)
            event = typedevent
        case "DoorAjarUpdate":
            typedevent := new(DoorAjarUpdate)
            err = json.Unmarshal(data[1], typedevent)
            event = typedevent
        case "DoorCommandEvent":
            typedevent := new(DoorCommandEvent)
            err = json.Unmarshal(data[1], typedevent)
            event = typedevent
        default:
            event = nil
            err = errors.New("unknown type")
    }
    return
}

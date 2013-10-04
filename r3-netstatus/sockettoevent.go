// (c) Bernhard Tittelbach, 2013

package main

import (
    pubsub "github.com/tuxychandru/pubsub"
    //~ "bufio"
    //~ "time"
    //~ "./brain"
    //~ "net"
    "encoding/json"
    r3events "svn.spreadspace.org/realraum/go.svn/r3-eventbroker_zmq/r3events"
    )

func ParseZMQr3Event(lines [][]byte, ps *pubsub.PubSub) { //, brn *brain.Brain) {
    //Debug_.Printf("ParseZMQr3Event: len: %d lines: %s", len(lines), lines)
    if len(lines) != 2 {
        return
    }
    switch string(lines[0]) {
        case "PresenceUpdate":
            evnt := new(r3events.PresenceUpdate)
            err := json.Unmarshal(lines[1],evnt)
            if err == nil {ps.Pub(*evnt, "presence")}
        case "IlluminationSensorUpdate" :
            evnt := new(r3events.IlluminationSensorUpdate)
            err := json.Unmarshal(lines[1],evnt)
            if err == nil {ps.Pub(*evnt, "sensors")}
        case "TempSensorUpdate" :
            evnt := new(r3events.TempSensorUpdate)
            err := json.Unmarshal(lines[1],evnt)
            if err == nil {ps.Pub(*evnt, "sensors")}
        case "MovementSensorUpdate" :
            evnt := new(r3events.MovementSensorUpdate)
            err := json.Unmarshal(lines[1],evnt)
            if err == nil {ps.Pub(*evnt, "movement")}
        case "BoreDoomButtonPressEvent" :
            evnt := new(r3events.BoreDoomButtonPressEvent)
            err := json.Unmarshal(lines[1],evnt)
            if err == nil {ps.Pub(*evnt, "buttons")}
        case "DoorLockUpdate" :
            evnt := new(r3events.DoorLockUpdate)
            err := json.Unmarshal(lines[1],evnt)
            if err == nil {ps.Pub(*evnt, "door")}
        case "DoorAjarUpdate" :
            evnt := new(r3events.DoorAjarUpdate)
            err := json.Unmarshal(lines[1],evnt)
            if err == nil {ps.Pub(*evnt, "door")}
         case "DoorCommandEvent" :
            evnt := new(r3events.DoorCommandEvent)
            err := json.Unmarshal(lines[1],evnt)
            if err == nil {ps.Pub(*evnt, "door")}
    }
}

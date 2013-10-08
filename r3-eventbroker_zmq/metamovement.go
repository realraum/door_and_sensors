// (c) Bernhard Tittelbach, 2013

package main

import (
    "time"
    pubsub "github.com/tuxychandru/pubsub"
    "container/ring"
    r3events "svn.spreadspace.org/realraum/go.svn/r3events"
    )


/// Movement Meta Event Injector:
///     threshold number of movements within gran_duration*granularity seconds -> SomethingReallyIsMoving{True}
///     No movement within 3 hours but movement within the last 6 hours -> SomethingReallyIsMoving{False}
///
/// Thus SomethingReallyIsMoving{True} fires regularly, but at most every gran_duration seconds
/// While SomethingReallyIsMoving{False} fires only once to assure us that everybody might really be gone


func MetaEventRoutine_Movement(ps *pubsub.PubSub, granularity, gran_duration int , threshold uint32) {
    var last_movement,last_movement1,last_movement2,last_movement3 int64
    var confidence uint8
    movement_window := ring.New(granularity+1)
    events_chan := ps.Sub("movement")
    defer ps.Unsub(events_chan, "movement")
    myticker := time.NewTicker(time.Duration(gran_duration) * time.Second)

    for { select {
        case event := <- events_chan:
            switch event.(type) {
                case r3events.MovementSensorUpdate:
                    if movement_window.Value == nil {
                        movement_window.Value = uint32(1)
                    } else {
                        movement_window.Value = uint32(movement_window.Value.(uint32)  + 1)
                    }
            }
        case gots := <- myticker.C:
            ts := gots.Unix()
            movement_window.Prev().Value = (uint32) (0)
            movement_window = movement_window.Next()
            var movsum uint32 = 0
            movement_window.Do(func(v interface{}){if v != nil {movsum += v.(uint32)}})
            if movsum > threshold {
                confidence = uint8(movsum)
                ps.Pub( r3events.SomethingReallyIsMoving{true, confidence ,ts}, "movement")
                last_movement = ts
                last_movement1 = ts
                last_movement2 = ts
                last_movement3 = ts
            }
            // this sucks.....
            if last_movement > 0 && ts - last_movement < 3600*6 {
                if ts - last_movement > 3600*3 {
                    last_movement = 0
                    ps.Pub( r3events.SomethingReallyIsMoving{false,99,ts}, "movement")
                } else if ts - last_movement > 3600 && last_movement3 > 0 {
                    last_movement3 = 0
                    ps.Pub( r3events.SomethingReallyIsMoving{false,50,ts}, "movement")
                } else if ts - last_movement > 1800 && last_movement2 > 0 {
                    last_movement2 = 0
                    ps.Pub( r3events.SomethingReallyIsMoving{false,20,ts}, "movement")
                } else if ts - last_movement > 120 && last_movement1 > 0 {
                    last_movement1 = 0
                    ps.Pub( r3events.SomethingReallyIsMoving{false,5,ts}, "movement")
                }
            }
    } }
}
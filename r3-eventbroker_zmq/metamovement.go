// (c) Bernhard Tittelbach, 2013

package main

import (
    "time"
    //~ "./brain"
    pubsub "github.com/tuxychandru/pubsub"
    "container/ring"
    "./r3events"
    )


func MetaEventRoutine_Movement(ps *pubsub.PubSub, granularity, gran_duration int , threshold uint32) {
    var last_movement int64
    movement_window := ring.New(granularity+1)
    events_chan := ps.Sub("movement")
    myticker := time.NewTicker(time.Duration(gran_duration) * time.Second)

    for { select {
        case event := <- events_chan:
            switch event.(type) {
                case r3events.MovementSensorUpdate:
                    movement_window.Value =  (uint32) (movement_window.Value.(uint32)  + 1)
            }
        case <- myticker.C:
            movement_window.Prev().Value = (uint32) (0)
            movement_window = movement_window.Next()
            var movsum uint32 = 0
            movement_window.Do(func(v interface{}){if v != nil {movsum += v.(uint32)}})
            ts :=  time.Now().Unix()
            if movsum > threshold {
                ps.Pub( r3events.SomethingReallyIsMoving{true,ts}, "movement")
                last_movement = ts
            }

            if last_movement > 0 && ts - last_movement < 3600*6 && ts - last_movement > 3600*3 {
                last_movement = 0
                ps.Pub( r3events.SomethingReallyIsMoving{false, ts}, "movement")
            }
    } }
}
// (c) Bernhard Tittelbach, 2013

package main

import (
    "time"
    //~ "./brain"
    pubsub "github.com/tuxychandru/pubsub"
    r3events "svn.spreadspace.org/realraum/go.svn/r3-eventbroker_zmq/r3events"
    )

func MetaEventRoutine_Presence(ps *pubsub.PubSub) {
    //~ var last_door_cmd *DoorCommandEvent
    var last_presence bool
    var last_movement, last_buttonpress int64
    var front_locked, front_shut, back_shut bool = true, true, true

    events_chan := ps.Sub("door", "doorcmd", "buttons", "movement")
    defer ps.Unsub(events_chan, "door", "doorcmd", "buttons", "movement")

    for event := range(events_chan) {
        //~ Debug_.Printf("Presence: %s - %s", event, doorstatemap)
        new_presence := last_presence
        ts := time.Now().Unix()
        switch evnt := event.(type) {
            case r3events.SomethingReallyIsMoving:
                if evnt.Movement {
                    last_movement = evnt.Ts
                } else {
                    last_movement = 0
                }
            case r3events.BoreDoomButtonPressEvent:
                last_buttonpress = evnt.Ts
                new_presence = true
            //~ case DoorCommandEvent:
                //~ last_door_cmd = &evnt
            case r3events.DoorLockUpdate:
                front_locked = evnt.Locked
            case r3events.DoorAjarUpdate:
                front_shut = evnt.Shut
            case r3events.BackdoorAjarUpdate:
                back_shut = evnt.Shut
        }

        any_door_unlocked := ! front_locked
        any_door_ajar := ! (front_shut && back_shut)

        if new_presence != last_presence {
            //... skip state check .. we had a definite presence event
        } else if any_door_unlocked || any_door_ajar {
            new_presence = true
        } else if last_movement != 0 || ts - last_buttonpress < 200 {
            new_presence = true
        } else {
            new_presence = false
        }
        //~ Debug_.Printf("Presence: new: %s , last:%s", new_presence, last_presence)
        if new_presence != last_presence {
            last_presence = new_presence
            ps.Pub(r3events.PresenceUpdate{new_presence, ts} , "presence")
        }
    }
}
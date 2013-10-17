// (c) Bernhard Tittelbach, 2013

package main

import (
    "time"
    //~ "./brain"
    pubsub "github.com/tuxychandru/pubsub"
    r3events "svn.spreadspace.org/realraum/go.svn/r3events"
    )

func MetaEventRoutine_Presence(ps *pubsub.PubSub, movement_timeout, button_timeout int64) {
    var last_door_cmd *r3events.DoorCommandEvent
    var last_presence bool
    var last_event_indicating_presence, last_frontlock_use, last_manual_lockhandling int64
    var front_locked, front_shut, back_shut bool = true, true, true

    events_chan := ps.Sub("door", "doorcmd", "buttons", "movement")
    defer ps.Unsub(events_chan, "door", "doorcmd", "buttons", "movement")

    for event := range(events_chan) {
        Debug_.Printf("Presence prior: %t : %T %+v", last_presence, event, event)
        new_presence := last_presence
        ts := time.Now().Unix()
        switch evnt := event.(type) {
            case r3events.SomethingReallyIsMoving:
                if evnt.Movement {
                    //ignore movements that happened just after locking door
                    if (evnt.Ts - last_event_indicating_presence) > movement_timeout {
                        new_presence = true
                    }
                    last_event_indicating_presence = evnt.Ts
                } else {
                    if last_presence { Syslog_.Printf("Presence: Mhh, SomethingReallyIsMoving{%+v} received but presence still true. Quite still a bunch we have here.", evnt) }
                    if front_locked && front_shut && back_shut && evnt.Confidence >= 20 && last_event_indicating_presence > 1800 {
                        new_presence = false
                    }
                }
            case r3events.BoreDoomButtonPressEvent:
                new_presence = true
                last_event_indicating_presence = evnt.Ts
            case r3events.DoorCommandEvent:
                last_door_cmd = &evnt
            case r3events.DoorManualMovementEvent:
                last_manual_lockhandling = evnt.Ts
                last_event_indicating_presence = evnt.Ts
            case r3events.DoorLockUpdate:
                front_locked = evnt.Locked
                last_frontlock_use = evnt.Ts
                last_event_indicating_presence = evnt.Ts
            case r3events.DoorAjarUpdate:
                if front_shut == false && evnt.Shut && front_locked && evnt.Ts - last_frontlock_use > 2 {
                    Syslog_.Print("Presence: ignoring frontdoor ajar event, since obviously someone is fooling around with the microswitch while the door is still open")
                } else {
                    front_shut = evnt.Shut
                }
                last_event_indicating_presence = evnt.Ts
            case r3events.BackdoorAjarUpdate:
                back_shut = evnt.Shut
                last_event_indicating_presence = evnt.Ts
        }

        any_door_unlocked := (front_locked == false)
        any_door_ajar := ! (front_shut && back_shut)

        if new_presence != last_presence {
            //... skip state check .. we had a definite presence event
        } else if any_door_unlocked || any_door_ajar {
            new_presence = true
        } else if last_door_cmd != nil && (last_door_cmd.Using == "Button"  || last_door_cmd.Ts < last_manual_lockhandling) {
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
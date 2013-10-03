// (c) Bernhard Tittelbach, 2013

package main

import (
    "time"
    //~ "./brain"
    pubsub "github.com/tuxychandru/pubsub"
    )

type PresenceUpdate struct {
    Present bool
    Ts int64
}

type doorstate struct {
    locked bool
    shut bool
}

func MetaEventRoutine_Presence(ps *pubsub.PubSub) {
    //~ var last_door_cmd *DoorCommandEvent
    var last_presence bool
    var last_movement, last_buttonpress int64
    doorstatemap := make(map[int]doorstate,1)

    events_chan := ps.Sub("door", "doorcmd", "buttons", "movement")

    for event := range(events_chan) {
        new_presence := last_presence
        ts := time.Now().Unix()
        switch evnt := event.(type) {
            case SomethingReallyIsMoving:
                if evnt.Movement {
                    last_movement = evnt.Ts
                } else {
                    last_movement = 0
                }
            case ButtonPressUpdate:
                last_buttonpress = evnt.Ts
                new_presence = true
            //~ case DoorCommandEvent:
                //~ last_door_cmd = &evnt
            case DoorLockUpdate:
                doorstatemap[evnt.DoorID]=doorstate{locked:evnt.Locked, shut:doorstatemap[evnt.DoorID].shut}
            case DoorAjarUpdate:
                doorstatemap[evnt.DoorID]=doorstate{locked:doorstatemap[evnt.DoorID].locked, shut:evnt.Shut}
        }

        any_door_unlocked := false
        any_door_ajar := false
        for _, ds := range(doorstatemap) {
            if ds.locked == false {any_door_unlocked = true }
            if ds.shut == false {any_door_ajar = true }
        }

        if new_presence != last_presence {
            //... skip state check .. we had a definite presence event
        } else if any_door_unlocked || any_door_ajar {
            new_presence = true
        } else if last_movement != 0 || ts - last_buttonpress < 200 {
            new_presence = true
        } else {
            new_presence = false
        }

        if new_presence != last_presence {
            last_presence = new_presence
            ps.Pub(PresenceUpdate{new_presence, ts} , "presence")
        }
    }
}
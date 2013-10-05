// (c) Bernhard Tittelbach, 2013

package main

import (
    zmq "github.com/vaughan0/go-zmq"
    r3events "svn.spreadspace.org/realraum/go.svn/r3-eventbroker_zmq/r3events"
)

type hippocampus map[string]interface{}

func BrainCenter( zmq_ctx *zmq.Context, listen_addr string, event_chan <- chan interface{} ) {
    zbrain_chans, err := ZmqsBindNewReplySocket(zmq_ctx, listen_addr)
    if err != nil { panic(err) }
    defer zbrain_chans.Close()
    h := make(hippocampus,5)

    for { select {
        case event, ec_still_open := <- event_chan:
            if ! ec_still_open { return }
            h[r3events.NameOfStruct(event)] = event
            Debug_.Printf("Brain: stored %s, %s", r3events.NameOfStruct(event), event)

        case brain_request := <- zbrain_chans.In():
            if len(brain_request) == 0 { continue }
            requested_eventname := string(brain_request[0])
            Debug_.Printf("Brain: received request: %s", requested_eventname)
            retr_event, is_in_map := h[requested_eventname]
            if is_in_map {
                data, err := r3events.MarshalEvent2ByteByte(retr_event)
                if err == nil {
                    zbrain_chans.Out() <- data
                    continue
                } else {
                    Syslog_.Print("BrainCenter", err)
                    Debug_.Print("BrainCenter", err)
                }
            }
            zbrain_chans.Out() <- [][]byte{[]byte("UNKNOWN")}
    } }
}

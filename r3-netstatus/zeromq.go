// (c) Bernhard Tittelbach, 2013

package main

import (
    zmq "github.com/vaughan0/go-zmq"
 )

// ---------- ZeroMQ Code -------------

func ZmqsInit(sub_port string)  (ctx *zmq.Context, sub_chans *zmq.Channels) {
    var err error
    ctx, err = zmq.NewContext()
    if err != nil {
        panic(err)
    }
    //close only on later panic, otherwise leave open:
    defer func(){ if r:= recover(); r != nil { ctx.Close(); panic(r) } }()

    if len(sub_port) > 0 {
        sub_sock, err := ctx.Socket(zmq.Sub)
        if err != nil {
            panic(err)
        }
        defer func() { if r:= recover(); r != nil { sub_sock.Close(); panic(r) } }()

        sub_sock.Subscribe([]byte{}) //subscribe empty filter -> aka to all messages

        if err = sub_sock.Connect(sub_port); err != nil {
            panic(err)
        }

        sub_chans = sub_sock.ChannelsBuffer(10)
        go zmqsHandleError(sub_chans)
    } else {
        sub_chans = nil
    }

    return
}

func zmqsHandleError(chans *zmq.Channels) {
    for error := range(chans.Errors()) {
        chans.Close()
        panic(error)
    }
}

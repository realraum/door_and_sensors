// (c) Bernhard Tittelbach, 2013

package main

import (
    zmq "github.com/vaughan0/go-zmq"
 )

// ---------- ZeroMQ Code -------------

func ZmqsInit(addrport string)  (ctx *zmq.Context, chans *zmq.Channels) {
    var err error
    ctx, err = zmq.NewContext()
    if err != nil {
        panic(err)
    }
    //close only on later panic, otherwise leave open:
    defer func(){ if r:= recover(); r != nil { ctx.Close(); panic(r) } }()

    if len(addrport) > 0 {
        sock, err := ctx.Socket(zmq.Rep)
        if err != nil {
            panic(err)
        }
        defer func() { if r:= recover(); r != nil { sock.Close(); panic(r) } }()

	    if err = sock.Bind(addrport); err != nil {
            panic(err)
        }

        chans = sock.ChannelsBuffer(10)
        go zmqsHandleError(chans)
    } else {
        chans = nil
    }

    return
}

func zmqsHandleError(chans *zmq.Channels) {
    for error := range(chans.Errors()) {
        chans.Close()
        panic(error)
    }
}
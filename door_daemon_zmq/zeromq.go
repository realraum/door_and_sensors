// (c) Bernhard Tittelbach, 2013

package main

import (
    zmq "github.com/vaughan0/go-zmq"
    "time"
 )

// ---------- ZeroMQ Code -------------

func ZmqsInit(cmd_port, pub_port string)  (ctx *zmq.Context, cmd_chans, pub_chans *zmq.Channels) {

    var err error
    ctx, err = zmq.NewContext()
    if err != nil {
        panic(err)
    }
    //close only on later panic, otherwise leave open:
    defer func(){ if r:= recover(); r != nil { ctx.Close(); panic(r) } }()

    if len(cmd_port) > 0 {
        cmd_sock, err := ctx.Socket(zmq.Rep)
        if err != nil {
            panic(err)
        }
        defer func() { if r:= recover(); r != nil { cmd_sock.Close(); panic(r) } }()

        cmd_sock.SetRecvTimeout(2 * time.Second)
        cmd_sock.SetSendTimeout(2 * time.Second)

	    if err = cmd_sock.Bind(cmd_port); err != nil {
            panic(err)
        }
    
        cmd_chans = cmd_sock.ChannelsBuffer(10)
        go zmqsHandleError(cmd_chans)
    } else {
        cmd_chans = nil
    }

    if len(pub_port) > 0 {
        pub_sock, err := ctx.Socket(zmq.Pub)
        if err != nil {
            panic(err)
        }
        defer func() { if r:= recover(); r != nil { pub_sock.Close(); panic(r) } }()

        if err = pub_sock.Bind(pub_port); err != nil {
            panic(err)
        }

        pub_chans = pub_sock.ChannelsBuffer(10)
        go zmqsHandleError(pub_chans)
    } else {
        pub_chans = nil
    }

    return
}

func zmqsHandleError(chans *zmq.Channels) {
    for error := range(chans.Errors()) {
        chans.Close()
        panic(error)    
    }
}
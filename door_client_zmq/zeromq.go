// (c) Bernhard Tittelbach, 2013

package main

import (
    zmq "github.com/vaughan0/go-zmq"
    "time"
 )

// ---------- ZeroMQ Code -------------

func ZmqsInit(cmd_port, sub_port string)  (ctx *zmq.Context, cmd_chans, sub_chans *zmq.Channels) {
    var err error
    ctx, err = zmq.NewContext()
    if err != nil {
        panic(err)
    }
    //close only on later panic, otherwise leave open:
    defer func(){ if r:= recover(); r != nil { ctx.Close(); panic(r) } }()

    if len(cmd_port) > 0 {
        cmd_sock, err := ctx.Socket(zmq.Req)
        if err != nil {
            panic(err)
        }
        defer func() { if r:= recover(); r != nil { cmd_sock.Close(); panic(r) } }()

        cmd_sock.SetRecvTimeout(2 * time.Second)
        cmd_sock.SetSendTimeout(2 * time.Second)

        if err = cmd_sock.Connect(cmd_port); err != nil {
            panic(err)
        }

        cmd_chans = cmd_sock.ChannelsBuffer(10)
        go zmqsHandleError(cmd_chans)
    } else {
        cmd_chans = nil
    }

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

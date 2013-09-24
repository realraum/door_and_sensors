// (c) Bernhard Tittelbach, 2013

package main

import (
    zmq "github.com/vaughan0/go-zmq"
    "time"
 )

// ---------- ZeroMQ Code -------------

func ZmqsInit(cmd_port, sub_port string)  (cmd_chans, sub_chans *zmq.Channels) {

    ctx, err := zmq.NewContext()
    if err != nil {
        panic(err)
    }
    //close only on panic, otherwise leave open:
    defer func(){ if r:= recover(); r != nil { ctx.Close(); panic(r) } }()

    cmd_sock, err := ctx.Socket(zmq.Req)
    if err != nil {
        panic(err)
    }
    defer func() { if r:= recover(); r != nil { cmd_sock.Close(); panic(r) } }()

    cmd_sock.SetRecvTimeout(2 * time.Second)
    cmd_sock.SetSendTimeout(2 * time.Second)

    sub_sock, err := ctx.Socket(zmq.Sub)
    if err != nil {
        panic(err)
    }
    defer func() { if r:= recover(); r != nil { sub_sock.Close(); panic(r) } }()

    if err = cmd_sock.Connect(cmd_port); err != nil {
        panic(err)
    }

    if err = sub_sock.Connect(sub_port); err != nil {
        panic(err)
    }

    cmd_chans = cmd_sock.ChannelsBuffer(10)
    sub_chans = cmd_sock.ChannelsBuffer(10)

    go zmqsHandleError(cmd_chans, sub_chans)
    return
}

func zmqsHandleError(cmd_chans, sub_chans *zmq.Channels) {
    for {
        select {
            case cmd_error := <- cmd_chans.Errors():
                cmd_chans.Close()
                sub_chans.Close()
                panic(cmd_error)
            case sub_error := <- sub_chans.Errors():
                cmd_chans.Close()
                sub_chans.Close()
                panic(sub_error)
        }
    }
}
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
    //close only on panic, otherwise leave open:
    defer func(){ if r:= recover(); r != nil { ctx.Close(); panic(r) } }()

    cmd_sock, err := ctx.Socket(zmq.Rep)
    if err != nil {
        panic(err)
    }
    defer func() { if r:= recover(); r != nil { cmd_sock.Close(); panic(r) } }()

    cmd_sock.SetRecvTimeout(2 * time.Second)
    cmd_sock.SetSendTimeout(2 * time.Second)

    pub_sock, err := ctx.Socket(zmq.Pub)
    if err != nil {
        panic(err)
    }
    defer func() { if r:= recover(); r != nil { pub_sock.Close(); panic(r) } }()

    if err = cmd_sock.Bind(cmd_port); err != nil { // "tcp://*:5555"
        panic(err)
    }

    if err = pub_sock.Bind(pub_port); err != nil { // "tcp://*:5556"
        panic(err)
    }

    cmd_chans = cmd_sock.ChannelsBuffer(10)
    pub_chans = cmd_sock.ChannelsBuffer(10)
    go zmqsHandleError(cmd_chans, pub_chans)
    return
}

func zmqsHandleError(cmd_chans, pub_chans *zmq.Channels) {
    for {
        select {
            case cmd_error := <- cmd_chans.Errors():
                cmd_chans.Close()
                pub_chans.Close()
                panic(cmd_error)
            case pub_error := <- pub_chans.Errors():
                cmd_chans.Close()
                pub_chans.Close()
                panic(pub_error)
        }
    }
}
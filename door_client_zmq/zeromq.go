// (c) Bernhard Tittelbach, 2013

package main

import (
    zmq "github.com/vaughan0/go-zmq"
 )

// ---------- ZeroMQ Code -------------

func ZmqsInit(cmd_port, sub_port string)  (cmd_chans, pub_chans *zmq.Channels) {

    cmd_ctx, err := zmq.NewContext()
    if err != nil {
        panic(err)
    }
    //close only on panic, otherwise leave open:
    defer func(){ if r:= recover(); r != nil { cmd_ctx.Close(); panic(r) } }()
    
    pub_ctx, err := zmq.NewContext()
    if err != nil {
        panic(err)
    }
    defer func() { if r:= recover(); r != nil { pub_ctx.Close(); panic(r) } }()
    
    cmd_sock, err := cmd_ctx.Socket(zmq.Req)
    if err != nil {
        panic(err)
    }
    defer func() { if r:= recover(); r != nil { cmd_sock.Close(); panic(r) } }()

    pub_sock, err := pub_ctx.Socket(zmq.Sub)
    if err != nil {
        panic(err)
    }
    defer func() { if r:= recover(); r != nil { pub_sock.Close(); panic(r) } }()

    if err = cmd_sock.Bind(cmd_port); err != nil {
        panic(err)
    }

    if err = pub_sock.Bind(sub_port); err != nil {
        panic(err)
    }
    
    cmd_chans = cmd_sock.Channels()
    pub_chans = cmd_sock.Channels()
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
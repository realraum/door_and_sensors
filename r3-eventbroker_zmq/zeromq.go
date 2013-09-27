// (c) Bernhard Tittelbach, 2013

package main

import (
    zmq "github.com/vaughan0/go-zmq"
    "bytes"
    "errors"
 )

// ---------- ZeroMQ Code -------------

func ZmqsInit(sub_connect_port, sub_listen_port, pub_port, keylookup_port string)  (ctx *zmq.Context, sub_chans *zmq.Channels, pub_sock *zmq.Socket, keylookup_sock *zmq.Socket) {
    var err error
    ctx, err = zmq.NewContext()
    if err != nil {
        panic(err)
    }
    //close only on later panic, otherwise leave open:
    defer func(){ if r:= recover(); r != nil { ctx.Close(); panic(r) } }()

    if len(sub_connect_port) > 0 && len(sub_listen_port) > 0 {
        sub_sock, err := ctx.Socket(zmq.Sub)
        if err != nil {
            panic(err)
        }
        defer func() { if r:= recover(); r != nil { sub_sock.Close(); panic(r) } }()

	    if err = sub_sock.Bind(sub_listen_port); err != nil {
            panic(err)
        }

	    if err = sub_sock.Connect(sub_connect_port); err != nil {
            panic(err)
        }

        sub_chans = sub_sock.ChannelsBuffer(10)
        go zmqsHandleError(sub_chans)
    } else {
        sub_chans = nil
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
    } else {
        pub_sock = nil
    }

    if len(keylookup_port) > 0 {
        keylookup_sock, err := ctx.Socket(zmq.Req)
        if err != nil {
            panic(err)
        }
        defer func() { if r:= recover(); r != nil { keylookup_sock.Close(); panic(r) } }()

        if err = keylookup_sock.Connect(keylookup_port); err != nil {
            panic(err)
        }
    } else {
        keylookup_sock = nil
    }

    return
}

func zmqsHandleError(chans *zmq.Channels) {
    for error := range(chans.Errors()) {
        chans.Close()
        panic(error)
    }
}

func (sock *zmq.Socket) ZmqsRequestAnswer(request [][]byte) (answer [][]byte) {
    if err := sock.Send(request); err != nil {
        panic(err)
    }
    parts, err := sock.Recv()
    if err != nil {
        panic(err)
    }
    return parts
}

func (s *zmq.Socket) LookupCardIdNick(hexbytes []byte) (nick string, error) {
    answ := s.ZmqsRequestAnswer([][]byte{hexbytes})
    if len(answ) == 0 {
        return "", errors.New("Empty reply received")
    }    
    if answ[0] == []byte("ERROR") {
        return "", errors.New(string(bytes.Join(answ[1:],[]byte(" "))))
    }
    if answ[0] !=  []byte("RESULT") || len(answ) != 3{
        return "", errors.New("Unknown reply received")
    }
    if answ[1] !=  hexbytes {
        return "", errors.New("Wrong reply received")
    }
    return string(answ[2]), nil
}
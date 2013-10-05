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

        sub_sock.Subscribe([]byte{}) //subscribe empty filter -> aka to all messages

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
        pub_sock, err = ctx.Socket(zmq.Pub)
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
        keylookup_sock, err = ctx.Socket(zmq.Req)
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

func ZmqsBindNewReplySocket(ctx *zmq.Context, addr string) (chans *zmq.Channels, err error) {
    if len(addr) == 0 {
        return nil, errors.New("No listen address given")
    }
    sock, err := ctx.Socket(zmq.Rep)
    if err != nil { return nil, err}

    if err = sock.Bind(addr); err != nil {
        sock.Close()
        return nil, err
    }

    chans = sock.ChannelsBuffer(10)
    go zmqsHandleError(chans)

    return chans, nil
}

func ZmqsRequestAnswer(sock *zmq.Socket, request [][]byte) (answer [][]byte) {
    if err := sock.Send(request); err != nil {
        panic(err)
    }
    parts, err := sock.Recv()
    if err != nil {
        panic(err)
    }
    return parts
}

func ZmqsAskQuestionsAndClose(ctx *zmq.Context, addr string, questions [][][]byte) [][][]byte {
    if len(addr) == 0 || ctx == nil { return nil }

    req_sock, err := ctx.Socket(zmq.Req)
    if err != nil {
        return nil
    }
    defer req_sock.Close()

    if err = req_sock.Connect(addr); err != nil {
        return nil
    }

    rv := make([][][]byte, len(questions))
    for index, q := range(questions) {
        rv[index] = ZmqsRequestAnswer(req_sock, q)
    }
    return rv
}

func LookupCardIdNick(s *zmq.Socket, hexbytes []byte) (string, error) {
    answ := ZmqsRequestAnswer(s, [][]byte{hexbytes})
    if len(answ) == 0 {
        return "", errors.New("Empty reply received")
    }
    if bytes.Compare(answ[0], []byte("ERROR")) == 0 {
        return "", errors.New(string(bytes.Join(answ[1:],[]byte(" "))))
    }
    if bytes.Compare(answ[0], []byte("RESULT")) != 0 || len(answ) != 3{
        return "", errors.New("Unknown reply received")
    }
    if bytes.Compare(answ[1], hexbytes) != 0 {
        return "", errors.New("Wrong reply received")
    }
    return string(answ[2]), nil
}
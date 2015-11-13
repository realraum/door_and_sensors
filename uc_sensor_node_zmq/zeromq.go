// (c) Bernhard Tittelbach, 2013

package main

import (
	zmq "github.com/pebbe/zmq4"
)

// ---------- ZeroMQ Code -------------

func ZmqsInit(pub_addr string) (ctx *zmq.Context, pub_sock *zmq.Socket) {
	var err error
	ctx, err = zmq.NewContext()
	if err != nil {
		panic(err)
	}
	//close only on later panic, otherwise leave open:
	defer func() {
		if r := recover(); r != nil {
			ctx.Term()
			panic(r)
		}
	}()

	if len(pub_addr) > 0 {
		pub_sock, err = ctx.NewSocket(zmq.PUB)
		if err != nil {
			panic(err)
		}
		defer func() {
			if r := recover(); r != nil {
				pub_sock.Close()
				panic(r)
			}
		}()

		if err = pub_sock.Connect(pub_addr); err != nil {
			panic(err)
		}
	} else {
		pub_sock = nil
	}

	return
}

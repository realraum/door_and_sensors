// (c) Bernhard Tittelbach, 2015
package main

import (
	"log"
	"net"
	"net/rpc"
)

type CmdAndReply struct {
	cmd      string
	backchan chan SerialLine
	errchan  chan error
}

type Frontdoor struct {
	ReqChan chan CmdAndReply
}

func (r *Frontdoor) SendCmd(cmd_w_args string, reply *SerialLine) error {
	backchan := make(chan SerialLine)
	errchan := make(chan error)
	r.ReqChan <- CmdAndReply{cmd_w_args, backchan, errchan}
	select {
	case *reply = <-backchan:
		return nil
	case err := <-errchan:
		return err
	}
}

func StartRPCServer(send_me_cmds chan CmdAndReply, socketpath string) {
	r := &Frontdoor{send_me_cmds}
	rpc.Register(r)
	l, e := net.Listen("unixpacket", socketpath)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	rpc.Accept(l) //this blocks forever
	log.Panic("rpc socket lost")
}

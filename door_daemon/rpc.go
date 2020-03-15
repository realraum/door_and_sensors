// (c) Bernhard Tittelbach, 2015
package main

import (
	"log"
	"net"
	"net/rpc"
	"os"
)

var rpcServerSocketPerm = 0777

type CmdAndReply struct {
	cmd      SerialLine
	backchan chan SerialLine
	errchan  chan error
}

type Frontdoor struct {
	ReqChan chan CmdAndReply
}

func (r *Frontdoor) SendCmd(cmd_w_args SerialLine, reply *SerialLine) error {
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

func (r *Frontdoor) ProgramKeys(new_keysfile []byte, reply *int) error {
	//TODO
	//(write keysfile with sanitized given data, if supplied)
	//reload keysfile if needed
	//check number and length of keys
	//program keys into eeprom
}

func StartRPCServer(send_me_cmds chan CmdAndReply, socketpath string) {
	r := &Frontdoor{send_me_cmds}
	rpc.Register(r)
	l, e := net.Listen("unixpacket", socketpath)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	if e := os.Chmod(socketpath, os.FileMode(rpcServerSocketPerm)); e != nil {
		log.Printf("Info: could not chmod %O on %s", rpcServerSocketPerm, socketpath)
	}
	rpc.Accept(l) //this blocks forever
	log.Panic("rpc socket lost")
}

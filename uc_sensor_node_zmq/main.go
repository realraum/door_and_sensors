// (c) Bernhard Tittelbach, 2013

package main

import (
    "flag"
)


// ---------- Main Code -------------

var (
    tty_dev_ string
    pub_addr string
    use_syslog_ bool    
    enable_debug_ bool    
)

func init() {
    flag.StringVar(&pub_addr, "brokeraddr", "tcp://torwaechter.realraum.at:4243", "zmq address to send stuff to")
    flag.StringVar(&tty_dev_, "ttydev", "/dev/ttyACM0", "path do tty uc device")
    flag.BoolVar(&use_syslog_, "syslog", false, "log to syslog local1 facility")    
    flag.BoolVar(&enable_debug_, "debug", false, "debugging messages on")    
    flag.Parse()
}

func main() {
    zmqctx, pub_sock := ZmqsInit(pub_addr)
    if pub_sock == nil { panic("zmq socket creation failed") }
    defer zmqctx.Close()   
    defer pub_sock.Close()

    if enable_debug_ {
        LogEnableDebuglog()
    } else if use_syslog_ {
        LogEnableSyslog()
        Syslog_.Print("started")
    }

    serial_wr, serial_rd, err := OpenAndHandleSerial(tty_dev_)
    if err != nil { panic(err) }    
    defer close(serial_wr)
    
    for incoming_ser_line := range(serial_rd) {
        Syslog_.Printf("%s",incoming_ser_line)
        if err := pub_sock.Send(incoming_ser_line); err != nil { panic(err) }
    }
}

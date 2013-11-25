// (c) Bernhard Tittelbach, 2013

package main

import (
    "os"
    "flag"
    "time"
)

//~ func StringArrayToByteArray(ss []string) [][]byte {
    //~ bb := make([][]byte, len(ss))
    //~ for index, s := range(ss) {
        //~ bb[index] = []byte(s)
    //~ }
    //~ return bb
//~ }

// ---------- Main Code -------------

var (
    cmd_port_ string
    pub_port_ string
    door_tty_path_ string
    enable_syslog_ bool
    enable_debug_ bool
)

func init() {
    flag.StringVar(&cmd_port_, "cmdport", "ipc:///run/tuer/door_cmd.ipc", "zmq command socket path")
    flag.StringVar(&pub_port_, "pubport", "tcp://*:4242", "zmq public/listen socket path")
    flag.StringVar(&door_tty_path_, "device", "/dev/door", "door tty device path")
    flag.BoolVar(&enable_syslog_, "syslog", false, "enable logging to syslog")
    flag.BoolVar(&enable_debug_, "debug", false, "enable debug output")
    flag.Parse()
}

func main() {
    if enable_syslog_ { LogEnableSyslog()}
    if enable_debug_ { LogEnableDebuglog()}
    Syslog_.Print("started")
    defer Syslog_.Print("exiting")

    zmqctx, cmd_chans, pub_chans := ZmqsInit(cmd_port_, pub_port_)
    defer cmd_chans.Close()
    defer pub_chans.Close()
    defer zmqctx.Close()

    serial_wr, serial_rd, err := OpenAndHandleSerial(door_tty_path_, 0)
    defer close(serial_wr)
    if err != nil {
        panic(err)
    }

    var next_incoming_serial_is_client_reply bool
    timeout_chan := make(chan bool)
    defer close(timeout_chan)
    for {
        select {
            case incoming_ser_line, is_notclosed := <- serial_rd:
                if is_notclosed {
                    //~ if Syslog_ != nil { Syslog_.Print(ByteArrayToString(incoming_ser_line)) }
                    if Syslog_ != nil { Syslog_.Printf("%s",incoming_ser_line) }
                    if next_incoming_serial_is_client_reply {
                        next_incoming_serial_is_client_reply = false
                        cmd_chans.Out() <- incoming_ser_line
                    }
                    pub_chans.Out() <- incoming_ser_line
                } else {
                    Syslog_.Print("serial device disappeared, exiting")
                    os.Exit(1)
                }
            case tv, timeout_notclosed := <- timeout_chan:
                if timeout_notclosed && tv && next_incoming_serial_is_client_reply {
                        next_incoming_serial_is_client_reply = false
                        cmd_chans.Out() <- [][]byte{[]byte("ERROR"), []byte("No reply from firmware")}
                }
            case incoming_request, ic_notclosed := <- cmd_chans.In():
                if ! ic_notclosed {
                    Syslog_.Print("zmq socket died, exiting")
                    os.Exit(2)
                }
                if string(incoming_request[0]) == "log" {
                    if len(incoming_request) < 2 {
                        cmd_chans.Out() <- [][]byte{[]byte("ERROR"), []byte("argument missing")}
                        continue
                    }
                    Syslog_.Printf("Log: %s",incoming_request[1:])
                    cmd_chans.Out() <- [][]byte{[]byte("Ok")}
                    continue
                }
                Syslog_.Printf("%s",incoming_request)
                 if err := HandleCommand(incoming_request, serial_wr, serial_rd); err != nil {
                    out_msg := [][]byte{[]byte("ERROR"), []byte(err.Error())}
                    cmd_chans.Out() <- out_msg
                 } else {
                    pub_chans.Out() <- incoming_request
                    next_incoming_serial_is_client_reply = true
                    go func(){time.Sleep(3*time.Second); timeout_chan <- true;}()
                 }
        }
    }
}

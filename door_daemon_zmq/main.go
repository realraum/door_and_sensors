// (c) Bernhard Tittelbach, 2013

package main

import (
    "fmt"
    "os"
    "flag"
    //~ "log"
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
)

func usage() {
    fmt.Fprintf(os.Stderr, "Usage: door_daemon_0mq <door tty device>\n")
    flag.PrintDefaults()
}

func init() {
    flag.StringVar(&cmd_port_, "cmdport", "tcp://127.0.01:3232", "zmq command socket path")
    flag.StringVar(&pub_port_, "pubport", "pgm://233.252.1.42:4242", "zmq public/listen socket path")
    flag.Usage = usage
    flag.Parse()
}

func main() {
    args := flag.Args()
    if len(args) < 1 {
        fmt.Fprintf(os.Stderr, "Input file is missing!\n");
        usage()
        os.Exit(1);
    }

    zmqctx, cmd_chans, pub_chans := ZmqsInit(cmd_port_, pub_port_)
    defer cmd_chans.Close()
    defer pub_chans.Close()
    defer zmqctx.Close()

    serial_wr, serial_rd, err := OpenAndHandleSerial(args[0])
    defer close(serial_wr)
    if err != nil {
        panic(err)
    }

    //~ serial_wr <- "f"
    //~ firmware_version := <- serial_rd
    //~ log.Print("Firmware version:", firmware_version)
    var next_incoming_serial_is_client_reply bool
    for {
        select {
            case incoming_ser_line, is_notclosed := <- serial_rd:
                if is_notclosed {
                    if next_incoming_serial_is_client_reply {
                        next_incoming_serial_is_client_reply = false
                        cmd_chans.Out() <- incoming_ser_line
                    }
                    pub_chans.Out() <- incoming_ser_line
                } else {
                    os.Exit(1)
                }
            case incoming_request, ic_notclosed := <- cmd_chans.In():
                if ! ic_notclosed {os.Exit(2)}
                //~ log.Print(incoming_request)
                 if err := HandleCommand(incoming_request, serial_wr, serial_rd); err != nil {
                    //~ log.Print(err)
                    out_msg := [][]byte{[]byte("ERROR"), []byte(err.Error())}
                    cmd_chans.Out() <- out_msg
                    //~ log.Print("sent error")
                 } else {
                    //~ log.Print(reply)
                    pub_chans.Out() <- incoming_request
                    next_incoming_serial_is_client_reply = true
                    //~ log.Print("sent reply")
                 }
        }
    }
}

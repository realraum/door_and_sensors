// (c) Bernhard Tittelbach, 2013

package main

import (
    "fmt"
    "os"
    "flag"
    "log"
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
    flag.StringVar(&cmd_port_, "cmdport", "tcp://localhost:5555", "zmq command socket path")
    flag.StringVar(&pub_port_, "pubport", "gmp://*:6666", "zmq public/listen socket path")
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
    
    cmd_chans, pub_chans := ZmqsInit(cmd_port_, pub_port_)   
    
    serial_wr, serial_rd, err := OpenAndHandleSerial(args[0], pub_chans.Out())
    if err != nil {
        close(serial_wr)
        panic(err)
    }
    
    serial_wr <- "f"
    firmware_version := <- serial_rd
    log.Print("Firmware version:", firmware_version)

    for incoming_request := range cmd_chans.In() {
        reply, err := HandleCommand(incoming_request, pub_chans.Out(), serial_wr, serial_rd)
         if err != nil {
            cmd_chans.Out() <- [][]byte{[]byte("ERROR"), []byte(err.Error())}
            log.Print(err)
         } else {
            cmd_chans.Out() <- reply
         }        
    }
}

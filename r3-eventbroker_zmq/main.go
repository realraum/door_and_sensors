// (c) Bernhard Tittelbach, 2013

package main

import (
    "fmt"
    "os"
    "flag"
    //~ "time"
    "log/syslog"
    "log"
    pubsub "github.com/tuxychandru/pubsub"
    "./r3events"
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
    doorsub_addr_ string
    sensorssub_port_ string
    pub_port_ string
    keylookup_addr_ string
    use_syslog_ bool
    Syslog_ *log.Logger
)

func usage() {
    fmt.Fprintf(os.Stderr, "Usage: zmq_broker_event_transformer [options]\n")
    flag.PrintDefaults()
}

func init() {
    flag.StringVar(&doorsub_addr_, "doorsubaddr", "tcp://torwaechter.realraum.at:4242", "zmq door event publish addr")
    flag.StringVar(&sensorssub_port_, "sensorsubport", "tcp://*:4243", "zmq public/listen socket addr for incoming sensor data")
    flag.StringVar(&pub_port_, "pubport", "tcp://*:4244", "zmq port publishing consodilated events")
    flag.StringVar(&keylookup_addr_, "keylookupaddr", "ipc:///run/tuer/door_keyname.ipc", "address to use for key/name lookups")
    flag.BoolVar(&use_syslog_, "syslog", false, "log to syslog local2 facility")
    flag.Usage = usage
    flag.Parse()
}

func main() {
    zmqctx, sub_in_chans, pub_out_socket, keylookup_socket := ZmqsInit(doorsub_addr_, sensorssub_port_, pub_port_, keylookup_addr_)
    if sub_in_chans != nil {defer sub_in_chans.Close()}
    defer zmqctx.Close()
    if pub_out_socket != nil {defer pub_out_socket.Close()}
    if keylookup_socket != nil {defer keylookup_socket.Close()}
    if sub_in_chans == nil || pub_out_socket == nil || keylookup_socket == nil {
        panic("zmq sockets must not be nil !!")
    }
    if use_syslog_ {
        var logerr error
        Syslog_, logerr = syslog.NewLogger(syslog.LOG_INFO | (18<<3), 0)
        //~ Syslog_, logerr = syslog.NewLogger(syslog.LOG_INFO | syslog.LOG_LOCAL2, 0)
        if logerr != nil { panic(logerr) }
        Syslog_.Print("started")
        defer Syslog_.Print("exiting")
    }

    ps := pubsub.New(3)
    defer ps.Shutdown()
    //~ ticker := time.NewTicker(time.Duration(5) * time.Minute)
    publish_these_events_chan := ps.Sub("door", "doorcmd", "presence", "sensors", "buttons", "movement")

    go MetaEventRoutine_Movement(ps, 10, 20, 10)
    go MetaEventRoutine_Presence(ps)

    for {
        select {
            case subin := <- sub_in_chans.In():
                ParseSocketInputLine(subin, ps, keylookup_socket)
            //~ case <- ticker.C:
                //~ MakeTimeTick(ps)
            case event_interface := <- publish_these_events_chan:
                data, err := r3events.MarshalEvent2ByteByte(event_interface)
                log.Printf("publishing %s",data)
                if err != nil {
                    if Syslog_ != nil {Syslog_.Print(err)}
                    continue
                }
                if err := pub_out_socket.Send(data); err != nil {
                    panic(err)
                }
        }
    }

}

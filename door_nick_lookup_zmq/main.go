// (c) Bernhard Tittelbach, 2013

package main

import (
    "os"
    "flag"
    "fmt"
    //~ "log/syslog"
    //~ "log"
)

// ---------- Main Code -------------

var (
    zmqport_ string
    door_keys_file_ string
    use_syslog_ bool
    start_server_ bool
    //~ syslog_ *log.Logger
)

func usage() {
    fmt.Fprintf(os.Stderr, "Usage: door_nick_lookup_zmq [options] [keyid1 [keyid2 [...]]]\n")
    flag.PrintDefaults()
}

func init() {
    flag.StringVar(&zmqport_, "zmqport", "ipc:///run/tuer/door_keyname.ipc", "zmq socket path")
    flag.StringVar(&door_keys_file_, "keysfile", "/flash/keys", "door keys file")
    //~ flag.BoolVar(&use_syslog_, "syslog", false, "log to syslog local2 facility")
    flag.BoolVar(&start_server_, "server", false, "open 0mq socket and listen to requests")
    flag.Usage = usage
    flag.Parse()
}

func getFileMTime(filename string ) (int64, error) {
    keysfile, err := os.Open(filename)
    if err != nil { return 0, err }
    defer keysfile.Close()
    stat, err := keysfile.Stat()
    if err != nil { return 0, err }
    return stat.ModTime().Unix(), nil
}

func main() {
    knstore := new(KeyNickStore)
    err := knstore.LoadKeysFile(door_keys_file_)
    if err != nil { panic(err) }
    door_keys_mtime, err := getFileMTime(door_keys_file_)
    if err != nil { panic(err) }
    
    for _, key := range(flag.Args()) {
        nick, err := knstore.LookupHexKeyNick(key)
        if err != nil {
            fmt.Printf("ERROR: %s for key %s\n", err.Error(), key)
        } else {
            fmt.Println(key,nick)
        }
    }

    if ! start_server_ {
       os.Exit(0)
    }

    zmqctx, zmqchans := ZmqsInit(zmqport_)
    if zmqchans == nil {
        os.Exit(0)
    }
    defer zmqchans.Close()
    defer zmqctx.Close()

    //~ if use_syslog_ {
        //~ var logerr error
        //~ syslog_, logerr = syslog.NewLogger(syslog.LOG_INFO | syslog.LOG_LOCAL2, 0)
        //~ if logerr != nil { panic(logerr) }
        //~ syslog_.Print("started")
        //~ defer syslog_.Print("exiting")
    //~ }

    for keybytes := range(zmqchans.In()) {
        current_door_keys_mtime, err := getFileMTime(door_keys_file_)
        if err == nil && current_door_keys_mtime > door_keys_mtime {
            door_keys_mtime = current_door_keys_mtime
            knstore.LoadKeysFile(door_keys_file_)
        }
        nick, err := knstore.LookupHexKeyNick(string(keybytes[0]))
        if err != nil {
            zmqchans.Out() <- [][]byte{[]byte("ERROR"), []byte(err.Error())}
        } else {
            zmqchans.Out() <- [][]byte{[]byte("RESULT"), keybytes[0], []byte(nick)}
        }
    }
}

// (c) Bernhard Tittelbach, 2013

package main

import (
    "fmt"
    "os"
    "flag"
    "log"
    "bufio"
    "bytes"
)


// ---------- Main Code -------------

var (
    cmd_port_ string
    sub_port_ string
)

func usage() {
    fmt.Fprintf(os.Stderr, "Usage: door_client_zmq\n")
    flag.PrintDefaults()
}

func init() {
    flag.StringVar(&cmd_port_, "cmdport", "tcp://127.0.0.1:3232", "zmq command socket path")
    flag.StringVar(&sub_port_, "pubport", "pgm://233.252.1.42:4242", "zmq subscribe/listen socket path")
    flag.Usage = usage
    flag.Parse()
}

func LineReader(out chan <- [][]byte, stdin * os.File) {
    linescanner := bufio.NewScanner(stdin)
    linescanner.Split(bufio.ScanLines)
    defer close(out)
    for linescanner.Scan() {
        if err := linescanner.Err(); err != nil {
            log.Print(err)
            return
        }
        //text := bytes.Fields(linescanner.Bytes()) //this returns a slice (aka pointer, no array deep-copy here)
        text := bytes.Fields([]byte(linescanner.Text())) //this allocates a string and slices it -> no race-condition with overwriting any data
        if len(text) == 0 {
            continue
        }
        out <- text
    }
}

func ByteArrayToString(bb [][]byte) string {
    b := bytes.Join(bb, []byte(" "))
    return string(b)
}

func main() {
    zmqctx, cmd_chans, sub_chans := ZmqsInit(cmd_port_, sub_port_)
    defer cmd_chans.Close()
    defer sub_chans.Close()
    defer zmqctx.Close()
    var listen bool
    var ignore_next bool

    user_input_chan := make(chan [][]byte, 1)
    go LineReader(user_input_chan, os.Stdin)
    defer os.Stdin.Close()

    for {
        select {
        case input, input_open := (<- user_input_chan):
            if input_open {
                if len(input) == 0 { continue }
                 switch string(input[0]) {
                    case "listen":
                        listen = true
                        fmt.Println("Now listening")
                    case "quit":
                        os.Exit(0)
                    default:
                        ignore_next = true
                        cmd_chans.Out() <- input
                        log.Print("input sent")
                        reply := <- cmd_chans.In()
                        log.Print("reply received")
                        fmt.Println(ByteArrayToString(reply))
                }
            } else {
                os.Exit(0)
            }
        case pubsubstuff := <- sub_chans.In():
            log.Print("pubsubstuff",pubsubstuff)
            if len(pubsubstuff) == 0 { continue}
            if ignore_next {
                ignore_next = false
                continue
            }
            if listen {
                fmt.Println(pubsubstuff)
            }
        }
    }
}

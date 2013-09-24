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
    flag.StringVar(&cmd_port_, "cmdport", "tcp://localhost:5555", "zmq command socket path")
    flag.StringVar(&sub_port_, "pubport", "gmp://*:6666", "zmq subscribe/listen socket path")
    flag.Usage = usage
    flag.Parse()
}

func LineReader(out chan <- [][]byte, stdin * os.File) {
    linescanner := bufio.NewScanner(stdin)
    linescanner.Split(bufio.ScanLines)
    for linescanner.Scan() {
        if err := linescanner.Err(); err != nil {
            log.Print(err)
            close(out)
            return
        }
        text := bytes.Fields([]byte(linescanner.Text()))
        if len(text) == 0 {
            continue
        }
        out <- text
    }
}

func main() { 
    cmd_chans, sub_chans := ZmqsInit(cmd_port_, sub_port_)
    defer cmd_chans.Close()
    defer sub_chans.Close()
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
                        break
                    default:
                        ignore_next = true
                        cmd_chans.Out() <- input
                        fmt.Println( <- cmd_chans.In())
                }
            } else {
                break
            }
        case pubsubstuff := <- sub_chans.In():
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

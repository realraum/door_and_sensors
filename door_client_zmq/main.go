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
    cmd_method_ string
    cmd_user_ string
)

func init() {
    flag.StringVar(&cmd_port_, "cmdport", "ipc:///run/tuer/door_cmd.ipc", "zmq command socket path")
    flag.StringVar(&sub_port_, "pubport", "tcp://zmqbroker.realraum.at:4242", "zmq subscribe/listen socket path")
    flag.StringVar(&cmd_method_, "method", "", "zmq cmd method")
    flag.StringVar(&cmd_user_, "usernick", "", "zmq cmd user identity")
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
    var ignore_next uint = 0

    user_input_chan := make(chan [][]byte, 1)
    go LineReader(user_input_chan, os.Stdin)
    defer os.Stdin.Close()

    for {
        select {
        case input, input_open := (<- user_input_chan):
            if input_open {
                if len(input) == 0 { continue }
                 switch string(input[0]) {
                    case "help", "?":
                        fmt.Println("Available Commands: help, listen, quit. Everything else is passed through to door daemon")
                    case "listen":
                        listen = true
                        fmt.Println("Now listening, @ are broadcasts")
                    case "quit":
                        os.Exit(0)
                    default:
                        ignore_next = 2
                        cmd_chans.Out() <- input
                        reply := <- cmd_chans.In()
                        fmt.Println(">",ByteArrayToString(reply))
                }
            } else {
                os.Exit(0)
            }
        case pubsubstuff := <- sub_chans.In():
            if len(pubsubstuff) == 0 { continue}
            if ignore_next > 0 {
                ignore_next--
                continue
            }
            if listen {
                fmt.Println("@",ByteArrayToString(pubsubstuff))
            }
        }
    }
}

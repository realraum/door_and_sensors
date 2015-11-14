// (c) Bernhard Tittelbach, 2013, 2015

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strings"
)

// ---------- Main Code -------------

type SerialLine []string

var (
	cmd_port_ string
	sub_port_ string
)

const DEFAULT_TUER_DOORCMD_SOCKETPATH string = "/run/tuer/door_cmd.unixpacket"

func init() {
	flag.StringVar(&cmd_port_, "cmdport", DEFAULT_TUER_DOORCMD_SOCKETPATH, "rpc command socket path")
	flag.Parse()
}

func LineReader(out chan<- SerialLine, stdin *os.File) {
	linescanner := bufio.NewScanner(stdin)
	linescanner.Split(bufio.ScanLines)
	defer close(out)
	for linescanner.Scan() {
		if err := linescanner.Err(); err != nil {
			log.Print(err)
			return
		}
		//text := bytes.Fields(linescanner.Bytes()) //this returns a slice (aka pointer, no array deep-copy here)
		text := strings.Fields(linescanner.Text()) //this allocates a string and slices it -> no race-condition with overwriting any data
		if len(text) == 0 {
			continue
		}
		out <- text
	}
}

func ConnectToRPCServer(socketpath string) (c *rpc.Client) {
	var err error
	c, err = rpc.Dial("unixpacket", socketpath)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func main() {
	rpcc := ConnectToRPCServer(cmd_port_)
	defer rpcc.Close()
	// var listen bool
	// var ignore_next uint = 0

	user_input_chan := make(chan SerialLine, 1)
	go LineReader(user_input_chan, os.Stdin)
	defer os.Stdin.Close()

	for {
		select {
		case input, input_open := (<-user_input_chan):
			if input_open {
				if len(input) == 0 {
					continue
				}
				switch input[0] {
				case "help", "?":
					fmt.Println("Available Commands: help, listen, quit. Everything else is passed through to door daemon")
				case "listen":
					// listen = true
					// fmt.Println("Now listening, @ are broadcasts")
					fmt.Println("Listening not supported anymore, please subscribe to mqtt 'door' messages")
				case "quit":
					os.Exit(0)
				default:
					// ignore_next = 2
					var reply SerialLine
					if err := rpcc.Call("Frontdoor.SendCmd", input, &reply); err == nil {
						fmt.Println(">", strings.Join(reply, " "))
					} else {
						fmt.Println("!!!", err.Error())
					}

				}
			} else {
				os.Exit(0)
			}
			// case pubsubstuff := <-sub_chans.In():
			// 	if len(pubsubstuff) == 0 {
			// 		continue
			// 	}
			// 	if ignore_next > 0 {
			// 		ignore_next--
			// 		continue
			// 	}
			// 	if listen {
			// 		fmt.Println("@", strings.Join(pubsubstuff, " "))
			// 	}
		}
	}
}

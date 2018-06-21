// (c) Bernhard Tittelbach, 2013, 2015, 2018

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
	cgiven_   bool //we ignore this
)

const DEFAULT_TUER_DOORCMD_SOCKETPATH string = "/run/tuer/door_cmd.unixpacket"

func init() {
	flag.StringVar(&cmd_port_, "cmdport", DEFAULT_TUER_DOORCMD_SOCKETPATH, "rpc command socket path")
	flag.BoolVar(&cgiven_, "c", false, "this is ignored") //filter out -c
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

	var append_to_all_fwd_cmds_ []string = flag.Args()
	user_input_chan := make(chan SerialLine, 1)
	defer os.Stdin.Close()

	ssh_orig_cmd := os.Getenv("SSH_ORIGINAL_COMMAND")
	if len(ssh_orig_cmd) == 0 || len(ssh_orig_cmd) > 50 {
		//read from stdin
		go LineReader(user_input_chan, os.Stdin)
	} else {
		//use first argument given in SSH_ORIGINAL_COMMAND as input, ignore the rest
		args := strings.Fields(ssh_orig_cmd)
		user_input_chan <- []string{args[0]}
	}

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
					fmt.Println("Listening not supported anymore, please subscribe to mqtt 'door' messages")
				case "quit":
					os.Exit(0)
				default:
					input = append(input, append_to_all_fwd_cmds_...)
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
		}
	}
}

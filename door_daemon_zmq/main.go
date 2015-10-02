// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	"os"
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
	cmd_port_      string
	pub_port_      string
	door_tty_path_ string
	enable_syslog_ bool
	enable_debug_  bool
)

// TUER_ZMQDOORCMD_ADDR
// TUER_ZMQDOOREVTS_LISTEN_ADDR
// TUER_TTY_PATH

const (
	DEFAULT_TUER_ZMQDOORCMD_ADDR         string = "ipc:///run/tuer/door_cmd.ipc"
	DEFAULT_TUER_ZMQDOOREVTS_LISTEN_ADDR string = "tcp://*:4242"
	DEFAULT_TUER_TTY_PATH                string = "/dev/door"
)

func init() {
	flag.BoolVar(&enable_syslog_, "syslog", false, "enable logging to syslog")
	flag.BoolVar(&enable_debug_, "debug", false, "enable debug output")
	flag.Parse()
}

func EnvironOrDefault(envvarname, defvalue string) string {
	if len(os.Getenv(envvarname)) > 0 {
		return os.Getenv(envvarname)
	} else {
		return defvalue
	}
}

func main() {
	if enable_syslog_ {
		LogEnableSyslog()
	}
	if enable_debug_ {
		LogEnableDebuglog()
	}
	Syslog_.Print("started")
	defer Syslog_.Print("exiting")

	zmqctx, cmd_chans, pub_chans := ZmqsInit(
		EnvironOrDefault("TUER_ZMQDOORCMD_ADDR", DEFAULT_TUER_ZMQDOORCMD_ADDR),
		EnvironOrDefault("TUER_ZMQDOOREVTS_LISTEN_ADDR", DEFAULT_TUER_ZMQDOOREVTS_LISTEN_ADDR))
	defer cmd_chans.Close()
	defer pub_chans.Close()
	defer zmqctx.Close()

	serial_wr, serial_rd, err := OpenAndHandleSerial(EnvironOrDefault("TUER_TTY_PATH", DEFAULT_TUER_TTY_PATH), 0)
	defer close(serial_wr)
	if err != nil {
		panic(err)
	}

	workaround_in_chan := WorkaroundFirmware(serial_wr)
	defer close(workaround_in_chan)

	var next_incoming_serial_is_client_reply bool
	timeout_chan := make(chan bool)
	defer close(timeout_chan)
	for {
		select {
		case incoming_ser_line, is_notclosed := <-serial_rd:
			if is_notclosed {
				//~ if Syslog_ != nil { Syslog_.Print(ByteArrayToString(incoming_ser_line)) }
				if Syslog_ != nil {
					Syslog_.Printf("%s", incoming_ser_line)
				}
				if next_incoming_serial_is_client_reply {
					next_incoming_serial_is_client_reply = false
					cmd_chans.Out() <- incoming_ser_line
				}
				pub_chans.Out() <- incoming_ser_line
				workaround_in_chan <- incoming_ser_line
			} else {
				Syslog_.Print("serial device disappeared, exiting")
				os.Exit(1)
			}
		case tv, timeout_notclosed := <-timeout_chan:
			if timeout_notclosed && tv && next_incoming_serial_is_client_reply {
				next_incoming_serial_is_client_reply = false
				cmd_chans.Out() <- [][]byte{[]byte("ERROR"), []byte("No reply from firmware")}
			}
		case incoming_request, ic_notclosed := <-cmd_chans.In():
			if !ic_notclosed {
				Syslog_.Print("zmq socket died, exiting")
				os.Exit(2)
			}
			if string(incoming_request[0]) == "log" {
				if len(incoming_request) < 2 {
					cmd_chans.Out() <- [][]byte{[]byte("ERROR"), []byte("argument missing")}
					continue
				}
				Syslog_.Printf("Log: %s", incoming_request[1:])
				cmd_chans.Out() <- [][]byte{[]byte("Ok")}
				continue
			}
			Syslog_.Printf("%s", incoming_request)
			if err := HandleCommand(incoming_request, serial_wr, serial_rd); err != nil {
				out_msg := [][]byte{[]byte("ERROR"), []byte(err.Error())}
				cmd_chans.Out() <- out_msg
			} else {
				pub_chans.Out() <- incoming_request
				next_incoming_serial_is_client_reply = true
				go func() { time.Sleep(3 * time.Second); timeout_chan <- true }()
			}
		}
	}
}

// (c) Bernhard Tittelbach, 2013, 2015

package main

import (
	"errors"
	"flag"
	"os"
	"time"

	"github.com/hishboy/gocommons/lang"
)

// ---------- Main Code -------------

var (
	enable_syslog_ bool
	enable_debug_  bool
)

type SerialLine []string

const (
	DEFAULT_TUER_DOORCMD_SOCKETPATH string = "/run/tuer/door_cmd.unixpacket"
	DEFAULT_R3_MQTT_BROKER          string = "tcp://mqtt.realraum.at:1883"
	DEFAULT_TUER_TTY_PATH           string = "/dev/door"
	DEFAULT_TUER_KEYSFILE_PATH      string = "/flash/keys/keys"
)

func init() {
	flag.BoolVar(&enable_syslog_, "syslog", false, "enable logging to syslog")
	flag.BoolVar(&enable_debug_, "debug", false, "enable debug output")
	flag.Parse()
}

func main() {
	// Logging
	if enable_syslog_ {
		LogEnableSyslog()
	}
	if enable_debug_ {
		LogEnableDebuglog()
	}
	Syslog_.Print("started")
	defer Syslog_.Print("exiting")

	// RPC Server
	send_me_cmds := make(chan CmdAndReply, 10)
	os.Remove(EnvironOrDefault("TUER_DOORCMD_SOCKETPATH", DEFAULT_TUER_DOORCMD_SOCKETPATH))
	go StartRPCServer(send_me_cmds, EnvironOrDefault("TUER_DOORCMD_SOCKETPATH", DEFAULT_TUER_DOORCMD_SOCKETPATH))
	defer os.Remove(EnvironOrDefault("TUER_DOORCMD_SOCKETPATH", DEFAULT_TUER_DOORCMD_SOCKETPATH))

	knstore, err := NewKeyNickStore(EnvironOrDefault("TUER_KEYSFILE_PATH", DEFAULT_TUER_KEYSFILE_PATH))
	if err != nil {
		panic(err) // todo: or not
	}

	// Connection to Door Firmware
	serial_wr, serial_rd, err := OpenAndHandleSerial(EnvironOrDefault("TUER_TTY_PATH", DEFAULT_TUER_TTY_PATH), 115200)
	defer close(serial_wr)
	if err != nil {
		panic(err)
	}

	// Connect to MQTT Broker
	// once we have connected, get status and distribute to MQTT Broker
	// Until we are connected, keep channels clean
	publish_line_as_event_to_mqtt := make(chan SerialLine, 50)
	run_on_connect := func() { HandleCommand([]string{"status"}, serial_wr, serial_rd) }
	go ConnectChannelToMQTT(publish_line_as_event_to_mqtt, EnvironOrDefault("R3_MQTT_BROKER", DEFAULT_R3_MQTT_BROKER), knstore, run_on_connect)

	// Start Workaround for Door Firmware "bug"
	workaround_in_chan := WorkaroundFirmware(serial_wr)
	defer close(workaround_in_chan)

	// Start Firmware answer timeout timer
	timeout_timer := time.NewTimer(0)
	timeout_timer.Stop()
	waiting_for_reply := lang.NewQueue()
	for {
		select {
		case incoming_ser_line, is_notclosed := <-serial_rd:
			if is_notclosed {
				//~ if Syslog_ != nil { Syslog_.Print(ByteArrayToString(incoming_ser_line)) }
				if Syslog_ != nil {
					Syslog_.Printf("%s", incoming_ser_line)
				}
				if waiting_for_reply.Len() > 0 {
					oldrequest := waiting_for_reply.Poll().(CmdAndReply)
					oldrequest.backchan <- incoming_ser_line
					close(oldrequest.errchan)
					close(oldrequest.backchan)
				}
				workaround_in_chan <- incoming_ser_line
				publish_line_as_event_to_mqtt <- incoming_ser_line
			} else {
				Syslog_.Print("serial device disappeared, exiting")
				os.Exit(1)
			}
		case _, timeout_notclosed := <-timeout_timer.C:
			if timeout_notclosed {
				for waiting_for_reply.Len() > 0 {
					oldrequest := waiting_for_reply.Poll().(CmdAndReply)
					oldrequest.errchan <- errors.New("Timeout reached. No reply from firmware")
					close(oldrequest.errchan)
					close(oldrequest.backchan)
				}
			}
		case incoming_cmd, ic_notclosed := <-send_me_cmds:
			if !ic_notclosed {
				Syslog_.Print("rpc chan closed, exiting")
				os.Exit(2)
			}
			if incoming_cmd.cmd[0] == "log" {
				if len(incoming_cmd.cmd) < 2 {
					incoming_cmd.errchan <- errors.New("argument missing")
					close(incoming_cmd.errchan)
					close(incoming_cmd.backchan)
					continue
				}
				Syslog_.Printf("Log: %s", incoming_cmd.cmd[1:])
				incoming_cmd.backchan <- SerialLine{"Ok"}
				close(incoming_cmd.errchan)
				close(incoming_cmd.backchan)
				continue
			}
			Syslog_.Printf("%s", incoming_cmd.cmd)
			if err := HandleCommand(incoming_cmd.cmd, serial_wr, serial_rd); err != nil {
				incoming_cmd.errchan <- err
				close(incoming_cmd.errchan)
				close(incoming_cmd.backchan)
			} else {
				publish_line_as_event_to_mqtt <- incoming_cmd.cmd
				waiting_for_reply.Push(incoming_cmd)
				timeout_timer.Reset(3 * time.Second)
			}
		}
	}
}

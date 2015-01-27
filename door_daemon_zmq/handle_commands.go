// (c) Bernhard Tittelbach, 2013

package main

import (
	"errors"
	"time"
)

type DoorCmdHandler struct {
	Checker      func([][]byte) error
	FirmwareChar string
}

var cmdToDoorCmdHandler = map[string]DoorCmdHandler{
	"open":   DoorCmdHandler{checkCmdDoorControl, "o"},
	"close":  DoorCmdHandler{checkCmdDoorControl, "c"},
	"toggle": DoorCmdHandler{checkCmdDoorControl, "t"},
	"status": DoorCmdHandler{checkCmdNoArgs, "s"},
}

// ---------- Talk with Firmware directly in response to stuff it sends ------------

func WorkaroundFirmware(serial_wr chan string) (in chan [][]byte) {
	in = make(chan [][]byte, 5)
	go func() {
		var last_state_time time.Time
		var last_door_state string
		for firmware_output := range in {
			Debug_.Printf("WorkaroundFirmware Input: %s", firmware_output)
			if len(firmware_output) > 1 && string(firmware_output[0]) == "State:" {
				last_state_time = time.Now()
				last_door_state = string(firmware_output[1])
			}
			if len(firmware_output) == 5 &&
				string(firmware_output[0]) == "Info(ajar):" &&
				string(firmware_output[4]) == "ajar" &&
				time.Now().Sub(last_state_time) < 30*time.Second &&
				last_door_state == "timeout_after_open" {
				//If we were in state "timeout_after_open" and within 30s the door was openend anyway,
				//we send another "open" command
				serial_wr <- cmdToDoorCmdHandler["open"].FirmwareChar
				Syslog_.Print("Telling Firmware to open, since door was ajar'ed after timeout_after_open")
			}
		}
	}()
	return in
}

// ---------- ZMQ Command Handling Code -------------

func checkCmdDoorControl(tokens [][]byte) error {
	doorctrl_usage := "syntax: <open|close|toggle> <method> <nickname>"
	if len(tokens) < 2 || len(tokens) > 3 {
		return errors.New(doorctrl_usage)
	}
	cmd := string(tokens[0])
	if !(cmd == "open" || cmd == "close" || cmd == "toggle") {
		return errors.New(doorctrl_usage)
	}
	method := string(tokens[1])
	if !(method == "Button" || method == "ssh" || method == "SSH" || method == "Phone") {
		return errors.New("method must be one either Button, SSH or Phone")
	}
	if (len(tokens) == 2 || len(tokens[2]) == 0) && method != "Button" {
		return errors.New("Operator nickname must be given")
	}
	return nil
}

func checkCmdNoArgs(tokens [][]byte) error {
	if len(tokens) != 1 {
		return errors.New("command does not accept arguments")
	}
	return nil
}

func HandleCommand(tokens [][]byte, serial_wr chan string, serial_rd chan [][]byte) error {
	if len(tokens) < 1 {
		return errors.New("No Command to handle")
	}

	dch, present := cmdToDoorCmdHandler[string(tokens[0])]
	if !present {
		return errors.New("Unknown Command")
	}

	if err := dch.Checker(tokens); err != nil {
		return err
	}

	serial_wr <- dch.FirmwareChar
	return nil
}

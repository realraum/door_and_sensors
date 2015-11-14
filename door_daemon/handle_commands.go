// (c) Bernhard Tittelbach, 2013

package main

import (
	"errors"
	"time"
)

type DoorCmdHandler struct {
	Checker      func(SerialLine) error
	FirmwareChar string
}

var cmdToDoorCmdHandler = map[string]DoorCmdHandler{
	"open":   DoorCmdHandler{checkCmdDoorControl, "o"},
	"close":  DoorCmdHandler{checkCmdDoorControl, "c"},
	"toggle": DoorCmdHandler{checkCmdDoorControl, "t"},
	"status": DoorCmdHandler{checkCmdNoArgs, "s"},
}

// ---------- Talk with Firmware directly in response to stuff it sends ------------

// The problem is this:
// If the key/motor turns just far enough so that the door is unlocked,
// but still get's blocked on the way (because the door clamped)
// the firmware will enter state "timeout_after_open" instead of "open"
// In this state, manual moving the key/cogwheel will not trigger the state "manual_movement"
// Users however will not notice that the door is in an error state and needs to manually be moved to the real open position,
// they have after all, already successfully entered the room.
// Without that information however, the r3-event-broker daemon, cannot tell if the door is being locked manuall from the inside
// or if someone else is locking the door for other reasons.
// Thus, if the door has been opened imperfectly and is then closed manually, it will trigger a presence=false event and switch off the lights.
//
// As Workaround, the door daemon watches for "timeout_after_open" events.
// If one is detected and followed by an "door is now ajar" info, we tell the firmware
// to open the door, causing it to move out of the error state and into the final open key/cogwheel position.
func WorkaroundFirmware(serial_wr chan string) (in chan SerialLine) {
	in = make(chan SerialLine, 5)
	go func() {
		var last_state_time time.Time
		var last_door_state string
		for firmware_output := range in {
			Debug_.Printf("WorkaroundFirmware Input: %s", firmware_output)
			if len(firmware_output) > 1 && firmware_output[0] == "State:" {
				last_state_time = time.Now()
				last_door_state = firmware_output[1]
			}
			if len(firmware_output) == 5 &&
				firmware_output[0] == "Info(ajar):" &&
				firmware_output[4] == "ajar" &&
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

func checkCmdDoorControl(tokens SerialLine) error {
	doorctrl_usage := "syntax: <open|close|toggle> <method> <nickname>"
	if len(tokens) < 2 || len(tokens) > 3 {
		return errors.New(doorctrl_usage)
	}
	cmd := tokens[0]
	if !(cmd == "open" || cmd == "close" || cmd == "toggle") {
		return errors.New(doorctrl_usage)
	}
	method := tokens[1]
	if !(method == "Button" || method == "ssh" || method == "SSH" || method == "Phone") {
		return errors.New("method must be one either Button, SSH or Phone")
	}
	if (len(tokens) == 2 || len(tokens[2]) == 0) && method != "Button" {
		return errors.New("Operator nickname must be given")
	}
	return nil
}

func checkCmdNoArgs(tokens SerialLine) error {
	if len(tokens) != 1 {
		return errors.New("command does not accept arguments")
	}
	return nil
}

func HandleCommand(tokens SerialLine, serial_wr chan string, serial_rd chan SerialLine) error {
	if len(tokens) < 1 {
		return errors.New("No Command to handle")
	}

	dch, present := cmdToDoorCmdHandler[tokens[0]]
	if !present {
		return errors.New("Unknown Command")
	}

	if err := dch.Checker(tokens); err != nil {
		return err
	}

	serial_wr <- dch.FirmwareChar
	return nil
}

// (c) Bernhard Tittelbach, 2013, 2018

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
	"open":             DoorCmdHandler{checkCmdDoorControl, "o"},
	"close":            DoorCmdHandler{checkCmdDoorControl, "c"},
	"toggle":           DoorCmdHandler{checkCmdDoorControl, "t"},
	"openfrominside":   DoorCmdHandler{checkCmdDoorControl, "o"},
	"closefrominside":  DoorCmdHandler{checkCmdDoorControl, "c"},
	"togglefrominside": DoorCmdHandler{checkCmdDoorControl, "t"},
	"status":           DoorCmdHandler{checkCmdIgnoreArgs, "s"},
}

// ---------- Talk with Firmware directly in response to stuff it sends ------------

// The problem is this:
// If the key/motor turns just far enough so that the door is unlocked,
// but still get's blocked on the way (because the door clamped)
// the firmware will enter state "timeout_after_open" instead of "open"
// In this state, manual moving the key/cogwheel will not trigger the state "manual_movement"
// Users however will not notice that the door is in an error state and needs to manually be moved to the real open position,
// they have after all, already successfully entered the room.
//
// While the firmware remains in the error-state, outside software like the r3-event-broker daemon, lacks the information to determine if the door is being locked manually from the inside since "manual_movement" is never sent.
//
// As a workaround for this behaviour, the door daemon watches now for "timeout_after_open" events.
// If one is detected and followed by an "door is now ajar" info, we tell the firmware
// to open the door, causing it to move out of the error state and into the final open key/cogwheel position.
//
//
//
// Another problem is the following:
// If there are pebbles in the door and the closed door is not completely flush with it's frame,
// the lock won't close properly. This causes problems later on.
//
// To encourage users to clean out the pebbles from the door until the door correctly locks,
// we open the door again, if there was a closing timeout.
//
// this is of course experimental.
// Also this is implemented in door_daemon and not in firmware, so this feature can be disabled just by unplugging USB if needed.
//
func WorkaroundFirmware(serial_wr chan string) (in chan SerialLine) {
	in = make(chan SerialLine, 5)
	go func() {
		var last_state_time time.Time
		var last_timeout_after_close_time time.Time
		var last_door_state string
		for firmware_output := range in {
			Debug_.Printf("WorkaroundFirmware Input: %s", firmware_output)
			if len(firmware_output) > 1 && firmware_output[0] == "State:" {
				if firmware_output[1] == "timeout_after_close" {
					if time.Now().Sub(last_timeout_after_close_time) > 90*time.Second {
						// only do this once for every error within 90s. e.g. ignore the next error in case people are just to tired to do the easy to do right thing
						serial_wr <- cmdToDoorCmdHandler["open"].FirmwareChar //open door again, so pebbles are cleaned out and door is shut again
					}
					last_timeout_after_close_time = time.Now()
				}
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

// ---------- Command Handling Code -------------

func checkCmdDoorControl(tokens SerialLine) error {
	doorctrl_usage := "syntax: <open|close|toggle>[frominside] <method> <nickname>"
	if len(tokens) < 2 || len(tokens) > 3 {
		return errors.New(doorctrl_usage)
	}
	cmd := tokens[0]
	if !(cmd == "open" || cmd == "close" || cmd == "toggle" || cmd == "openfrominside" || cmd == "closefrominside" || cmd == "togglefrominside") {
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

func checkCmdIgnoreArgs(tokens SerialLine) error {
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

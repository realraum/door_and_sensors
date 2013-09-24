// (c) Bernhard Tittelbach, 2013

package main

import (
    "errors"
)

type DoorCmdHandler struct {
    Checker func([][]byte)(error)
    FirmwareChar string
}

var cmdToDoorCmdHandler = map[string]DoorCmdHandler {
  "open": DoorCmdHandler{ checkCmdDoorControl, "o"},
  "close": DoorCmdHandler{ checkCmdDoorControl, "c"},
  "toggle": DoorCmdHandler{ checkCmdDoorControl, "t"},
  "status": DoorCmdHandler{ checkCmdDoorControl, "s"},
}

// ---------- Command Handling Code -------------

func checkCmdDoorControl(tokens [][]byte) (error) {
    doorctrl_usage := "syntax: <open|close|toggle> <method> <nickname>"
    if len(tokens) != 3 {
        return errors.New(doorctrl_usage)
    }
    cmd := string(tokens[0])
    if ! (cmd == "open" || cmd == "close" || cmd == "toggle") {
        return errors.New(doorctrl_usage)
    }
    method := string(tokens[1])
    if ! (method == "Button" || method == "ssh" || method == "SSH" || method == "Phone") {
        return errors.New("method must be one either Button, SSH or Phone")
    }
    if len(tokens[2]) == 0 && method != "Button" {
        return errors.New("Operator nickname must be given")
    }
    return nil
}

func checkCmdStatus(tokens [][]byte) (error) {
    if len(tokens) != 1 {
        return errors.New("status command does not accept arguments")
    }
    return nil
}

func HandleCommand(tokens [][]byte, serial_wr chan string, serial_rd chan [][]byte) error {
    if len(tokens) < 1 {
        return errors.New("No Command to handle")
    }

    dch, present := cmdToDoorCmdHandler[string(tokens[0])]
    if ! present {
        return errors.New("Unknown Command")
    }

    if err := dch.Checker(tokens); err != nil {
        //return error to sender
        return nil
    }

    serial_wr <- dch.FirmwareChar
    return nil
}

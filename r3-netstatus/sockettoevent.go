// (c) Bernhard Tittelbach, 2013

package main

import (
    pubsub "github.com/tuxychandru/pubsub"
    "regexp"
    "strconv"
    "bufio"
    "time"
    //~ "./brain"
    "net"
    )

var (
	re_presence_    *regexp.Regexp     = regexp.MustCompile("Presence: (yes|no)(?:, (opened|closed), (.+))?")
	re_state_      *regexp.Regexp     = regexp.MustCompile("State: (closed|opened|manual movement|error|reset|timeout after open|timeout after close|opening|closing).*")
	re_infocard_      *regexp.Regexp     = regexp.MustCompile("Info\(card\): card\(([a-fA-F0-9]+)\) (found|not found).*")
	re_infoajar_      *regexp.Regexp     = regexp.MustCompile("Info\(ajar\): door is now (ajar|shut)")
	re_command_     *regexp.Regexp     = regexp.MustCompile("(open|close|toggle|reset)(?: +(Card|Phone|SSH|ssh))?(?: +(.+))?")
	re_button_      *regexp.Regexp     = regexp.MustCompile("PanicButton|button\\d?")
	re_temp_        *regexp.Regexp     = regexp.MustCompile("temp0: (\\d+\\.\\d+)")
	re_photo_       *regexp.Regexp     = regexp.MustCompile("photo0: (\\d+)")
)


type PresenceUpdate struct {
    Present bool
    Ts int64
}

type DoorStatusUpdate struct {
    Locked bool
    Shut bool
    Ts int64
}

type DoorCommandEvent struct {
    Command string
    Using string
    Who string
    Ts int64
}

type ButtonPressUpdate struct {
    Buttonindex int
    Ts int64
}

type TempSensorUpdate struct {
    Sensorindex int
    Value float64
    Ts int64
}

type IlluminationSensorUpdate struct {
    Sensorindex int
    Value int64
    Ts int64
}

type TimeTick struct {
    Ts int64
}

type MovementSensorUpdate struct {
    Sensorindex int
    Ts int64
}

func ParseSocketInputLine(line string, ps *pubsub.PubSub) { //, brn *brain.Brain) {
    match_presence := re_presence_.FindStringSubmatch(line)
    match_status := re_status_.FindStringSubmatch(line)
    match_command := re_command_.FindStringSubmatch(line)
    match_button := re_button_.FindStringSubmatch(line)
    match_temp := re_temp_.FindStringSubmatch(line)
    match_photo := re_photo_.FindStringSubmatch(line)

    //~ log.Println("ParseSocketInputLine",line)
    var tidbit interface{}
    ts := time.Now().Unix()
    if match_presence != nil {
        if match_presence[2] != "" { ps.Pub(DoorStatusUpdate{match_presence[2] == "closed", true, ts}, "door"); }
        tidbit = PresenceUpdate{match_presence[1] == "yes", ts}
        //~ brn.Oboite("presence", tidbit)
        ps.Pub(tidbit, "presence")
	} else if match_status != nil {
        tidbit = DoorStatusUpdate{match_status[1] == "closed", match_status[3] == "shut", ts}
        //~ brn.Oboite("door", tidbit)
        ps.Pub(tidbit, "door")
	} else if match_command != nil {
        tidbit = DoorCommandEvent{match_command[1], match_command[2], match_command[3], ts}
        //~ brn.Oboite("doorcmd", tidbit)
        ps.Pub(tidbit, "door")
	} else if match_button != nil {
        //~ brn.Oboite("button0", ts)
        ps.Pub(ButtonPressUpdate{0, ts}, "buttons")
	} else if match_temp != nil {
		newtemp, err := strconv.ParseFloat((match_temp[1]), 32)
		if err == nil {
            //~ brn.Oboite( "temp0", newtemp)
            ps.Pub(TempSensorUpdate{0, newtemp, ts}, "sensors")
		}
	} else if match_photo != nil {
		newphoto, err := strconv.ParseInt(match_photo[1], 10, 32)
		if err == nil {
            //~ brn.Oboite("photo0", newphoto)
            ps.Pub(IlluminationSensorUpdate{0, newphoto, ts}, "sensors")
		}
	} else if line == "movement" {
        //~ brn.Oboite("movement", ts)
        ps.Pub(MovementSensorUpdate{0, ts}, "movements")
	}
}

func ReadFromUSocket(path string, c chan string) {
ReOpenSocket:
	for {
		presence_socket, err := net.Dial("unix", path)
		if err != nil {
			//Waiting on Socket
			time.Sleep(5 * time.Second)
			continue ReOpenSocket
		}
		presence_reader := bufio.NewReader(presence_socket)
		for {
			line, err := presence_reader.ReadString('\n')
			if err != nil {
				//Socket closed
				presence_socket.Close()
				continue ReOpenSocket
			}
			c <- line
		}
	}
}
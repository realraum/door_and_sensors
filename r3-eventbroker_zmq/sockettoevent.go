// (c) Bernhard Tittelbach, 2013

package main

import (
    "regexp"
    "strconv"
    "time"
    //~ "./brain"
    "encoding/json"
    pubsub "github.com/tuxychandru/pubsub"
    zmq "github.com/vaughan0/go-zmq"    
    "log"
    )

var (
	re_presence_    *regexp.Regexp     = regexp.MustCompile("Presence: (yes|no)(?:, (opened|closed), (.+))?")
	re_state_      *regexp.Regexp     = regexp.MustCompile("State: (closed|opened|manual movement|error|reset|timeout after open|timeout after close|opening|closing).*")
	re_status_      *regexp.Regexp     = regexp.MustCompile("Status: (closed|opened) (closed|opened|manual movement|error|reset|timeout after open|timeout after close|opening|closing) (ajar|shut).*")
	re_infocard_      *regexp.Regexp     = regexp.MustCompile("Info\\(card\\): card\\(([a-fA-F0-9]+)\\) (found|not found).*")
	re_cardid_      *regexp.Regexp     = regexp.MustCompile("card\\(([a-fA-F0-9]+)\\)")
	re_infoajar_      *regexp.Regexp     = regexp.MustCompile("Info\\(ajar\\): door is now (ajar|shut)")
	re_command_     *regexp.Regexp     = regexp.MustCompile("(open|close|toggle|reset)(?: +(Card|Phone|SSH|ssh))?(?: +(.+))?")
	re_button_      *regexp.Regexp     = regexp.MustCompile("PanicButton|button\\d?")
	re_temp_        *regexp.Regexp     = regexp.MustCompile("temp0: (\\d+\\.\\d+)")
	re_photo_       *regexp.Regexp     = regexp.MustCompile("photo0: (\\d+)")
)


type DoorLockUpdate struct {
    DoorID byte
    Locked bool
    Ts int64
}

type DoorAjarUpdate struct {
    DoorID byte
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

func parseSocketInputLine_State(lines [][]byte, ps *pubsub.PubSub, ts int64) {
    switch string(lines[0]) {
        case "closed":
            ps.Pub(DoorLockUpdate{0, true, ts}, "door")
        case "opened":
            ps.Pub(DoorLockUpdate{0, false, ts}, "door")
        case "manual":   //movement
        case "error":
        case "reset":
            ps.Pub(DoorLockUpdate{0, true, ts}, "door")
        case "timeout":   //after open | after close
        case "opening":
        case "closing":
        default:    
    }
}


func ParseSocketInputLine(lines [][]byte, ps *pubsub.PubSub, keylookup_socket *zmq.Socket) { //, brn *brain.Brain) {
    var tidbit interface{}
    ts := time.Now().Unix()
    if len(lines) < 1 { return }
    log.Print("ParseSocketInputLine",string(lines[0]))
    switch string(lines[0]) {
        case "State:":
            if len(lines) < 2 { return }
            parseSocketInputLine_State(lines[1:], ps, ts)
        case "Status:":
            if len(lines) < 3 { return }
            tidbit = DoorLockUpdate{0, string(lines[1]) == "closed", ts}
            //~ brn.Oboite("door", tidbit)
            ps.Pub(tidbit, "door")
            tidbit = DoorAjarUpdate{0, string(lines[len(lines)-2]) == "shut", ts}
            //~ brn.Oboite("door", tidbit)
            ps.Pub(tidbit, "door")            
        case "Info(card):":
            if len(lines) < 3 { return }
            if string(lines[2]) != "found" { return }
            match_cardid := re_cardid_.FindSubmatch(lines[1])
            if len(match_cardid) > 1 {
                // PreCondition: same thread/goroutinge as created keylookup_socket !!!!
                nick, err := LookupCardIdNick(keylookup_socket, match_cardid[1])
                if err != nil {
                    Syslog_.Print("CardID Lookup Error",err)
                    nick = "Unresolvable KeyID"
                }
                // new event: toggle by user nick using card
                ps.Pub(DoorCommandEvent{"toggle", "Card", nick, ts},"doorcmd")
            }
        case "Info(ajar):":
            if len(lines) < 5 { return }
            tidbit = DoorAjarUpdate{0, string(lines[4]) == "shut", ts}
            //~ brn.Oboite("door", tidbit)
            ps.Pub(tidbit, "door")                    
        case "open", "close", "toggle", "reset":
            ps.Pub(DoorCommandEvent{string(lines[0]), string(lines[1]), string(lines[2]), ts},"doorcmd")
        case "photo0":
            newphoto, err := strconv.ParseInt(string(lines[1]), 10, 32)
            if err == nil {
                // brn.Oboite("photo0", newphoto)
                ps.Pub(IlluminationSensorUpdate{0, newphoto, ts}, "sensors")
            }
    }
}

func MakeTimeTick(ps *pubsub.PubSub) {
    ps.Pub(TimeTick{time.Now().Unix()},"time")
}

func FormatEventForSocket(event_interface interface{}) (data [][]byte, err error) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}
    return [][]byte{msg}, nil
}

    //~ match_presence := re_presence_.FindStringSubmatch(line)
    //~ match_status := re_status_.FindStringSubmatch(line)
    //~ match_command := re_command_.FindStringSubmatch(line)
    //~ match_button := re_button_.FindStringSubmatch(line)
    //~ match_temp := re_temp_.FindStringSubmatch(line)
    //~ match_photo := re_photo_.FindStringSubmatch(line)
	//~ if match_button != nil {
        //~ // brn.Oboite("button0", ts)
        //~ ps.Pub(ButtonPressUpdate{0, ts}, "buttons")
	//~ } else if match_temp != nil {
		//~ newtemp, err := strconv.ParseFloat((match_temp[1]), 32)
		//~ if err == nil {
            //~ // brn.Oboite( "temp0", newtemp)
            //~ ps.Pub(TempSensorUpdate{0, newtemp, ts}, "sensors")
		//~ }
	//~ } else if match_photo != nil {
		//~ newphoto, err := strconv.ParseInt(match_photo[1], 10, 32)
		//~ if err == nil {
            //~ // brn.Oboite("photo0", newphoto)
            //~ ps.Pub(IlluminationSensorUpdate{0, newphoto, ts}, "sensors")
		//~ }
	//~ } else if line == "movement" {
        //~ // brn.Oboite("movement", ts)
        //~ ps.Pub(MovementSensorUpdate{0, ts}, "movements")
	//~ }
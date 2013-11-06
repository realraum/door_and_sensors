// (c) Bernhard Tittelbach, 2013

package main

import (
    "regexp"
    "strconv"
    "time"
    "bytes"
    //~ "./brain"
    pubsub "github.com/tuxychandru/pubsub"
    zmq "github.com/vaughan0/go-zmq"
    r3events "svn.spreadspace.org/realraum/go.svn/r3events"
    )

var (
	//~ re_presence_    *regexp.Regexp     = regexp.MustCompile("Presence: (yes|no)(?:, (opened|closed), (.+))?")
	//~ re_state_      *regexp.Regexp     = regexp.MustCompile("State: (closed|opened|manual movement|error|reset|timeout after open|timeout after close|opening|closing).*")
	//~ re_status_      *regexp.Regexp     = regexp.MustCompile("Status: (closed|opened), (closed|opened|manual movement|error|reset|timeout after open|timeout after close|opening|closing), (ajar|shut).*")
	//~ re_infocard_      *regexp.Regexp     = regexp.MustCompile("Info\\(card\\): card\\(([a-fA-F0-9]+)\\) (found|not found).*")
	re_cardid_      *regexp.Regexp     = regexp.MustCompile("card\\(([a-fA-F0-9]+)\\)")
	//~ re_infoajar_      *regexp.Regexp     = regexp.MustCompile("Info\\(ajar\\): door is now (ajar|shut)")
	//~ re_command_     *regexp.Regexp     = regexp.MustCompile("(open|close|toggle|reset)(?: +(Card|Phone|SSH|ssh))?(?: +(.+))?")
	//~ re_button_      *regexp.Regexp     = regexp.MustCompile("PanicButton|button\\d?")
	//~ re_temp_        *regexp.Regexp     = regexp.MustCompile("temp0: (\\d+\\.\\d+)")
	//~ re_photo_       *regexp.Regexp     = regexp.MustCompile("photo0: (\\d+)")
)


func parseSocketInputLine_State(lines [][]byte, ps *pubsub.PubSub, ts int64) {
    switch string(lines[0]) {
        case "closed":
            ps.Pub(r3events.DoorLockUpdate{true, ts}, "door")
        case "opened":
            ps.Pub(r3events.DoorLockUpdate{false, ts}, "door")
        case "manual", "manual_movement":   //movement
            ps.Pub(r3events.DoorManualMovementEvent{ts}, "door")
        case "error":
            ps.Pub(r3events.DoorProblemEvent{100, string(bytes.Join(lines,[]byte(" "))),  ts}, "door")
        case "reset":
            ps.Pub(r3events.DoorLockUpdate{true, ts}, "door")
        case "timeout_after_open":
            ps.Pub(r3events.DoorProblemEvent{10, string(lines[0]), ts}, "door")
            ps.Pub(r3events.DoorLockUpdate{false, ts}, "door")
        case "timeout_after_close":
            ps.Pub(r3events.DoorProblemEvent{20, string(lines[0]), ts}, "door")
            // can't say for sure that door is locked if we ran into timeout while closing
            //~ ps.Pub(r3events.DoorLockUpdate{true, ts}, "door")
        case "opening":
        case "closing":
        default:
            Syslog_.Print("parseSocketInputLine_State: Unexpected State:", lines)
    }
}


func ParseSocketInputLine(lines [][]byte, ps *pubsub.PubSub, keylookup_socket *zmq.Socket) { //, brn *brain.Brain) {
    ts := time.Now().Unix()
    if len(lines) < 1 { return }
    Debug_.Printf("ParseSocketInputLine: %s %s",string(lines[0]), lines[1:])
    switch string(lines[0]) {
        case "State:":
            if len(lines) < 2 { return }
            parseSocketInputLine_State(lines[1:], ps, ts)
        case "Status:":
            if len(lines) < 3 { return }
            if len(lines[1]) < 4 { return }
            ps.Pub(r3events.DoorLockUpdate{string(lines[1])[0:4] != "open", ts}, "door")
            ps.Pub(r3events.DoorAjarUpdate{string(lines[len(lines)-1]) == "shut", ts}, "door")
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
                ps.Pub(r3events.DoorCommandEvent{"toggle", "Card", nick, ts},"doorcmd")
            }
        case "Info(ajar):":
            if len(lines) < 5 { return }
            ps.Pub(r3events.DoorAjarUpdate{string(lines[4]) == "shut", ts}, "door")
        case "open", "close", "toggle", "reset":
            ps.Pub(r3events.DoorCommandEvent{string(lines[0]), string(lines[1]), string(lines[2]), ts},"doorcmd")
        case "BackdoorInfo(ajar):":
            ps.Pub(r3events.BackdoorAjarUpdate{string(lines[len(lines)-1]) == "shut", ts},"door")
        case "temp0:","temp1:", "temp2:", "temp3:":
            sensorid, err := strconv.ParseInt(string(lines[0][4]), 10, 32)
            if err != nil {return }
            newtemp, err := strconv.ParseFloat(string(lines[1]), 10)
            if err != nil {return }
            ps.Pub(r3events.TempSensorUpdate{int(sensorid), newtemp, ts}, "sensors")
        case "photo0:","photo1:", "photo2:", "photo3:":
            sensorid, err := strconv.ParseInt(string(lines[0][5]), 10, 32)
            if err != nil {return }
            newphoto, err := strconv.ParseInt(string(lines[1]), 10, 32)
            if err != nil {return }
            ps.Pub(r3events.IlluminationSensorUpdate{int(sensorid), newphoto, ts}, "sensors")
        case "rh0:":
            //~ sensorid, err := strconv.ParseInt(string(lines[0][4]), 10, 32)
            //~ if err != nil {return }
            relhumid, err := strconv.ParseInt(string(lines[1]), 10, 32)
            if err != nil {return }
            ps.Pub(r3events.RelativeHumiditySensorUpdate{0, int(relhumid), ts}, "sensors")
        case "dust0:","dust1:","dust2:":
            sensorid, err := strconv.ParseInt(string(lines[0][4]), 10, 32)
            if err != nil {return }
            dustlvl, err := strconv.ParseInt(string(lines[1]), 10, 32)
            if err != nil {return }
            ps.Pub(r3events.DustSensorUpdate{int(sensorid), dustlvl, ts}, "sensors")
        default:
            evnt, pubsubcat, err := r3events.UnmarshalByteByte2Event(lines)
            if err == nil {
                ps.Pub(evnt, pubsubcat)
            }
    }
}

func MakeTimeTick(ps *pubsub.PubSub) {
    ps.Pub(r3events.TimeTick{time.Now().Unix()},"time")
}

    //~ match_presence := re_presence_.FindStringSubmatch(line)
    //~ match_status := re_status_.FindStringSubmatch(line)
    //~ match_command := re_command_.FindStringSubmatch(line)
    //~ match_button := re_button_.FindStringSubmatch(line)
    //~ match_temp := re_temp_.FindStringSubmatch(line)
    //~ match_photo := re_photo_.FindStringSubmatch(line)
	//~ if match_button != nil {
        //~ // brn.Oboite("button0", ts)
        //~ ps.Pub(BoreDoomButtonPressEvent{0, ts}, "buttons")
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

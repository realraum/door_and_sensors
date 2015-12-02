// (c) Bernhard Tittelbach, 2013

package main

import (
	"fmt"
	"time"

	"./r3xmppbot"
	r3events "github.com/realraum/door_and_sensors/r3events"
)

func composeDoorLockMessage(locked bool, frontshut r3events.DoorAjarUpdate, doorcmd r3events.DoorCommandEvent, ts int64) string {
	var ajarstring string = ""
	if frontshut.Shut == false && frontshut.Ts < doorcmd.Ts {
		ajarstring = " (still ajar)"
	}
	if ts-doorcmd.Ts < 30 {
		if len(doorcmd.Who) == 0 || doorcmd.Who == "-" {
			return fmt.Sprintf("The%s frontdoor was %s by %s at %s.", ajarstring, IfThenElseStr(locked, "locked", "unlocked"), doorcmd.Using, time.Unix(ts, 0).String())
		} else {
			return fmt.Sprintf("%s %s the%s frontdoor by %s at %s.", doorcmd.Who, IfThenElseStr(locked, "locked", "unlocked"), ajarstring, doorcmd.Using, time.Unix(ts, 0).String())
		}
	} else {
		return fmt.Sprintf("The%s frontdoor was %s manually at %s.", ajarstring, IfThenElseStr(locked, "locked", "unlocked"), time.Unix(ts, 0).String())
	}
}

// compose a message string from presence state
func composePresence(present bool, temp_cx float64, light_lothr, last_buttonpress int64) r3xmppbot.XMPPStatusEvent {
	present_msg := "Somebody is present"
	notpresent_msg := "Nobody is here"
	button_msg := "The button has been pressed :-)"
	msg := "%s (CX: %.2f°C, LoTHR light: %d)"

	if present {
		if last_buttonpress > 0 && time.Now().Unix()-last_buttonpress < button_press_timeout_ {
			return r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowFreeForChat, fmt.Sprintf(msg, button_msg, temp_cx, light_lothr)}
		} else {
			return r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowOnline, fmt.Sprintf(msg, present_msg, temp_cx, light_lothr)}
		}
	} else {
		return r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowNotAvailabe, fmt.Sprintf(msg, notpresent_msg, temp_cx, light_lothr)}
	}
}

// gets r3events and sends corresponding XMPP messages and XMPP Presence/Status Updates
func EventToXMPP(bot *r3xmppbot.XmppBot, events <-chan interface{}, xmpp_presence_events_chan chan<- interface{}) {
	button_msg := "Dooom ! The button has been pressed ! Propably someone is bored and in need of company ! ;-)"
	defer func() {
		if x := recover(); x != nil {
			//defer ist called _after_ EventToXMPP function has returned. Thus we recover after returning from this function.
			Syslog_.Printf("handleIncomingXMPPStanzas: run time panic: %v", x)
		}
	}()

	var present, frontlock bool = false, true
	var last_buttonpress, light_lothr int64 = 0, 0
	var temp_cx float64 = 0.0
	var last_door_cmd r3events.DoorCommandEvent
	var last_frontdoor_ajar r3events.DoorAjarUpdate = r3events.DoorAjarUpdate{true, 0}
	var standard_distribute_level r3xmppbot.R3JIDDesire = r3xmppbot.R3DebugInfo // initial state, changed after startup finished event recieved

	xmpp_presence_events_chan <- r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowNotAvailabe, "Nobody is here"}

	for eventinterface := range events {
		Debug_.Printf("event2xmpp: %T %+v", eventinterface, eventinterface)
		switch event := eventinterface.(type) {
		case EventToXMPPStartupFinished:
			//after we received all events from QueryLatestEventsAndInjectThem, we get this event and start sending new events normally
			standard_distribute_level = r3xmppbot.R3OnlineOnlyInfo
		case r3events.PresenceUpdate:
			present = event.Present
			if !present {
				last_buttonpress = 0
			}
			xmpp_presence_events_chan <- composePresence(present, temp_cx, light_lothr, last_buttonpress)
		case r3events.DoorCommandEvent:
			last_door_cmd = event
			xmpp_presence_events_chan <- fmt.Sprintln("DoorCommand:", event.Command, "using", event.Using, "by", event.Who, time.Unix(event.Ts, 0))
		case r3events.DoorLockUpdate:
			if frontlock != event.Locked {
				xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: composeDoorLockMessage(event.Locked, last_frontdoor_ajar, last_door_cmd, event.Ts), DistributeLevel: standard_distribute_level, RememberAsStatus: true}
			}
			frontlock = event.Locked
		case r3events.DoorAjarUpdate:
			if last_frontdoor_ajar.Shut != event.Shut {
				xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Frontdoor is %s  (%s)", IfThenElseStr(event.Shut, "now shut.", "ajar."), time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3DebugInfo, RememberAsStatus: false}
			}
			last_frontdoor_ajar = event
		case r3events.BackdoorAjarUpdate:
			xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Backdoor is %s  (%s)", IfThenElseStr(event.Shut, "now shut.", "ajar!"), time.Unix(event.Ts, 0).String()), DistributeLevel: standard_distribute_level, RememberAsStatus: false}
		case r3events.GasLeakAlert:
			xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("ALERT !! GasLeak Detected !!! (%s)", time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false}
		case r3events.UPSPowerUpdate:
			if event.OnBattery {
				xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("ALERT !! UPS reports power has been lost. Battery at %d%% (%s)", event.PercentBattery, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false}
			} else if event.PercentBattery < 100 {
				xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("UPS reports power restored. Battery at %d%% (%s)", event.PercentBattery, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false}
			}
		case r3events.TempOverThreshold:
			xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("ALERT !! Temperature %s exceeded limit at %f°C (%s)", event.Location, event.Value, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false}
		case r3events.SensorLost:
			xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Mhhh. Apparently sensor %s disappeared. (last seen: %s, usual average update interval: %ds) (%s)", event.Topic, time.Unix(event.LastSeen, 0).String(), event.UsualInterval, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false}
		case r3events.IlluminationSensorUpdate:
			light_lothr = event.Value
		case r3events.TempSensorUpdate:
			if event.Location == "CX" {
				temp_cx = event.Value
			}
		case r3events.BoreDoomButtonPressEvent:
			last_buttonpress = event.Ts
			xmpp_presence_events_chan <- composePresence(present, temp_cx, light_lothr, last_buttonpress)
			xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: button_msg, DistributeLevel: standard_distribute_level}
		case r3events.TimeTick:
			// update presence text with sensor and button info
			xmpp_presence_events_chan <- composePresence(present, temp_cx, light_lothr, last_buttonpress)
			// Try to XMPP Ping the server and if that fails, quit XMPPBot
			if bot.PingServer(XMPP_PING_TIMEOUT) == false && bot.PingServer(XMPP_PING_TIMEOUT) == false && bot.PingServer(XMPP_PING_TIMEOUT) == false && bot.PingServer(XMPP_PING_TIMEOUT) == false && bot.PingServer(XMPP_PING_TIMEOUT) == false {
				return
			}
		case r3events.DoorProblemEvent:
			xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Door Problem: %s. SeverityLevel: %d (%s)", event.Problem, event.Severity, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3OnlineOnlyInfo, RememberAsStatus: false}
		}
	}
}

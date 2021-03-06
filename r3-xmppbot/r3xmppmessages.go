// (c) Bernhard Tittelbach, 2013

package main

import (
	"fmt"
	"time"

	r3events "../r3events"
	"./r3xmppbot"
)

func composeW2DoorLockMessage(locked bool) string {
	return fmt.Sprintf("Flat#2 is %s", IfThenElseStr(locked, "locked", "unlocked"))
}

func composeDoorLockMessage(w1locked, w2locked bool, frontshut r3events.DoorAjarUpdate, doorcmd r3events.DoorCommandEvent, ts int64) string {
	var ajarstring string = ""
	if frontshut.Shut == false && frontshut.Ts < doorcmd.Ts {
		ajarstring = " (still ajar)"
	}
	var frominside bool = doorcmd.Using == "Button" || (len(doorcmd.Command) > 10 && doorcmd.Command[len(doorcmd.Command)-10:] == "frominside")
	var whounknown bool = len(doorcmd.Who) == 0 || doorcmd.Who == "-"
	if ts-doorcmd.Ts < 30 {
		if whounknown {
			return fmt.Sprintf("The%s frontdoor was %s by %s %sand %s as of %s.", ajarstring, IfThenElseStr(w1locked, "locked", "unlocked"), doorcmd.Using, IfThenElseStr(frominside, "from the inside ", ""), composeW2DoorLockMessage(w2locked), time.Unix(ts, 0).String())
		} else {
			return fmt.Sprintf("%s %s the%s frontdoor by %s %sand %s as of %s.", doorcmd.Who, IfThenElseStr(w1locked, "locked", "unlocked"), ajarstring, doorcmd.Using, IfThenElseStr(frominside, "from the inside ", ""), composeW2DoorLockMessage(w2locked), time.Unix(ts, 0).String())
		}
	} else {
		return fmt.Sprintf("The%s frontdoor was %s manually and %s as of %s.", ajarstring, IfThenElseStr(w1locked, "locked", "unlocked"), composeW2DoorLockMessage(w2locked), time.Unix(ts, 0).String())
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
func EventToXMPP(bot *r3xmppbot.XmppBot, events <-chan *r3events.R3MQTTMsg, xmpp_presence_events_chan chan<- interface{}, watchdog_timeout time.Duration) {
	button_msg := "Dooom ! The button has been pressed ! Probably someone is bored and in need of company ! ;-)"
	defer func() {
		if x := recover(); x != nil {
			//defer ist called _after_ EventToXMPP function has returned. Thus we recover after returning from this function.
			Syslog_.Printf("EventToXMPP: run time panic: %v", x)
		}
	}()

	//the watchdog timer is watching for hanging for loops
	watchdog := time.AfterFunc(watchdog_timeout, func() { panic("Event2Xmpp Watchdog timed out") })
	//make sure we don't panic when we exit EventToXMPP for other reasons
	defer watchdog.Stop()

	var present, frontlock, w2frontlock bool = false, true, true
	var last_buttonpress, light_lothr int64 = 0, 0
	var temp_cx float64 = 0.0
	var last_door_cmd r3events.DoorCommandEvent
	var last_frontdoor_ajar r3events.DoorAjarUpdate = r3events.DoorAjarUpdate{true, 0}
	var last_backdoor_ajar r3events.DoorAjarUpdate = r3events.DoorAjarUpdate{true, 0}
	var standard_distribute_level r3xmppbot.R3JIDDesire = r3xmppbot.R3DebugInfo // initial state, changed after startup finished event recieved

	xmpp_presence_events_chan <- r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowNotAvailabe, "Nobody is here"}

	ticker := time.NewTicker(time.Duration(6) * time.Minute)

	for {
		select {
		case r3msg := <-events:
			eventinterface := r3msg.Event
			Debug_.Printf("event2xmpp: %T %+v", eventinterface, eventinterface)
			watchdog.Reset(watchdog_timeout)
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
				if len(r3msg.Topic) < 3 {
					continue
				}
				some_lock_changed := false
				switch r3msg.Topic[1] {
				case r3events.CLIENTID_FRONTDOOR:
					if frontlock != event.Locked {
						some_lock_changed = true
					}
					frontlock = event.Locked
				case r3events.CLIENTID_W2FRONTDOOR:
					if w2frontlock != event.Locked {
						some_lock_changed = true
					}
					w2frontlock = event.Locked
				}
				if some_lock_changed {
					xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: composeDoorLockMessage(frontlock, w2frontlock, last_frontdoor_ajar, last_door_cmd, event.Ts), DistributeLevel: standard_distribute_level, RememberAsStatus: true}
				}
			case r3events.DoorAjarUpdate:
				if len(r3msg.Topic) < 3 {
					continue
				}
				switch r3msg.Topic[1] {
				case r3events.CLIENTID_FRONTDOOR:
					if last_frontdoor_ajar.Shut != event.Shut {
						xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Frontdoor is %s  (%s)", IfThenElseStr(event.Shut, "now shut.", "ajar."), time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3DebugInfo, RememberAsStatus: false, RelevantFilter: r3xmppbot.JDFieldNoFrontdoorUpdates}
					}
					last_frontdoor_ajar = event
				case r3events.CLIENTID_BACKDOOR:
					//ignore persistant messages resends we get if mqtt reconnects
					if last_backdoor_ajar.Ts != event.Ts {
						xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Backdoor is %s  (%s)", IfThenElseStr(event.Shut, "now shut.", "ajar!"), time.Unix(event.Ts, 0).String()), DistributeLevel: standard_distribute_level, RememberAsStatus: false, RelevantFilter: r3xmppbot.JDFieldNoBackdoorUpdates}
					}
					last_backdoor_ajar = event
				}
			case r3events.GasLeakAlert:
				xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("ALERT !! GasLeak Detected !!! (%s)", time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false, RelevantFilter: r3xmppbot.JDFieldNoGasAlertUpdates}
			case r3events.UPSPowerUpdate:
				if event.OnBattery {
					xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("ALERT !! UPS reports power has been lost. Battery at %.1f%% (%s)", event.PercentBattery, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false, RelevantFilter: r3xmppbot.JDFieldNoSensorUpdates}
				} else if event.PercentBattery < 100 {
					xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("UPS reports power restored. Battery at %.1f%% (%s)", event.PercentBattery, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false, RelevantFilter: r3xmppbot.JDFieldNoSensorUpdates}
				}
			case r3events.TempOverThreshold:
				xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("ALERT !! Temperature %s exceeded limit at %.2f°C (%s)", event.Location, event.Value, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false, RelevantFilter: r3xmppbot.JDFieldNoFreezerAlarmUpdates}
			case r3events.SensorLost:
				xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Mhhh. Apparently sensor %s disappeared. (last seen: %s, usual average update interval: %ds) (%s)", event.Topic, time.Unix(event.LastSeen, 0).String(), event.UsualInterval, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3NeverInfo, RememberAsStatus: false, RelevantFilter: r3xmppbot.JDFieldNoSensorUpdates}
			case r3events.IlluminationSensorUpdate:
				light_lothr = event.Value
			case r3events.TempSensorUpdate:
				if event.Location == "CX" {
					temp_cx = event.Value
				}
			case r3events.BoreDoomButtonPressEvent:
				last_buttonpress = event.Ts
				xmpp_presence_events_chan <- composePresence(present, temp_cx, light_lothr, last_buttonpress)
				xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: button_msg, DistributeLevel: standard_distribute_level, RelevantFilter: r3xmppbot.JDFieldNoButtonUpdates}
			case r3events.DoorProblemEvent:
				xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Door Problem: %s. SeverityLevel: %d (%s)", event.Problem, event.Severity, time.Unix(event.Ts, 0).String()), DistributeLevel: r3xmppbot.R3OnlineOnlyInfo, RememberAsStatus: false, RelevantFilter: r3xmppbot.JDFieldNoFrontdoorUpdates}
			}

		case <-ticker.C:
			// update presence text with sensor and button info
			xmpp_presence_events_chan <- composePresence(present, temp_cx, light_lothr, last_buttonpress)
			// Try to XMPP Ping the server and if that fails, quit XMPPBot
			if bot.PingServer(XMPP_PING_TIMEOUT) == false && bot.PingServer(XMPP_PING_TIMEOUT) == false && bot.PingServer(XMPP_PING_TIMEOUT) == false && bot.PingServer(XMPP_PING_TIMEOUT) == false {
				return
			}
		}
	}
}

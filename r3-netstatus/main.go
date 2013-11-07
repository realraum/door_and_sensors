// (c) Bernhard Tittelbach, 2013

package main

import (
    "./r3xmppbot"
    pubsub "github.com/tuxychandru/pubsub"
    "flag"
    "time"
    "fmt"
    //~ "./brain"
    r3events "svn.spreadspace.org/realraum/go.svn/r3events"
    zmq "github.com/vaughan0/go-zmq"
)

type SpaceState struct {
    present           bool
    buttonpress_until int64
    door_locked bool
    door_shut bool
}

var (
    xmpp_login_ struct {jid string; pass string}
    xmpp_bot_authstring_ string
    xmpp_state_save_dir_ string
    r3eventssub_port_ string
    button_press_timeout_ int64 = 3600
    brain_connect_addr_ string
    enable_syslog_, enable_debug_ bool
)

//-------

func init() {
    flag.StringVar(&xmpp_login_.jid, "xjid", "realrauminfo@realraum.at/Tuer", "XMPP Bot Login JID")
    flag.StringVar(&xmpp_login_.pass, "xpass", "", "XMPP Bot Login Password")
    flag.StringVar(&xmpp_bot_authstring_, "xbotauth", "", "String that user use to authenticate themselves to the bot")
    flag.StringVar(&xmpp_state_save_dir_,"xstatedir","/flash/var/lib/r3netstatus/",  "Directory to save XMPP bot state in")
    flag.StringVar(&r3eventssub_port_, "eventsubport", "tcp://wuzzler.realraum.at:4244", "zmq address to subscribe r3events")
    flag.StringVar(&brain_connect_addr_, "brainconnect", "tcp://wuzzler.realraum.at:4245", "address to ask about most recent stored events")
    flag.BoolVar(&enable_syslog_, "syslog", false, "enable logging to syslog")
    flag.BoolVar(&enable_debug_, "debug", false, "enable debug output")
    flag.Parse()
}

//-------

func IfThenElseStr(c bool, strue, sfalse string) string {
    if c {return strue} else {return sfalse}
}

func composeDoorLockMessage(locked bool, frontshut r3events.DoorAjarUpdate, doorcmd r3events.DoorCommandEvent, ts int64) string {
    var ajarstring string = "";
    if frontshut.Shut == false && frontshut.Ts < doorcmd.Ts {
        ajarstring = " (still ajar)"
    }
    if ts - doorcmd.Ts < 30 {
        if len(doorcmd.Who) == 0 || doorcmd.Who == "-" {
            return fmt.Sprintf("The%s frontdoor was %s by %s at %s.",ajarstring, IfThenElseStr(locked, "locked","unlocked"), doorcmd.Using, time.Unix(ts,0).String())
        } else {
            return fmt.Sprintf("%s %s the%s frontdoor by %s at %s.",doorcmd.Who, IfThenElseStr(locked, "locked","unlocked"), ajarstring, doorcmd.Using, time.Unix(ts,0).String())
        }
    } else {
        return fmt.Sprintf("The%s frontdoor was %s manually at %s.", ajarstring, IfThenElseStr(locked, "locked","unlocked"), time.Unix(ts,0).String())
    }
}

func EventToXMPP(bot *r3xmppbot.XmppBot, events <- chan interface{}, xmpp_presence_events_chan chan <- interface{}) {

    defer func() {
        if x := recover(); x != nil {
            //defer ist called _after_ EventToXMPP function has returned. Thus we recover after returning from this function.
            Syslog_.Printf("handleIncomingXMPPStanzas: run time panic: %v", x)
        }
    }()

    var present, frontlock bool = false, true
    var last_buttonpress int64 = 0
    var last_door_cmd r3events.DoorCommandEvent;
    var last_frontdoor_ajar r3events.DoorAjarUpdate = r3events.DoorAjarUpdate{true, 0};
    button_msg := "Dooom ! The button has been pressed ! Propably someone is bored and in need of company ! ;-)"
    present_status := r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowOnline,"Somebody is present"}
    notpresent_status := r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowNotAvailabe,"Nobody is here"}
    button_status := r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowFreeForChat, "The button has been pressed :-)"}

    xmpp_presence_events_chan <- r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowNotAvailabe, "Nobody is here"}

    for eventinterface := range(events) {
        Debug_.Printf("event2xmpp: %T %+v", eventinterface, eventinterface)
        switch event := eventinterface.(type) {
            case r3events.PresenceUpdate:
                present = event.Present
                if present {
                    xmpp_presence_events_chan <- present_status
                } else {
                    xmpp_presence_events_chan <- notpresent_status
                }
            case r3events.DoorCommandEvent:
                last_door_cmd = event
                xmpp_presence_events_chan <- fmt.Sprintln("DoorCommand:",event.Command, "using", event.Using, "by", event.Who, time.Unix(event.Ts,0))
            case r3events.DoorLockUpdate:
                if frontlock != event.Locked {
                    xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: composeDoorLockMessage(event.Locked, last_frontdoor_ajar, last_door_cmd, event.Ts), DistributeLevel: r3xmppbot.R3OnlineOnlyInfo, RememberAsStatus: true}
                }
                frontlock = event.Locked
           case r3events.DoorAjarUpdate:
                if last_frontdoor_ajar.Shut != event.Shut {
                    xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Frontdoor is %s  (%s)",IfThenElseStr(event.Shut,"now shut.","ajar."),time.Unix(event.Ts,0).String()), DistributeLevel: r3xmppbot.R3DebugInfo, RememberAsStatus: false}
                }
                last_frontdoor_ajar = event
           case r3events.BackdoorAjarUpdate:
                xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Backdoor is %s  (%s)",IfThenElseStr(event.Shut,"now shut.","ajar!"),time.Unix(event.Ts,0).String()), DistributeLevel: r3xmppbot.R3OnlineOnlyInfo, RememberAsStatus: false}
            case r3events.BoreDoomButtonPressEvent:
                xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: button_msg, DistributeLevel: r3xmppbot.R3OnlineOnlyInfo}
                xmpp_presence_events_chan <- button_status
                last_buttonpress = event.Ts
            case r3events.TimeTick:
                if present && last_buttonpress > 0 && time.Now().Unix() - last_buttonpress > button_press_timeout_ {
                    xmpp_presence_events_chan <- present_status
                    last_buttonpress = 0
                }
                
            case r3events.DoorProblemEvent:
                xmpp_presence_events_chan <- r3xmppbot.XMPPMsgEvent{Msg: fmt.Sprintf("Door Problem: %s. SeverityLevel: %d (%s)",event.Problem, event.Severity, time.Unix(event.Ts,0).String()), DistributeLevel: r3xmppbot.R3OnlineOnlyInfo, RememberAsStatus: false}
        }
	}
}

func RunXMPPBot(ps *pubsub.PubSub, zmqctx *zmq.Context) {
    var xmpperr error
    var bot *r3xmppbot.XmppBot
    var xmpp_presence_events_chan chan interface{}
    for {
        bot, xmpp_presence_events_chan, xmpperr = r3xmppbot.NewStartedBot(xmpp_login_.jid, xmpp_login_.pass, xmpp_bot_authstring_, xmpp_state_save_dir_, true)
        if xmpperr == nil {
            Syslog_.Printf("Successfully (re)started XMPP Bot")
            // subscribe before QueryLatestEventsAndInjectThem and EventToXMPP
            psevents := ps.Sub("presence","door","buttons","updateinterval")
            QueryLatestEventsAndInjectThem(ps, zmqctx)
            EventToXMPP(bot, psevents, xmpp_presence_events_chan)
            // unsubscribe right away, since we don't known when reconnect will succeed and we don't want to block PubSub
            ps.Unsub(psevents, "presence","door","buttons","updateinterval")
            bot.StopBot()
        } else {
            Syslog_.Printf("Error starting XMPP Bot: %s", xmpperr.Error())
        }
        time.Sleep(5 * time.Second)
    }
}

func ParseZMQr3Event(lines [][]byte, ps *pubsub.PubSub) {
    evnt, pubsubcat, err := r3events.UnmarshalByteByte2Event(lines)
    Debug_.Printf("ParseZMQr3Event: %s %s %s", evnt, pubsubcat, err)
    if err != nil { return }
    ps.Pub(evnt, pubsubcat)
}

func QueryLatestEventsAndInjectThem(ps *pubsub.PubSub, zmqctx *zmq.Context) {
    answ := ZmqsAskQuestionsAndClose(zmqctx, brain_connect_addr_, [][][]byte{
        [][]byte{[]byte("BackdoorAjarUpdate")},
        [][]byte{[]byte("DoorCommandEvent")},
        [][]byte{[]byte("DoorLockUpdate")},
        [][]byte{[]byte("DoorAjarUpdate")},
        [][]byte{[]byte("PresenceUpdate")},
        [][]byte{[]byte("IlluminationSensorUpdate")},
        [][]byte{[]byte("TempSensorUpdate")}})
    for _, a := range(answ) {
        ParseZMQr3Event(a, ps)
    }
}

func main() {
    if enable_syslog_ { LogEnableSyslog(); r3xmppbot.LogEnableSyslog() }
    if enable_debug_ { LogEnableDebuglog(); r3xmppbot.LogEnableDebuglog() }
    Syslog_.Print("started")
    defer Syslog_.Print("exiting")
    zmqctx, zmqsub := ZmqsInit(r3eventssub_port_)
    defer zmqctx.Close()
    if zmqsub != nil {defer zmqsub.Close()}
    if zmqsub == nil {
        panic("zmq sockets must not be nil !!")
    }

    ps := pubsub.New(10)
    defer ps.Shutdown()
    //~ brn := brain.New()
    //~ defer brn.Shutdown()

    go EventToWeb(ps)
    // --- get update on most recent events ---
    QueryLatestEventsAndInjectThem(ps, zmqctx)
    go RunXMPPBot(ps, zmqctx)

    // --- receive and distribute events ---
    ticker := time.NewTicker(time.Duration(7) * time.Minute)
    for {
    select {
        case e := <-zmqsub.In():
            ParseZMQr3Event(e, ps) //, brn)
        case <-ticker.C:
            ps.Pub(r3events.TimeTick{time.Now().Unix()}, "updateinterval")
        }
    }
}

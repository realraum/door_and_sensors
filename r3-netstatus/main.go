// (c) Bernhard Tittelbach, 2013

package main

import (
    "./r3xmppbot"
    pubsub "github.com/tuxychandru/pubsub"
    "flag"
    "time"
    "fmt"
    //~ "./brain"
)

type SpaceState struct {
    present           bool
    buttonpress_until int64
    door_locked bool
    door_shut bool
}

var (
    presence_socket_path_ string
    xmpp_presence_events_chan_     chan interface{}
    xmpp_login_ struct {jid string; pass string}
    xmpp_bot_authstring_ string
    xmpp_state_save_dir_ string
    button_press_timeout_ int64 = 3600
)

//-------

func init() {
    flag.StringVar(&xmpp_login_.jid, "xjid", "realrauminfo@realraum.at/Tuer", "XMPP Bot Login JID")
    flag.StringVar(&xmpp_login_.pass, "xpass", "", "XMPP Bot Login Password")
    flag.StringVar(&xmpp_bot_authstring_, "xbotauth", "", "String that user use to authenticate themselves to the bot")
    flag.StringVar(&presence_socket_path_,"presencesocket", "/var/run/tuer/presence.socket",  "Path to presence socket")
    flag.StringVar(&xmpp_state_save_dir_,"xstatedir","/flash/var/lib/r3netstatus/",  "Directory to save XMPP bot state in")
    flag.Parse()
}

//-------

func IfThenElseStr(c bool, strue, sfalse string) string {
    if c {return strue} else {return sfalse}
}

func composeMessage(present, locked, shut bool, who string, ts int64) string {
    return fmt.Sprintf("%s (Door is %s and %s and was last used%s at %s)",
        IfThenElseStr(present,  "Somebody is present!" , "Everybody left."),
        IfThenElseStr(locked, "locked","unlocked"),
        IfThenElseStr(shut, "shut","ajar"),
        IfThenElseStr(len(who) == 0,"", " by " + who),
        time.Unix(ts,0).String())
}

func EventToXMPP(ps *pubsub.PubSub, xmpp_presence_events_chan_ chan <- interface{}) {
    events := ps.Sub("presence","door","buttons","updateinterval")

    defer func() {
        if x := recover(); x != nil {
            fmt.Printf("handleIncomingXMPPStanzas: run time panic: %v", x)
            ps.Unsub(events, "presence","door","buttons","updateinterval")
            close(xmpp_presence_events_chan_)
        }
    }()

    var present, locked, shut bool = false, true, true
    var last_buttonpress int64 = 0
    var who string
    button_msg := "The button has been pressed ! Propably someone is bored and in need of company ! ;-)"
    present_status := r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowOnline,"Somebody is present"}
    notpresent_status := r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowNotAvailabe,"Nobody is here"}
    button_status := r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowFreeForChat, "The button has been pressed :-)"}
    
    xmpp_presence_events_chan_ <- r3xmppbot.XMPPStatusEvent{r3xmppbot.ShowNotAvailabe, "Nobody is here"}
    
    for eventinterface := range(events) {
        switch event := eventinterface.(type) {
            case PresenceUpdate:
                present = event.Present
                xmpp_presence_events_chan_ <- r3xmppbot.XMPPMsgEvent{Msg: composeMessage(present, locked, shut, who, event.Ts), DistributeLevel: r3xmppbot.R3OnlineOnlyInfo, RememberAsStatus: true}
                if present {
                    xmpp_presence_events_chan_ <- present_status
                } else {
                    xmpp_presence_events_chan_ <- notpresent_status
                }           
            case DoorCommandEvent:
                if len(event.Who) > 0 && len(event.Using) > 0 {
                    who = fmt.Sprintf("%s (%s)",event.Who, event.Using)
                } else {
                    who = event.Who
                }
                xmpp_presence_events_chan_ <- fmt.Sprintln("DoorCommand:",event.Command, "using", event.Using, "by", event.Who, time.Unix(event.Ts,0))
            case DoorStatusUpdate:
                locked = event.Locked
                shut = event.Shut
                xmpp_presence_events_chan_ <- r3xmppbot.XMPPMsgEvent{Msg: composeMessage(present, locked, shut, who, event.Ts), DistributeLevel: r3xmppbot.R3DebugInfo, RememberAsStatus: true}
            case ButtonPressUpdate:
                xmpp_presence_events_chan_ <- r3xmppbot.XMPPMsgEvent{Msg: button_msg, DistributeLevel: r3xmppbot.R3OnlineOnlyInfo}
                xmpp_presence_events_chan_ <- button_status
                last_buttonpress = event.Ts
            case TimeTick:
                if present && last_buttonpress > 0 && time.Now().Unix() - last_buttonpress > button_press_timeout_ {
                    xmpp_presence_events_chan_ <- present_status
                    last_buttonpress = 0
                }
        }
	}
}

func main() {
    var xmpperr error
    var bot *r3xmppbot.XmppBot
    bot, xmpp_presence_events_chan_, xmpperr = r3xmppbot.NewStartedBot(xmpp_login_.jid, xmpp_login_.pass, xmpp_bot_authstring_, xmpp_state_save_dir_, true)

    newlinequeue := make(chan string, 1)
    ps := pubsub.New(1)
    //~ brn := brain.New()
    defer close(newlinequeue)
    defer ps.Shutdown()
    //~ defer brn.Shutdown()

    go EventToWeb(ps)
    if xmpperr == nil {
        defer bot.StopBot()
        go EventToXMPP(ps, xmpp_presence_events_chan_)
    } else {
        fmt.Println(xmpperr)
        fmt.Println("XMPP Bot disabled")
    }
    go ReadFromUSocket(presence_socket_path_, newlinequeue)
    ticker := time.NewTicker(time.Duration(7) * time.Minute)

    for {
    select {
        case e := <-newlinequeue:
            ParseSocketInputLine(e, ps) //, brn)
        case <-ticker.C:
            ps.Pub(TimeTick{time.Now().Unix()}, "updateinterval")
        }
    }
}

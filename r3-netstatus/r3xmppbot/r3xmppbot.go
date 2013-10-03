// (c) Bernhard Tittelbach, 2013

package r3xmppbot

import (
	xmpp "code.google.com/p/goexmpp"
    "log"
    "crypto/tls"
    "os"
    "time"
    "encoding/json"
    "path"
)

//~ type StdLogger struct {
//~ }

//~ func (s *StdLogger) Log(v ...interface{}) {
        //~ log.Println(v...)
//~ }

//~ func (s *StdLogger) Logf(fmt string, v ...interface{}) {
        //~ log.Printf(fmt, v...)
//~ }


func (botdata *XmppBot) makeXMPPMessage(to string, message interface{}, subject interface{}) *xmpp.Message {
    xmppmsgheader := xmpp.Header{To: to,
                                                            From: botdata.my_jid_,
                                                            Id: <-xmpp.Id,
                                                            Type: "chat",
                                                            Lang: "",
                                                            Innerxml: "",
                                                            Error: nil,
                                                            Nested: make([]interface{},0)}

    var msgsubject, msgbody *xmpp.Generic
    switch cast_msg := message.(type) {
        case string:
            msgbody = &xmpp.Generic{Chardata: cast_msg}
        case *string:
            msgbody = &xmpp.Generic{Chardata: *cast_msg}
        case *xmpp.Generic:
            msgbody = cast_msg
        default:
            msgbody = &xmpp.Generic{}
    }
    switch cast_msg := subject.(type) {
        case string:
            msgsubject = &xmpp.Generic{Chardata: cast_msg}
        case *string:
            msgsubject = &xmpp.Generic{Chardata: *cast_msg}
        case *xmpp.Generic:
            msgsubject = cast_msg
        default:
            msgsubject = &xmpp.Generic{}
    }
    return &xmpp.Message{Header: xmppmsgheader , Subject: msgsubject, Body: msgbody, Thread: &xmpp.Generic{}}
}

func (botdata *XmppBot) makeXMPPPresence(to, ptype, show, status string) *xmpp.Presence {
    xmppmsgheader := xmpp.Header{To: to,
                                                            From: botdata.my_jid_,
                                                            Id: <-xmpp.Id,
                                                            Type: ptype,
                                                            Lang: "",
                                                            Innerxml: "",
                                                            Error: nil,
                                                            Nested: make([]interface{},0)}
    var gen_show, gen_status *xmpp.Generic
    if len(show) == 0 {
        gen_show = nil
    } else {
        gen_show = &xmpp.Generic{Chardata: show}
    }
    if len(status) == 0 {
        gen_status = nil
    } else {
        gen_status = &xmpp.Generic{Chardata: status}
    }
    return &xmpp.Presence{Header: xmppmsgheader, Show: gen_show, Status: gen_status}
}

type R3JIDDesire int

const (
    R3NoChange R3JIDDesire = -1
    R3NeverInfo R3JIDDesire = iota // ignore first value by assigning to blank identifier
    R3OnlineOnlyInfo
    R3OnlineOnlyWithRecapInfo
    R3AlwaysInfo
    R3DebugInfo
)

const (
    ShowOnline string = ""
    ShowAway string = "away"
    ShowNotAvailabe string = "xa"
    ShowDoNotDisturb string = "dnd"
    ShowFreeForChat string = "chat"
)

type JidData struct {
	Online  bool
    Wants   R3JIDDesire
}

type JabberEvent struct {
    JID      string
    Online   bool
    Wants    R3JIDDesire
    StatusNow bool
}

type XMPPMsgEvent struct {
    Msg string
    DistributeLevel R3JIDDesire
    RememberAsStatus bool
}

type XMPPStatusEvent struct {
    Show string
    Status string
}

type RealraumXmppNotifierConfig map[string]JidData

type XmppBot struct {
    jid_lastauthtime_ map[string]int64
    realraum_jids_ RealraumXmppNotifierConfig
    password_ string
    auth_cmd_ string
    auth_cmd2_ string
    my_jid_ string
    auth_timeout_ int64
    config_file_ string
    my_login_password_ string
    xmppclient_ *xmpp.Client
    presence_events_ *chan interface{}
}


func (data RealraumXmppNotifierConfig) saveTo(filepath string) () {
    fh, err := os.Create(filepath)
    if err != nil {
        log.Println(err)
        return
    }
    defer fh.Close()
    enc := json.NewEncoder(fh)
    if err = enc.Encode(&data); err != nil {
        log.Println(err)
        return
    }
}

func (data RealraumXmppNotifierConfig) loadFrom(filepath string) () {
    fh, err := os.Open(filepath)
    if err != nil {
        log.Println(err)
        return
    }
    defer fh.Close()
    dec := json.NewDecoder(fh)
    if err = dec.Decode(&data); err != nil {
        log.Println(err)
        return
    }
    for to, jiddata := range data  {
        jiddata.Online = false
        data[to]=jiddata
    }
}


func init() {
        //~ logger := &StdLogger{}
        //~ xmpp.Debug = logger
        //~ xmpp.Info = logger
        //~ xmpp.Warn = logger
}

func (botdata *XmppBot) handleEventsforXMPP(xmppout chan <- xmpp.Stanza, presence_events <- chan interface{}, jabber_events <- chan JabberEvent) {
    var last_status_msg *string

    defer func() {
        if x := recover(); x != nil {
            log.Printf("handleEventsforXMPP: run time panic: %v", x)
        }
    }()

	for {
		select {
		case pe := <-presence_events:
            switch pec := pe.(type) {
                case xmpp.Stanza:
                    xmppout <- pec
                    continue
                case string:
                    for to, jiddata := range botdata.realraum_jids_  {
                        if  jiddata.Wants >= R3DebugInfo {
                            xmppout <-  botdata.makeXMPPMessage(to, pec, nil)
                        }
                    }

                case XMPPStatusEvent:
                    xmppout <- botdata.makeXMPPPresence("", "", pec.Show, pec.Status)

                case XMPPMsgEvent:
                    if pec.RememberAsStatus {
                        last_status_msg = &pec.Msg
                    }
                    for to, jiddata := range botdata.realraum_jids_  {
                        if  jiddata.Wants >= pec.DistributeLevel && ((jiddata.Wants >= R3OnlineOnlyInfo && jiddata.Online) || jiddata.Wants >= R3AlwaysInfo) {
                            xmppout <-  botdata.makeXMPPMessage(to, pec.Msg, nil)
                        }
                    }
                default:
                    break
                }

		case je := <-jabber_events:
            simple_jid := removeJIDResource(je.JID)
            jid_data, jid_in_map := botdata.realraum_jids_[simple_jid]
            if jid_in_map {
                if last_status_msg != nil && (je.StatusNow || (! jid_data.Online && je.Online && jid_data.Wants == R3OnlineOnlyWithRecapInfo) ) {
                    xmppout <-  botdata.makeXMPPMessage(je.JID, last_status_msg, nil)
                }
                jid_data.Online = je.Online
                if je.Wants > R3NoChange {
                    jid_data.Wants = je.Wants
                }
                botdata.realraum_jids_[simple_jid] = jid_data
                botdata.realraum_jids_.saveTo(botdata.config_file_)
            } else if je.Wants > R3NoChange {
                botdata.realraum_jids_[simple_jid] = JidData{je.Online, je.Wants}
                botdata.realraum_jids_.saveTo(botdata.config_file_)
            }
		}
	}
}

func removeJIDResource(jid string) string {
    var jidjid xmpp.JID
    jidjid.Set(jid)
    jidjid.Resource = ""
    return jidjid.String()
}

func (botdata *XmppBot) isAuthenticated(jid string) bool {
    authtime, in_map := botdata.jid_lastauthtime_[jid]
    //~ log.Println("isAuthenticated", in_map, authtime, time.Now().Unix(), auth_timeout_, time.Now().Unix() - authtime > auth_timeout_)
    return in_map && time.Now().Unix() - authtime < botdata.auth_timeout_
}

const help_text_ string = "\n*auth*<password>* ...Enables you to use more commands.\n*time* ...Returns bot time."
const help_text_auth string = "You are authorized to use the following commands:\n*off* ...You will no longer receive notifications.\n*on* ...You will be notified of r3 status changes while you are online.\n*on_with_recap* ...Like *on* but additionally you will receive the current status when you come online.\n*on_while_offline* ...You will receive all r3 status changes, wether you are online or offline.\n*status* ...Use it to query the current status.\n*time* ...Returns bot time.\n*bye* ...Logout."

//~ var re_msg_auth_    *regexp.Regexp     = regexp.MustCompile("auth\s+(\S+)")

func (botdata *XmppBot) handleIncomingMessageDialog(inmsg xmpp.Message, xmppout chan<- xmpp.Stanza, jabber_events chan JabberEvent) {
    if inmsg.Body == nil || inmsg.GetHeader() == nil {
        return
    }
    bodytext :=inmsg.Body.Chardata
    //~ log.Println("Message Body:", bodytext)
    if botdata.isAuthenticated(inmsg.GetHeader().From) {
        switch bodytext {
            case "on", "*on*":
                jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3OnlineOnlyInfo, false}
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Receive r3 status updates while online." , "Your New Status")
            case "off", "*off*":
                jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3NeverInfo, false}
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Do not receive anything." , "Your New Status")
            case "on_with_recap", "*on_with_recap*":
                jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3OnlineOnlyWithRecapInfo, false}
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Receive r3 status updates while and current status on coming, online." , "Your New Status")
            case "on_while_offline", "*on_while_offline*":
                jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3AlwaysInfo, false}
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Receive all r3 status updates, even if you are offline." , "Your New Status")
            case "debug":
                jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3DebugInfo, false}
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Debug mode enabled" , "Your New Status")
            case "bye", "Bye", "quit", "logout", "*bye*":
                botdata.jid_lastauthtime_[inmsg.GetHeader().From] = 0
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Bye Bye !" ,nil)
            case "open","close":
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Sorry, I can't operate the door for you." ,nil)
            case "status", "*status*":
                jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3NoChange, true}
            case "time", "*time*":
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, time.Now().String() , nil)
            default:
                //~ auth_match = re_msg_auth_.FindStringSubmatch(inmsg.Body.Chardata)
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, help_text_auth, nil)
        }
    } else {
        switch bodytext {
            case "Hilfe","hilfe","help","Help","?","hallo","Hallo","Yes","yes","ja","ja bitte","bitte","sowieso":
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, help_text_, "Available Commands")
            case botdata.auth_cmd_, botdata.auth_cmd2_:
                botdata.jid_lastauthtime_[inmsg.GetHeader().From] = time.Now().Unix()
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, help_text_auth, nil)
            case "status", "*status*", "off", "*off*", "on", "*on*", "on_while_offline", "*on_while_offline*", "on_with_recap", "*on_with_recap*":
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Sorry, you need to be authorized to do that." , nil)
            case "time", "*time*":
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, time.Now().String() , nil)
            default:
                //~ auth_match = re_msg_auth_.FindStringSubmatch(inmsg.Body.Chardata)
                xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "A nice day to you too !\nDo you need \"help\" ?", nil)
        }
    }
}

func (botdata *XmppBot) handleIncomingXMPPStanzas(xmppin <- chan xmpp.Stanza, xmppout chan<- xmpp.Stanza, jabber_events chan JabberEvent) {

    defer func() {
        if x := recover(); x != nil {
            log.Printf("handleIncomingXMPPStanzas: run time panic: %v", x)
            close(jabber_events)
        }
    }()

    var incoming_stanza interface{}
    for incoming_stanza = range xmppin {
        switch stanza := incoming_stanza.(type) {
            case *xmpp.Message:
                botdata.handleIncomingMessageDialog(*stanza, xmppout, jabber_events)
            case *xmpp.Presence:
                if stanza.GetHeader() == nil {
                    continue
                }
                switch stanza.GetHeader().Type {
                    case "subscribe":
                        xmppout <- botdata.makeXMPPPresence(stanza.GetHeader().From, "subscribed", "", "")
                        jabber_events <- JabberEvent{stanza.GetHeader().From, true, R3NoChange, false}
                        xmppout <- botdata.makeXMPPPresence(stanza.GetHeader().From, "subscribe", "", "")
                    case "unsubscribe", "unsubscribed":
                        jabber_events <- JabberEvent{stanza.GetHeader().From, false, R3NeverInfo, false}
                        botdata.jid_lastauthtime_[stanza.GetHeader().From] = 0 //logout
                        xmppout <- botdata.makeXMPPPresence(stanza.GetHeader().From, "unsubscribe", "","")
                    case "unavailable":
                        jabber_events <- JabberEvent{stanza.GetHeader().From, false, R3NoChange, false}
                        botdata.jid_lastauthtime_[stanza.GetHeader().From] = 0 //logout
                    default:
                        jabber_events <- JabberEvent{stanza.GetHeader().From, true, R3NoChange, false}
                }
            case *xmpp.Iq:
                if stanza.GetHeader() == nil {
                    continue
                }
        }
    }
}

func NewStartedBot(loginjid, loginpwd, password, state_save_dir string, insecuretls bool) (*XmppBot, chan interface{}, error) {
    var err error
    botdata := new(XmppBot)

    botdata.realraum_jids_ = make(map[string]JidData, 1)
    botdata.jid_lastauthtime_ = make(map[string]int64,1)
    botdata.auth_cmd_ = "auth " + password
    botdata.auth_cmd2_ = "*auth*" + password+"*"
    botdata.my_jid_ = loginjid
    botdata.my_login_password_ = loginpwd
    botdata.auth_timeout_ = 3600*2

    botdata.config_file_ = path.Join(state_save_dir, "r3xmpp."+removeJIDResource(loginjid)+".json")

    //~ log.Println(botdata.config_file_)

    //~ logger := &StdLogger{}
    //~ xmpp.Debug = logger
    //~ xmpp.Info = logger
    //~ xmpp.Warn = logger

    xmpp.TlsConfig = tls.Config{InsecureSkipVerify: insecuretls}
    botdata.realraum_jids_.loadFrom(botdata.config_file_)

    client_jid := new(xmpp.JID)
    client_jid.Set(botdata.my_jid_)
    botdata.xmppclient_, err = xmpp.NewClient(client_jid, botdata.my_login_password_, nil)
    if err != nil {
        log.Println("Error connecting to xmpp server", err)
        return nil, nil, err
    }

    err = botdata.xmppclient_.StartSession(true, &xmpp.Presence{})
    if err != nil {
        log.Println("'Error StartSession:", err)
        return nil, nil, err
    }

    roster := xmpp.Roster(botdata.xmppclient_)
    for _, entry := range roster {
        if entry.Subscription == "from" {
            botdata.xmppclient_.Out <- botdata.makeXMPPPresence(entry.Jid, "subscribe", "","")
        }
        if entry.Subscription == "none" {
            delete(botdata.realraum_jids_, entry.Jid)
        }
    }

    presence_events := make(chan interface{},1)
    jabber_events := make(chan JabberEvent,1)

    go botdata.handleEventsforXMPP(botdata.xmppclient_.Out, presence_events, jabber_events)
    go botdata.handleIncomingXMPPStanzas(botdata.xmppclient_.In, botdata.xmppclient_.Out, jabber_events)

    botdata.presence_events_ = &presence_events

    return botdata, presence_events, nil
}

func (botdata *XmppBot) StopBot() {
    if botdata.xmppclient_ != nil {
        close(botdata.xmppclient_.Out)
    }
    if botdata.presence_events_ != nil {
        *botdata.presence_events_ <- false
        close(*botdata.presence_events_)
    }
}

// (c) Bernhard Tittelbach, 2013

package r3xmppbot

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"os"
	"path"
	"strings"
	"time"

	xmpp "github.com/curzonj/goexmpp"
)

func (botdata *XmppBot) makeXMPPMessage(to string, message interface{}, subject interface{}) *xmpp.Message {
	xmppmsgheader := xmpp.Header{To: to,
		From:     botdata.my_jid_,
		Id:       <-xmpp.Id,
		Type:     "chat",
		Lang:     "",
		Innerxml: "",
		Error:    nil,
		Nested:   make([]interface{}, 0)}

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
	return &xmpp.Message{Header: xmppmsgheader, Subject: msgsubject, Body: msgbody, Thread: &xmpp.Generic{}}
}

func (botdata *XmppBot) makeXMPPPresence(to, ptype, show, status string) *xmpp.Presence {
	xmppmsgheader := xmpp.Header{To: to,
		From:     botdata.my_jid_,
		Id:       <-xmpp.Id,
		Type:     ptype,
		Lang:     "",
		Innerxml: "",
		Error:    nil,
		Nested:   make([]interface{}, 0)}
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
	R3NoChange  R3JIDDesire = -1
	R3NeverInfo R3JIDDesire = iota // ignore first value by assigning to blank identifier
	R3OnlineOnlyInfo
	R3OnlineOnlyWithRecapInfo
	R3AlwaysInfo
	R3DebugInfo
)

const (
	ShowOnline       string = ""
	ShowAway         string = "away"
	ShowNotAvailabe  string = "xa"
	ShowDoNotDisturb string = "dnd"
	ShowFreeForChat  string = "chat"
)

const XMPP_MAX_ERROR_COUNT = 49

type JidData struct {
	Online     bool
	Wants      R3JIDDesire
	ErrorCount int64
}

type JabberEvent struct {
	JID          string
	Online       bool
	Wants        R3JIDDesire
	StatusNow    bool
	ErrorOccured bool
}

type XMPPMsgEvent struct {
	Msg              string
	DistributeLevel  R3JIDDesire
	RememberAsStatus bool
}

type XMPPStatusEvent struct {
	Show   string
	Status string
}

type RealraumXmppNotifierConfig map[string]JidData

type XmppBot struct {
	jid_lastauthtime_  map[string]int64
	realraum_jids_     RealraumXmppNotifierConfig
	password_          string
	my_jid_            string
	auth_timeout_      int64
	config_file_       string
	my_login_password_ string
	xmppclient_        *xmpp.Client
	presence_events_   *chan interface{}
}

func (data RealraumXmppNotifierConfig) saveTo(filepath string) {
	fh, err := os.Create(filepath)
	if err != nil {
		Syslog_.Println(err)
		return
	}
	defer fh.Close()
	enc := json.NewEncoder(fh)
	if err = enc.Encode(&data); err != nil {
		Syslog_.Println(err)
		return
	}
}

func (data RealraumXmppNotifierConfig) loadFrom(filepath string) {
	fh, err := os.Open(filepath)
	if err != nil {
		Syslog_.Println(err)
		return
	}
	defer fh.Close()
	dec := json.NewDecoder(fh)
	if err = dec.Decode(&data); err != nil {
		Syslog_.Println(err)
		return
	}
	for to, jiddata := range data {
		//set status to offline. We're going to get current online status of everyone who is currently online at reconnect
		jiddata.Online = false
		data[to] = jiddata
	}
}

func (botdata *XmppBot) handleEventsforXMPP(xmppout chan<- xmpp.Stanza, presence_events <-chan interface{}, jabber_events <-chan JabberEvent) {
	var last_status_msg *string

	defer func() {
		if x := recover(); x != nil {
			Syslog_.Printf("handleEventsforXMPP: run time panic: %v", x)
		}
		for _ = range jabber_events {
		} //cleanout jabber_events queue
	}()

	// the settle period is the time during which we receive precence updates from the jabber server right after connecting
	// presence states we want to remember but not act upon with a recap-message
	settletimer := time.NewTimer(900 * time.Millisecond)
	withinSettlePeriod := true

	for {
		select {
		case pe, pe_still_open := <-presence_events:
			if !pe_still_open {
				return
			}
			Debug_.Printf("handleEventsforXMPP<-presence_events: %T %+v", pe, pe)
			switch pec := pe.(type) {
			case xmpp.Stanza:
				xmppout <- pec
				continue
			case string:
				for to, jiddata := range botdata.realraum_jids_ {
					if jiddata.Wants >= R3DebugInfo {
						xmppout <- botdata.makeXMPPMessage(to, pec, nil)
					}
				}

			case XMPPStatusEvent:
				xmppout <- botdata.makeXMPPPresence("", "", pec.Show, pec.Status)

			case XMPPMsgEvent:
				if pec.RememberAsStatus {
					last_status_msg = &pec.Msg
				}
				for to, jiddata := range botdata.realraum_jids_ {
					if jiddata.Wants >= pec.DistributeLevel && ((jiddata.Wants >= R3OnlineOnlyInfo && jiddata.Online) || jiddata.Wants >= R3AlwaysInfo) {
						xmppout <- botdata.makeXMPPMessage(to, pec.Msg, nil)
					}
				}
			case bool:
				Debug_.Println("handleEventsforXMPP<-presence_events: shutdown received: quitting")
				return
			default:
				Debug_.Println("handleEventsforXMPP<-presence_events: unknown type received: quitting")
				return
			}

		case <-settletimer.C:
			withinSettlePeriod = false
		case je, je_still_open := <-jabber_events:
			if !je_still_open {
				return
			}
			Debug_.Printf("handleEventsforXMPP<-jabber_events: %T %+v", je, je)
			simple_jid := removeResourceFromJIDString(je.JID)
			jid_data, jid_in_map := botdata.realraum_jids_[simple_jid]

			//send status if requested, even if user never changed any settings and thus is not in map
			if last_status_msg != nil && je.StatusNow {
				xmppout <- botdata.makeXMPPMessage(je.JID, last_status_msg, nil)
			}

			// if user is already known
			if jid_in_map {
				//if R3OnlineOnlyWithRecapInfo, we want a status update when coming online
				if last_status_msg != nil && !withinSettlePeriod && !jid_data.Online && je.Online && jid_data.Wants == R3OnlineOnlyWithRecapInfo {
					xmppout <- botdata.makeXMPPMessage(je.JID, last_status_msg, nil)
				}
				jid_data.Online = je.Online
				if je.Wants > R3NoChange {
					jid_data.Wants = je.Wants
				}

				if je.ErrorOccured {
					jid_data.ErrorCount++
					if jid_data.ErrorCount > XMPP_MAX_ERROR_COUNT {
						jid_data.Wants = R3NeverInfo
					}
				}

				//save data
				botdata.realraum_jids_[simple_jid] = jid_data
				botdata.realraum_jids_.saveTo(botdata.config_file_)
			} else if je.Wants > R3NoChange {
				//save data
				botdata.realraum_jids_[simple_jid] = JidData{Online: je.Online, Wants: je.Wants}
				botdata.realraum_jids_.saveTo(botdata.config_file_)
			}
		}
	}
}

func removeResourceFromJIDString(jid string) string {
	var jidjid xmpp.JID
	jidjid.Set(jid)
	jidjid.Resource = ""
	return jidjid.String()
}

func (botdata *XmppBot) isAuthenticated(jid string) bool {
	authtime, in_map := botdata.jid_lastauthtime_[jid]
	return in_map && time.Now().Unix()-authtime < botdata.auth_timeout_
}

const help_text_ string = "\n*auth*<password>* ...Enables you to use more commands.\n*time* ...Returns bot time."
const help_text_auth string = "You are authorized to use the following commands:\n*off* ...You will no longer receive notifications.\n*on* ...You will be notified of r3 status changes while you are online.\n*on_with_recap* ...Like *on* but additionally you will receive the current status when you come online.\n*on_while_offline* ...You will receive all r3 status changes, wether you are online or offline.\n*status* ...Use it to query the current status.\n*time* ...Returns bot time.\n*bye* ...Logout."

//~ var re_msg_auth_    *regexp.Regexp     = regexp.MustCompile("auth\s+(\S+)")

func (botdata *XmppBot) handleIncomingMessageDialog(inmsg xmpp.Message, xmppout chan<- xmpp.Stanza, jabber_events chan JabberEvent) {
	if inmsg.Body == nil || inmsg.GetHeader() == nil {
		return
	}
	bodytext_args := strings.Split(strings.Replace(inmsg.Body.Chardata, "*", " ", -1), " ")
	for len(bodytext_args) > 1 && len(bodytext_args[0]) == 0 {
		bodytext_args = bodytext_args[1:len(bodytext_args)] //get rid of empty first strings resulting from " text"
	}
	bodytext_lc_cmd := strings.ToLower(bodytext_args[0])
	if botdata.isAuthenticated(inmsg.GetHeader().From) {
		switch bodytext_lc_cmd {
		case "on":
			jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3OnlineOnlyInfo, false, false}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Receive r3 status updates while online.", "Your New Status")
		case "off":
			jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3NeverInfo, false, false}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Do not receive anything.", "Your New Status")
		case "on_with_recap":
			jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3OnlineOnlyWithRecapInfo, false, false}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Receive r3 status updates while and current status on coming, online.", "Your New Status")
		case "on_while_offline":
			jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3AlwaysInfo, false, false}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Receive all r3 status updates, even if you are offline.", "Your New Status")
		case "debug":
			jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3DebugInfo, false, false}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Debug mode enabled", "Your New Status")
		case "bye", "quit", "logout":
			botdata.jid_lastauthtime_[inmsg.GetHeader().From] = 0
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Bye Bye.", nil)
		case "open", "close":
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Sorry, I'm just weak software, not strong enough to operate the door for you.", nil)
		case "status":
			jabber_events <- JabberEvent{inmsg.GetHeader().From, true, R3NoChange, true, false}
		case "time":
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, time.Now().String(), nil)
		case "ping":
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Pong with auth", nil)
		default:
			//~ auth_match = re_msg_auth_.FindStringSubmatch(inmsg.Body.Chardata)
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, help_text_auth, nil)
		}
	} else {
		switch bodytext_lc_cmd {
		case "hilfe", "help", "?", "hallo", "yes", "ja", "ja bitte", "bitte", "sowieso":
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, help_text_, "Available Commands")
		case "auth":
			authindex := 1
			for len(bodytext_args) > authindex && len(bodytext_args[authindex]) == 0 {
				authindex++
			}
			if len(bodytext_args) > authindex && bodytext_args[authindex] == botdata.password_ {
				botdata.jid_lastauthtime_[inmsg.GetHeader().From] = time.Now().Unix()
				xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, help_text_auth, nil)
			}
		case "status", "off", "on", "on_while_offline", "on_with_recap":
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Sorry, you need to be authorized to do that.", nil)
		case "time":
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, time.Now().String(), nil)
		case "ping":
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Pong", nil)
		case "":
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "You're a quiet one, aren't you?", nil)
		default:
			//~ auth_match = re_msg_auth_.FindStringSubmatch(inmsg.Body.Chardata)
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "A nice day to you too!\nDo you need \"help\"?", nil)
		}
	}
}

func (botdata *XmppBot) handleIncomingXMPPStanzas(xmppin <-chan xmpp.Stanza, xmppout chan<- xmpp.Stanza, jabber_events chan JabberEvent) {

	defer func() {
		if x := recover(); x != nil {
			Syslog_.Printf("handleIncomingXMPPStanzas: run time panic: %v", x)
		}
	}()

	var error_count int = 0
	var incoming_stanza interface{}

	handleStanzaError := func() bool {
		error_count++
		if error_count > 15 {
			Syslog_.Println("handleIncomingXMPPStanzas: too many errors in series.. bailing out")
			botdata.StopBot()
			return true
		}
		return false
	}

	for incoming_stanza = range xmppin {
		switch stanza := incoming_stanza.(type) {
		case *xmpp.Message:
			if stanza.GetHeader() == nil {
				continue
			}
			if stanza.Type == "error" || stanza.Error != nil {
				Syslog_.Printf("XMPP %T Error: %s", stanza, stanza)
				if stanza.Error.Type == "cancel" {
					// asume receipient not reachable -> increase error count
					Syslog_.Printf("Error reaching %s. Disabling user, please reenable manually", stanza.From)
					jabber_events <- JabberEvent{JID: stanza.From, Wants: R3NoChange, ErrorOccured: true, Online: false}
					continue
				}
				if handleStanzaError() {
					return
				}
				continue
			} else {
				error_count = 0
			}
			botdata.handleIncomingMessageDialog(*stanza, xmppout, jabber_events)
		case *xmpp.Presence:
			if stanza.GetHeader() == nil {
				continue
			}
			if stanza.Type == "error" || stanza.Error != nil {
				Syslog_.Printf("XMPP %T Error: %s", stanza, stanza)
				if handleStanzaError() {
					return
				}
				continue
			} else {
				error_count = 0
			}
			switch stanza.GetHeader().Type {
			case "subscribe":
				xmppout <- botdata.makeXMPPPresence(stanza.GetHeader().From, "subscribed", "", "")
				jabber_events <- JabberEvent{stanza.GetHeader().From, true, R3NoChange, false, false}
				xmppout <- botdata.makeXMPPPresence(stanza.GetHeader().From, "subscribe", "", "")
			case "unsubscribe", "unsubscribed":
				jabber_events <- JabberEvent{stanza.GetHeader().From, false, R3NeverInfo, false, false}
				botdata.jid_lastauthtime_[stanza.GetHeader().From] = 0 //logout
				xmppout <- botdata.makeXMPPPresence(stanza.GetHeader().From, "unsubscribe", "", "")
			case "unavailable":
				jabber_events <- JabberEvent{stanza.GetHeader().From, false, R3NoChange, false, false}
				botdata.jid_lastauthtime_[stanza.GetHeader().From] = 0 //logout
			default:
				jabber_events <- JabberEvent{stanza.GetHeader().From, true, R3NoChange, false, false}
			}

		case *xmpp.Iq:
			if stanza.GetHeader() == nil {
				continue
			}
			if stanza.Type == "error" || stanza.Error != nil {
				Syslog_.Printf("XMPP %T Error: %s", stanza, stanza)
				if handleStanzaError() {
					return
				}
				continue
			} else {
				error_count = 0
			}

			if HandleServerToClientPing(stanza, xmppout) {
				continue
			} //if true then routine handled it and we can continue
			Debug_.Printf("Unhandled Iq: %s", stanza)
		}
	}
}

func init() {
	//~ xmpp.Debug = &XMPPDebugLogger{}
	xmpp.Info = &XMPPDebugLogger{}
	xmpp.Warn = &XMPPLogger{}
}

func NewStartedBot(loginjid, loginpwd, password, state_save_dir string, insecuretls bool) (*XmppBot, chan interface{}, error) {
	var err error
	botdata := new(XmppBot)

	connect_timeout := time.AfterFunc(20*time.Second, func() {
		panic("NewStartedBot: connection timeout reached, no error or reply from goexmpp lib ... exiting")
	})

	botdata.realraum_jids_ = make(map[string]JidData, 1)
	botdata.jid_lastauthtime_ = make(map[string]int64, 1)
	botdata.my_jid_ = loginjid
	botdata.my_login_password_ = loginpwd
	botdata.password_ = password
	botdata.auth_timeout_ = 3600 * 2

	botdata.config_file_ = path.Join(state_save_dir, "r3xmpp."+removeResourceFromJIDString(loginjid)+".json")

	xmpp.TlsConfig = tls.Config{InsecureSkipVerify: insecuretls}
	botdata.realraum_jids_.loadFrom(botdata.config_file_)

	client_jid := new(xmpp.JID)
	client_jid.Set(botdata.my_jid_)
	botdata.xmppclient_, err = xmpp.NewClient(client_jid, botdata.my_login_password_, nil)
	if err != nil {
		Syslog_.Println("Error connecting to xmpp server", err)
		return nil, nil, err
	}
	if botdata.xmppclient_ == nil {
		Syslog_.Println("xmpp.NewClient returned nil without error")
		return nil, nil, errors.New("No answer from xmpp server")
	}

	err = botdata.xmppclient_.StartSession(true, &xmpp.Presence{})
	if err != nil {
		Syslog_.Println("'Error StartSession:", err)
		return nil, nil, err
	}

	roster := xmpp.Roster(botdata.xmppclient_)
	for _, entry := range roster {
		Debug_.Print(entry)
		if entry.Subscription == "from" {
			botdata.xmppclient_.Out <- botdata.makeXMPPPresence(entry.Jid, "subscribe", "", "")
		}
		if entry.Subscription == "none" {
			delete(botdata.realraum_jids_, entry.Jid)
		}
	}

	connect_timeout.Stop() // if we reach here, connection should have succeeded
	Syslog_.Println("NewStartedBot established connection")

	presence_events := make(chan interface{}, 1)
	jabber_events := make(chan JabberEvent, 1)

	go func() {
		for { //auto recover from panic
			if botdata.xmppclient_ == nil || botdata.xmppclient_.Out == nil {
				break
			}
			botdata.handleEventsforXMPP(botdata.xmppclient_.Out, presence_events, jabber_events)
			time.Sleep(50 * time.Millisecond)
		}
	}()
	go func() {
		for { //auto recover from panic
			if botdata.xmppclient_ == nil || botdata.xmppclient_.In == nil || botdata.xmppclient_.Out == nil {
				break
			}
			botdata.handleIncomingXMPPStanzas(botdata.xmppclient_.In, botdata.xmppclient_.Out, jabber_events)
			time.Sleep(50 * time.Millisecond)
		}
	}()

	botdata.presence_events_ = &presence_events

	return botdata, presence_events, nil
}

func (botdata *XmppBot) StopBot() {
	Syslog_.Println("Stopping XMPP Bot")
	if botdata.xmppclient_ != nil {
		close(botdata.xmppclient_.Out)
	}
	if botdata.presence_events_ != nil {
		select {
		case *botdata.presence_events_ <- false:
		default:
		}
		close(*botdata.presence_events_)
	}
	botdata.config_file_ = ""
	botdata.realraum_jids_ = nil
	botdata.xmppclient_ = nil
}

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

	"github.com/mitchellh/mapstructure"

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

const (
	JDFieldOnline              = "Online"
	JDFieldWants               = "Wants              "
	JDFieldFrontdoorUpdates    = "FrontdoorUpdates"
	JDFieldBackdoorUpdates     = "BackdoorUpdates"
	JDFieldSensorUpdates       = "SensorUpdates"
	JDFieldButtonUpdates       = "ButtonUpdates"
	JDFieldFreezerAlarmUpdates = "FreezerAlarmUpdates"
	JDFieldGasAlertUpdates     = "GasAlertUpdates"
	JDFieldFoodOrderUpdates    = "FoodOrderUpdates"
	JDFieldErrorCount          = "ErrorCount"
	JEvtStatusNow              = "StatusNow"
)

type JidData struct {
	Online                bool
	Wants                 R3JIDDesire
	NoFrontdoorUpdates    bool
	NoBackdoorUpdates     bool
	NoSensorUpdates       bool
	NoButtonUpdates       bool
	NoFreezerAlarmUpdates bool
	NoGasAlertUpdates     bool
	NoFoodOrderUpdates    bool
}

type JidDataUpdate struct {
	JID     string
	Updates JidDataUpdatesMap
}

type JidDataUpdatesMap map[string]interface{}

func updateJidData(jdata *JidData, jupdate JidDataUpdatesMap) {
	mapstructure.Decode(jupdate, jdata)
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

func (botdata *XmppBot) handleEventsforXMPP(xmppout chan<- xmpp.Stanza, presence_events <-chan interface{}, jabber_events <-chan JidDataUpdate) {
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
			jeWants, jeWants_in_map := je.Updates[JDFieldWants]

			//send status if requested, even if user never changed any settings and thus is not in map
			if _, status_now := je.Updates[JEvtStatusNow]; last_status_msg != nil && status_now {
				xmppout <- botdata.makeXMPPMessage(je.JID, last_status_msg, nil)
			}

			// if user is already known
			if jid_in_map {
				user_previously_online := jid_data.Online

				//update user info
				updateJidData(&jid_data, je.Updates)

				user_now_online := jid_data.Online

				//if R3OnlineOnlyWithRecapInfo, we want a status update when coming online
				if last_status_msg != nil && !withinSettlePeriod && !user_previously_online && user_now_online && jid_data.Wants == R3OnlineOnlyWithRecapInfo {
					xmppout <- botdata.makeXMPPMessage(je.JID, last_status_msg, nil)
				}
				//save data
				botdata.realraum_jids_[simple_jid] = jid_data
				botdata.realraum_jids_.saveTo(botdata.config_file_)

			} else if jeWants_in_map && jeWants.(R3JIDDesire) > R3NoChange {
				//new user wants to be enabled defaults
				jid_data = JidData{NoBackdoorUpdates: true}
				//save data
				updateJidData(&jid_data, je.Updates)
				//save data
				botdata.realraum_jids_[simple_jid] = jid_data
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

func (botdata *XmppBot) handleIncomingMessageDialog(inmsg xmpp.Message, xmppout chan<- xmpp.Stanza, jabber_events chan JidDataUpdate) {
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
		case "backdoorinfo":
			switch strings.ToLower(bodytext_args[1]) {
			case "on", "1", "ein", "ja":
				xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "You will receive updates about the backdoor", "Your New Status")
				jabber_events <- JidDataUpdate{inmsg.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: true, JDFieldWants: R3OnlineOnlyInfo}}
			case "off", "0", "aus", "nein":
				xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "You will no longer receive updates about the backdoor", "Your New Status")
			}
		case "on":
			jabber_events <- JidDataUpdate{inmsg.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: true, JDFieldWants: R3OnlineOnlyInfo}}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Receive r3 status updates while online.", "Your New Status")
		case "off":
			jabber_events <- JidDataUpdate{inmsg.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: true, JDFieldWants: R3NeverInfo}}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Do not receive anything.", "Your New Status")
		case "on_with_recap":
			jabber_events <- JidDataUpdate{inmsg.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: true, JDFieldWants: R3OnlineOnlyWithRecapInfo}}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Receive r3 status updates while and current status on coming, online.", "Your New Status")
		case "on_while_offline":
			jabber_events <- JidDataUpdate{inmsg.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: true, JDFieldWants: R3AlwaysInfo}}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Receive all r3 status updates, even if you are offline.", "Your New Status")
		case "debug":
			jabber_events <- JidDataUpdate{inmsg.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: true, JDFieldWants: R3DebugInfo}}
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Debug mode enabled", "Your New Status")
		case "bye", "quit", "logout":
			botdata.jid_lastauthtime_[inmsg.GetHeader().From] = 0
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Bye Bye.", nil)
		case "open", "close":
			xmppout <- botdata.makeXMPPMessage(inmsg.GetHeader().From, "Sorry, I'm just weak software, not strong enough to operate the door for you.", nil)
		case "status":
			jabber_events <- JidDataUpdate{inmsg.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: true, JEvtStatusNow: true}}
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

func (botdata *XmppBot) handleIncomingXMPPStanzas(xmppin <-chan xmpp.Stanza, xmppout chan<- xmpp.Stanza, jabber_events chan JidDataUpdate) {

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
					jabber_events <- JidDataUpdate{JID: stanza.From, Updates: JidDataUpdatesMap{JDFieldOnline: false}}
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
				jabber_events <- JidDataUpdate{stanza.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: true}}
				xmppout <- botdata.makeXMPPPresence(stanza.GetHeader().From, "subscribe", "", "")
			case "unsubscribe", "unsubscribed":
				jabber_events <- JidDataUpdate{stanza.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: false, JDFieldWants: R3NeverInfo}}
				botdata.jid_lastauthtime_[stanza.GetHeader().From] = 0 //logout
				xmppout <- botdata.makeXMPPPresence(stanza.GetHeader().From, "unsubscribe", "", "")
			case "unavailable":
				jabber_events <- JidDataUpdate{stanza.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: false}}
				botdata.jid_lastauthtime_[stanza.GetHeader().From] = 0 //logout
			default:
				jabber_events <- JidDataUpdate{stanza.GetHeader().From, JidDataUpdatesMap{JDFieldOnline: true}}
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
	jabber_events := make(chan JidDataUpdate, 1)

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

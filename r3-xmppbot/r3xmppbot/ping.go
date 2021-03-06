// (c) Bernhard Tittelbach, 2013

package r3xmppbot

import (
	"encoding/xml"
	"time"

	xmpp "github.com/curzonj/goexmpp"
)

// XMPP Ping
type XMPPPing struct {
	XMLName xml.Name `xml:"urn:xmpp:ping ping"`
}

func HandleServerToClientPing(iq *xmpp.Iq, xmppout chan<- xmpp.Stanza) bool {
	///<iq from='juliet@capulet.lit/balcony' to='capulet.lit' id='s2c1' type='result'/>
	if iq.Type != "get" {
		return false
	}
	for _, ele := range iq.Nested {
		if _, ok := ele.(*XMPPPing); ok {
			xmppout <- &xmpp.Iq{Header: xmpp.Header{To: iq.From, From: iq.To, Id: iq.Id, Type: "result"}}
			return true
		}
	}
	return false
}

//Uses XMPP Ping to check if server is still there
// returns either false or true
func (botdata *XmppBot) PingServer(timeout time.Duration) (is_up bool) {
	///<iq from='juliet@capulet.lit/balcony' to='capulet.lit' id='c2s1' type='get'>
	///  <ping xmlns='urn:xmpp:ping'/>
	///</iq>
	server_jid := new(xmpp.JID)
	server_jid.Set(botdata.my_jid_)
	iqping := &xmpp.Iq{Header: xmpp.Header{To: server_jid.Domain,
		From:   botdata.my_jid_,
		Id:     <-xmpp.Id,
		Type:   "get",
		Nested: []interface{}{XMPPPing{}}}}
	pong := make(chan bool, 1) // store one element, means if we take out exaclty one, we can put in two without blocking
	defer close(pong)          // not strictly needed, as garbage collector will take care of it sooner or later unless the functions with this jid is _never_ called
	f := func(v xmpp.Stanza) bool {
		defer func() {
			if x := recover(); x != nil {
				//if error is nil, it means we did NOT panic when trying to send false to pong channel which was already closed
				Syslog_.Printf("ping reply (or cancel) was received too late")
			}
		}() //recover from writing to possibly already closed chan.
		let_others_handle_stanza := false
		iq, ok := v.(*xmpp.Iq)
		if !ok {
			Syslog_.Printf("response to iq ping wasn't iq: %s", v)
			pong <- false
			return true //let other handlers process reply
		}
		if iq.Type == "error" && iq.Error != nil && iq.Error.Type == "cancel" {
			Debug_.Printf("response to iq ping was cancel, server does not support ping")
			//server does not support ping, but at least we know server is still there
		} else if iq.Type != "result" {
			Syslog_.Printf("response to iq ping was not pong: %s", v)
			let_others_handle_stanza = true //let other handlers process reply
		}
		pong <- true
		return let_others_handle_stanza // return false so that Stanza v will not be appear in xmppclient_.Out()
	}
	botdata.xmppclient_.HandleStanza(iqping.Id, f)
	botdata.xmppclient_.Out <- iqping
	go func() {
		defer func() {
			if x := recover(); x == nil {
				//if error is nil, it means we did NOT panic when trying to send false to pong channel which was already closed
				Syslog_.Printf("response to iq ping timed out !!")
			}
		}() //recover from writing to possibly already closed chan. If we did not need to recover, then Handler did not receive reply
		time.Sleep(timeout)
		pong <- false //xmpp ping timed out, if this succeeds. Really we expect the channel to be closed already and this statement to panic
	}()
	return <-pong
}

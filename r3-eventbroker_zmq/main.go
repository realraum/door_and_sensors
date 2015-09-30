// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	//~ "time"
	r3events "github.com/realraum/door_and_sensors/r3events"
	pubsub "github.com/tuxychandru/pubsub"
)

//~ func StringArrayToByteArray(ss []string) [][]byte {
//~ bb := make([][]byte, len(ss))
//~ for index, s := range(ss) {
//~ bb[index] = []byte(s)
//~ }
//~ return bb
//~ }

// ---------- Main Code -------------

var (
	doorsub_addr_      string
	sensorssub_port_   string
	pub_port_          string
	keylookup_addr_    string
	brain_listen_addr_ string
	door_cmd_addr_     string
	use_syslog_        bool
	enable_debuglog_   bool
)

func init() {
	flag.StringVar(&door_cmd_addr_, "doorcmdaddr", "ipc:///run/tuer/door_cmd.ipc", "zmq door event publish addr")
	flag.StringVar(&doorsub_addr_, "doorsubaddr", "tcp://zmqbroker.realraum.at:4242", "zmq door event publish addr")
	flag.StringVar(&sensorssub_port_, "sensorsubport", "tcp://*:4243", "zmq public/listen socket addr for incoming sensor data")
	flag.StringVar(&pub_port_, "pubport", "tcp://*:4244", "zmq port publishing consodilated events")
	flag.StringVar(&keylookup_addr_, "keylookupaddr", "ipc:///run/tuer/door_keyname.ipc", "address to use for key/name lookups")
	flag.StringVar(&brain_listen_addr_, "brainlisten", "tcp://*:4245", "address to listen for requests about latest stored event")
	flag.BoolVar(&use_syslog_, "syslog", false, "log to syslog local2 facility")
	flag.BoolVar(&enable_debuglog_, "debug", false, "enable debug logging")
	flag.Parse()
}

func main() {
	if enable_debuglog_ {
		LogEnableDebuglog()
	}
	if use_syslog_ {
		LogEnableSyslog()
		Syslog_.Print("started")
		defer Syslog_.Print("exiting")
	}

	zmqctx, sub_in_chans, pub_out_socket, keylookup_socket := ZmqsInit(doorsub_addr_, sensorssub_port_, pub_port_, keylookup_addr_)
	if sub_in_chans != nil {
		defer sub_in_chans.Close()
	}
	defer zmqctx.Close()
	if pub_out_socket != nil {
		defer pub_out_socket.Close()
	}
	if keylookup_socket != nil {
		defer keylookup_socket.Close()
	}
	if sub_in_chans == nil || pub_out_socket == nil || keylookup_socket == nil {
		panic("zmq sockets must not be nil !!")
	}

	ps := pubsub.New(10)
	defer ps.Shutdown() // ps.Shutdown should be called before zmq_ctx.Close(), since it will cause goroutines to shutdown and close zqm_sockets which is needed for zmq_ctx.Close() to return
	//~ ticker := time.NewTicker(time.Duration(5) * time.Minute)

	store_these_events_chan := ps.Sub("door", "doorcmd", "presence", "sensors", "buttons", "movement")
	go BrainCenter(zmqctx, brain_listen_addr_, store_these_events_chan)

	go MetaEventRoutine_Movement(ps, 10, 20, 10)
	go MetaEventRoutine_Presence(ps, 21, 200)

	// --- get update on most recent status ---
	answ := ZmqsAskQuestionsAndClose(zmqctx, door_cmd_addr_, [][][]byte{[][]byte{[]byte("status")}})
	for _, a := range answ {
		ParseSocketInputLine(a, ps, keylookup_socket)
	}

	publish_these_events_chan := ps.Sub("door", "doorcmd", "presence", "sensors", "buttons", "movement")
	for {
		select {
		case subin := <-sub_in_chans.In():
			ParseSocketInputLine(subin, ps, keylookup_socket)
		//~ case <- ticker.C:
		//~ MakeTimeTick(ps)
		case event_interface := <-publish_these_events_chan:
			data, err := r3events.MarshalEvent2ByteByte(event_interface)
			Debug_.Printf("publishing %s", data)
			if err != nil {
				Syslog_.Print(err)
				continue
			}
			if err := pub_out_socket.Send(data); err != nil {
				panic(err)
			}
		}
	}

}

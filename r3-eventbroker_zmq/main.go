// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	"os"
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
	use_syslog_      bool
	enable_debuglog_ bool
)

//-------
// available Config Environment Variables
// TUER_ZMQDOORCMD_ADDR
// TUER_ZMQDOOREVTS_ADDR
// TUER_R3EVENTS_ZMQBROKERINPUT_ADDR
// TUER_R3EVENTS_ZMQBROKERINPUT_LISTEN_ADDR
// TUER_R3EVENTS_ZMQBROKER_LISTEN_ADDR
// TUER_R3EVENTS_ZMQBRAIN_LISTEN_ADDR
// TUER_ZMQKEYNAMELOOKUP_ADDR

const (
	DEFAULT_TUER_ZMQDOORCMD_ADDR                     string = "ipc:///run/tuer/door_cmd.ipc"
	DEFAULT_TUER_ZMQDOOREVTS_ADDR                    string = "tcp://zmqbroker.realraum.at:4242"
	DEFAULT_TUER_R3EVENTS_ZMQBROKERINPUT_ADDR        string = "tcp://zmqbroker.realraum.at:4243"
	DEFAULT_TUER_R3EVENTS_ZMQBROKERINPUT_LISTEN_ADDR string = "tcp://*:4243"
	DEFAULT_TUER_R3EVENTS_ZMQBROKER_LISTEN_ADDR      string = "tcp://*:4244"
	DEFAULT_TUER_R3EVENTS_ZMQBRAIN_LISTEN_ADDR       string = "tcp://*:4245"
	DEFAULT_TUER_ZMQKEYNAMELOOKUP_ADDR               string = "ipc:///run/tuer/door_keyname.ipc"
)

func init() {
	flag.BoolVar(&use_syslog_, "syslog", false, "log to syslog local2 facility")
	flag.BoolVar(&enable_debuglog_, "debug", false, "enable debug logging")
	flag.Parse()
}

func EnvironOrDefault(envvarname, defvalue string) string {
	if len(os.Getenv(envvarname)) > 0 {
		return os.Getenv(envvarname)
	} else {
		return defvalue
	}
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

	zmqctx, sub_in_chans, pub_out_socket, keylookup_socket := ZmqsInit(
		EnvironOrDefault("TUER_ZMQDOOREVTS_ADDR", DEFAULT_TUER_ZMQDOOREVTS_ADDR),
		EnvironOrDefault("TUER_R3EVENTS_ZMQBROKERINPUT_LISTEN_ADDR", DEFAULT_TUER_R3EVENTS_ZMQBROKERINPUT_LISTEN_ADDR),
		EnvironOrDefault("TUER_R3EVENTS_ZMQBROKER_LISTEN_ADDR", DEFAULT_TUER_R3EVENTS_ZMQBROKER_LISTEN_ADDR),
		EnvironOrDefault("TUER_ZMQKEYNAMELOOKUP_ADDR", DEFAULT_TUER_ZMQKEYNAMELOOKUP_ADDR))
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
	go BrainCenter(zmqctx,
		EnvironOrDefault("TUER_R3EVENTS_ZMQBRAIN_LISTEN_ADDR", DEFAULT_TUER_R3EVENTS_ZMQBRAIN_LISTEN_ADDR),
		store_these_events_chan)

	go MetaEventRoutine_Movement(ps, 10, 20, 10)
	go MetaEventRoutine_Presence(ps, 21, 200)

	// --- get update on most recent status ---
	answ := ZmqsAskQuestionsAndClose(zmqctx, EnvironOrDefault("TUER_ZMQDOORCMD_ADDR", DEFAULT_TUER_ZMQDOORCMD_ADDR), [][][]byte{[][]byte{[]byte("status")}})
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

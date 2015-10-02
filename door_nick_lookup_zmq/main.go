// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	"fmt"
	"os"
	//~ "log/syslog"
	//~ "log"
)

// ---------- Main Code -------------

var (
	use_syslog_   bool
	start_server_ bool
	//~ syslog_ *log.Logger
)

const (
	DEFAULT_TUER_KEYSFILE_PATH         string = "/flash/keys"
	DEFAULT_TUER_ZMQKEYNAMELOOKUP_ADDR string = "ipc:///run/tuer/door_keyname.ipc"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: door_nick_lookup_zmq [options] [keyid1 [keyid2 [...]]]\n")
	flag.PrintDefaults()
}

func init() {
	flag.BoolVar(&start_server_, "server", false, "open 0mq socket and listen to requests")
	flag.Usage = usage
	flag.Parse()
}

func EnvironOrDefault(envvarname, defvalue string) string {
	if len(os.Getenv(envvarname)) > 0 {
		return os.Getenv(envvarname)
	} else {
		return defvalue
	}
}

func getFileMTime(filename string) (int64, error) {
	keysfile, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer keysfile.Close()
	stat, err := keysfile.Stat()
	if err != nil {
		return 0, err
	}
	return stat.ModTime().Unix(), nil
}

func main() {
	knstore := new(KeyNickStore)
	err := knstore.LoadKeysFile(EnvironOrDefault("DEFAULT_TUER_KEYSFILE_PATH", DEFAULT_TUER_KEYSFILE_PATH))
	if err != nil {
		panic(err)
	}
	door_keys_mtime, err := getFileMTime(EnvironOrDefault("DEFAULT_TUER_KEYSFILE_PATH", DEFAULT_TUER_KEYSFILE_PATH))
	if err != nil {
		panic(err)
	}

	for _, key := range flag.Args() {
		nick, err := knstore.LookupHexKeyNick(key)
		if err != nil {
			fmt.Printf("ERROR: %s for key %s\n", err.Error(), key)
		} else {
			fmt.Println(key, nick)
		}
	}

	if !start_server_ {
		os.Exit(0)
	}

	zmqctx, zmqchans := ZmqsInit(EnvironOrDefault("TUER_ZMQKEYNAMELOOKUP_ADDR", DEFAULT_TUER_ZMQKEYNAMELOOKUP_ADDR))
	if zmqchans == nil {
		os.Exit(0)
	}
	defer zmqchans.Close()
	defer zmqctx.Close()

	//~ if use_syslog_ {
	//~ var logerr error
	//~ syslog_, logerr = syslog.NewLogger(syslog.LOG_INFO | syslog.LOG_LOCAL2, 0)
	//~ if logerr != nil { panic(logerr) }
	//~ syslog_.Print("started")
	//~ defer syslog_.Print("exiting")
	//~ }

	for keybytes := range zmqchans.In() {
		current_door_keys_mtime, err := getFileMTime(EnvironOrDefault("DEFAULT_TUER_KEYSFILE_PATH", DEFAULT_TUER_KEYSFILE_PATH))
		if err == nil && current_door_keys_mtime > door_keys_mtime {
			door_keys_mtime = current_door_keys_mtime
			knstore.LoadKeysFile(EnvironOrDefault("DEFAULT_TUER_KEYSFILE_PATH", DEFAULT_TUER_KEYSFILE_PATH))
		}
		nick, err := knstore.LookupHexKeyNick(string(keybytes[0]))
		if err != nil {
			zmqchans.Out() <- [][]byte{[]byte("ERROR"), []byte(err.Error())}
		} else {
			zmqchans.Out() <- [][]byte{[]byte("RESULT"), keybytes[0], []byte(nick)}
		}
	}
}

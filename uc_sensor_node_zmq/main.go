// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	"time"

	zmq "github.com/vaughan0/go-zmq"
)

// ---------- Main Code -------------

var (
	tty_dev_      string
	pub_addr      string
	use_syslog_   bool
	enable_debug_ bool
	serial_speed_ uint
)

const exponential_backof_activation_threshold int64 = 4

func init() {
	flag.StringVar(&pub_addr, "brokeraddr", "tcp://zmqbroker.realraum.at:4243", "zmq address to send stuff to")
	flag.StringVar(&tty_dev_, "ttydev", "/dev/ttyACM0", "path do tty uc device")
	flag.UintVar(&serial_speed_, "serspeed", 0, "tty baudrate (0 to disable setting a baudrate e.g. in case of ttyACM)")
	flag.BoolVar(&use_syslog_, "syslog", false, "log to syslog local1 facility")
	flag.BoolVar(&enable_debug_, "debug", false, "debugging messages on")
	flag.Parse()
}

func ConnectSerialToZMQ(pub_sock *zmq.Socket, timeout time.Duration) {
	defer func() {
		if x := recover(); x != nil {
			Syslog_.Println(x)
		}
	}()

	serial_wr, serial_rd, err := OpenAndHandleSerial(tty_dev_, serial_speed_)
	if err != nil {
		panic(err)
	}
	defer close(serial_wr)

	t := time.NewTimer(timeout)
	for {
		select {
		case incoming_ser_line, seropen := <-serial_rd:
			if !seropen {
				return
			}
			t.Reset(timeout)
			Syslog_.Printf("%s", incoming_ser_line)
			if err := pub_sock.Send(incoming_ser_line); err != nil {
				Syslog_.Println(err.Error())
			}

		case <-t.C:
			Syslog_.Print("Timeout, no message for 120 seconds")
		}
	}
}

func main() {
	zmqctx, pub_sock := ZmqsInit(pub_addr)
	if pub_sock == nil {
		panic("zmq socket creation failed")
	}
	defer zmqctx.Close()
	defer pub_sock.Close()

	if enable_debug_ {
		LogEnableDebuglog()
	} else if use_syslog_ {
		LogEnableSyslog()
		Syslog_.Print("started")
	}

	var backoff_exp uint32 = 0
	for {
		start_time := time.Now().Unix()
		ConnectSerialToZMQ(pub_sock, time.Second*120)
		run_time := time.Now().Unix() - start_time
		if run_time > exponential_backof_activation_threshold {
			backoff_exp = 0
		}
		time.Sleep(150 * (1 << backoff_exp) * time.Millisecond)
		if backoff_exp < 12 {
			backoff_exp++
		}
	}
}

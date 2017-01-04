// (c) Bernhard Tittelbach, 2013

package main

import (
	"flag"
	"os/exec"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/realraum/door_and_sensors/r3events"
)

// ---------- Main Code -------------

var (
	tty_dev_      string
	pub_addr      string
	use_syslog_   bool
	enable_debug_ bool
	serial_speed_ uint
)

type SerialLine []string

const exponential_backof_activation_threshold int64 = 4

const (
	DEFAULT_R3_MQTT_BROKER              string = "tcp://mqtt.realraum.at:1883"
	DEFAULT_R3_AJARSENSOR_TTY_PATH      string = "/dev/backdoor"
	DEFAULT_R3_GASLEAK2SMS_MININTERVAL  string = "45"
	DEFAULT_R3_GASLEAK2SMS_DESTINATIONS string = "livesclose"
)

func init() {
	flag.BoolVar(&use_syslog_, "syslog", false, "log to syslog local1 facility")
	flag.BoolVar(&enable_debug_, "debug", false, "debugging messages on")
	flag.Parse()
}

func SendSMS(groups []string, text string) {
	cmd := exec.Command("/usr/local/bin/send_group_sms.sh", groups...)
	stdinpipe, err := cmd.StdinPipe()
	if err != nil {
		Syslog_.Printf("Error sending text to smsscript: %s", err)
		return
	}
	err = cmd.Start()
	if err != nil {
		Syslog_.Printf("Error sending sms: %s", err)
		return
	}
	stdinpipe.Write([]byte(text))
	stdinpipe.Close()
}

func ConnectSerialToMQTT(mc mqtt.Client, timeout time.Duration) {
	defer func() {
		if x := recover(); x != nil {
			Syslog_.Println(x)
		}
	}()

	var gasleak_min_interval time.Duration
	var err error
	if gasleak_min_interval, err = time.ParseDuration(EnvironOrDefault("R3_GASLEAK2SMS_MININTERVAL", DEFAULT_R3_GASLEAK2SMS_MININTERVAL)); err != nil {
		gasleak_min_interval = time.Duration(30 * time.Minute) //minutes between notifications per sms
	}
	gasleak_last_time := time.Now().Add(-1 * gasleak_min_interval)
	gasleak_smsnotification_destinations := strings.Fields(EnvironOrDefault("R3_GASLEAK2SMS_DESTINATIONS", DEFAULT_R3_GASLEAK2SMS_DESTINATIONS))

	serial_wr, serial_rd, err := OpenAndHandleSerial(EnvironOrDefault("R3_AJARSENSOR_TTY_PATH", DEFAULT_R3_AJARSENSOR_TTY_PATH), 57600)
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
			var tk mqtt.Token
			switch incoming_ser_line[0] {
			case "temp1:", "temp2:", "temp0:":
				temp, err := strconv.ParseFloat(incoming_ser_line[1], 64)
				if err != nil {
					Syslog_.Print("Error parsing float", err)
					continue
				}
				payload := r3events.MarshalEvent2ByteOrPanic(r3events.TempSensorUpdate{Location: "CX", Ts: time.Now().Unix(), Value: temp})
				tk = mc.Publish(r3events.TOPIC_BACKDOOR_TEMP, 0, true, payload)
			case "BackdoorInfo(ajar):":
				payload := r3events.MarshalEvent2ByteOrPanic(r3events.BackdoorAjarUpdate{Shut: incoming_ser_line[1] == "shut", Ts: time.Now().Unix()})
				tk = mc.Publish(r3events.TOPIC_BACKDOOR_AJAR, 2, true, payload)
			case "GasLeakAlert":
				if time.Now().Sub(gasleak_last_time) >= gasleak_min_interval {
					gasleak_last_time = time.Now()
					SendSMS(gasleak_smsnotification_destinations, "r3 ALERT: possible GAS LEAK detected")
				}
				payload := r3events.MarshalEvent2ByteOrPanic(r3events.GasLeakAlert{Ts: time.Now().Unix()})
				tk = mc.Publish(r3events.TOPIC_BACKDOOR_GASALERT, 2, false, payload)
			default:
				Syslog_.Printf("Received unknown line")
			}
			if tk != nil {
				tk.Wait()
				if tk.Error() != nil {
					Syslog_.Print("mqtt publish error", tk.Error())
				}
			}
		case <-t.C:
			Syslog_.Print("Timeout, no message for 120 seconds")
		}
	}
}

func main() {
	if enable_debug_ {
		LogEnableDebuglog()
	} else if use_syslog_ {
		LogEnableSyslog()
		Syslog_.Print("started")
	}

	options := mqtt.NewClientOptions().AddBroker(EnvironOrDefault("R3_MQTT_BROKER", DEFAULT_R3_MQTT_BROKER)).SetAutoReconnect(true).SetProtocolVersion(4).SetCleanSession(true)
	mqttclient := mqtt.NewClient(options)
	ctk := mqttclient.Connect()
	ctk.Wait()
	if ctk.Error() != nil {
		Syslog_.Fatal("Error connecting to MQTT broker", ctk.Error())
	}

	var backoff_exp uint32 = 0
	for {
		start_time := time.Now().Unix()
		ConnectSerialToMQTT(mqttclient, time.Second*120)
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

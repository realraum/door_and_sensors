// (c) Bernhard Tittelbach, 2013

package r3xmppbot

import "os"
import "log"
import "log/syslog"

type NullWriter struct {}
func (n *NullWriter) Write(p []byte) (int, error) {return len(p),nil}

var (
    Syslog_ *log.Logger
    Debug_ *log.Logger
)

func init() {
    Syslog_ = log.New(&NullWriter{}, "", 0)
    Debug_ = log.New(&NullWriter{}, "", 0)
}

func LogEnableSyslog() {
    var logerr error
    Syslog_, logerr = syslog.NewLogger(syslog.LOG_INFO | (18<<3), 0)
    if logerr != nil { panic(logerr) }
}

func LogEnableDebuglog() {
    Syslog_ = log.New(os.Stdout, "", log.LstdFlags)
    Debug_ = log.New(os.Stderr, "DEBUG ", log.LstdFlags)
}

type XMPPLogger struct {
}

func (s *XMPPLogger) Log(v ...interface{}) {
        Syslog_.Println(v...)
}

func (s *XMPPLogger) Logf(fmt string, v ...interface{}) {
        Syslog_.Printf(fmt, v...)
}

type XMPPDebugLogger struct {
}

func (s *XMPPDebugLogger) Log(v ...interface{}) {
        Debug_.Println(v...)
}

func (s *XMPPDebugLogger) Logf(fmt string, v ...interface{}) {
        Debug_.Printf(fmt, v...)
}
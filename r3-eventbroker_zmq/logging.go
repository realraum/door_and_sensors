// (c) Bernhard Tittelbach, 2013

package main

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
    Syslog_ = log.New(os.Stdout, "", log.LstdFlags)
    Debug_ = log.New(&NullWriter{}, "", 0)
}

func LogEnableSyslog() {
    var logerr error
    Syslog_, logerr = syslog.NewLogger(syslog.LOG_INFO | (18<<3), 0)
    if logerr != nil { panic(logerr) }
}

func LogEnableDebuglog() {
    Debug_ = log.New(os.Stderr, "DEBUG", log.LstdFlags)
}
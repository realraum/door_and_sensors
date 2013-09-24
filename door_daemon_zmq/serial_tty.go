// (c) Bernhard Tittelbach, 2013

package main

import (
    "fmt"
    "bufio"
    "bytes"
    "os"
    "svn.spreadspace.org/realraum/go.svn/termios"
    "log"
    "sync"
)

// ---------- Serial TTY Code -------------

func openTTY(name string) (*os.File, error) {
    file, err := os.OpenFile(name,os.O_RDWR, 0600) // For read access.
    if err != nil {
        log.Println(err.Error())
        return nil, err
    }
    termios.Ttyfd(file.Fd())
    termios.SetRaw()
    return file, nil
}

func serialWriter(in <- chan string, serial * os.File) {
    for totty := range(in) {
        serial.WriteString(totty)
        serial.Sync()
    }
}

var last_read_serial_input [][]byte = [][]byte{{}}
var last_read_mutex sync.Mutex

func serialReader(topub chan <- [][]byte, serial * os.File) {
    linescanner := bufio.NewScanner(serial)
    linescanner.Split(bufio.ScanLines)
    for linescanner.Scan() {
        if err := linescanner.Err(); err != nil {
            panic(fmt.Sprintf("Error in read from serial: %v\n",err.Error()))
        }
        fmt.Println("read text", linescanner.Text())
        text := bytes.Fields([]byte(linescanner.Text()))
        if len(text) == 0 {
            continue
        }
        //~ for len(serial_read) > 5 {
            //~ //drain channel before putting new line into it
            //~ //thus we make sure "out" only ever holds the last line
            //~ //thus the buffer never blocks and we don't need to read from out unless we need it
            //~ // BUT: don't drain the chan dry, or we might have a race condition resulting in a deadlock
            //~ <- serial_read
        //~ }
        last_read_mutex.Lock()
        last_read_serial_input = text
        fmt.Println("Put Text", text)
        last_read_mutex.Unlock()
        topub <- text
    }
}

//TODO: improve this, make it work for multiple open serial devices
func GetLastSerialLine() [][]byte {
    var last_line_pointer [][]byte
    last_read_mutex.Lock()
    last_line_pointer = last_read_serial_input
    last_read_mutex.Unlock()
    fmt.Println("Retrieve Text", last_line_pointer)
    return last_line_pointer
}

func OpenAndHandleSerial(filename string, topub chan <- [][]byte) (chan string, error) {
    serial, err :=openTTY(filename)
    if err != nil {
        return nil, err
    }
    wr := make(chan string)
    go serialWriter(wr, serial)
    go serialReader(topub, serial)
    return wr, nil
}

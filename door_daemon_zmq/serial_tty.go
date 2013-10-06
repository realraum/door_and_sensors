// (c) Bernhard Tittelbach, 2013

package main

import (
    "bufio"
    "bytes"
    "os"
    "svn.spreadspace.org/realraum/go.svn/termios"
)


// ---------- Serial TTY Code -------------

func openTTY(name string) (*os.File, error) {
    file, err := os.OpenFile(name,os.O_RDWR, 0600) // For read access.
    if err != nil {
        Syslog_.Println(err.Error())
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
    serial.Close()
}

func serialReader(out chan <- [][]byte, serial * os.File) {
    linescanner := bufio.NewScanner(serial)
    linescanner.Split(bufio.ScanLines)
    for linescanner.Scan() {
        if err := linescanner.Err(); err != nil {
            panic(err.Error())
        }
        text := bytes.Fields([]byte(linescanner.Text()))
        if len(text) == 0 {
            continue
        }
        out <- text
    }
}

func OpenAndHandleSerial(filename string) (chan string, chan [][]byte, error) {
    serial, err :=openTTY(filename)
    if err != nil {
        return nil, nil, err
    }
    wr := make(chan string, 1)
	rd := make(chan [][]byte, 20)
    go serialWriter(wr, serial)
    go serialReader(rd, serial)
    return wr, rd, nil
}

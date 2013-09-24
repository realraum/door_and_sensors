// (c) Bernhard Tittelbach, 2013

package main

import (
    "fmt"
    "bufio"
    "bytes"
    "os"
    "svn.spreadspace.org/realraum/go.svn/termios"
    "log"
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

func SerialWriter(in <- chan string, serial * os.File) {
    for totty := range(in) {
        serial.WriteString(totty)
        serial.Sync()
    }
}

func SerialReader(out chan <- [][]byte, topub chan <- [][]byte, serial * os.File) {
    linescanner := bufio.NewScanner(serial)
    linescanner.Split(bufio.ScanLines)
    for linescanner.Scan() {
        if err := linescanner.Err(); err != nil {
            panic(fmt.Sprintf("Error in read from serial: %v\n",err.Error()))
        }
        text := bytes.Fields([]byte(linescanner.Text()))
        if len(text) == 0 {
            continue
        }
        out <- text
        topub <- text
    }
}

func OpenAndHandleSerial(filename string, topub chan <- [][]byte) (chan string, chan [][]byte, error) {
    serial, err :=openTTY(filename)
    if err != nil {
        return nil, nil, err
    }
    var wr chan string
    var rd chan [][]byte
    go SerialWriter(wr, serial)
    go SerialReader(rd, topub, serial)
    return wr, rd, nil
}

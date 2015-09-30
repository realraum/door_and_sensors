// (c) Bernhard Tittelbach, 2013

package main

import (
	"bufio"
	"bytes"
	"errors"
	"os"

	"github.com/realraum/go/termios"
)

// ---------- Serial TTY Code -------------

func openTTY(name string, speed uint) (file *os.File, err error) {
	file, err = os.OpenFile(name, os.O_RDWR, 0600)
	if err != nil {
		return
	}
	if err = termios.SetRawFile(file); err != nil {
		return
	}
	switch speed {
	case 0: // set no baudrate
	case 1200:
		err = termios.SetSpeedFile(file, termios.B1200)
	case 2400:
		err = termios.SetSpeedFile(file, termios.B2400)
	case 4800:
		err = termios.SetSpeedFile(file, termios.B4800)
	case 9600:
		err = termios.SetSpeedFile(file, termios.B9600)
	case 19200:
		err = termios.SetSpeedFile(file, termios.B19200)
	case 38400:
		err = termios.SetSpeedFile(file, termios.B38400)
	case 57600:
		err = termios.SetSpeedFile(file, termios.B57600)
	case 115200:
		err = termios.SetSpeedFile(file, termios.B115200)
	case 230400:
		err = termios.SetSpeedFile(file, termios.B230400)
	default:
		file.Close()
		err = errors.New("Unsupported Baudrate, use 0 to disable setting a baudrate")
	}
	return
}

func serialWriter(in <-chan string, serial *os.File) {
	for totty := range in {
		serial.WriteString(totty)
		serial.Sync()
	}
	serial.Close()
}

func serialReader(out chan<- [][]byte, serial *os.File) {
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

func OpenAndHandleSerial(filename string, serspeed uint) (chan string, chan [][]byte, error) {
	serial, err := openTTY(filename, serspeed)
	if err != nil {
		return nil, nil, err
	}
	wr := make(chan string, 1)
	rd := make(chan [][]byte, 20)
	go serialWriter(wr, serial)
	go serialReader(rd, serial)
	return wr, rd, nil
}

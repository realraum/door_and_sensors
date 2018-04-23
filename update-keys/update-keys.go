package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"regexp"
	"syscall"

	"github.com/schleibinger/sio"
)

var re_keynickline *regexp.Regexp = regexp.MustCompile("^\\s*([0-9a-fA-F]+).*")

//max UID-length of 7 bytes + 1 checksum bytes
const KEYSLOT_LEN = 8
const EEPROM_SIZE = 1024
const NUM_KEYSLOTS = EEPROM_SIZE / KEYSLOT_LEN

var FW_EXPECTED_E_ANSW = []byte("Info(keystore): flashing\r\n")

type KeySlotT []byte

var doortty string

func init() {
	flag.StringVar(&doortty, "doortty", "/dev/door", "tuer tty device")
	flag.Parse()
}

func readStdinAndFilterLines(new_keys_chan chan<- KeySlotT) {
	linescanner := bufio.NewScanner(os.Stdin)
	linescanner.Split(bufio.ScanLines)
	for linescanner.Scan() {
		line := linescanner.Bytes()
		if len(line) == 0 {
			continue
		}
		matches := re_keynickline.FindSubmatch(line)
		if matches == nil {
			fmt.Println("rejecting line (not matching regex):", string(line))
			continue
		}
		hexkey := matches[1]
		if hexkey != nil && len(hexkey)%2 == 0 && len(hexkey) > 0 && len(hexkey) <= 2*KEYSLOT_LEN {
			newkey := make(KeySlotT, KEYSLOT_LEN)
			//bytes in newkey are automagically initialized with 0 by golang
			//so remaining bytes of e.g. 4 bytes key are 0
			nchars, err := hex.Decode(newkey, hexkey)
			if err == nil && nchars == len(hexkey)/2 {
				generateFletcher8Checksum(newkey)
				// fmt.Println("read key:", newkey)
				new_keys_chan <- newkey
			}
		} else {
			fmt.Println("rejecting line (invalid keylen", len(hexkey), "):", string(line))
		}
	}
	if err := linescanner.Err(); err != nil {
		panic(err)
	}
	close(new_keys_chan)
}

func generateFletcher8Checksum(key KeySlotT) {
	var sum1 byte = 0x0f
	var sum2 byte = 0x0f

	for x := 0; x < KEYSLOT_LEN-1; x++ {
		sum1 += key[x]
		sum2 += sum1
	}
	csum := sum2<<4 | sum1
	//add checksum to key
	key[KEYSLOT_LEN-1] = csum
}

func writeKeysToSerial(serial *sio.Port, new_keys_chan <-chan KeySlotT) {
	num_keys_written := 0
	// serial.Write([]byte{'e', '\r', '\n'})
	serial.Write([]byte{'e'})
	rxbuf := make([]byte, len(FW_EXPECTED_E_ANSW))
	serial.Read(rxbuf)
	if bytes.Compare(rxbuf, FW_EXPECTED_E_ANSW) != 0 {
		panic("unexpected response to eeprom programming request from firmware")
	}
	for keyslotdata := range new_keys_chan {
		firmware_answer := make([]byte, 1)
		serial.Write(keyslotdata[0:KEYSLOT_LEN])
		// fmt.Println(keyslotdata[0:KEYSLOT_LEN])
		serial.Read(firmware_answer)
		if firmware_answer[0] == '0' {
			panic("send_key failed at keyslot")
		} else if firmware_answer[0] == '.' {
			fmt.Printf("key %d sent\n", num_keys_written)
		} else {
			fmt.Printf("Unknown reply from firmware: %c\n", firmware_answer[0])
		}
		num_keys_written++
		if num_keys_written > NUM_KEYSLOTS {
			panic("EEPROM full, could not write any more keys")
		}
	}
	//fill the rest of the buffer with invalid keys
	invalid_key := make(KeySlotT, KEYSLOT_LEN)
	for z := 0; z < KEYSLOT_LEN; z++ {
		invalid_key[z] = 0xff
	}
	for ; num_keys_written < NUM_KEYSLOTS; num_keys_written++ {
		serial.Write(invalid_key)
		fmt.Printf("empty key %d sent\n", num_keys_written)
	}
	serial.Close()
}

func main() {
	serial, err := sio.Open(doortty, syscall.B115200)
	if err != nil {
		panic(err)
	}
	new_keys_chan := make(chan KeySlotT, 40)
	go readStdinAndFilterLines(new_keys_chan)
	writeKeysToSerial(serial, new_keys_chan)
}

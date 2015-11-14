// (c) Bernhard Tittelbach, 2013

package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"regexp"
	"strconv"
)

var re_keynickline *regexp.Regexp = regexp.MustCompile("^\\s*([0-9a-fA-F]+)\\s+((?:\\p{Latin}|[[:graph:]]|[[:digit:]])+).*")

type KeyNickStore struct {
	key_nick_map map[uint64]string
	filename     string
	file_mtime   int64
}

func NewKeyNickStore(filename string) (*KeyNickStore, error) {
	rv := &KeyNickStore{make(map[uint64]string), filename, 0}
	if err := rv.LoadKeysFileIfNeeded(); err != nil {
		return nil, err
	}
	return rv, nil
}

func (keystore *KeyNickStore) LoadKeysFileIfNeeded() error {
	current_door_keys_mtime, err := getFileMTime(keystore.filename)
	if err != nil {
		return err
	}
	if current_door_keys_mtime <= keystore.file_mtime {
		return nil
	}
	keystore.file_mtime = current_door_keys_mtime

	keysfile, err := os.OpenFile(keystore.filename, os.O_RDONLY, 0400) // For read access.
	defer keysfile.Close()
	if err != nil {
		return err
	}

	//clear map
	keystore.key_nick_map = make(map[uint64]string)

	linescanner := bufio.NewScanner(keysfile)
	linescanner.Split(bufio.ScanLines)
	for linescanner.Scan() {
		m := re_keynickline.FindStringSubmatch(linescanner.Text())
		if len(m) > 2 {
			if kuint, err := strconv.ParseUint(m[1], 16, 64); err == nil {
				(keystore.key_nick_map)[kuint] = m[2]
			} else {
				log.Print("Error converting hex-cardid:", m[0])
			}
		}
	}
	return nil
}

func (keystore *KeyNickStore) LookupHexKeyNick(key string) (string, error) {
	if err := keystore.LoadKeysFileIfNeeded(); err != nil {
		return "", err
	}
	kuint, err := strconv.ParseUint(key, 16, 64)
	if err != nil {
		return "", errors.New("Invalid Hex-Card-Id")
	}
	nick, present := (keystore.key_nick_map)[kuint]
	if present {
		return nick, nil
	} else {
		return "", errors.New("Key Unknown")
	}
}

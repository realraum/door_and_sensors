// (c) Bernhard Tittelbach, 2013

package main

import (
    "log"
    "regexp"
    "strconv"
    "errors"
    "os"
    "bufio"
)

var re_keynickline *regexp.Regexp = regexp.MustCompile("^\\s*([0-9a-fA-F]+)\\s+((?:\\p{Latin}|[[:graph:]]|[[:digit:]])+).*")

type KeyNickStore map[uint64]string

func (key_nick_map *KeyNickStore) LoadKeysFile(filename string) error {
    keysfile, err := os.OpenFile(filename, os.O_RDONLY, 0400) // For read access.
    defer keysfile.Close()
    if err != nil {
        return err
    }
    
    //clear map
    *key_nick_map = make(KeyNickStore)
    
    linescanner := bufio.NewScanner(keysfile)
    linescanner.Split(bufio.ScanLines)
    for linescanner.Scan() {
        m := re_keynickline.FindStringSubmatch(linescanner.Text())
        if len(m) > 2 {
            if kuint, err := strconv.ParseUint(m[1], 16, 64); err == nil {
                (*key_nick_map)[kuint] = m[2]
            } else {
                log.Print("Error converting hex-cardid:",m[0])
            }
        }
    }
    return nil
}

func (key_nick_map *KeyNickStore) LookupHexKeyNick( key string ) (string, error) {
    kuint, err := strconv.ParseUint(key, 16, 64)
    if err != nil {
        return "", errors.New("Invalid Hex-Card-Id")
    }
    nick, present := (*key_nick_map)[kuint]
    if present {
        return nick, nil
    } else {
        return "", errors.New("Key Unknown")
    }
}
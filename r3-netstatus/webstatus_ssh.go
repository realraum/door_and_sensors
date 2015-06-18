// (c) Bernhard Tittelbach, Christian Pointner, 2015

package main

import (
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

var ssh_webstatus_client_ *ssh.Client

func connectWebStatusSSHConnection() (*ssh.Client, error) {
	privateBytes, err := ioutil.ReadFile(environOrDefault("SSH_ID_FILE", "/flash/tuer/id_rsa"))
	if err != nil {
		Syslog_.Println("Error: Failed to load ssh private key:", err.Error())
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey([]byte(privateBytes))
	if err != nil {
		Syslog_.Println("Error: Failed to parse ssh private key:", err.Error())
		return nil, err
	}
	config := &ssh.ClientConfig{
		User: environOrDefault("SSH_USER", "www-data"),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}
	client, err := ssh.Dial("tcp", environOrDefault("SSH_HOST_PORT", "vex.realraum.at:2342"), config)
	if err != nil {
		Syslog_.Println("Error: Failed to connect to ssh host:", err.Error())
		return nil, err
	}
	return client, nil
}

func getWebStatusSSHSession() (session *ssh.Session) {
	var err error
	for attempts := 2; attempts > 0; attempts-- {
		if ssh_webstatus_client_ == nil {
			ssh_webstatus_client_, err = connectWebStatusSSHConnection()
			if err != nil {
				continue
			}
		}
		session, err = ssh_webstatus_client_.NewSession()
		if err != nil {
			Syslog_.Println("Error: Failed to create ssh session:", err.Error())
			ssh_webstatus_client_.Close()
			ssh_webstatus_client_ = nil
			continue
		}
		break
	}
	return session
}

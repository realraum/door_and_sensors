// (c) Bernhard Tittelbach, Christian Pointner, 2015

package main

import (
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

var ssh_webstatus_client_ *ssh.Client

func connectWebStatusSSHConnection() (*ssh.Client, error) {
	privateBytes, err := ioutil.ReadFile(EnvironOrDefault("TUER_STATUSPUSH_SSH_ID_FILE", DEFAULT_TUER_STATUSPUSH_SSH_ID_FILE))
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
		User: EnvironOrDefault("TUER_STATUSPUSH_SSH_USER", DEFAULT_TUER_STATUSPUSH_SSH_USER),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}
	client, err := ssh.Dial("tcp", EnvironOrDefault("TUER_STATUSPUSH_SSH_HOST_PORT", DEFAULT_TUER_STATUSPUSH_SSH_HOST_PORT), config)
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

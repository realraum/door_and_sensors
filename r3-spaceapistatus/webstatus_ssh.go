// (c) Bernhard Tittelbach, Christian Pointner, 2015

package main

import (
	"fmt"
	"io/ioutil"
	"time"

	"golang.org/x/crypto/ssh"
)

var session_request_chan_ chan session_request

type session_request struct {
	Future chan *ssh.Session
}

func init() {
	session_request_chan_ = make(chan session_request)
	go goCreateSSHSessions()
}

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
	// hostpubkey, err := ssh.ParsePublicKey([]byte(EnvironOrDefault("TUER_STATUSPUSH_SSH_REMOTEHOSTKEY", "")))
	// if err != nil {
	// 	Syslog_.Fatalf("could not parse ssh remote host pubkey: %s", err.Error())
	// }
	config := &ssh.ClientConfig{
		User: EnvironOrDefault("TUER_STATUSPUSH_SSH_USER", DEFAULT_TUER_STATUSPUSH_SSH_USER),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //FIXME: verify against vex HostKey provided via EnvironOrDefault path
		// HostKeyCallback: ssh.FixedHostKey(hostpubkey),
		Timeout: 3 * time.Second,
	}
	client, err := ssh.Dial("tcp", EnvironOrDefault("TUER_STATUSPUSH_SSH_HOST_PORT", DEFAULT_TUER_STATUSPUSH_SSH_HOST_PORT), config)
	if err != nil {
		Syslog_.Println("Error: Failed to connect to ssh host:", err.Error())
		return nil, err
	}
	return client, nil
}

func goCreateSSHSessions() {
	var ssh_webstatus_client *ssh.Client
NEXTSREQ:
	for sreq := range session_request_chan_ {
		var err error
		for attempts := 2; attempts > 0; attempts-- {
			session_chan := make(chan *ssh.Session)
			err_chan := make(chan error)
			timeout_tmr := time.NewTimer(6 * time.Second)
			go func() {
				defer func() {
					if x := recover(); x != nil {
						err_chan <- fmt.Errorf("recovered from %s", x)
					}
				}()
				if ssh_webstatus_client == nil {
					ssh_webstatus_client, err = connectWebStatusSSHConnection()
					if err != nil || ssh_webstatus_client == nil {
						Syslog_.Println("Error: Failed to connect to ssh daemon:", err.Error())
						ssh_webstatus_client = nil
						err_chan <- err
						return
					}
				}
				session, err := ssh_webstatus_client.NewSession()
				if err == nil {
					session_chan <- session
					return
				} else {
					ssh_webstatus_client.Close()
					err_chan <- err
					return
				}
			}()
			select {
			case <-timeout_tmr.C:
				Syslog_.Println("Error: Failed to create ssh session in time")
				ssh_webstatus_client = nil
			case err = <-err_chan:
				Syslog_.Println("Error: Failed to create ssh session:", err.Error())
				ssh_webstatus_client = nil
			case session := <-session_chan:
				sreq.Future <- session
				close(sreq.Future)
				continue NEXTSREQ
			}
		}
		sreq.Future <- nil
		close(sreq.Future)
		continue NEXTSREQ
	}
}

func getWebStatusSSHSession() (session *ssh.Session) {
	mysession := make(chan *ssh.Session)
	session_request_chan_ <- session_request{mysession}
	return <-mysession
}

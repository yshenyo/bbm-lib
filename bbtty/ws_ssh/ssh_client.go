package ws_ssh

import (
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

type SShClientConfig struct {
	Ip       string
	Port     int
	UserName string
	Password string
}

type SSHClient struct {
	Client  *ssh.Client
	Session *ssh.Session
}

func NewWsSSHClient(cfg SShClientConfig) (*SSHClient, error) {
	config := &ssh.ClientConfig{
		Timeout:         time.Second * 5,
		User:            cfg.UserName,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	config.Auth = []ssh.AuthMethod{ssh.Password(cfg.Password)}

	addr := fmt.Sprintf("%s:%d", cfg.Ip, cfg.Port)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	conn := &SSHClient{
		Client:  sshClient,
		Session: sshSession,
	}
	return conn, nil
}

package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type CommonSSH struct {
	userName     string
	addr         string
	session      *ssh.Session
	client       *ssh.Client
	clientConfig *ssh.ClientConfig
	auth         []ssh.AuthMethod
	timeOut      time.Duration
}

type CommonSSHInterface interface {
	SetConnect() error
	CombinedOutput(cmd string) (out string, err error) //运行命令，并返回标准输出和标准错误
	Run(cmd string) (out string, err error)            //开始指定命令并且等待他执行结束
	Close()
}

func NewCommonSSH(userName, passWord, host, port, sshKey string) (CommonSSHInterface, error) {
	var commonSSh CommonSSH
	commonSSh.timeOut = 10 * time.Second
	commonSSh.userName = userName
	commonSSh.addr = fmt.Sprintf("%s:%s", host, port)
	if sshKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(sshKey))
		if err != nil {
			return nil, err
		}
		commonSSh.auth = append(commonSSh.auth, ssh.PublicKeys(signer))
	} else {
		commonSSh.auth = append(commonSSh.auth, ssh.Password(passWord))
	}
	return &commonSSh, nil
}

func (c *CommonSSH) SetConnect() error {
	var err error
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
	c.clientConfig = &ssh.ClientConfig{
		User:            c.userName,
		Auth:            c.auth,
		Timeout:         c.timeOut,
		HostKeyCallback: hostKeyCallback,
	}
	retry := 3
	for i := 0; i < retry; i++ {
		c.client, err = ssh.Dial("tcp", c.addr, c.clientConfig)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	for i := 0; i < retry; i++ {
		c.session, err = c.client.NewSession()
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *CommonSSH) CombinedOutput(cmd string) (out string, err error) {
	outByte, err := c.session.CombinedOutput(cmd)
	output := string(outByte[:])
	if err != nil {
		if output != "" {
			return output, nil
		} else {
			return
		}
	}
	return output, nil
}

func (c *CommonSSH) Run(cmd string) (out string, err error) {
	var stdOut, stdErr bytes.Buffer
	c.session.Stdout = &stdOut
	c.session.Stderr = &stdErr
	err = c.session.Run(cmd)
	if err != nil && stdErr.Len() == 0 {
		return
	}
	if stdErr.Len() > 0 {
		return stdOut.String(), errors.New(stdErr.String())
	}
	return stdOut.String(), nil
}

func (c *CommonSSH) Close() {
	c.client.Close()
	c.session.Close()
}

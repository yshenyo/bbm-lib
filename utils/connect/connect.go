package connect

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/spf13/cast"
	"github.com/zjbobingtech/bbm-lib/utils/encryption"
	"golang.org/x/crypto/ssh"
)

func TelnetAddress(address string) error {
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	if err != nil {
		return err
	}
	if conn == nil {
		return fmt.Errorf("%v cant connect", address)
	}
	return nil
}

func TelnetIPPort(ip string, port int) error {
	address := net.JoinHostPort(ip, cast.ToString(port))
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	if err != nil {
		return err
	}
	if conn == nil {
		return fmt.Errorf("%v cant connect", address)
	}
	return nil
}

func PingIPV4(ip string) error {
	conn, err := net.DialTimeout("ip4:icmp", ip, 10*time.Second)
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	if err != nil {
		return err
	}
	if conn == nil {
		return errors.New("IP地址无法Ping通")
	}
	return nil
}

func SShConnectTest(username string, password string, key string, ip string, port interface{}) error {
	if key != "" {
		return SShConnectWithKeyTest(username, key, ip, port)
	}
	return SShConnectWithPasswordTest(username, password, ip, port)
}

func SShConnectWithPasswordTest(username string, password string, ip string, port interface{}) error {
	var sc *ssh.ClientConfig
	p := encryption.PasswordDecode(password)
	sc = &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(p),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%v", ip, port), sc)
	if err != nil {
		return err
	}
	defer client.Close()
	return nil
}

func SShConnectWithKeyTest(username string, key string, ip string, port interface{}) error {
	var sc *ssh.ClientConfig
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {

	}
	sc = &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         8 * time.Second,
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%v", ip, port), sc)
	if err != nil {
		return err
	}
	defer client.Close()
	return nil
}

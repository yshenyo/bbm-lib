package bbtty

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/gorilla/websocket"
)

const maxPacket = 1 << 15

type AllClients struct {
	client            map[string]*ssh.Client //map[uniqueKey]*ssh.Client
	lastActiveTime    map[string]int64       //map[uniqueKey]timestamp
	keepAliveTime     map[string]int64       //map[uniqueKey]int64
	closeCallBackFunc map[string]func()      //map[uniqueKey]callFunc 关闭回调
	mu                sync.RWMutex
	logicSshWsSession map[string]*LogicSshWsSession
}

var allClients *AllClients

func init() {
	allClients = &AllClients{
		client:            make(map[string]*ssh.Client),
		lastActiveTime:    make(map[string]int64),
		closeCallBackFunc: make(map[string]func()),
		logicSshWsSession: make(map[string]*LogicSshWsSession),
	}
}

func (cli *AllClients) GetLogicSshWsSession(uniqueKey string, wsConn *websocket.Conn) *LogicSshWsSession {
	if uniqueKey == "" {
		return nil
	}
	allClients.mu.Lock()
	defer allClients.mu.Unlock()
	if logic, ok := allClients.logicSshWsSession[uniqueKey]; ok {
		logic.wsConn = wsConn
		return logic
	}
	return nil
}

func (cli *AllClients) AddLogicSshWsSession(uniqueKey string, c *LogicSshWsSession) {
	if uniqueKey == "" {
		return
	}
	allClients.mu.Lock()
	defer allClients.mu.Unlock()
	if _, ok := allClients.logicSshWsSession[uniqueKey]; ok {
		return
	}
	allClients.logicSshWsSession[uniqueKey] = c
	go cli.AutoRecycle(uniqueKey)
}

func (cli *AllClients) AddClient(uniqueKey string, c *ssh.Client, callBackFunc func(), keepAlive int64) {
	if uniqueKey == "" {
		return
	}
	allClients.mu.Lock()
	defer allClients.mu.Unlock()
	if _, ok := allClients.client[uniqueKey]; ok {
		return
	}
	allClients.keepAliveTime[uniqueKey] = keepAlive
	allClients.client[uniqueKey] = c
	allClients.lastActiveTime[uniqueKey] = time.Now().Unix()
	allClients.closeCallBackFunc[uniqueKey] = callBackFunc
	go cli.AutoRecycle(uniqueKey)
}

func (cli *AllClients) GetClient(uniqueKey string) *ssh.Client {
	if uniqueKey == "" {
		return nil
	}
	allClients.mu.Lock()
	defer allClients.mu.Unlock()
	if cli, ok := allClients.client[uniqueKey]; ok {
		allClients.lastActiveTime[uniqueKey] = time.Now().Unix()
		return cli
	}
	return nil
}

func (cli *AllClients) DelClient(uniqueKey string) {
	allClients.mu.Lock()
	defer allClients.mu.Unlock()
	if _, ok := allClients.client[uniqueKey]; !ok {
		return
	}
	delete(allClients.client, uniqueKey)
	delete(allClients.lastActiveTime, uniqueKey)
	delete(allClients.closeCallBackFunc, uniqueKey)
	delete(allClients.logicSshWsSession, uniqueKey)
}

func (cli *AllClients) UpdateLastActiveTime(uniqueKey string) {
	allClients.mu.Lock()
	defer allClients.mu.Unlock()
	if _, ok := allClients.client[uniqueKey]; !ok {
		return
	}
	allClients.lastActiveTime[uniqueKey] = time.Now().Unix()
}

func (cli *AllClients) AutoRecycle(uniqueKey string) {
	for {
		lastActiveTime_Interval := int64(0)
		sleepTime := time.Duration(lastActiveTime_Interval)
		allClients.mu.Lock()
		lastActiveTime, ok := allClients.lastActiveTime[uniqueKey]
		if !ok {
			allClients.mu.Unlock()
			time.Sleep(sleepTime * time.Second)
			return
		}
		aliveTime, ok := allClients.keepAliveTime[uniqueKey]
		if ok {
			lastActiveTime_Interval = aliveTime
			sleepTime = time.Duration(lastActiveTime_Interval)
		}
		if time.Now().Unix()-lastActiveTime >= lastActiveTime_Interval {
			//执行关闭回调
			if f, ok := allClients.closeCallBackFunc[uniqueKey]; ok {
				f()
			}
			delete(allClients.client, uniqueKey)
			delete(allClients.lastActiveTime, uniqueKey)
			delete(allClients.closeCallBackFunc, uniqueKey)
			delete(allClients.logicSshWsSession, uniqueKey)
			allClients.mu.Unlock()
			time.Sleep(sleepTime * time.Second)
			continue
		}
		allClients.mu.Unlock()
		time.Sleep(sleepTime * time.Second)
	}
}

func NewSshClient(ip string, port int, user string, password string, uniqueKey string, closeCallBackFunc func(), keepAlive int64) (*ssh.Client, error) {
	if cli := allClients.GetClient(uniqueKey); cli != nil {
		return cli, nil
	}

	config := &ssh.ClientConfig{
		Timeout:         time.Second * 5,
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //这个可以， 但是不够安全
		//HostKeyCallback: hostKeyCallBackFunc(h.Host),
	}

	config.Auth = []ssh.AuthMethod{ssh.Password(password)}

	addr := fmt.Sprintf("%s:%d", ip, port)
	c, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	allClients.AddClient(uniqueKey, c, closeCallBackFunc, keepAlive)
	return c, nil
}

func NewSshClientSingle(ip string, port int, user string, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		Timeout:         time.Second * 5,
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //这个可以， 但是不够安全
		//HostKeyCallback: hostKeyCallBackFunc(h.Host),
	}

	config.Auth = []ssh.AuthMethod{ssh.Password(password)}

	addr := fmt.Sprintf("%s:%d", ip, port)
	c, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func RunCommand(client *ssh.Client, command string) (stdout string, err error) {
	session, err := client.NewSession()
	if err != nil {
		//log.Print(err)
		return
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf
	session.Stderr = &buf
	err = session.Run(command)
	if err != nil {
		return
	}
	stdout = string(buf.Bytes())
	return
}

func NewSftpClient(ip string, port int, user string, password string, keeyAlive int64) (*sftp.Client, error) {
	conn, err := NewSshClient(ip, port, user, password, "", nil, keeyAlive)
	if err != nil {
		return nil, err
	}
	return sftp.NewClient(conn, sftp.MaxPacket(maxPacket))
}
func toUnixPath(path string) string {
	return filepath.Clean(path)
}

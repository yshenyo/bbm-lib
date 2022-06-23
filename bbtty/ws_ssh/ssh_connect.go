package ws_ssh

import (
	"io"
	"time"

	"golang.org/x/crypto/ssh"

	"bobingtech/inspect/bblog"
	"bobingtech/inspect/common/bbtty"
)

type WsSSH struct {
	ssh         *SSHClient
	stdinPipe   io.WriteCloser
	comboOutput *bbtty.SafeBuffer //ssh 终端混合输出
	isClose     bool
}

type WsSShConnectConfig struct {
	SShConfig      SShClientConfig
	Cols           int
	Rows           int
	SSHReceiveChan chan []byte //接收数据的通道 消费者在SSH client
	SSHSendChan    chan []byte //ssh 发送数据的通道
	Close          chan bool   //通知外部是否关闭
}

func NewWsSShConnect(cfg WsSShConnectConfig) (*WsSSH, error) {
	sshClient, err := NewWsSSHClient(cfg.SShConfig)
	if err != nil {
		return nil, err
	}
	stdinP, err := sshClient.Session.StdinPipe()
	if err != nil {
		return nil, err
	}

	comboWriter := new(bbtty.SafeBuffer)

	//ssh 终端内容写入
	sshClient.Session.Stdout = comboWriter
	sshClient.Session.Stderr = comboWriter

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := sshClient.Session.RequestPty("xterm", cfg.Rows, cfg.Cols, modes); err != nil {
		return nil, err
	}

	if err := sshClient.Session.Shell(); err != nil {
		return nil, err
	}
	SSHReceiveChan := cfg.SSHReceiveChan
	SSHSendChan := cfg.SSHSendChan
	conn := &WsSSH{
		ssh:         sshClient,
		stdinPipe:   stdinP,
		comboOutput: comboWriter,
		isClose:     false,
	}

	go conn.sshReceive(SSHReceiveChan)
	go conn.sshSend(SSHSendChan)
	go conn.wait(cfg.Close)
	return conn, nil
}

//ssh 接收到外部数据 丢给服务器
func (ws *WsSSH) sshReceive(receive chan []byte) {
	for {
		if ws.isClose {
			return
		}
		content := <-receive
		if len(content) > 0 {
			if _, err := ws.stdinPipe.Write(content); err != nil {
				bblog.Logger.ZapLog.Error("sshReceive error", err.Error())
			}
		}
	}
}

//ssh 接收到服务器返回的值丢入需要send通道
func (ws *WsSSH) sshSend(send chan []byte) {
	//every 120ms write combine output bytes into websocket response
	tick := time.NewTicker(time.Millisecond * time.Duration(60))
	//for range time.Tick(120 * time.Millisecond){}
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if ws.comboOutput == nil {
				return
			}
			if ws.isClose {
				return
			}
			bs := ws.comboOutput.Bytes()
			if len(bs) > 0 {
				send <- bs
				ws.comboOutput.Buffer.Reset()
			}
		}
	}
}

//修改窗口
func (ws *WsSSH) WindowChange(rows, cols int) error {
	if err := ws.ssh.Session.WindowChange(rows, cols); err != nil {
		bblog.Logger.ZapLog.Error("WsSSH WindowChange error", err.Error())
		return err
	}
	return nil
}

//监听session 情况
func (ws *WsSSH) wait(close chan bool) {
	if err := ws.ssh.Session.Wait(); err != nil {
		bblog.Logger.ZapLog.Error("WsSSH Wait error", err.Error())
		ws.Close()
		close <- true
	}
}

//主动关闭
func (ws *WsSSH) Close() {
	ws.isClose = true
	ws.ssh.Session.Close()
	ws.ssh.Client.Close()
}

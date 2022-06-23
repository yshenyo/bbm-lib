package bbtty

import (
	"bobingtech/inspect/bblog"
	"encoding/base64"
	"encoding/json"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"io"
	"strings"
	"time"
)

type LogicSshWsSession struct {
	stdinPipe       io.WriteCloser
	comboOutput     *SafeBuffer //ssh 终端混合输出
	logBuff         *SafeBuffer //保存session的日志
	inputFilterBuff *SafeBuffer //用来过滤输入的命令和ssh_filter配置对比的
	session         *ssh.Session
	wsConn          *websocket.Conn
	isAdmin         bool
	IsFlagged       bool `comment:"当前session是否包含禁止命令"`
	FilterMap       map[string]struct{}
	SshUniqueKey    string //ssh 连接的唯一id
}

func NewLogicSshWsSession(cols, rows int, isAdmin bool, sshClient *ssh.Client, wsConn *websocket.Conn, filterMap map[string]struct{}, sshUniqueKey string) (*LogicSshWsSession, error) {

	if s := allClients.GetLogicSshWsSession(sshUniqueKey, wsConn); s != nil {
		return s, nil
	}
	sshSession, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}

	stdinP, err := sshSession.StdinPipe()
	if err != nil {
		return nil, err
	}

	comboWriter := new(SafeBuffer)
	logBuf := new(SafeBuffer)
	inputBuf := new(SafeBuffer)
	//ssh.stdout and stderr will write output into comboWriter
	sshSession.Stdout = comboWriter
	sshSession.Stderr = comboWriter

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := sshSession.RequestPty("xterm", rows, cols, modes); err != nil {
		return nil, err
	}
	// Start remote shell
	if err := sshSession.Shell(); err != nil {
		return nil, err
	}
	logic := &LogicSshWsSession{
		stdinPipe:       stdinP,
		comboOutput:     comboWriter,
		logBuff:         logBuf,
		inputFilterBuff: inputBuf,
		session:         sshSession,
		wsConn:          wsConn,
		isAdmin:         isAdmin,
		IsFlagged:       false,
		FilterMap:       filterMap,
		SshUniqueKey:    sshUniqueKey,
	}
	allClients.AddLogicSshWsSession(sshUniqueKey, logic)
	return logic, nil
}

//Close 关闭
func (sws *LogicSshWsSession) Close() {
	if sws.session != nil {
		sws.session.Close()
	}
	if sws.logBuff != nil {
		sws.logBuff = nil
	}
	if sws.comboOutput != nil {
		sws.comboOutput = nil
	}
}
func (sws *LogicSshWsSession) Start(quitChan chan bool) {
	go sws.receiveWsMsg(quitChan)
	go sws.sendComboOutput(quitChan)
}

//receiveWsMsg  receive websocket msg do some handling then write into ssh.session.stdin
func (sws *LogicSshWsSession) receiveWsMsg(exitCh chan bool) {
	//tells other go routine quit
	defer setQuit(exitCh)
	for {
		select {
		case <-exitCh:
			return
		default:
			//read websocket msg
			_, wsData, err := sws.wsConn.ReadMessage()
			if err != nil {
				bblog.Logger.Error("reading webSocket message failed", zap.Error(err))
				return
			}
			//unmashal bytes into struct
			msgObj := WebsocketMsg{}
			if err := json.Unmarshal(wsData, &msgObj); err != nil {
				bblog.Logger.Error("unmarshal websocket message failed", zap.Error(err), zap.String("wsData", string(wsData)))
			}
			switch msgObj.Type {
			case wsMsgResize:
				//handle xterm.js size change
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := sws.session.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						bblog.Logger.Error("ssh pty change windows size failed", zap.Error(err))
					}
				}
			case WsConfirm:
				cs, err := base64.StdEncoding.DecodeString("DQ==")
				if err != nil {
					bblog.Logger.Error("base64 DecodeString fail", zap.Error(err))
				}
				sws.inputFilterBuff.Reset()
				sws.sendWebsocketInputCommandToSshSessionStdinPipe(cs)
			case wsMsgCmd:
				decodeBytes, err := base64.StdEncoding.DecodeString(msgObj.Cmd)
				if err != nil {
					bblog.Logger.Error("websocket cmd string base64 decoding failed", zap.Error(err))
				}
				sws.inputFilterBuff.Write(decodeBytes)
				if msgObj.Cmd == "Aw==" {
					sws.inputFilterBuff.Reset()
				}
				if msgObj.Cmd == "DQ==" || decodeBytes[len(decodeBytes)-1] == '\r' {
					cmd := sws.inputFilterBuff.Buffer.String()
					cmd = strings.TrimSpace(cmd)

					//cb, err := base64.StdEncoding.DecodeString(cmd)
					//if err != nil {
					//	continue
					//}
					cf := false
					for f, _ := range sws.FilterMap {
						if strings.Contains(cmd, f) {
							cf = true
							break
						}
					}
					//if _, ok := sws.FilterMap[cmd]; ok {
					if cf {
						//cs, err := base64.StdEncoding.DecodeString("Aw==")
						//if err != nil {
						//
						//}
						//decodeBytes = cs
						//sws.sendWebsocketInputCommandToSshSessionStdinPipe(cs)
						//time.Sleep(time.Microsecond * 100)
						data := WebsocketMsg{
							Type: WsConfirm,
							Cmd:  "高危命令禁止操作!是否继续？",
						}
						errMsg, err := json.Marshal(data)
						if err != nil {
							bblog.Logger.Error("ssh json.Marsha failed", zap.Error(err))
						}
						err = sws.wsConn.WriteMessage(websocket.TextMessage, errMsg)
						if err != nil {
							bblog.Logger.Error("ssh sending combo output to webSocket failed", zap.Error(err))
						}
						sws.inputFilterBuff.Reset()
						continue
					}
					sws.inputFilterBuff.Reset()
				}
				sws.sendWebsocketInputCommandToSshSessionStdinPipe(decodeBytes)
			}
		}
	}
}

//sendWebsocketInputCommandToSshSessionStdinPipe
func (sws *LogicSshWsSession) sendWebsocketInputCommandToSshSessionStdinPipe(cmdBytes []byte) {
	if _, err := sws.stdinPipe.Write(cmdBytes); err != nil {
		bblog.Logger.Error("ws cmd bytes write to ssh.stdin pipe failed", zap.Error(err))
	}
}

func (sws *LogicSshWsSession) sendComboOutput(exitCh chan bool) {
	//todo 优化成一个方法
	//tells other go routine quit
	defer setQuit(exitCh)
	//every 120ms write combine output bytes into websocket response
	tick := time.NewTicker(time.Millisecond * time.Duration(60))
	//for range time.Tick(120 * time.Millisecond){}
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if sws.comboOutput == nil {
				return
			}
			bs := sws.comboOutput.Bytes()
			if len(bs) > 0 {
				data := WebsocketMsg{
					Type: wsMsgCmd,
					Cmd:  string(bs),
				}
				bs, err := json.Marshal(data)
				if err != nil {
					bblog.Logger.Error("ssh json.Marsha failed", zap.Error(err))
				}
				err = sws.wsConn.WriteMessage(websocket.TextMessage, bs)
				if err != nil {
					bblog.Logger.Error("ssh sending combo output to webSocket failed", zap.Error(err))
				}
				_, err = sws.logBuff.Write(bs)
				if err != nil {
					bblog.Logger.Error("combo output to log buffer failed", zap.Error(err))
				}
				sws.comboOutput.Buffer.Reset()
				//记录最后一次的活动时间
				go allClients.UpdateLastActiveTime(sws.SshUniqueKey)
			}

		case <-exitCh:
			return
		}
	}
}

func (sws *LogicSshWsSession) Wait(quitChan chan bool) {
	if err := sws.session.Wait(); err != nil {
		bblog.Logger.Error("ssh session wait failed", zap.Error(err))
		setQuit(quitChan)
	}
}

func (sws *LogicSshWsSession) LogString() string {
	return sws.logBuff.Buffer.String()
}

func setQuit(ch chan bool) {
	ch <- true
}

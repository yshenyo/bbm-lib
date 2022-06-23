package bbtty

import (
	"bobingtech/inspect/bblog"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/pkg/errors"
)

// WebTTY bridges a PTY slave and its PTY master.
// To support text-based streams and side channel commands such as
// terminal resizing, WebTTY uses an original protocol.
type WebTTY struct {
	// PTY Master, which probably a connection to browser
	MasterConn Master
	// PTY Slave
	Slave           Slave
	LogBuffer       *SafeBuffer
	inputFilterBuff *SafeBuffer
	WindowTitle     []byte
	PermitWrite     bool
	Columns         int
	Rows            int
	Reconnect       int // in seconds
	MasterPrefs     []byte

	BufferSize int
	WriteMutex sync.Mutex
	StartFlag  string
	UniqueKey  string // 连接的唯一id

	ErrCache  [][]byte //发送错误的时候返回值
	Uuid      string
	keepAlive int64 //连接保留时间
}

type AllWebTTY struct {
	webTTY         map[string]*WebTTY
	lastActiveTime map[string]int64
	mu             sync.RWMutex
}

var allWebTTY *AllWebTTY

func init() {
	allWebTTY = &AllWebTTY{
		webTTY:         make(map[string]*WebTTY),
		lastActiveTime: make(map[string]int64),
	}
}

func (a *AllWebTTY) addWebTTY(uniqueKey string, tty *WebTTY) {
	allWebTTY.mu.Lock()
	defer allWebTTY.mu.Unlock()
	if _, ok := allWebTTY.webTTY[uniqueKey]; ok {
		return
	}
	allWebTTY.webTTY[uniqueKey] = tty
	allWebTTY.lastActiveTime[uniqueKey] = time.Now().Unix()
	go a.AutoRecycle(uniqueKey)
}

func (a *AllWebTTY) getWebTTY(uniqueKey string) *WebTTY {
	allWebTTY.mu.Lock()
	defer allWebTTY.mu.Unlock()
	if tty, ok := allWebTTY.webTTY[uniqueKey]; ok {
		return tty
	}
	return nil
}

func (a *AllWebTTY) UpdateLastActiveTime(uniqueKey string) {
	allWebTTY.mu.Lock()
	defer allWebTTY.mu.Unlock()
	if _, ok := allWebTTY.lastActiveTime[uniqueKey]; !ok {
		return
	}
	allWebTTY.lastActiveTime[uniqueKey] = time.Now().Unix()
}

func (a *AllWebTTY) AutoRecycle(uniqueKey string) {
	for {
		allWebTTY.mu.Lock()
		lastActiveTime_Interval := int64(0)
		sleepTime := time.Duration(lastActiveTime_Interval)
		lastActiveTime, ok := allWebTTY.lastActiveTime[uniqueKey]
		if !ok {
			allWebTTY.mu.Unlock()
			time.Sleep(sleepTime * time.Second)
			return
		}
		webTTY, ok := allWebTTY.webTTY[uniqueKey]
		if ok {
			lastActiveTime_Interval = webTTY.keepAlive
			sleepTime = time.Duration(lastActiveTime_Interval)
		}
		if time.Now().Unix()-lastActiveTime >= lastActiveTime_Interval {
			delete(allWebTTY.lastActiveTime, uniqueKey)
			delete(allWebTTY.webTTY, uniqueKey)
			allSlaves.DelSlave(uniqueKey)
			allWebTTY.mu.Unlock()
			time.Sleep(sleepTime * time.Second)
			continue
		}
		allWebTTY.mu.Unlock()
		time.Sleep(sleepTime * time.Second)
	}
}

func (a *AllWebTTY) DelWebTTY(uniqueKey string) {
	_, ok := allWebTTY.webTTY[uniqueKey]
	if !ok {
		return
	}

	allWebTTY.mu.Lock()
	defer allWebTTY.mu.Unlock()
	delete(allWebTTY.lastActiveTime, uniqueKey)
	delete(allWebTTY.webTTY, uniqueKey)
	allSlaves.DelSlave(uniqueKey)

}

func NewWebTTY(masterConn Master, slave Slave, cols, rows int, uniqueKey string, keepAlive int64) (*WebTTY, error) {

	if tty := allWebTTY.getWebTTY(uniqueKey); tty != nil {
		tty.WriteMutex.Lock()
		defer tty.WriteMutex.Unlock()
		//将新的socket赋值到原连接上
		tty.MasterConn = masterConn
		tty.Columns = cols
		tty.Rows = rows
		tty.BufferSize = 1024 * 1024 * 8
		tty.LogBuffer = &SafeBuffer{}
		tty.inputFilterBuff = &SafeBuffer{}
		tty.PermitWrite = true
		return tty, nil
	}
	wt := &WebTTY{
		MasterConn:      masterConn,
		Slave:           slave,
		LogBuffer:       &SafeBuffer{},
		inputFilterBuff: &SafeBuffer{},
		PermitWrite:     true,
		Columns:         cols,
		Rows:            rows,
		BufferSize:      1024 * 1024 * 8,
		UniqueKey:       uniqueKey,
		Uuid:            uuid.New().String(),
		keepAlive:       keepAlive,
	}
	allWebTTY.addWebTTY(uniqueKey, wt)

	return wt, nil
}

// Run starts the main process of the WebTTY.
// This method blocks until the context is canceled.
// Note that the master and slave are left intact even
// after the context is canceled. Closing them is caller's
// responsibility.
// If the connection to one end gets closed, returns ErrSlaveClosed or ErrMasterClosed.
func (wt *WebTTY) Run(ctx context.Context, exitCache bool) error {
	err := wt.sendInitializeMessage()
	if err != nil {
		return errors.Wrapf(err, "failed to send initializing message")
	}
	errs := make(chan error, 2)

	//db
	go func() {
		errs <- func() error {
			Output := false
			if wt.StartFlag == "" {
				Output = true
			}

			if exitCache {
				Output = true
			}
			for {
				buffer := make([]byte, wt.BufferSize)

				n, err := wt.Slave.Read(buffer)

				if err != nil {
					return ErrSlaveClosed
				}
				//记录最后一次的活动时间
				go allSlaves.UpdateLastActiveTime(wt.UniqueKey)

				if !Output {
					rl := strings.Split(string(buffer[:n]), "\n")
					for _, r := range rl {
						if strings.HasPrefix(r, "ORA-") {
							err = wt.handleSlaveReadEvent([]byte(r + "\n"))
							if err != nil {
								return err
							}
							return errors.New("db connection failed")
						}

						if strings.HasPrefix(r, wt.StartFlag) {
							Output = true
							err = wt.handleSlaveReadEvent([]byte(rl[len(rl)-1]))
							if err != nil {
								return err
							}
							break
						}
					}
					if Output {
						continue
					}
				}
				if !Output {
					continue
				}
				err = wt.handleSlaveReadEvent(buffer[:n])
				if err != nil {
					return err
				}

			}
		}()
	}()

	//客户端
	go func() {
		errs <- func() error {
			for {
				go allWebTTY.UpdateLastActiveTime(wt.UniqueKey)
				buffer := make([]byte, wt.BufferSize)
				n, err := wt.MasterConn.Read(buffer)
				if err != nil {
					return ErrMasterClosed
				}
				err = wt.handleMasterReadEvent(buffer[:n])
				if err != nil {
					bblog.Logger.Error("handleMasterReadEvent error", zap.Error(err))
					return err
				}
			}
		}()
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-errs:
	}
	return err
}

func (wt *WebTTY) WriteMaster(msg string) {
	wm := WebsocketMsg{
		Type: WsMsgCmd,
		Cmd:  base64.StdEncoding.EncodeToString([]byte(msg)) + "DQ==",
	}
	wmb, _ := json.Marshal(wm)
	wt.handleMasterReadEvent(append([]byte{}, wmb...))
}

func (wt *WebTTY) WriteSlave(msg string) error {
	eb, err := base64.StdEncoding.DecodeString("DQ==")
	if err != nil {
		return errors.Wrapf(err, "failed to write received data to slave")
	}
	_, err = wt.Slave.Write(append([]byte(msg), eb...))
	if err != nil {
		return errors.Wrapf(err, "failed to write received data to slave")
	}
	return nil
}

func (wt *WebTTY) WriteEnterToMaster() {
	wm := WebsocketMsg{
		Type: WsMsgCmd,
		Cmd:  "DQ==",
	}
	wmb, _ := json.Marshal(wm)
	wt.handleMasterReadEvent(append([]byte{}, wmb...))
}

func (wt *WebTTY) sendInitializeMessage() error {
	err := wt.masterWrite(append([]byte{}, wt.WindowTitle...))
	if err != nil {
		return errors.Wrapf(err, "failed to send window title")
	}

	if wt.Reconnect > 0 {
		reconnect, _ := json.Marshal(wt.Reconnect)
		err := wt.masterWrite(append([]byte{}, reconnect...))
		if err != nil {
			return errors.Wrapf(err, "failed to set reconnect")
		}
	}

	if wt.MasterPrefs != nil {
		err := wt.masterWrite(append([]byte{}, wt.MasterPrefs...))
		if err != nil {
			return errors.Wrapf(err, "failed to set preferences")
		}
	}

	return nil
}

func (wt *WebTTY) handleSlaveReadEvent(data []byte) error {
	//safeMessage := base64.StdEncoding.EncodeToString(data)
	//safeMessage := string(data)
	err := wt.masterWrite(append([]byte{}, data...))
	if err != nil {
		return errors.Wrapf(err, "failed to send message to master")
	}
	_, err = wt.LogBuffer.Write(data)
	if err != nil {
		bblog.Logger.Error("combo output to log buffer failed", zap.Error(err))
	}

	return nil
}
func (wt *WebTTY) masterWrite(data []byte) error {
	//if len(wt.ErrCache) > 0 {
	//	for _, v := range wt.ErrCache {
	//		_, _ = wt.MasterConn.Write(v[:])
	//	}
	//}
	//
	resp := WebsocketMsg{}
	resp.Type = wsMsgCmd
	resp.Cmd = string(data[:])

	if len(data) > 0 {
		if strings.Contains(string(data[:]), "root@") {
			resp.Type = WsError
			resp.Cmd = "断开连接"
			wt.ErrCache = [][]byte{}
			defer allWebTTY.DelWebTTY(wt.UniqueKey)
		}
	}

	respByte, err := json.Marshal(resp)
	if err != nil {
		return errors.Wrapf(err, "failed to  json.Marshal")
	}
	_, err = wt.MasterConn.Write(respByte[:])
	if err != nil {
		wt.ErrCache = append(wt.ErrCache, data)
		return errors.Wrapf(err, "failed to write to master")
	}
	return nil
}

func (wt *WebTTY) handleMasterReadEvent(data []byte) error {
	if len(data) == 0 {
		return errors.New("unexpected zero length read from master")
	}
	args := WebsocketMsg{}
	err := json.Unmarshal(data[:], &args)
	if err != nil {
		return errors.New("json.Unmarshal error" + err.Error())
	}
	switch args.Type {
	case WsMsgCmd:

		if !wt.PermitWrite {
			return nil
		}
		//if len(data) <= 1 {
		//	return nil
		//}
		decodeBytes, err := base64.StdEncoding.DecodeString(args.Cmd)
		if err != nil {
			return errors.Wrapf(err, "websocket cmd string base64 decoding failed")
		}
		//过滤 退出命令
		_, err = wt.Slave.Write(decodeBytes)
		if err != nil {
			wt.inputFilterBuff.Reset()
			return errors.Wrapf(err, "failed to write received data to slave")
		}
	case WsPing:
		err := wt.masterWrite([]byte{})
		if err != nil {
			return errors.Wrapf(err, "failed to return Pong message to master")
		}
	case WsMsgResize:
		if args.Cols > 0 && args.Rows > 0 {
			wt.Slave.ResizeTerminal(args.Cols, args.Rows)
		} else {
			if wt.Columns > 0 && wt.Rows > 0 {
				wt.Slave.ResizeTerminal(wt.Columns, wt.Rows)
			}
		}
	default:
		return errors.Errorf("unknown message type `%c`", data[0])
	}

	return nil
}

func (wt *WebTTY) LogString() string {
	return wt.LogBuffer.Buffer.String()
}

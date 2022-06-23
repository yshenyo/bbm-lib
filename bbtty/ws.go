package bbtty

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type WsWrapper struct {
	*websocket.Conn
}

type WebsocketMsg struct {
	Type    string `json:"type,omitempty"`
	Cmd     string `json:"cmd,omitempty"`
	Cols    int    `json:"cols,omitempty"`
	Rows    int    `json:"rows,omitempty"`
	Confirm int    `json:"confirm,omitempty"` //0 正常请求 1 继续执行
}

type WebsocketResponse struct {
	Content string `json:"content"`
	Type    int    `json:"type"` //0 正常返回 1 waring
	Info    string `json:"info"`
}

func NewUpGrader(protocol []string) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024 * 1024 * 8 * 2,
		WriteBufferSize: 1024 * 1024 * 8 * 2,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		Subprotocols: protocol,
	}
}

func (wsw *WsWrapper) Write(p []byte) (n int, err error) {
	writer, err := wsw.Conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	defer writer.Close()
	return writer.Write(p)
}

func (wsw *WsWrapper) Read(p []byte) (n int, err error) {
	for {
		msgType, reader, err := wsw.Conn.NextReader()
		if err != nil {
			return 0, err
		}
		if msgType != websocket.TextMessage {
			continue
		}
		return reader.Read(p)
	}
}

package bbtty

const (
	wsMsgCmd    = "cmd"
	wsMsgResize = "resize"

	WsMsgResize = "resize"
	WsMsgCmd    = "cmd"
	WsPing      = "ping"
	WsConfirm   = "confirm"

	WsMsgConnect   = "connect"
	WsMsgReConnect = "reConnect"

	WsError = "error" //error 情况下 直接断开socket

	CmdWarning  = 1
	CmdOutput   = 0
	CmdInput    = 0
	CmdContinue = 1
)

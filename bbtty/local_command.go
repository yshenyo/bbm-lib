package bbtty

import (
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/creack/pty"
	"github.com/pkg/errors"
)

const (
	DefaultCloseSignal  = syscall.SIGINT
	DefaultCloseTimeout = 10 * time.Second
)

type LocalCommand struct {
	command string
	argv    []string

	closeSignal  syscall.Signal
	closeTimeout time.Duration

	cmd       *exec.Cmd
	pty       *os.File
	ptyClosed chan struct{}
	keepAlive int64
}

type AllSlaves struct {
	slaves            map[string]*LocalCommand //map[uniqueKey]*LocalCommand
	lastActiveTime    map[string]int64         //map[uniqueKey]timestamp
	closeCallBackFunc map[string]func()        //map[uniqueKey]callFunc 关闭回调
	mu                sync.RWMutex
}

var allSlaves *AllSlaves

func init() {
	allSlaves = &AllSlaves{
		slaves:            make(map[string]*LocalCommand),
		lastActiveTime:    make(map[string]int64),
		closeCallBackFunc: make(map[string]func()),
	}
}

func (slv *AllSlaves) AddSlave(uniqueKey string, lc *LocalCommand, closeCallBackFunc func()) {
	allSlaves.mu.Lock()
	defer allSlaves.mu.Unlock()
	if _, ok := allSlaves.slaves[uniqueKey]; ok {
		return
	}
	allSlaves.slaves[uniqueKey] = lc
	allSlaves.lastActiveTime[uniqueKey] = time.Now().Unix()
	allSlaves.closeCallBackFunc[uniqueKey] = closeCallBackFunc
	go slv.AutoRecycle(uniqueKey)
}

func (slv *AllSlaves) GetSlave(uniqueKey string) *LocalCommand {
	allSlaves.mu.Lock()
	defer allSlaves.mu.Unlock()
	if slave, ok := allSlaves.slaves[uniqueKey]; ok {
		allSlaves.lastActiveTime[uniqueKey] = time.Now().Unix()
		return slave
	}
	return nil
}

func (slv *AllSlaves) DelSlave(uniqueKey string) {
	allSlaves.mu.Lock()
	defer allSlaves.mu.Unlock()
	if _, ok := allSlaves.slaves[uniqueKey]; !ok {
		return
	}
	delete(allSlaves.slaves, uniqueKey)
	delete(allSlaves.lastActiveTime, uniqueKey)
	delete(allSlaves.closeCallBackFunc, uniqueKey)
}

func (slv *AllSlaves) UpdateLastActiveTime(uniqueKey string) {
	allSlaves.mu.Lock()
	defer allSlaves.mu.Unlock()
	if _, ok := allSlaves.slaves[uniqueKey]; !ok {
		return
	}
	allSlaves.lastActiveTime[uniqueKey] = time.Now().Unix()
}

func (slv *AllSlaves) AutoRecycle(uniqueKey string) {
	for {
		allSlaves.mu.Lock()
		lastActiveTime_Interval := int64(0)
		sleepTime := time.Duration(lastActiveTime_Interval)
		lastActiveTime, ok := allSlaves.lastActiveTime[uniqueKey]
		if !ok {
			allSlaves.mu.Unlock()
			time.Sleep(sleepTime * time.Second)
			return
		}
		slaves, ok := allSlaves.slaves[uniqueKey]
		if ok {
			lastActiveTime_Interval = slaves.keepAlive
			sleepTime = time.Duration(lastActiveTime_Interval)
		}

		if time.Now().Unix()-lastActiveTime >= lastActiveTime_Interval {
			//执行关闭回调
			if f, ok := allSlaves.closeCallBackFunc[uniqueKey]; ok {
				f()
			}
			delete(allSlaves.slaves, uniqueKey)
			delete(allSlaves.lastActiveTime, uniqueKey)
			delete(allSlaves.closeCallBackFunc, uniqueKey)
			allSlaves.mu.Unlock()
			time.Sleep(sleepTime * time.Second)
			continue
		}
		allSlaves.mu.Unlock()
		time.Sleep(sleepTime * time.Second)
	}
}

func NewLocalCommand(command string, argv []string, uniqueKey string, closeCallBackFunc func(), keepAlive int64) (*LocalCommand, error, bool) {
	if slv := allSlaves.GetSlave(uniqueKey); slv != nil {
		return slv, nil, true
	}

	cmd := exec.Command(command, argv...)

	ty, err := pty.Start(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start command `%s`", command), false
	}
	ptyClosed := make(chan struct{})

	lc := &LocalCommand{
		command: command,
		argv:    argv,

		closeSignal:  DefaultCloseSignal,
		closeTimeout: DefaultCloseTimeout,

		cmd:       cmd,
		pty:       ty,
		ptyClosed: ptyClosed,
		keepAlive: keepAlive,
	}

	// When the process is closed by the user,
	// close pty so that Read() on the pty breaks with an EOF.
	go func() {
		defer func() {
			_ = lc.pty.Close()
			close(lc.ptyClosed)
		}()

		_ = lc.cmd.Wait()
	}()

	allSlaves.AddSlave(uniqueKey, lc, closeCallBackFunc)
	return lc, nil, false
}

func NewLocalCommandSingle(command string, argv []string) (*LocalCommand, error) {
	cmd := exec.Command(command, argv...)

	ty, err := pty.Start(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to start command `%s`", command)
	}
	ptyClosed := make(chan struct{})

	lc := &LocalCommand{
		command: command,
		argv:    argv,

		closeSignal:  DefaultCloseSignal,
		closeTimeout: DefaultCloseTimeout,

		cmd:       cmd,
		pty:       ty,
		ptyClosed: ptyClosed,
	}

	// When the process is closed by the user,
	// close pty so that Read() on the pty breaks with an EOF.
	go func() {
		defer func() {
			_ = lc.pty.Close()
			close(lc.ptyClosed)
		}()

		_ = lc.cmd.Wait()
	}()

	return lc, nil
}

func (lc *LocalCommand) Read(p []byte) (n int, err error) {
	return lc.pty.Read(p)
}

func (lc *LocalCommand) Write(p []byte) (n int, err error) {
	return lc.pty.Write(p)
}

func (lc *LocalCommand) Close() error {
	if lc.cmd != nil && lc.cmd.Process != nil {
		_ = lc.pty.Close()
		_ = lc.cmd.Process.Signal(lc.closeSignal)
	}
	for {
		select {
		case <-lc.ptyClosed:
			return nil
		case <-lc.closeTimeoutC():
			_ = lc.cmd.Process.Signal(syscall.SIGKILL)
		}
	}
}

func (lc *LocalCommand) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{
		"command": lc.command,
		"argv":    lc.argv,
		"pid":     lc.cmd.Process.Pid,
	}
}

func (lc *LocalCommand) ResizeTerminal(width int, height int) error {
	window := struct {
		row uint16
		col uint16
		x   uint16
		y   uint16
	}{
		uint16(height),
		uint16(width),
		0,
		0,
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		lc.pty.Fd(),
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&window)),
	)
	if errno != 0 {
		return errno
	} else {
		return nil
	}
}

func (lc *LocalCommand) closeTimeoutC() <-chan time.Time {
	if lc.closeTimeout >= 0 {
		return time.After(lc.closeTimeout)
	}

	return make(chan time.Time)
}

func (lc *LocalCommand) SetCloseSignal(signal syscall.Signal) {
	lc.closeSignal = signal
}

func (lc *LocalCommand) SetCloseTimeout(timeout time.Duration) {
	lc.closeTimeout = timeout
}

package shell

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/creack/pty"

	"bobingtech/inspect/bblog"
)

type Shell struct {
	pty     *os.File
	cmd     *exec.Cmd
	isClose bool
}

type ShellConfig struct {
	Command             string   //连接命令
	Argv                []string //参数
	ShellReceiveMsgChan chan []byte
	ShellSendMsgChan    chan []byte
	Close               chan bool //通知外部是否关闭
}

func NewShell(cfg ShellConfig) (cli *Shell, err error) {
	c := exec.Command(cfg.Command, cfg.Argv...)
	ptmx, err := pty.Start(c)
	if err != nil {
		return nil, err
	}
	cli = &Shell{
		pty:     ptmx,
		cmd:     c,
		isClose: false,
	}

	go cli.wait(cfg.Close)
	go cli.read(cfg.ShellReceiveMsgChan)
	go cli.write(cfg.ShellSendMsgChan)

	return cli, nil
}

func (s *Shell) write(send chan []byte) {
	for {
		if s.isClose {
			return
		}
		buffer := make([]byte, 1024*8)
		n, err := s.pty.Read(buffer)
		if err != nil {
			//pty have already close
			bblog.Logger.ZapLog.Error("shell_write_error", err.Error())
		}
		if n > 0 {
			send <- buffer[:n]
		}
	}
	//}

}

func (s *Shell) read(receive chan []byte) {
	for {
		if s.isClose {
			return
		}
		content := <-receive
		if len(content) > 0 {
			_, err := s.pty.Write(content)
			if err != nil {
				bblog.Logger.ZapLog.Error("shell_read_error", err.Error())
			}
		}
	}
}

func (s *Shell) wait(close chan bool) {
	if err := s.cmd.Wait(); err != nil {
		s.isClose = true
		s.Close()
		close <- true
	}
}

func (s *Shell) WindowChange(rows, cols int) error {
	window := struct {
		row uint16
		col uint16
		x   uint16
		y   uint16
	}{
		uint16(rows),
		uint16(cols),
		0,
		0,
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		s.pty.Fd(),
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&window)),
	)
	if errno != 0 {
		return errno
	} else {
		return nil
	}
}

func (s *Shell) Close() {
	s.isClose = true
	_ = s.cmd.Process.Signal(syscall.SIGKILL)
	_ = s.pty.Close()
}

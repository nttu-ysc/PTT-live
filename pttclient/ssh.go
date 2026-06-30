package pttclient

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"ptt-live/ptterror"
	"regexp"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// SSHConnection implements Connection over SSH to ptt.cc:22.
type SSHConnection struct {
	ctx      context.Context
	client   *ssh.Client
	session  *ssh.Session
	stdin    io.WriteCloser
	stdout   *customOut
	reconnCh chan struct{}
}

func NewSSHConnection() *SSHConnection {
	return &SSHConnection{
		reconnCh: make(chan struct{}, 1),
	}
}

func (s *SSHConnection) Connect(ctx context.Context) {
	s.ctx = ctx

	config := &ssh.ClientConfig{
		User:            "bbsu",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	var err error
	s.client, err = ssh.Dial("tcp", "ptt.cc:22", config)
	if err != nil {
		log.Fatalf("unable to connect: %s", err)
	}

	s.session, err = s.client.NewSession()
	if err != nil {
		log.Fatalf("unable to create session: %s", err)
	}

	s.stdin, _ = s.session.StdinPipe()

	reader := make(chan []byte, 10)
	out := new(customOut)
	out.reader = reader
	s.stdout = out

	s.session.Stdout = out
	s.session.Stderr = os.Stderr

	terminalModes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	fileDescriptor := int(os.Stdin.Fd())
	if term.IsTerminal(fileDescriptor) {
		originalState, err := term.MakeRaw(fileDescriptor)
		if err != nil {
			panic(err)
		}
		defer term.Restore(fileDescriptor, originalState)

		termWidth, termHeight, err := term.GetSize(fileDescriptor)
		if err != nil {
			panic(err)
		}

		if err = s.session.RequestPty("xterm-256color", termHeight, termWidth, terminalModes); err != nil {
			panic(err)
		}
	}

	if err = s.session.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}

	go func() {
		s.session.Wait()
		to := time.NewTicker(time.Second)
		select {
		case <-to.C:
			runtime.MessageDialog(s.ctx, runtime.MessageDialogOptions{
				Type:    runtime.ErrorDialog,
				Title:   "PTT live",
				Message: "偵測到重複登入，即將關閉程式",
			})
			os.Exit(1)
		case <-s.reconnCh:
			// Reconnect() is managing a new session; exit this goroutine.
			return
		}
	}()
}

func (s *SSHConnection) Reconnect() {
	s.reconnCh <- struct{}{}
	s.Connect(s.ctx)
}

func (s *SSHConnection) Write(p []byte) (int, error) {
	errCh := make(chan error, 1)
	nCh := make(chan int, 1)

	go func() {
		n, err := s.stdin.Write(p)
		errCh <- err
		nCh <- n
	}()

	select {
	case err := <-errCh:
		return <-nCh, err
	case <-time.After(5 * time.Second):
		log.Println("SSH Write Timeout! TCP Connection is likely dead. Triggering Reconnect.")
		go s.Reconnect()
		return 0, ptterror.Timeout
	}
}

func (s *SSHConnection) Read(t time.Duration) ([]byte, error) {
	return s.stdout.Read(t)
}

func (s *SSHConnection) Close() {
	if s.session != nil {
		s.session.Close()
	}
	if s.client != nil {
		s.client.Close()
	}
}

// customOut buffers SSH stdout for polling reads, cleaning ANSI codes on each write.
type customOut struct {
	reader chan []byte
}

func (w *customOut) Read(t time.Duration) ([]byte, error) {
	select {
	case p := <-w.reader:
		return p, nil
	case <-time.After(t):
		return nil, os.ErrDeadlineExceeded
	}
}

func (w *customOut) Write(p []byte) (n int, err error) {
	newP := cleanData(p)

	// Non-blocking send with ring-buffer behavior: drop oldest if full.
	select {
	case w.reader <- newP:
	default:
		select {
		case <-w.reader:
		default:
		}
		w.reader <- newP
	}

	return os.Stdout.Write(p)
}

var (
	ctrlCharReg    = regexp.MustCompile(`[\x00-\x07\x0B\x0C\x0E-\x1A\x1C-\x1F\x7F]+`)
	ansiEscReg     = regexp.MustCompile(`\x1B`)
	ansiColorReg   = regexp.MustCompile(`\[[\d+;]*m`)
	ansiPosLineReg = regexp.MustCompile(`\[\d+;[0-4]H`)
	ansiPosLeftReg = regexp.MustCompile(`\[[\d;]*[HrJK]`)
)

func cleanData(data []byte) []byte {
	data = ctrlCharReg.ReplaceAll(data, nil)
	data = ansiEscReg.ReplaceAll(data, nil)
	data = ansiColorReg.ReplaceAll(data, nil)
	data = ansiPosLineReg.ReplaceAll(data, []byte("\n"))
	data = ansiPosLeftReg.ReplaceAll(data, nil)
	data = bytes.ReplaceAll(data, []byte{'\r'}, nil)
	data = bytes.ReplaceAll(data, []byte{' ', '\x08'}, nil)
	return data
}

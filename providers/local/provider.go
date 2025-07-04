package local

import (
	"io"
	"os"
	"os/exec"
	"syscall"

	"go.alt-gnome.ru/capytest"
)

type localProvider struct{}

func Provider() *localProvider {
	return &localProvider{}
}

type session struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	stdoutC chan string
	stderrC chan string
	done    chan error
}

func (p *localProvider) StartCommand(cmd []string) (capytest.InteractiveSession, error) {
	c := exec.Command(cmd[0], cmd[1:]...)
	stdin, err := c.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, err
	}

	sess := &session{
		cmd:     c,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		stdoutC: make(chan string),
		stderrC: make(chan string),
		done:    make(chan error, 1),
	}

	if err := c.Start(); err != nil {
		return nil, err
	}

	go sess.readPipe(sess.stdout, sess.stdoutC)
	go sess.readPipe(sess.stderr, sess.stderrC)
	go func() {
		sess.done <- c.Wait()
		close(sess.stdoutC)
		close(sess.stderrC)
	}()

	return sess, nil
}

func (s *session) readPipe(r io.Reader, ch chan string) {
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			ch <- string(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func (s *session) Write(input string) error {
	_, err := io.WriteString(s.stdin, input)
	if f, ok := s.stdin.(*os.File); ok {
		f.Sync()
	}
	return err
}

func (s *session) Stdout() <-chan string {
	return s.stdoutC
}

func (s *session) Stderr() <-chan string {
	return s.stderrC
}

func (s *session) Wait() (int, error) {
	err := <-s.done
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), nil
	}
	return -1, err
}

func (s *session) Interrupt() error {
	if s.cmd.Process == nil {
		return os.ErrInvalid
	}
	return s.cmd.Process.Signal(syscall.SIGINT)
}

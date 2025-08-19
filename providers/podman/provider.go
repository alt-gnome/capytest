package podman

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
	"go.alt-gnome.ru/capytest"
)

var DefaultPodmanCli string = "podman"
var DefaultImage string = "ubuntu:latest"

type PodmanOption func(*podmanProvider)

func WithImage(image string) PodmanOption {
	return func(p *podmanProvider) {
		p.image = image
	}
}

func WithWorkdir(workdir string) PodmanOption {
	return func(p *podmanProvider) {
		p.workdir = workdir
	}
}

func WithVolumes(volumes ...string) PodmanOption {
	return func(p *podmanProvider) {
		p.volumes = append(p.volumes, volumes...)
	}
}

func WithEnvVars(envVars ...string) PodmanOption {
	return func(p *podmanProvider) {
		p.envVars = append(p.envVars, envVars...)
	}
}

func WithNetwork(network string) PodmanOption {
	return func(p *podmanProvider) {
		p.network = network
	}
}

func WithPrivileged(privileged bool) PodmanOption {
	return func(p *podmanProvider) {
		p.privileged = privileged
	}
}

type podmanProvider struct {
	image       string
	workdir     string
	volumes     []string
	envVars     []string
	network     string
	privileged  bool
	containerID string
	prepared    bool
}

func Provider(opts ...PodmanOption) *podmanProvider {
	p := &podmanProvider{
		image: DefaultImage,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Подготовка: создаем и запускаем контейнер
func (p *podmanProvider) Prepare() error {
	if p.prepared {
		return nil
	}

	// Проверяем, есть ли образ
	exists, err := p.ImageExists()
	if err != nil {
		return err
	}
	if !exists {
		if err := p.PullImage(); err != nil {
			return err
		}
	}

	// Создаем контейнер
	containerID, err := p.createContainer()
	if err != nil {
		return err
	}
	p.containerID = containerID

	// Запускаем контейнер
	if err := p.startContainer(); err != nil {
		return err
	}

	p.prepared = true
	return nil
}

func (p *podmanProvider) Cleanup() error {
	if !p.prepared || p.containerID == "" {
		return nil
	}

	stopCmd := exec.Command(DefaultPodmanCli, "stop", p.containerID)
	stopCmd.Run()

	rmCmd := exec.Command(DefaultPodmanCli, "rm", p.containerID)
	if err := rmCmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", p.containerID, err)
	}

	p.containerID = ""
	p.prepared = false
	return nil
}

func (p *podmanProvider) PullImage() error {
	cmd := exec.Command(DefaultPodmanCli, "pull", p.image)
	return cmd.Run()
}

func (p *podmanProvider) ImageExists() (bool, error) {
	cmd := exec.Command(DefaultPodmanCli, "image", "exists", p.image)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *podmanProvider) createContainer() (string, error) {
	createCmd := []string{DefaultPodmanCli, "create", "--init"}

	// Добавляем опции
	if p.workdir != "" {
		createCmd = append(createCmd, "--workdir", p.workdir)
	}

	for _, volume := range p.volumes {
		createCmd = append(createCmd, "-v", volume)
	}

	for _, env := range p.envVars {
		createCmd = append(createCmd, "-e", env)
	}

	if p.network != "" {
		createCmd = append(createCmd, "--network", p.network)
	}

	if p.privileged {
		createCmd = append(createCmd, "--privileged")
	}

	// Добавляем образ и команду для поддержания контейнера живым
	createCmd = append(createCmd, p.image, "sleep", "infinity")

	cmd := exec.Command(createCmd[0], createCmd[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	containerID := string(output)
	// Удаляем trailing newline
	if len(containerID) > 0 && containerID[len(containerID)-1] == '\n' {
		containerID = containerID[:len(containerID)-1]
	}

	return containerID, nil
}

func (p *podmanProvider) startContainer() error {
	cmd := exec.Command(DefaultPodmanCli, "start", p.containerID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start container %s: %w", p.containerID, err)
	}

	// Ждем, пока контейнер полностью запустится
	for i := 0; i < 10; i++ {
		if running, err := p.isContainerRunning(); err != nil {
			return err
		} else if running {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("container %s failed to start within timeout", p.containerID)
}

func (p *podmanProvider) isContainerRunning() (bool, error) {
	cmd := exec.Command(DefaultPodmanCli, "container", "inspect", p.containerID, "--format", "{{.State.Running}}")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return string(output) == "true\n", nil
}

func (p *podmanProvider) StartCommand(cmd []string) (capytest.NotInteractiveSession, error) {
	if !p.prepared {
		if err := p.Prepare(); err != nil {
			return nil, fmt.Errorf("failed to prepare container: %w", err)
		}
	}

	execCmd := []string{DefaultPodmanCli, "exec", "-i", p.containerID}
	execCmd = append(execCmd, cmd...)

	c := exec.Command(execCmd[0], execCmd[1:]...)
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

	sess := &notInteractiveSession{
		cmd:         c,
		containerID: p.containerID,
		stdin:       stdin,
		stdout:      stdout,
		stderr:      stderr,
		stdoutC:     make(chan string),
		stderrC:     make(chan string),
		done:        make(chan error, 1),
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

func (p *podmanProvider) StartInteractiveCommand(cmd []string) (capytest.InteractiveSession, error) {
	if !p.prepared {
		if err := p.Prepare(); err != nil {
			return nil, fmt.Errorf("failed to prepare container: %w", err)
		}
	}

	execCmd := []string{DefaultPodmanCli, "exec", "-it", p.containerID}
	execCmd = append(execCmd, cmd...)

	c := exec.Command(execCmd[0], execCmd[1:]...)

	ptmx, err := pty.Start(c)
	if err != nil {
		return nil, fmt.Errorf("failed to start interactive command: %w", err)
	}

	sess := &interactiveSession{
		cmd:    c,
		pty:    ptmx,
		output: make(chan string),
		done:   make(chan error, 1),
	}

	go func() {
		defer close(sess.output)
		buf := make([]byte, 1024)
		for {
			n, err := sess.pty.Read(buf)
			if n > 0 {
				sess.output <- string(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		sess.done <- c.Wait()
	}()

	return sess, nil
}

type notInteractiveSession struct {
	cmd         *exec.Cmd
	containerID string
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	stderr      io.ReadCloser

	stdoutC chan string
	stderrC chan string
	done    chan error
}

func (s *notInteractiveSession) Write(input string) error {
	_, err := io.WriteString(s.stdin, input)
	if f, ok := s.stdin.(*os.File); ok {
		f.Sync()
	}
	return err
}

func (s *notInteractiveSession) Stdout() <-chan string {
	return s.stdoutC
}

func (s *notInteractiveSession) Stderr() <-chan string {
	return s.stderrC
}

func (s *notInteractiveSession) Wait() (int, error) {
	err := <-s.done
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode := exitErr.ExitCode()

		// podman exec usually returns the actual program codes,
		// but we still check the podman-specific codes
		switch exitCode {
		case 125:
			return -1, fmt.Errorf("podman exec internal error: %w", err)
		case 126:
			return -1, fmt.Errorf("cannot invoke command in container: %w", err)
		case 127:
			return -1, fmt.Errorf("command not found in container: %w", err)
		default:
			// For all other codes, we return it as it is.
			return exitCode, nil
		}
	}
	return -1, err
}

func (s *notInteractiveSession) Interrupt() error {
	if s.cmd.Process == nil {
		return os.ErrInvalid
	}

	return s.cmd.Process.Signal(syscall.SIGINT)
}

func (s *notInteractiveSession) readPipe(r io.Reader, ch chan string) {
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

type interactiveSession struct {
	cmd    *exec.Cmd
	pty    *os.File
	output chan string
	done   chan error
}

func (s *interactiveSession) Write(input []byte) error {
	_, err := s.pty.Write(input)
	return err
}

func (s *interactiveSession) Output() <-chan string {
	return s.output
}

func (s *interactiveSession) Wait() (int, error) {
	err := <-s.done
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		switch code {
		case 125:
			return -1, fmt.Errorf("podman exec internal error: %w", err)
		case 126:
			return -1, fmt.Errorf("cannot invoke command in container: %w", err)
		case 127:
			return -1, fmt.Errorf("command not found in container: %w", err)
		default:
			return code, nil
		}
	}
	return -1, err
}

func (s *interactiveSession) Interrupt() error {
	if s.cmd.Process == nil {
		return os.ErrInvalid
	}
	return s.cmd.Process.Signal(syscall.SIGINT)
}

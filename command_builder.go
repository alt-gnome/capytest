package capytest

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

type CommandBuilder interface {
	Executable

	WithTimeout(duration time.Duration) CommandBuilder

	ExpectExitCode(code int) CommandBuilder
	ExpectSuccess() CommandBuilder // shorthand ExpectExitCode(0)
	ExpectFailure() CommandBuilder // shorthand ExpectExitCode != 0

	ExpectStdoutContains(substr string) CommandBuilder
	ExpectStderrContains(substr string) CommandBuilder
	ExpectStdoutRegex(pattern string) CommandBuilder
	ExpectStderrRegex(pattern string) CommandBuilder
	ExpectStdoutEmpty() CommandBuilder
	ExpectStderrEmpty() CommandBuilder

	Do() StepBuilder
}

type commandBuilder struct {
	provider Provider
	cmd      []string
	timeout  time.Duration

	expectedExitCode   *int
	expectFailure      bool
	stdoutExpectations []string
	stderrExpectations []string
	stdoutRegexes      []string
	stderrRegexes      []string
	expectStdoutEmpty  bool
	expectStderrEmpty  bool

	steps []step
}

func (c *commandBuilder) WithTimeout(duration time.Duration) CommandBuilder {
	c.timeout = duration
	return c
}

func (c *commandBuilder) ExpectExitCode(code int) CommandBuilder {
	c.expectedExitCode = &code
	c.expectFailure = false
	return c
}

func (c *commandBuilder) ExpectSuccess() CommandBuilder {
	code := 0
	c.expectedExitCode = &code
	c.expectFailure = false
	return c
}

func (c *commandBuilder) ExpectFailure() CommandBuilder {
	c.expectedExitCode = nil
	c.expectFailure = true
	return c
}

func (c *commandBuilder) ExpectStdoutContains(substr string) CommandBuilder {
	c.stdoutExpectations = append(c.stdoutExpectations, substr)
	return c
}

func (c *commandBuilder) ExpectStderrContains(substr string) CommandBuilder {
	c.stderrExpectations = append(c.stderrExpectations, substr)
	return c
}

func (c *commandBuilder) ExpectStdoutRegex(pattern string) CommandBuilder {
	c.stdoutRegexes = append(c.stdoutRegexes, pattern)
	return c
}

func (c *commandBuilder) ExpectStderrRegex(pattern string) CommandBuilder {
	c.stderrRegexes = append(c.stderrRegexes, pattern)
	return c
}

func (c *commandBuilder) ExpectStdoutEmpty() CommandBuilder {
	c.expectStdoutEmpty = true
	return c
}

func (c *commandBuilder) ExpectStderrEmpty() CommandBuilder {
	c.expectStderrEmpty = true
	return c
}

func (c *commandBuilder) Do() StepBuilder {
	return &stepBuilder{
		parent:      c,
		currentStep: &step{},
	}
}

func (c *commandBuilder) Run(t *testing.T) {
	t.Helper()

	session, err := c.provider.StartCommand(c.cmd)
	if err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	var stdoutBuf, stderrBuf strings.Builder

	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})

	go func() {
		defer close(stdoutDone)
		for out := range session.Stdout() {
			stdoutBuf.WriteString(out)
		}
	}()

	go func() {
		defer close(stderrDone)
		for errOut := range session.Stderr() {
			stderrBuf.WriteString(errOut)
		}
	}()

	for _, step := range c.steps {
		if err := c.executeStep(session, step, &stdoutBuf, &stderrBuf, t); err != nil {
			t.Fatalf("failed to execute step: %v", err)
		}
	}

	exitCode, err := session.Wait()
	if err != nil {
		t.Fatalf("error waiting for process: %v", err)
	}

	<-stdoutDone
	<-stderrDone

	c.validateResults(exitCode, stdoutBuf.String(), stderrBuf.String(), t)
}

func (c *commandBuilder) executeStep(session InteractiveSession, step step, stdoutBuf, stderrBuf *strings.Builder, t *testing.T) error {
	switch step.action {
	case sendAction:
		if err := session.Write(step.data); err != nil {
			return fmt.Errorf("failed to write to stdin: %v", err)
		}
	case waitAction:
		time.Sleep(step.duration)
	case interruptAction:
		if err := session.Interrupt(); err != nil {
			return fmt.Errorf("failed to interrupt process: %v", err)
		}
	}

	c.validateStepExpectations(step.expectation, stdoutBuf, stderrBuf, t)

	return nil
}

func (c *commandBuilder) validateStepExpectations(exp expectation, stdoutBuf, stderrBuf *strings.Builder, t *testing.T) {
	if exp.stdout != "" {
		if !waitForSubstring(stdoutBuf, exp.stdout, 5) {
			t.Errorf("stdout does not contain %q\nstdout: %q", exp.stdout, stdoutBuf.String())
		}
	}

	if exp.stderr != "" {
		if !waitForSubstring(stderrBuf, exp.stderr, 5) {
			t.Errorf("stderr does not contain %q\nstderr: %q", exp.stderr, stderrBuf.String())
		}
	}

	if exp.stdoutRegex != "" {
		if matched, _ := regexp.MatchString(exp.stdoutRegex, stdoutBuf.String()); !matched {
			t.Errorf("stdout does not match regex %q\nstdout: %q", exp.stdoutRegex, stdoutBuf.String())
		}
	}

	if exp.stderrRegex != "" {
		if matched, _ := regexp.MatchString(exp.stderrRegex, stderrBuf.String()); !matched {
			t.Errorf("stderr does not match regex %q\nstderr: %q", exp.stderrRegex, stderrBuf.String())
		}
	}
}

func (c *commandBuilder) validateResults(exitCode int, stdout, stderr string, t *testing.T) {
	t.Helper()
	// Check exit code
	if c.expectedExitCode != nil {
		if exitCode != *c.expectedExitCode {
			t.Errorf("unexpected exit code: got %d, want %d", exitCode, *c.expectedExitCode)
		}
	} else if c.expectFailure {
		if exitCode == 0 {
			t.Errorf("expected failure but got success (exit code 0)")
		}
	}

	// Check stdout
	for _, expected := range c.stdoutExpectations {
		if !strings.Contains(stdout, expected) {
			t.Errorf("stdout does not contain %q\nstdout: %q", expected, stdout)
		}
	}

	// Check stderr
	for _, expected := range c.stderrExpectations {
		if !strings.Contains(stderr, expected) {
			t.Errorf("stderr does not contain %q\nstderr: %q", expected, stderr)
		}
	}

	// Check regex for stdout
	for _, pattern := range c.stdoutRegexes {
		if matched, _ := regexp.MatchString(pattern, stdout); !matched {
			t.Errorf("stdout does not match regex %q\nstdout: %q", pattern, stdout)
		}
	}

	// Check regex for stderr
	for _, pattern := range c.stderrRegexes {
		if matched, _ := regexp.MatchString(pattern, stderr); !matched {
			t.Errorf("stderr does not match regex %q\nstderr: %q", pattern, stderr)
		}
	}

	// Check empty stdout
	if c.expectStdoutEmpty && stdout != "" {
		t.Errorf("expected stdout to be empty but got: %q", stdout)
	}

	// Check empty stderr
	if c.expectStderrEmpty && stderr != "" {
		t.Errorf("expected stderr to be empty but got: %q", stderr)
	}
}

func waitForSubstring(buf *strings.Builder, substr string, timeoutSeconds int) bool {
	timeout := time.After(time.Duration(timeoutSeconds) * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return false
		case <-ticker.C:
			if strings.Contains(buf.String(), substr) {
				return true
			}
		}
	}
}

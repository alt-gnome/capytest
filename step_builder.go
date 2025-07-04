package capytest

import "time"

type StepBuilder interface {
	Send(input string) StepBuilder
	SendLine(line string) StepBuilder
	Wait(duration time.Duration) StepBuilder
	Interrupt() StepBuilder
	Terminate() StepBuilder

	ExpectStdoutContains(substr string) StepBuilder
	ExpectStderrContains(substr string) StepBuilder
	ExpectStdoutRegex(pattern string) StepBuilder
	ExpectStderrRegex(pattern string) StepBuilder

	Then() StepBuilder
	Done() CommandBuilder
}

type stepAction int

const (
	sendAction stepAction = iota
	sendLineAction
	waitAction
	interruptAction
	terminateAction
)

type expectation struct {
	stdout      string
	stderr      string
	stdoutRegex string
	stderrRegex string
}

type step struct {
	action      stepAction
	data        string
	duration    time.Duration
	expectation expectation
}

type stepBuilder struct {
	parent      *commandBuilder
	currentStep *step
}

func (s *stepBuilder) Send(input string) StepBuilder {
	s.currentStep.action = sendAction
	s.currentStep.data = input
	return s
}

func (s *stepBuilder) SendLine(line string) StepBuilder {
	s.currentStep.action = sendAction
	s.currentStep.data = line + "\n"
	return s
}

func (s *stepBuilder) Wait(duration time.Duration) StepBuilder {
	s.currentStep.action = waitAction
	s.currentStep.duration = duration
	return s
}

func (s *stepBuilder) Interrupt() StepBuilder {
	s.currentStep.action = interruptAction
	return s
}

func (s *stepBuilder) Terminate() StepBuilder {
	s.currentStep.action = terminateAction
	return s
}

func (s *stepBuilder) ExpectStdoutContains(substr string) StepBuilder {
	s.currentStep.expectation.stdout = substr
	return s
}

func (s *stepBuilder) ExpectStderrContains(substr string) StepBuilder {
	s.currentStep.expectation.stderr = substr
	return s
}

func (s *stepBuilder) ExpectStdoutRegex(pattern string) StepBuilder {
	s.currentStep.expectation.stdoutRegex = pattern
	return s
}

func (s *stepBuilder) ExpectStderrRegex(pattern string) StepBuilder {
	s.currentStep.expectation.stderrRegex = pattern
	return s
}

func (s *stepBuilder) Then() StepBuilder {
	s.parent.steps = append(s.parent.steps, *s.currentStep)

	return &stepBuilder{
		parent:      s.parent,
		currentStep: &step{},
	}
}

func (s *stepBuilder) Done() CommandBuilder {
	s.parent.steps = append(s.parent.steps, *s.currentStep)
	return s.parent
}

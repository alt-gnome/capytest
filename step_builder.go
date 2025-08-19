package capytest

import "time"

type StepBuilder interface {
	Send(input []byte) StepBuilder
	SendString(input string) StepBuilder
	SendLine(line string) StepBuilder
	Wait(duration time.Duration) StepBuilder
	Interrupt() StepBuilder
	Terminate() StepBuilder

	ExpectOutputContains(substr string) StepBuilder
	ExpectOutputRegex(pattern string) StepBuilder

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
	outputContains string
	outputRegex    string
}

type step struct {
	action      stepAction
	data        []byte
	duration    time.Duration
	expectation expectation
}

type stepBuilder struct {
	parent      *commandBuilder
	currentStep *step
}

func (s *stepBuilder) Send(input []byte) StepBuilder {
	s.currentStep.action = sendAction
	s.currentStep.data = input
	return s
}

func (s *stepBuilder) SendString(input string) StepBuilder {
	s.currentStep.action = sendAction
	s.currentStep.data = []byte(input)
	return s
}

func (s *stepBuilder) SendLine(line string) StepBuilder {
	s.currentStep.action = sendAction
	s.currentStep.data = []byte(line + "\n")
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

func (s *stepBuilder) ExpectOutputContains(substr string) StepBuilder {
	s.currentStep.expectation.outputContains = substr
	return s
}

func (s *stepBuilder) ExpectOutputRegex(pattern string) StepBuilder {
	s.currentStep.expectation.outputRegex = pattern
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

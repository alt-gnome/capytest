package capytest

import "testing"

type Runner interface {
	Command(name string, args ...string) CommandBuilder
}

type Executable interface {
	Run(t *testing.T)
}

type runner struct {
	p Provider
}

func (r *runner) Command(name string, args ...string) CommandBuilder {
	return &commandBuilder{provider: r.p, cmd: append([]string{name}, args...)}
}

func NewRunner(p Provider) Runner {
	return &runner{p}
}

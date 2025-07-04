package capytest

type Provider interface {
	StartCommand(cmd []string) (InteractiveSession, error)
}

type PreparableProvider interface {
	Provider
	Prepare() error
	Cleanup() error
}

type InteractiveSession interface {
	Write(input string) error

	Stdout() <-chan string
	Stderr() <-chan string

	Wait() (exitCode int, err error)
	Interrupt() error
}

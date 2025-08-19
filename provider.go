package capytest

type Provider interface {
	StartCommand(cmd []string) (NotInteractiveSession, error)
	StartInteractiveCommand(cmd []string) (InteractiveSession, error)
}

type PreparableProvider interface {
	Provider
	Prepare() error
	Cleanup() error
}

type Session interface {
	Wait() (exitCode int, err error)
	Interrupt() error
}

type NotInteractiveSession interface {
	Session

	Write(input string) error

	Stdout() <-chan string
	Stderr() <-chan string
}

type InteractiveSession interface {
	Session
	Write([]byte) error
	Output() <-chan string
}

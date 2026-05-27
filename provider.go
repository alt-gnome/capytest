package capytest

// CommandOptions carries per-command execution options passed from the
// CommandBuilder to a Provider. The struct is intentionally extensible so
// new options can be added without breaking the Provider interface.
type CommandOptions struct {
	// Env is a list of "KEY=VALUE" entries to set for the command. Later
	// entries override earlier entries with the same key.
	Env []string
}

type Provider interface {
	StartCommand(cmd []string, opts CommandOptions) (NotInteractiveSession, error)
	StartInteractiveCommand(cmd []string, opts CommandOptions) (InteractiveSession, error)
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

# capytest

<p align="center">
  <img src="https://altlinux.space/alt-gnome/capytest/media/branch/main/image.png" alt="capytest logo: a cute cartoon capybara sitting upright holding a terminal icon with '>_' and the text 'capytest' below in a modern rounded font" width="250"/>
</p>

<p align="center">
  <b>capytest</b> is a simple, friendly library for E2E testing CLI applications in Go.<br/>
  It lets you write clear, step-by-step scenarios for both interactive and non-interactive command-line programs.
</p>

<p align="center">
  <a href="https://pkg.go.dev/go.alt-gnome.ru/capytest"><img src="https://pkg.go.dev/badge/go.alt-gnome.ru/capytest.svg" alt="Go Reference"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
</p>

## Features

- Simple step-by-step scenarios
- Supports interactive and non-interactive CLIs
- Simulate interrupts and signals
- Check stdout, stderr, exit codes
- Pluggable providers (local, Podman or your own)

## Installation

```bash
# Install the core
go get -u go.alt-gnome.ru/capytest

# Install the necessary provider(s)
go get -u go.alt-gnome.ru/capytest/providers/local
```

## Example

```go
package main

import (
	"testing"

	"go.alt-gnome.ru/capytest"
	"go.alt-gnome.ru/capytest/providers/local"
)

func TestExample(t *testing.T) {
	ts := capytest.NewTestSuite(t, local.Provider())

	// Interactive scenario
	ts.Run("bc is works", func(t *testing.T, r capytest.Runner) {
		r.Command("bc").
			Do().SendLine("2+2").ExpectOutputContains("4").
			Then().SendLine("2*3").ExpectOutputContains("6").
			Then().Send([]byte{4}). // Ctrl-D
			Done().ExpectExitCode(0).
			Run(t)
	})

	// Non-interactive scenario
	ts.Run("bash --version contains GNU", func(t *testing.T, r capytest.Runner) {
		r.Command("bash", "--version").
			ExpectStdoutContains("GNU").
			ExpectStdoutMatchesSnapshot().
			ExpectStderrEmpty().
			Run(t)
	})
}
```

## License

[MIT License © 2025 Maxim Slipenko](./LICENSE)

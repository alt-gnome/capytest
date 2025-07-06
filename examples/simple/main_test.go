package simple_test

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
			Do().SendLine("2+2").ExpectStdoutContains("4").
			Then().SendLine("2*3").ExpectStdoutContains("6").
			Then().Interrupt().
			Done().ExpectExitCode(-1).
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

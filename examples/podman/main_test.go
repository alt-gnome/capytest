package podman_test

import (
	"testing"

	"go.alt-gnome.ru/capytest"
	"go.alt-gnome.ru/capytest/providers/podman"
)

func TestPodman(t *testing.T) {
	ts := capytest.NewTestSuite(t, podman.Provider(
		podman.WithImage("registry.altlinux.org/sisyphus/alt"),
	))

	ts.Run("bash --version", func(t *testing.T, r capytest.Runner) {
		r.Command("bash", "--version").
			ExpectStdoutRegex("GNU").
			Run(t)
	})

	ts.Run("bc is works", func(t *testing.T, r capytest.Runner) {
		r.Command("sh", "-c", "apt-get update && apt-get install -y bc").
			ExpectSuccess().
			Run(t)

		r.Command("bc").
			Do().SendLine("2+2").ExpectOutputContains("4").
			Then().SendLine("2*3").ExpectOutputContains("6").
			Then().Send([]byte{4}). // Ctrl-D
			Done().ExpectExitCode(0).
			Run(t)
	})
}

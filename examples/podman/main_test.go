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
}

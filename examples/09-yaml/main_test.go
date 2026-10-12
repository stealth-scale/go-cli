// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"testing"

	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli/clitest"
)

// files are the config files of the example.
var files = os.DirFS(".")

// runs are the command lines of the transcript of the example.
var runs = []clitest.Input{
	{Args: []string{"send"}, Files: files},
	{Args: []string{"-c", "notify.yml", "send"}, Files: files},
	{Args: []string{"send", "--flag-debug"}, Files: files},
	{Args: []string{"-c", "broken.yaml", "send"}, Files: files},
	{Args: []string{"send"}, Files: files, FailStdout: true},
}

// TestYAML checks the transcript of the runs of the example.
func TestYAML(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "notify", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

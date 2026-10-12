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
	{Args: []string{"backup"}, Files: files},
	{Vars: map[string]string{"SCHEDULE_BACKUP_EVERY": "6h"}, Args: []string{"backup", "--flag-debug"}, Files: files},
	{Args: []string{"-c", "broken.toml", "backup"}, Files: files},
	{Args: []string{"backup"}, Files: files, FailStdout: true},
}

// TestTOML checks the transcript of the runs of the example.
func TestTOML(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "schedule", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

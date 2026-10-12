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
	{Args: []string{"serve", "--help"}, Files: files},
	{Args: []string{"serve"}, Files: files},
	{Args: []string{"migrate"}, Files: files},
	{Args: []string{"-c", "relay.ini", "serve"}, Files: files},
	{Args: []string{"-c", "relay.conf", "serve"}, Files: files},
	{Vars: map[string]string{"RELAY_CONFIG": "relay.ini"}, Args: []string{"migrate", "--steps", "1"}, Files: files},
	{Args: []string{"-c", "typo.json", "serve"}, Files: files},
	{Args: []string{"serve"}, Files: files, FailStdout: true},
	{Args: []string{"migrate"}, Files: files, FailStdout: true},
}

// TestConfig checks the transcript of the runs of the example.
func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "relay", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

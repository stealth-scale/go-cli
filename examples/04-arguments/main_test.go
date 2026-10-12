// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli/clitest"
)

// files are the input files of the runs of add.
var files = clitest.Files{
	"app.log":  "GET /\nGET /health\n",
	"db.log":   "SELECT 1\n",
	"-odd.log": "odd\n",
	"--level":  "level\n",
}

// runs are the command lines of the transcript of the example.
var runs = []clitest.Input{
	{Args: []string{"create", "--help"}},
	{Args: []string{"add", "--help"}},
	{Args: []string{"create", "logs"}},
	{Args: []string{"create", "logs", "rotated daily"}},
	{Args: []string{"create"}},
	{Args: []string{"create", "logs", "daily", "extra"}},
	{Args: []string{"add", "logs", "app.log", "-l", "9", "db.log"}, Files: files},
	{Args: []string{"add", "logs"}},
	{Args: []string{"add", "logs", "--", "-odd.log", "--level"}, Files: files},
	{Stdin: "GET /\n", Args: []string{"add", "logs", "-"}},
	{Args: []string{"add", "logs", "missing.log"}, Files: files},
	{Args: []string{"list"}},
	{Args: []string{"list", "*.log", "app-*"}},
	{Args: []string{"create", "logs"}, FailStdout: true},
}

// TestArguments checks the transcript of the runs of the example.
func TestArguments(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "pack", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

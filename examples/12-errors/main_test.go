// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"testing"

	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli/clitest"
)

// runs are the command lines of the transcript of the example.
var runs = []clitest.Input{
	{Args: []string{"run", "--strict", "src/main.go", "src/util.go"}},
	{Args: []string{"run", "--strict", "legacy/old.go"}},
	{Args: []string{"rn", "src"}},
	{Args: []string{"run", "--strikt", "src"}},
	{Args: []string{"run"}},
	{Args: []string{"run", "--max-issues", "many", "src"}},
	{Vars: map[string]string{"LINT_RUN_FORMAT": "xml"}, Args: []string{"run", "src"}},
	{Args: []string{"-c", "lint.json", "run", "src"}, Files: os.DirFS(".")},
	{Args: []string{"-c", "missing.json", "run", "src"}},
	{Args: []string{"run", "src"}, FailStdout: true},
}

// TestErrors checks the transcript of the runs of the example.
func TestErrors(t *testing.T) {
	t.Parallel()

	t.Run("run", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "lint", run, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

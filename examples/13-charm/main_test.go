// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli/clitest"
)

// runs are the command lines of the transcript of the example.
var runs = []clitest.Input{
	{Args: []string{"--help"}},
	{Args: []string{"serve", "--help"}},
	{Vars: map[string]string{"COLUMNS": "60"}, Args: []string{"serve", "--help"}},
	{Args: []string{"serve", "--adr", ":9000"}},
	{Args: []string{"serve"}},
	{Vars: map[string]string{"SRV_SERVE_TOKEN": "s3cret"}, Args: []string{"serve", "store-2", "--flag-debug"}},
	{Vars: map[string]string{"CLICOLOR_FORCE": "1", "SRV_SERVE_TOKEN": "s3cret"}, Args: []string{"serve"}},
	{Args: []string{"migrate", "-n"}},
	{Args: []string{"migrate"}, FailStdout: true},
}

// TestCharm checks the transcript of the runs of the example.
func TestCharm(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "srv", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli/examples/internal/transcript"
)

// runs are the command lines of the transcript of the example.
var runs = []transcript.Run{
	{Args: []string{"--help"}},
	{Args: []string{"plan", "--env", "prod"}},
	{Args: []string{"-v", "apply", "-e", "prod", "-y"}},
	{Args: []string{"apply", "-e", "prod", "-y", "--verbose"}},
	{Args: []string{"apply", "-e", "prod"}},
	{Args: []string{"rollback", "-e", "dev", "--region", "us-east-1", "41"}},
	{Args: []string{"plan"}},
	{Args: []string{"plan", "--env", "staging"}},
	{Args: []string{"aply", "-e", "dev"}},
	{Args: []string{"rollback", "--help"}},
	{Args: []string{"plan", "-e", "dev"}, Full: true},
}

// TestCommands checks the transcript of the runs of the example.
func TestCommands(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := transcript.Write(t.Context(), "deploy", runs, app().Main)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

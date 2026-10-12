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
	{Args: []string{"create", "--help"}},
	{Args: []string{"add", "--help"}},
	{Args: []string{"create", "logs"}},
	{Args: []string{"create", "logs", "rotated daily"}},
	{Args: []string{"create"}},
	{Args: []string{"create", "logs", "daily", "extra"}},
	{Args: []string{"add", "logs", "app.log", "-l", "9", "db.log"}},
	{Args: []string{"add", "logs"}},
	{Args: []string{"add", "logs", "--", "-odd.log", "--level"}},
	{Args: []string{"add", "logs", "-"}},
	{Args: []string{"list"}},
	{Args: []string{"list", "*.log", "app-*"}},
	{Args: []string{"create", "logs"}, Full: true},
}

// TestArguments checks the transcript of the runs of the example.
func TestArguments(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := transcript.Write(t.Context(), "pack", runs, app().Main)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

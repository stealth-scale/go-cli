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
	{Args: []string{"count", "bolts"}},
	{Args: []string{"--format", "json", "count", "bolts"}},
	{Args: []string{"count", "--help"}},
	{Args: []string{"--format", "json", "count", "--help"}},
	{Args: []string{"count"}},
	{Args: []string{"--format", "json", "count"}},
	{Args: []string{"--format", "json", "cuont", "bolts"}},
	{
		Comment: "The variable selects the format of a run whose command line fails.",
		Vars:    map[string]string{"STOCK_FORMAT": "json"},
		Args:    []string{"count", "bolts", "--bogus"},
	},
	{
		Comment: "A flag after the first fault does not count.",
		Args:    []string{"count", "bolts", "--bogus", "--format", "json"},
	},
	{Args: []string{"--format", "json", "count", "bolts", "-w", "east", "--flag-debug"}},
}

// TestRenderer checks the transcript of the runs of the example.
func TestRenderer(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "stock", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

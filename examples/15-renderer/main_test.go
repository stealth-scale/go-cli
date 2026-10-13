// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"go.dokimi.dev/assert/expect"
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

// TestRenderer checks the transcript of the runs of the example and the streams of a failure.
func TestRenderer(t *testing.T) {
	t.Parallel()

	t.Run("run", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "stock", run, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})

		t.Run("writes a failure in JSON to the standard output", func(t *testing.T) {
			t.Parallel()
			res := clitest.Main(t.Context(), run, &clitest.Input{Args: []string{"--format", "json", "count"}})
			expect.Equal(t, res.Status, exitUsage, "the run exits with the status of a usage error")
			expect.HasPrefix(t, res.Stdout, "{\n  \"app\": \"stock\",", "the standard output contains the failure")
			expect.Empty(t, res.Stderr, "the standard error is empty")
		})

		t.Run("writes a failure in text to the standard error", func(t *testing.T) {
			t.Parallel()
			res := clitest.Main(t.Context(), run, &clitest.Input{Args: []string{"count"}})
			expect.Equal(t, res.Status, exitUsage, "the run exits with the status of a usage error")
			expect.Equal(
				t,
				res.Stderr,
				"stock: count: missing argument ITEM\n",
				"the standard error contains the failure",
			)
			expect.Empty(t, res.Stdout, "the standard output is empty")
		})
	})
}

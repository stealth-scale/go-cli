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
	{Args: []string{"hello"}},
	{Args: []string{"hello", "Ada", "-g", "Hi", "--times", "2"}},
	{Args: []string{"-v", "hello", "Grace"}},
	{Args: []string{"hello", "--times", "twice"}},
	{Args: []string{"--help"}},
	{Args: []string{"hello", "-h"}},
	{Args: nil},
	{Args: []string{"hello"}, FailStdout: true},
}

// TestQuickstart checks the transcript of the runs of the example.
func TestQuickstart(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "greet", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

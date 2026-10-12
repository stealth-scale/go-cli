// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli/clitest"
)

// asJSON contains the variable that selects the output in JSON.
var asJSON = map[string]string{"OUTPUT": "json"}

// runs are the command lines of the transcript of the example.
var runs = []clitest.Input{
	{Args: []string{"count", "bolts"}},
	{Args: []string{"count", "--help"}},
	{Vars: asJSON, Args: []string{"count", "--help"}},
	{Args: []string{"count"}},
	{Vars: asJSON, Args: []string{"count"}},
	{Vars: asJSON, Args: []string{"cuont", "bolts"}},
	{Vars: asJSON, Args: []string{"count", "bolts", "-w", "east", "--flag-debug"}},
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

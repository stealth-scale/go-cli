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
	{Args: []string{"serve", "--help"}},
	{Args: []string{"serve"}},
	{Args: []string{"migrate"}},
	{Args: []string{"-c", "relay.ini", "serve"}},
	{Args: []string{"-c", "relay.conf", "serve"}},
	{Vars: map[string]string{"RELAY_CONFIG": "relay.ini"}, Args: []string{"migrate", "--steps", "1"}},
	{Args: []string{"-c", "typo.json", "serve"}},
	{Args: []string{"serve"}, Full: true},
	{Args: []string{"migrate"}, Full: true},
}

// TestConfig checks the transcript of the runs of the example.
func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := transcript.Write(t.Context(), "relay", runs, app().Main)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

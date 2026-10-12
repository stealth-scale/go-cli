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
	{Args: []string{"backup"}},
	{Vars: map[string]string{"SCHEDULE_BACKUP_EVERY": "6h"}, Args: []string{"backup", "--flag-debug"}},
	{Args: []string{"-c", "broken.toml", "backup"}},
	{Args: []string{"backup"}, Full: true},
}

// TestConfig checks the transcript of the runs of the example.
func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := transcript.Write(t.Context(), "schedule", runs, app().Main)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

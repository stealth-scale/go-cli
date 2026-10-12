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
	{Args: []string{"run", "--help"}},
	{Args: []string{"run"}},
	{Args: []string{"run", "--bucket", "nightly"}},
	{Args: []string{"run", "--bucket", "nightly", "--retain", "3", "-z", "gzip", "-n"}},
	{Args: []string{"run", "--bucket", "nightly", "--schedule", "0 3 * * *"}},
	{
		Vars: map[string]string{"BACKUP_TOKEN": "s3cret", "BACKUP_RUN_RETAIN": "14"},
		Args: []string{"run", "--bucket", "nightly", "--flag-debug"},
	},
	{Args: []string{"run", "--bucket", "nightly", "-z", "rar"}},
	{Args: []string{"run", "--bucket", "nightly"}, Full: true},
}

// TestTags checks the transcript of the runs of the example.
func TestTags(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := transcript.Write(t.Context(), "backup", runs, app().Main)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

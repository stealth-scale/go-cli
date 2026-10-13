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
	{Args: []string{"run", "--help"}},
	{Args: []string{"run"}},
	{Args: []string{"run", "--bucket", "nightly"}},
	{Args: []string{"run", "--bucket", "nightly", "--retain", "3", "-z", "gzip", "-n"}},
	{Args: []string{"run", "--bucket", "nightly", "--schedule", "0 3 * * *"}},
	{
		Args:  []string{"-c", "backup.json", "run", "--bucket", "nightly"},
		Files: clitest.Files{"backup.json": `{"storage": {"endpoint": "https://minio.internal"}}`},
	},
	{
		Vars: map[string]string{"BACKUP_TOKEN": "s3cret", "BACKUP_RUN_RETAIN": "14"},
		Args: []string{"run", "--bucket", "nightly", "--flag-debug"},
	},
	{Args: []string{"run", "--bucket", "nightly", "-z", "rar"}},
	{Args: []string{"run", "--bucket", "nightly"}, FailStdout: true},
}

// TestTags checks the transcript of the runs of the example.
func TestTags(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "backup", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

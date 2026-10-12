// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"testing"

	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli/clitest"
)

// files are the config files of the example.
var files = os.DirFS(".")

// runs are the command lines of the transcript of the example.
var runs = []clitest.Input{
	{Args: []string{"get"}, Files: files},
	{Vars: map[string]string{"FETCH_GET_TIMEOUT": "1m"}, Args: []string{"get"}, Files: files},
	{Vars: map[string]string{"FETCH_GET_TIMEOUT": "1m"}, Args: []string{"get", "--timeout", "2m"}, Files: files},
	{Vars: map[string]string{"FETCH_CONFIG": ""}, Args: []string{"get"}, Files: files},
	{Args: []string{"get", "--flag-debug", "--agent", "curl/8.10"}, Files: files},
	{Vars: map[string]string{"FETCH_FLAG_DEBUG": "true"}, Args: []string{"get", "--retries", "0"}, Files: files},
	{Args: []string{"get", "--token", "s3cret"}, Files: files},
	{Vars: map[string]string{"FETCH_GET_TOKEN": "s3cret"}, Args: []string{"get"}, Files: files},
	{Args: []string{"get"}, Files: files, FailStdout: true},
}

// TestSources checks the transcript of the runs of the example.
func TestSources(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "fetch", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

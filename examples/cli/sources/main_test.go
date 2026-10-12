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
	{Args: []string{"get"}},
	{Vars: map[string]string{"FETCH_GET_TIMEOUT": "1m"}, Args: []string{"get"}},
	{Vars: map[string]string{"FETCH_GET_TIMEOUT": "1m"}, Args: []string{"get", "--timeout", "2m"}},
	{Vars: map[string]string{"FETCH_CONFIG": ""}, Args: []string{"get"}},
	{Args: []string{"get", "--flag-debug", "--agent", "curl/8.10"}},
	{Vars: map[string]string{"FETCH_FLAG_DEBUG": "true"}, Args: []string{"get", "--retries", "0"}},
	{Args: []string{"get"}, Full: true},
}

// TestSources checks the transcript of the runs of the example.
func TestSources(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := transcript.Write(t.Context(), "fetch", runs, app().Main)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

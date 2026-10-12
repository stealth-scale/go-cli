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
	{Args: []string{"show", "--help"}},
	{Args: []string{"show"}},
	{Args: []string{
		"show", "--name", "Ada", "--enabled", "--level", "-8", "--port", "8080", "--ratio", "0.25",
		"--timeout", "1m30s", "--since", "2026-10-12T09:30:00Z", "--addr", "10.0.0.1", "--log", "warn",
		"--upstream", "https://api.example.com/v1", "--mirror", "https://mirror.example.com",
		"--zone", "Europe/Amsterdam", "--network", "10.0.0.0/8", "--peer", "192.168.0.0/16",
		"--mode", "0640", "--workers", "4", "--tags", "a,b", "--ports", "80,443", "--limits", "cpu=2,mem=8",
		"--limits", "disk=100",
	}},
	{Args: []string{"show", "--port", "70000"}},
	{Args: []string{"show", "--level", "1.5"}},
	{Args: []string{"show", "--enabled=maybe"}},
	{Args: []string{"show", "--timeout", "90"}},
	{Args: []string{"show", "--since", "yesterday"}},
	{Args: []string{"show", "--addr", "10.0.0"}},
	{Args: []string{"show", "--log", "loud"}},
	{Args: []string{"show", "--upstream", "%zz"}},
	{Args: []string{"show", "--zone", "Mars/Base"}},
	{Args: []string{"show", "--network", "10.0.0.0/33"}},
	{Args: []string{"show", "--mode", "0999"}},
	{Args: []string{"show", "--limits", "cpu"}},
	{Args: []string{"show"}, FailStdout: true},
}

// TestTypes checks the transcript of the runs of the example.
func TestTypes(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "types", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

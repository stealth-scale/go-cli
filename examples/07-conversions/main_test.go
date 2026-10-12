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
	{Args: []string{"check", "--help"}},
	{Args: []string{"check"}},
	{Args: []string{
		"check", "--max", "64KB", "--sizes", "512B,1MB", "--version", "v2.3.4", "--target", "19.5C",
		"--origin", "3,4", "--webhook", "https://hooks.example.com/alerts",
	}},
	{Args: []string{"check", "--max", "5TB"}},
	{Args: []string{"check", "--max", "1.5MB"}},
	{Args: []string{"check", "--max", "5"}},
	{Args: []string{"check", "--version", "2.0"}},
	{Args: []string{"check", "--version", "v1.x.3"}},
	{Args: []string{"check", "--target", "70F"}},
	{Args: []string{"check", "--target", "warmC"}},
	{Args: []string{"check", "--origin", "3"}},
	{Args: []string{"check", "--webhook", "http://hooks.example.com/alerts"}},
	{Args: []string{"check", "--webhook", "%zz"}},
	{Args: []string{"check"}, Full: true},
}

// TestConversions checks the transcript of the runs of the example.
func TestConversions(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := transcript.Write(t.Context(), "units", runs, app().Main)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

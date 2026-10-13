// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli/clitest"
)

// files are the workspaces of the runs. The file .git marks the top of a repository.
var files = clitest.Files{
	"ws/.acme.json":     `{"version": 1, "strict": false, "plans": {"api": {"enabled": false}, "web": {"enabled": true}}}`,
	"strict/.acme.json": `{"version": 1, "strict": true, "plans": {"api": {"enabled": false}}}`,
	"typo/.acme.json":   `{"version": 1, "strickt": true}`,
	"repo/.git":         "",
	".acme.json":        `{"version": 1, "plans": {"root": {"enabled": true}}}`,
}

// runs are the command lines of the transcript of the example.
var runs = []clitest.Input{
	{Args: []string{"--chdir", "ws/svc/store", "plans"}, Files: files},
	{Args: []string{"--chdir", "ws/svc/store", "--flag-debug", "plans"}, Files: files},
	{Comment: "The workspace file sets the option strict.", Args: []string{"--chdir", "strict", "plans"}, Files: files},
	{Args: []string{"--chdir", "strict", "--strict=false", "plans"}, Files: files},
	{
		Comment: "The search stops at the directory with .git.",
		Args:    []string{"--chdir", "repo/pkg", "plans"},
		Files:   files,
	},
	{Args: []string{"--chdir", "typo", "plans"}, Files: files},
	{
		Comment: "A path of the flag --config needs no search.",
		Args:    []string{"-c", ".acme.json", "plans"},
		Files:   files,
	},
}

// TestWorkspace checks the transcript of the runs of the example.
func TestWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "acme", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

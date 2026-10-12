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
	{Args: []string{"--help"}},
	{Args: []string{"plan", "--env", "prod"}},
	{Args: []string{"-v", "apply", "-e", "prod", "-y"}},
	{Args: []string{"apply", "-e", "prod", "-y", "--verbose"}},
	{Args: []string{"apply", "-e", "prod"}},
	{Args: []string{"rollback", "-e", "dev", "--region", "us-east-1", "41"}},
	{Args: []string{"plan"}},
	{Args: []string{"plan", "--env", "staging"}},
	{Args: []string{"aply", "-e", "dev"}},
	{Args: []string{"rollback", "--help"}},
	{Args: []string{"plan", "-e", "dev"}, FailStdout: true},
	{Args: []string{"registry"}},
	{Args: []string{"-v", "registry", "push", "app:v2"}},
	{Args: []string{"registry", "--url", "registry.internal", "prune", "--user", "ci", "--keep", "2"}},
	{Vars: map[string]string{"DEPLOY_REGISTRY_USER": "ci"}, Args: []string{"registry", "push", "app:v3"}},
	{Args: []string{"registry", "pusj", "app:v2"}},
	{Args: []string{"registry", "push"}},
	{Args: []string{"registry", "--keep", "2", "prune"}},
	{Args: []string{"registry", "push", "--help"}},
}

// TestCommands checks the transcript of the runs of the example.
func TestCommands(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "deploy", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

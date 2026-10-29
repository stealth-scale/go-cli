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
	{Comment: "The help lists the variable of each option.", Args: []string{"serve", "--help"}},
	{
		Comment: "STORE_SERVE_PORT sets the option Port.",
		Vars:    map[string]string{"STORE_SERVE_PORT": "9000"},
		Args:    []string{"serve"},
	},
	{
		Comment: "The program does not read a variable that starts with its name.",
		Vars:    map[string]string{"STORECTL_SERVE_PORT": "9000"},
		Args:    []string{"serve"},
	},
	{
		Comment: "The tag env of DatabaseURL replaces the prefix with the whole variable DATABASE_URL.",
		Vars:    map[string]string{"DATABASE_URL": "postgres://store@db/store", "STORE_SERVE_WORKERS": "8"},
		Args:    []string{"serve", "--flag-debug"},
	},
	{
		Comment: "STORE_FLAG_DEBUG turns on the flag --flag-debug.",
		Vars:    map[string]string{"STORE_FLAG_DEBUG": "true"},
		Args:    []string{"serve", "-p", "9001"},
	},
	{Args: []string{"serve"}, FailStdout: true},
}

// TestEnvironment checks the transcript of the runs of the example.
func TestEnvironment(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "storectl", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

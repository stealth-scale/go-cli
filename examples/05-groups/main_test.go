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
	{Args: []string{"serve", "--help"}},
	{Args: []string{"serve", "--addr", ":443", "--tls-cert", "public.pem", "--tls-key", "public.key"}},
	{
		Vars: map[string]string{"PROXY_SERVE_ADMIN_TLS_CERT": "rotated.pem", "PROXY_SERVE_BACKEND_TIMEOUT": "5s"},
		Args: []string{"-c", "proxy.json", "serve", "--cache-dir", "/var/cache/proxy", "--flag-debug"},
	},
	{Args: []string{"serve"}, Full: true},
}

// TestGroups checks the transcript of the runs of the example.
func TestGroups(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := transcript.Write(t.Context(), "proxy", runs, app().Main)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})
	})
}

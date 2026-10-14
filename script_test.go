// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"context"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli"
)

// TestScript checks the completion scripts that the command completion writes.
func TestScript(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
			t.Run("writes the script of "+shell, func(t *testing.T) {
				t.Parallel()
				app := cli.New[struct{}]("my-app", appHelp)
				app.Command(
					command,
					commandHelp,
					func(context.Context, *cli.Run[struct{}, struct{}]) error { return nil },
				)
				res := execute(t, app, []string{"completion", shell}, nil, nil)
				assert.NoError(t, res.err, "Execute writes the script")
				golden.Match(t, "completion/"+shell+".txt", []byte(res.stdout), golden.ShouldUpdate())
			})
		}

		t.Run("returns an error that wraps the error of the standard output", func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			err := completing(t).Execute(t.Context(), &cli.Env{
				Stdout: failingWriter{}, Stderr: &stderr, Args: []string{"completion", "zsh"},
			})
			assert.ErrorIs(t, err, errWrite, "Execute returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the completion script: disk full", "the error states the script")
		})
	})
}

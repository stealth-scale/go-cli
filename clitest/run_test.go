// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package clitest_test

import (
	"context"
	"io"
	"testing"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/clitest"
)

// contextKey is the key of the value that a test puts into the context of Main.
type contextKey struct{}

// TestRun checks one run of a program and its result.
func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("Main", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the exit status of the program", func(t *testing.T) {
			t.Parallel()
			res := clitest.Main(t.Context(), say(), &clitest.Input{Args: []string{"echo", "--fial"}})
			assert.Equal(t, res.Status, cli.ExitUsage, "Main returns the status of the program")
		})

		t.Run("returns the standard output and the standard error of the run", func(t *testing.T) {
			t.Parallel()
			res := clitest.Main(t.Context(), say(), &clitest.Input{Args: []string{"echo", "a"}, Stdin: "in\n"})
			assert.Equal(t, res.Status, cli.ExitOK, "Main returns ExitOK")
			assert.Equal(t, res.Stdout, "A\n", "Main returns the standard output")
			assert.Equal(t, res.Stderr, "in\n", "Main returns the standard error")
		})

		t.Run("passes the values of the context of the caller to the program", func(t *testing.T) {
			t.Parallel()
			var got any
			program := func(ctx context.Context, _ *cli.Env) int {
				got = ctx.Value(contextKey{})
				return cli.ExitOK
			}
			clitest.Main(context.WithValue(t.Context(), contextKey{}, "value"), program, &clitest.Input{})
			assert.Equal(t, got, any("value"), "the program reads the value of the context of the caller")
		})

		t.Run("cancels the context of the run when the program returns", func(t *testing.T) {
			t.Parallel()
			var ended func() error
			program := func(ctx context.Context, env *cli.Env) int {
				ended = ctx.Err
				_, err := io.WriteString(env.Stdout, "done\n")
				assert.NoError(t, err, "the program writes")
				return cli.ExitOK
			}
			clitest.Main(t.Context(), program, &clitest.Input{})
			assert.ErrorIs(t, ended(), context.Canceled, "the context of the run has ended")
		})
	})
}

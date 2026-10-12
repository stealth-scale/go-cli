// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/clitest"
)

// input is the standard input of the tests: three lines of five words.
const input = "one two\nthree four\nfive\n"

// runs are the command lines of the transcript of the example.
var runs = []clitest.Input{
	{Comment: "Without an option, wc writes every number.", Stdin: input},
	{Comment: "The flag --words selects the number of words.", Stdin: input, Args: []string{"--words"}},
	{Comment: "Short flags combine in one argument.", Stdin: input, Args: []string{"-lc"}},
	{Comment: "An empty standard input has no lines.", Args: []string{"--lines"}},
	{Comment: "An unknown flag fails with the exit status 2.", Stdin: input, Args: []string{"--chars"}},
	{Comment: "A failed write fails with the exit status 1.", Stdin: input, FailStdout: true},
	{Comment: "A failed read fails with the exit status 1.", FailStdin: true},
}

// newRun returns a run of the options config with an Env of in, and the output of the run. The
// options of the app are the options of its handler, so config is both.
func newRun(t *testing.T, config *Count, in *clitest.Input) (*cli.Run[Count, Count], *clitest.Output) {
	t.Helper()
	env, out := clitest.NewEnv(in)
	run, err := cli.NewRun(config, config, env)
	assert.NoError(t, err, "NewRun returns a run")
	return run, out
}

// TestTesting checks the transcript of the runs, one run and the error of an unknown flag through
// clitest. It also tests the handler count on its own.
func TestTesting(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "wc", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})

		t.Run("writes the number of words of the standard input", func(t *testing.T) {
			t.Parallel()
			res := clitest.Main(t.Context(), app().Main, &clitest.Input{Stdin: input, Args: []string{"-w"}})
			assert.Equal(t, res.Status, cli.ExitOK, "Main returns ExitOK")
			assert.Equal(t, res.Stdout, "5\n", "the run writes the number of words")
		})

		t.Run("returns an *Error for an unknown flag", func(t *testing.T) {
			t.Parallel()
			env, out := clitest.NewEnv(&clitest.Input{Args: []string{"--chars"}})
			err := app().Execute(t.Context(), env)
			e := assert.ErrorAs[*cli.Error](t, err, "Execute returns an *Error")
			assert.ErrorIs(t, e, cli.ErrUnknownFlag, "the error wraps ErrUnknownFlag")
			assert.Equal(t, e.Input, "--chars", "the input of the error is the flag")
			assert.Empty(t, out.Stdout(), "the run writes nothing to its standard output")
		})
	})

	t.Run("count", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the number of words of the standard input", func(t *testing.T) {
			t.Parallel()
			run, out := newRun(t, &Count{Words: true}, &clitest.Input{Stdin: input})
			assert.NoError(t, count(t.Context(), run), "count reads the standard input")
			assert.Equal(t, out.Stdout(), "5\n", "count writes the number of words")
		})

		t.Run("returns the error of a context that ended", func(t *testing.T) {
			t.Parallel()
			run, _ := newRun(t, &Count{}, &clitest.Input{Stdin: input})
			assert.HonoursCancellation(t, func(ctx context.Context) error { return count(ctx, run) },
				"count stops when its context ends")
		})

		t.Run("returns the error of a standard input that fails", func(t *testing.T) {
			t.Parallel()
			run, _ := newRun(t, &Count{}, &clitest.Input{FailStdin: true})
			assert.ErrorIs(t, count(t.Context(), run), clitest.ErrStdin, "count returns the error of the read")
		})
	})
}

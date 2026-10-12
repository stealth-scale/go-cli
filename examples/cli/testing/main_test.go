// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/examples/internal/transcript"
)

// input is the standard input of the tests: three lines of five words.
const input = "one two\nthree four\nfive\n"

// errRead is the error of every read of brokenInput.
var errRead = errors.New("main: the input is gone")

// runs are the command lines of the transcript of the example.
var runs = []transcript.Run{
	{Stdin: input, Args: []string{"count"}},
	{Stdin: input, Args: []string{"count", "--words"}},
	{Stdin: input, Args: []string{"count", "-lc"}},
	{Args: []string{"count", "--lines"}},
	{Stdin: input, Args: []string{"count", "--chars"}},
	{Stdin: input, Args: []string{"count"}, Full: true},
}

// brokenInput is a standard input whose every read fails with errRead.
type brokenInput struct{}

// Read returns errRead.
func (brokenInput) Read([]byte) (int, error) {
	return 0, errRead
}

// newRun returns a run of the options config with the standard input stdin. It also returns the
// buffer of the standard output of the run.
func newRun(t *testing.T, config *Count, stdin io.Reader) (*cli.Run[struct{}, Count], *bytes.Buffer) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	run, err := cli.NewRun(&struct{}{}, config, &cli.Env{Stdin: stdin, Stdout: &stdout, Stderr: &stderr})
	assert.NoError(t, err, "NewRun returns a run")
	return run, &stdout
}

// TestTesting checks the transcript of the runs of the example, the handler count on its own, and
// the error of an unknown flag.
func TestTesting(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := transcript.Write(t.Context(), "wc", runs, app().Main)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})

		t.Run("returns an *Error for an unknown flag", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := app().Execute(t.Context(), &cli.Env{Stdout: &out, Stderr: &out, Args: []string{"count", "--chars"}})
			e := assert.ErrorAs[*cli.Error](t, err, "Execute returns an *Error")
			assert.ErrorIs(t, e, cli.ErrUnknownFlag, "the error wraps ErrUnknownFlag")
			assert.Equal(t, e.Input, "--chars", "the input of the error is the flag")
		})
	})

	t.Run("count", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the number of words of the standard input", func(t *testing.T) {
			t.Parallel()
			run, stdout := newRun(t, &Count{Words: true}, strings.NewReader(input))
			assert.NoError(t, count(t.Context(), run), "count reads the standard input")
			assert.Equal(t, stdout.String(), "5\n", "count writes the number of words")
		})

		t.Run("returns the error of a context that ended", func(t *testing.T) {
			t.Parallel()
			run, _ := newRun(t, &Count{}, strings.NewReader(input))
			assert.HonoursCancellation(t, func(ctx context.Context) error { return count(ctx, run) },
				"count stops when its context ends")
		})

		t.Run("returns the error of a standard input that fails", func(t *testing.T) {
			t.Parallel()
			run, _ := newRun(t, &Count{}, brokenInput{})
			assert.ErrorIs(t, count(t.Context(), run), errRead, "count returns the error of the read")
		})
	})
}

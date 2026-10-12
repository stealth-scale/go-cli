// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"testing/fstest"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli"
)

// handlerText is what the handler of the tests of --flag-debug writes to the standard error.
const handlerText = "handler\n"

// errMark is the error of the MarshalText of a mark.
var errMark = errors.New("cli_test: a mark has no text")

// mark is a string whose MarshalText fails, so --flag-debug writes the result of its String.
type mark string

// MarshalText returns errMark.
func (mark) MarshalText() ([]byte, error) {
	return nil, errMark
}

// String returns the mark in brackets.
func (m mark) String() string {
	return "[" + string(m) + "]"
}

// The options of the app in the tests of --flag-debug.
type (
	debugGlobal struct {
		Config  cli.ConfigFile
		Verbose bool
	}
	debugOptions struct {
		Target   string `arg:""`
		Since    time.Time
		Labels   map[string]int
		Workers  *int
		Limit    *int
		Allow    []net.IPNet
		Tags     []string
		Password string `secret:""`
		Addr     string `default:":8080"`
		Mark     mark
		Timeout  time.Duration
		Port     int
	}
)

// debugApp returns the app srv with the command serve, whose handler writes handlerText to the
// standard error of its run.
func debugApp(tb assert.TB) *cli.App[debugGlobal] {
	tb.Helper()
	app := cli.New[debugGlobal](appName, appHelp)
	app.Command(command, commandHelp, func(_ context.Context, run *cli.Run[debugGlobal, debugOptions]) error {
		_, err := run.Stderr().Write([]byte(handlerText))
		return err
	})
	return app
}

// TestDebug checks the table of the values and the sources of the options that --flag-debug writes.
func TestDebug(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the value and the source of each option before the handler runs", func(t *testing.T) {
			t.Parallel()
			files := fstest.MapFS{configPath: {Data: []byte(`{"serve": {"timeout": "5s", "labels": {"env": 1}}}`)}}
			args := []string{
				"--config", configPath, "--flag-debug", command, "--allow", "10.0.0.0/8,192.168.0.0/16",
				"--limit", "8", "--password", "hunter2", "--since", "2026-10-12T09:30:00Z", "--tags", "a b,c",
				"--mark", "m", "v2",
			}
			res := execute(t, debugApp(t), args, map[string]string{"SRV_SERVE_PORT": "81"}, files)
			assert.NoError(t, res.err, "Execute runs the command")
			golden.Match(t, "debug/table.txt", []byte(res.stderr), golden.ShouldUpdate())
		})

		t.Run("writes the table for the environment variable of --flag-debug", func(t *testing.T) {
			t.Parallel()
			res := execute(t, debugApp(t), []string{command, "v2"}, map[string]string{"SRV_FLAG_DEBUG": "true"}, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.HasPrefix(t, res.stderr, "  OPTION", "the run writes the table")
		})

		t.Run("writes no table without --flag-debug", func(t *testing.T) {
			t.Parallel()
			res := execute(t, debugApp(t), []string{command, "v2"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, res.stderr, handlerText, "the run writes the output of the handler alone")
		})

		t.Run("converts --flag-debug by its kind under a conversion of bool", func(t *testing.T) {
			t.Parallel()
			app := debugApp(t)
			app.Convert(func(string) (bool, error) { return false, errors.New("main: no bool") })
			res := execute(t, app, []string{"--flag-debug", command, "v2"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.HasPrefix(t, res.stderr, "  OPTION", "the run writes the table")
		})

		t.Run("lists the options of the ancestors of a subcommand before its own options", func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			app, _, _ := nesting[commandGlobal, Store, stepsOptions](t)
			app.Render(r)
			res := execute(t, app, []string{parent, subcommand, "--flag-debug", "--steps", "3"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.Equal(t, r.rows, []cli.DebugRow{
				{Option: "--dsn", Value: `""`},
				{Option: "--verbose", Value: "false"},
				{Option: "--steps", Value: "3", Source: cli.Source{Kind: cli.Flag, Name: "--steps"}},
			}, "the rows list the options of the ancestors and then the options of the subcommand")
		})

		t.Run("returns the error of Stderr when the table fails to write", func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			env := &cli.Env{Stdout: &stdout, Stderr: failingWriter{}, Args: []string{"--flag-debug", command, "v2"}}
			err := debugApp(t).Execute(t.Context(), env)
			assert.ErrorIs(t, err, errWrite, "Execute returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the output of --flag-debug: disk full", "the error names the flag")
		})
	})
}

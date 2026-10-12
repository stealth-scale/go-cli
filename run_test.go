// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// runGlobal and runOptions are the options of the app in the tests of Run.
type (
	runGlobal struct {
		Verbose bool
	}
	runOptions struct {
		Addr string `default:":8080"`
		Port int
	}
)

// TestRun checks the options, the streams and the sources that a handler reads from its run.
func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("NewRun", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a run of the options", func(t *testing.T) {
			t.Parallel()
			global, config := &runGlobal{Verbose: true}, &runOptions{Addr: ":80"}
			var out bytes.Buffer
			run, err := cli.NewRun(global, config, &cli.Env{Stdout: &out, Stderr: &out})
			assert.NoError(t, err, "NewRun returns a run")
			assert.Equal(t, run.Global(), global, "the run returns the global options", assert.ByIdentity())
			assert.Equal(t, run.Config(), config, "the run returns the options of the command", assert.ByIdentity())
		})

		t.Run("returns ErrEnv for a nil Env", func(t *testing.T) {
			t.Parallel()
			_, err := cli.NewRun(&runGlobal{}, &runOptions{}, nil)
			assert.ErrorIs(t, err, cli.ErrEnv, "NewRun returns ErrEnv")
		})

		t.Run("returns ErrDeclaration for nil global options", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			_, err := cli.NewRun[runGlobal](nil, &runOptions{}, &cli.Env{Stdout: &out, Stderr: &out})
			assert.ErrorIs(t, err, cli.ErrDeclaration, "NewRun returns ErrDeclaration")
		})

		t.Run("returns ErrDeclaration for nil options of the command", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			_, err := cli.NewRun[runGlobal, runOptions](&runGlobal{}, nil, &cli.Env{Stdout: &out, Stderr: &out})
			assert.ErrorIs(t, err, cli.ErrDeclaration, "NewRun returns ErrDeclaration")
		})
	})

	t.Run("Global", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the global options after every source is applied", func(t *testing.T) {
			t.Parallel()
			app, got := serving[runGlobal, runOptions](t)
			res := execute(t, app, []string{command, "--verbose"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.True(t, got.Global().Verbose, "the run returns the global flag")
		})
	})

	t.Run("Config", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the options of the command after every source is applied", func(t *testing.T) {
			t.Parallel()
			app, got := serving[runGlobal, runOptions](t)
			res := execute(t, app, []string{command, "--port", "9"}, map[string]string{"SRV_SERVE_ADDR": ":80"}, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(
				t,
				*got.Config(),
				runOptions{Addr: ":80", Port: 9},
				"the run returns the options of every source",
			)
		})
	})

	t.Run("Stdin", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the Stdin of the Env", func(t *testing.T) {
			t.Parallel()
			in, out := strings.NewReader("data"), &bytes.Buffer{}
			run, err := cli.NewRun(&runGlobal{}, &runOptions{}, &cli.Env{Stdin: in, Stdout: out, Stderr: out})
			assert.NoError(t, err, "NewRun returns a run")
			assert.Equal(t, run.Stdin(), io.Reader(in), "Stdin returns the Stdin of the Env", assert.ByIdentity())
		})

		t.Run("returns an empty reader for an Env without Stdin", func(t *testing.T) {
			t.Parallel()
			out := &bytes.Buffer{}
			run, err := cli.NewRun(&runGlobal{}, &runOptions{}, &cli.Env{Stdout: out, Stderr: out})
			assert.NoError(t, err, "NewRun returns a run")
			data, err := io.ReadAll(run.Stdin())
			assert.NoError(t, err, "the input reads")
			assert.Empty(t, data, "the input is empty")
		})
	})

	t.Run("Stdout", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the Stdout of the Env", func(t *testing.T) {
			t.Parallel()
			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
			run, err := cli.NewRun(&runGlobal{}, &runOptions{}, &cli.Env{Stdout: stdout, Stderr: stderr})
			assert.NoError(t, err, "NewRun returns a run")
			assert.Equal(
				t,
				run.Stdout(),
				io.Writer(stdout),
				"Stdout returns the Stdout of the Env",
				assert.ByIdentity(),
			)
		})
	})

	t.Run("Stderr", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the Stderr of the Env", func(t *testing.T) {
			t.Parallel()
			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
			run, err := cli.NewRun(&runGlobal{}, &runOptions{}, &cli.Env{Stdout: stdout, Stderr: stderr})
			assert.NoError(t, err, "NewRun returns a run")
			assert.Equal(
				t,
				run.Stderr(),
				io.Writer(stderr),
				"Stderr returns the Stderr of the Env",
				assert.ByIdentity(),
			)
		})
	})

	t.Run("Environ", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the environment of the Env", func(t *testing.T) {
			t.Parallel()
			out := &bytes.Buffer{}
			env := &cli.Env{Environ: cli.Environ(map[string]string{"TERM": "xterm"}), Stdout: out, Stderr: out}
			run, err := cli.NewRun(&runGlobal{}, &runOptions{}, env)
			assert.NoError(t, err, "NewRun returns a run")
			assert.Equal(
				t,
				run.Environ(),
				[]string{"TERM=xterm"},
				"Environ returns the result of the Environ of the Env",
			)
		})

		t.Run("returns nil for an Env without Environ", func(t *testing.T) {
			t.Parallel()
			out := &bytes.Buffer{}
			run, err := cli.NewRun(&runGlobal{}, &runOptions{}, &cli.Env{Stdout: out, Stderr: out})
			assert.NoError(t, err, "NewRun returns a run")
			assert.Nil(t, run.Environ(), "Environ returns nil")
		})
	})

	t.Run("Source", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the source of an option of the command", func(t *testing.T) {
			t.Parallel()
			app, got := serving[runGlobal, runOptions](t)
			res := execute(t, app, []string{command, "--port", "9"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			src, ok := got.Source(&got.Config().Port)
			assert.True(t, ok, "Source reports the option")
			assert.Equal(t, src, cli.Source{Kind: cli.Flag, Name: "--port"}, "Source returns the flag")
		})

		t.Run("returns the source of a global option", func(t *testing.T) {
			t.Parallel()
			app, got := serving[runGlobal, runOptions](t)
			res := execute(t, app, []string{command}, map[string]string{"SRV_VERBOSE": "true"}, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			src, ok := got.Source(&got.Global().Verbose)
			assert.True(t, ok, "Source reports the option")
			assert.Equal(t, src, cli.Source{Kind: cli.Environment, Name: "SRV_VERBOSE"}, "Source returns the variable")
		})

		t.Run("reports false for a pointer that is no option of the run", func(t *testing.T) {
			t.Parallel()
			app, got := serving[runGlobal, runOptions](t)
			res := execute(t, app, []string{command}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			_, ok := got.Source(new(int))
			assert.False(t, ok, "Source reports no source")
		})

		t.Run("reports false for an option of a run from NewRun", func(t *testing.T) {
			t.Parallel()
			config, out := &runOptions{}, &bytes.Buffer{}
			run, err := cli.NewRun(&runGlobal{}, config, &cli.Env{Stdout: out, Stderr: out})
			assert.NoError(t, err, "NewRun returns a run")
			_, ok := run.Source(&config.Addr)
			assert.False(t, ok, "Source reports no source")
		})
	})
}

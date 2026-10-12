// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// errRender is the error of a renderer that fails.
var errRender = errors.New("main: the terminal is gone")

// recorder is a Renderer that records what the package passes to it, and returns err.
type recorder struct {
	err     error
	help    *cli.Help
	failure *cli.Failure
	environ []string
	rows    []cli.DebugRow
}

// Help records h and environ.
func (r *recorder) Help(_ io.Writer, environ []string, h *cli.Help) error {
	r.help, r.environ = h, environ
	return r.err
}

// Error records f and environ.
func (r *recorder) Error(_ io.Writer, environ []string, f *cli.Failure) error {
	r.failure, r.environ = f, environ
	return r.err
}

// Debug records rows and environ.
func (r *recorder) Debug(_ io.Writer, environ []string, rows []cli.DebugRow) error {
	r.rows, r.environ = rows, environ
	return r.err
}

// globalsHelp are the global options of an app of the options appGlobal in its help.
var globalsHelp = []cli.HelpOption{
	{Name: "--config", Short: "-c", Placeholder: "file", Env: "SRV_CONFIG"},
	{
		Name: "--flag-debug", Text: "Print the value and the source of each option before the command runs.",
		Env: "SRV_FLAG_DEBUG",
	},
	{Name: "--help", Short: "-h", Text: "Show the help."},
	{Name: "--verbose", Short: "-v", Env: "SRV_VERBOSE", Key: "verbose"},
}

// rendering returns the app srv with the command serve of the options appOptions, whose output goes
// to r, and whose handler returns err.
func rendering(r cli.Renderer, err error) *cli.App[appGlobal] {
	app := cli.New[appGlobal](appName, appHelp)
	app.Command(command, commandHelp, func(context.Context, *cli.Run[appGlobal, appOptions]) error { return err })
	app.Render(r)
	return app
}

// TestRender checks the registration of a renderer, what the package passes to it, and the text
// that the renderer Plain writes.
func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		t.Run("passes the help of the app to the renderer", func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			res := execute(t, rendering(r, nil), []string{"-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.Equal(t, r.help, &cli.Help{
				App:      appName,
				Usage:    []string{"[global options]", "<command>", "[options]"},
				Text:     appHelp,
				Commands: []cli.HelpCommand{{Name: command, Text: commandHelp}},
				Globals:  globalsHelp,
			}, "the renderer receives the usage, the text, the commands and the global options of the app")
		})

		t.Run("passes the help of a command to the renderer", func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			res := execute(t, rendering(r, nil), []string{command, "--help"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.Equal(t, r.help, &cli.Help{
				App:       appName,
				Command:   command,
				Usage:     []string{"[options]", "[TARGET]"},
				Text:      commandHelp,
				Arguments: []cli.HelpOption{{Name: "TARGET", Default: `"head"`}},
				Options: []cli.HelpOption{
					{Name: "--labels", Placeholder: "string=int,...", Env: "SRV_SERVE_LABELS", Key: "serve.labels"},
					{Name: "--port", Short: "-p", Placeholder: "int", Env: "SRV_SERVE_PORT", Key: "serve.port"},
					{Name: "--tags", Placeholder: "string,...", Env: "SRV_SERVE_TAGS", Key: "serve.tags"},
				},
				Globals: globalsHelp,
			}, "the renderer receives the usage, the arguments, the options and the global options of the command")
		})

		t.Run("passes the rows of --flag-debug to the renderer", func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			res := execute(t, rendering(r, nil), []string{command, "--flag-debug", "-p", "8"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, r.rows, []cli.DebugRow{
				{Option: "--config", Value: `""`},
				{Option: "--verbose", Value: "false"},
				{Option: "--labels", Value: "map[]"},
				{Option: "--port", Value: "8", Source: cli.Source{Name: "--port", Kind: cli.Flag}},
				{Option: "--tags", Value: "[]"},
				{Option: "TARGET", Value: `"head"`},
			}, "the renderer receives the global options and then the options of the command")
		})

		t.Run("passes the failure of an invalid command line to the renderer", func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			var out bytes.Buffer
			args := []string{command, "--prot", "8"}
			status := rendering(r, nil).Main(t.Context(), &cli.Env{Stdout: &out, Stderr: &out, Args: args})
			assert.Equal(t, status, cli.ExitUsage, "Main returns ExitUsage")
			assert.ErrorIs(t, r.failure.Err, cli.ErrUnknownFlag, "the failure contains the error of the run")
			r.failure.Err = nil
			assert.Equal(t, r.failure, &cli.Failure{
				App: appName, Command: command, Message: "unknown flag --prot", Suggestion: "--port", Usage: true,
			}, "the renderer receives the parts of the message")
		})

		t.Run("passes the failure of a handler to the renderer", func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			var out bytes.Buffer
			app := rendering(r, errHandler)
			status := app.Main(t.Context(), &cli.Env{Stdout: &out, Stderr: &out, Args: []string{command}})
			assert.Equal(t, status, cli.ExitFailure, "Main returns ExitFailure")
			assert.Equal(
				t,
				r.failure.Err,
				errHandler,
				"the failure contains the error of the handler",
				assert.ByIdentity(),
			)
			r.failure.Err = nil
			assert.Equal(t, r.failure, &cli.Failure{App: appName, Message: "main: the store is down"},
				"the renderer receives the text of the error as the message")
		})

		t.Run("passes the environment of the Env to the renderer", func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			var out bytes.Buffer
			env := &cli.Env{Environ: cli.Environ(map[string]string{"NO_COLOR": "1"}), Stdout: &out, Stderr: &out}
			env.Args = []string{"--help"}
			assert.NoError(t, rendering(r, nil).Execute(t.Context(), env), "Execute writes the help")
			assert.Equal(t, r.environ, []string{"NO_COLOR=1"}, "the renderer receives the result of Environ")
		})

		t.Run("returns the error of the help of the renderer unchanged", func(t *testing.T) {
			t.Parallel()
			res := execute(t, rendering(&recorder{err: errRender}, nil), []string{"-h"}, nil, nil)
			assert.Equal(t, res.err, errRender, "Execute returns the error of Help", assert.ByIdentity())
		})

		t.Run("returns the error of the help of a command line without a command unchanged", func(t *testing.T) {
			t.Parallel()
			res := execute(t, rendering(&recorder{err: errRender}, nil), nil, nil, nil)
			assert.Equal(t, res.err, errRender, "Execute returns the error of Help", assert.ByIdentity())
		})

		t.Run("returns the error of the table of the renderer unchanged", func(t *testing.T) {
			t.Parallel()
			res := execute(t, rendering(&recorder{err: errRender}, nil), []string{command, "--flag-debug"}, nil, nil)
			assert.Equal(t, res.err, errRender, "Execute returns the error of Debug", assert.ByIdentity())
		})

		t.Run("returns the exit status of the run when the renderer fails to write the failure", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			app := rendering(&recorder{err: errRender}, nil)
			status := app.Main(t.Context(), &cli.Env{Stdout: &out, Stderr: &out, Args: []string{"migrate"}})
			assert.Equal(t, status, cli.ExitUsage, "Main returns ExitUsage")
		})

		t.Run("uses the renderer of the last call", func(t *testing.T) {
			t.Parallel()
			first, last := &recorder{}, &recorder{}
			app := rendering(first, nil)
			app.Render(last)
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.Nil(t, first.help, "the first renderer receives nothing")
			assert.NotNil(t, last.help, "the last renderer receives the help")
		})

		t.Run("returns ErrDeclaration for a nil renderer", func(t *testing.T) {
			t.Parallel()
			res := execute(t, rendering(nil, nil), []string{command}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(
				t,
				res.err.Error(),
				"cli: invalid declaration: the renderer is nil",
				"the error names the renderer",
			)
		})

		t.Run("writes the failure of a nil renderer with Plain", func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			status := rendering(nil, nil).Main(t.Context(), &cli.Env{Stdout: &stdout, Stderr: &stderr})
			assert.Equal(t, status, cli.ExitFailure, "Main returns ExitFailure")
			assert.Equal(t, stderr.String(), "srv: cli: invalid declaration: the renderer is nil\n",
				"Main writes the error with Plain")
		})
	})

	t.Run("Help", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the usage, the text and the commands of an app", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := cli.Plain{}.Help(&out, nil, &cli.Help{
				App:      appName,
				Usage:    []string{"[global options]", "<command>", "[options]"},
				Text:     appHelp,
				Commands: []cli.HelpCommand{{Name: command, Text: commandHelp}, {Name: "migrate"}},
				Globals:  []cli.HelpOption{{Name: "--help", Short: "-h", Text: "Show the help."}},
			})
			assert.NoError(t, err, "Help writes the help")
			assert.Equal(t, out.String(), "Usage: srv [global options] <command> [options]\n\nServe the store.\n\n"+
				"Commands:\n  serve    Run the server.\n  migrate\n\nGlobal options:\n  -h, --help  Show the help.\n",
				"Help writes the usage, the text, the commands and the global options in columns")
		})

		t.Run("writes the arguments, the options and the notes of a command", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := cli.Plain{}.Help(&out, nil, &cli.Help{
				App:       appName,
				Command:   command,
				Usage:     []string{"[options]", "[NAME]"},
				Text:      commandHelp,
				Arguments: []cli.HelpOption{{Name: "NAME", Text: "Name of the store.", Default: `"main"`}},
				Options: []cli.HelpOption{
					{
						Name: "--addr", Short: "-a", Placeholder: "string", Text: "Address.", Default: `":8080"`,
						Env: "SRV_SERVE_ADDR", Key: "serve.addr",
					},
					{Name: "--token", Placeholder: "string", Text: "Token.", Required: true},
				},
			})
			assert.NoError(t, err, "Help writes the help")
			assert.Equal(t, out.String(), "Usage: srv serve [options] [NAME]\n\nRun the server.\n\n"+
				"Arguments:\n      NAME  Name of the store.  default \"main\"\n\n"+
				"Options:\n  -a, --addr string   Address.  default \":8080\", env SRV_SERVE_ADDR, key serve.addr\n"+
				"      --token string  Token.    required\n",
				"Help writes the arguments and the options with their notes in columns")
		})

		t.Run("leaves out the text of a help without text", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := cli.Plain{}.Help(&out, nil, &cli.Help{App: appName, Usage: []string{"<command>"}})
			assert.NoError(t, err, "Help writes the help")
			assert.Equal(t, out.String(), "Usage: srv <command>\n", "Help writes the usage alone")
		})

		t.Run("returns an error that wraps the error of the writer", func(t *testing.T) {
			t.Parallel()
			err := cli.Plain{}.Help(failingWriter{}, nil, &cli.Help{App: appName})
			assert.ErrorIs(t, err, errWrite, "Help returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the help: disk full", "the error states the help")
		})
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give *cli.Failure
			want string
		}{
			{
				name: "writes the app, the command, the message and the suggestion",
				give: &cli.Failure{
					App: appName, Command: command, Message: "unknown flag --adr", Suggestion: "--addr", Usage: true,
				},
				want: "srv: serve: unknown flag --adr (did you mean --addr?)\n",
			},
			{
				name: "writes the message after the app for a failure without a command",
				give: &cli.Failure{App: appName, Message: "main: the store is down"},
				want: "srv: main: the store is down\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var out bytes.Buffer
				assert.NoError(t, cli.Plain{}.Error(&out, nil, tt.give), "Error writes the failure")
				assert.Equal(t, out.String(), tt.want, "Error writes one line")
			})
		}

		t.Run("returns an error that wraps the error of the writer", func(t *testing.T) {
			t.Parallel()
			err := cli.Plain{}.Error(failingWriter{}, nil, &cli.Failure{App: appName})
			assert.ErrorIs(t, err, errWrite, "Error returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the failure: disk full", "the error states the failure")
		})
	})

	t.Run("Debug", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the rows in columns below a header", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := cli.Plain{}.Debug(&out, nil, []cli.DebugRow{
				{Option: "--addr", Value: `":80"`, Source: cli.Source{Name: "--addr", Kind: cli.Flag}},
				{Option: "--port", Value: "8080"},
			})
			assert.NoError(t, err, "Debug writes the table")
			assert.Equal(t, out.String(),
				"  OPTION  VALUE  SOURCE\n  --addr  \":80\"  flag --addr\n  --port  8080   default\n",
				"Debug writes the option, the value and the source of each row")
		})

		t.Run("returns an error that wraps the error of the writer", func(t *testing.T) {
			t.Parallel()
			err := cli.Plain{}.Debug(failingWriter{}, nil, nil)
			assert.ErrorIs(t, err, errWrite, "Debug returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the output of --flag-debug: disk full",
				"the error states the flag")
		})
	})
}

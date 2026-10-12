// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// executeContract is the contract of the property that FuzzExecute fuzzes and TestApp checks.
const executeContract = "Execute returns nil or an *Error for any command line of a valid app"

// errHandler is the error of a handler that fails.
var errHandler = errors.New("main: the store is down")

// contextKey is the key of the value that a test puts into the context of Execute.
type contextKey struct{}

// The options of the app in the tests of App.
type (
	appGlobal struct {
		Config  cli.ConfigFile `short:"c"`
		Verbose bool           `short:"v"`
	}
	appOptions struct {
		Labels map[string]int
		Target string `arg:"" default:"head"`
		Tags   []string
		Port   int `                      short:"p"`
	}
	filesGlobal struct {
		Files []string `arg:""`
	}
)

// executeBody runs the app of fuzzApp with a command line of tokens that the app declares and of
// other text, and checks that Execute returns nil or an *Error.
func executeBody(c *prop.Case) {
	token := prop.OneOf(
		prop.SampledFrom(command, "--port", "-p", "-vp8", "--tags", "--labels", "a=1", "--", "-", "-h", "--help",
			"-c", "srv.json", "--port=x", "--verbose=maybe", "--flag-debug"),
		prop.String(prop.MaxSize(6)),
	)
	args := c.Draw(prop.List(token, prop.MaxSize(8)), "args")
	app := cli.New[appGlobal](appName, appHelp)
	app.Command(command, commandHelp, func(context.Context, *cli.Run[appGlobal, appOptions]) error { return nil })
	var out bytes.Buffer
	files := fstest.MapFS{"srv.json": {Data: []byte(`{"serve": {"port": 7, "tags": ["a"]}}`)}}
	err := app.Execute(c.Context(), &cli.Env{Open: files.Open, Stdout: &out, Stderr: &out, Args: args})
	if err != nil {
		_ = assert.ErrorAs[*cli.Error](c, err, "Execute returns an *Error for an invalid command line")
	}
}

// FuzzExecute fuzzes the command line of a valid app.
func FuzzExecute(f *testing.F) {
	prop.Fuzz(f, executeContract, executeBody)
}

// TestApp checks the app that New returns, the registration of a command, a run, and the exit
// status of Main.
func TestApp(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			givePath string
			giveData string
			want     int
		}{
			{
				name:     "registers JSON for the extension .json",
				givePath: "srv.json",
				giveData: `{"serve": {"port": 7}}`,
				want:     7,
			},
			{
				name:     "registers INI for the extension .ini",
				givePath: "srv.ini",
				giveData: "[serve]\nport = 8\n",
				want:     8,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[appGlobal, appOptions](t)
				files := fstest.MapFS{tt.givePath: {Data: []byte(tt.giveData)}}
				res := execute(t, app, []string{"-c", tt.givePath, command}, nil, files)
				assert.NoError(t, res.err, "Execute reads the config file")
				assert.Equal(t, got.Config().Port, tt.want, "the run reads the key of the file")
			})
		}
	})

	t.Run("Command", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the handler of the command of the command line", func(t *testing.T) {
			t.Parallel()
			var ran []string
			app := cli.New[appGlobal](appName, appHelp)
			for _, name := range []string{"serve", "migrate"} {
				app.Command(name, "", func(context.Context, *cli.Run[appGlobal, appOptions]) error {
					ran = append(ran, name)
					return nil
				})
			}
			res := execute(t, app, []string{"migrate"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, ran, []string{"migrate"}, "Execute runs the handler of migrate alone")
		})

		t.Run("passes the context of Execute to the handler", func(t *testing.T) {
			t.Parallel()
			var got any
			app := cli.New[appGlobal](appName, appHelp)
			app.Command(command, commandHelp, func(ctx context.Context, _ *cli.Run[appGlobal, appOptions]) error {
				got = ctx.Value(contextKey{})
				return nil
			})
			var out bytes.Buffer
			ctx := context.WithValue(t.Context(), contextKey{}, command)
			err := app.Execute(ctx, &cli.Env{Stdout: &out, Stderr: &out, Args: []string{command}})
			assert.NoError(t, err, "Execute runs the command")
			assert.Equal(t, got, any(command), "the handler reads the value of the context of Execute")
		})

		t.Run("returns the error of the handler unchanged", func(t *testing.T) {
			t.Parallel()
			app := cli.New[appGlobal](appName, appHelp)
			app.Command(command, commandHelp, func(context.Context, *cli.Run[appGlobal, appOptions]) error {
				return errHandler
			})
			res := execute(t, app, []string{command}, nil, nil)
			assert.Equal(t, res.err, errHandler, "Execute returns the error of the handler", assert.ByIdentity())
		})

		t.Run("returns the command that it adds", func(t *testing.T) {
			t.Parallel()
			ran := false
			app := cli.New[appGlobal](appName, appHelp)
			cmd := app.Command[struct{}](parent, parentHelp, nil)
			cmd.Command(subcommand, subcommandHelp, func(context.Context, *cli.Run[appGlobal, struct{}]) error {
				ran = true
				return nil
			})
			res := execute(t, app, []string{parent, subcommand}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.True(t, ran, "the subcommand of the returned command runs")
		})

		t.Run("returns ErrDeclaration for a nil handler of a command without subcommands", func(t *testing.T) {
			t.Parallel()
			app := cli.New[appGlobal](appName, appHelp)
			app.Command[appOptions](command, commandHelp, nil)
			res := execute(t, app, []string{command}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, res.err.Error(),
				"cli: invalid declaration: the command serve has no handler and no subcommands",
				"the error states the command")
		})
	})

	t.Run("Handle", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the handler of the app for a command line without a command", func(t *testing.T) {
			t.Parallel()
			var got *cli.Run[appGlobal, appGlobal]
			app := cli.New[appGlobal](appName, appHelp)
			app.Handle(func(_ context.Context, run *cli.Run[appGlobal, appGlobal]) error {
				got = run
				return nil
			})
			res := execute(t, app, []string{"-v"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the handler of the app")
			assert.True(t, got.Global().Verbose, "the handler receives the global options")
		})

		t.Run("passes the global options as the options of the run", func(t *testing.T) {
			t.Parallel()
			var got *cli.Run[appGlobal, appGlobal]
			app := cli.New[appGlobal](appName, appHelp)
			app.Handle(func(_ context.Context, run *cli.Run[appGlobal, appGlobal]) error {
				got = run
				return nil
			})
			res := execute(t, app, nil, nil, nil)
			assert.NoError(t, res.err, "Execute runs the handler of the app")
			assert.Equal(t, got.Config(), got.Global(), "Config returns the global options", assert.ByIdentity())
		})

		t.Run("reads the positional arguments of the global options of an app without commands", func(t *testing.T) {
			t.Parallel()
			var got []string
			app := cli.New[filesGlobal](appName, appHelp)
			app.Handle(func(_ context.Context, run *cli.Run[filesGlobal, filesGlobal]) error {
				got = run.Global().Files
				return nil
			})
			res := execute(t, app, []string{"a.txt", "b.txt"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the handler of the app")
			assert.Equal(t, got, []string{"a.txt", "b.txt"}, "the handler receives the positional arguments")
		})

		t.Run("runs a command of an app with a handler", func(t *testing.T) {
			t.Parallel()
			ranApp := false
			app, got := serving[appGlobal, appOptions](t)
			app.Handle(func(context.Context, *cli.Run[appGlobal, appGlobal]) error {
				ranApp = true
				return nil
			})
			res := execute(t, app, []string{command, "-p", "8"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Port, 8, "the command receives its options")
			assert.False(t, ranApp, "the handler of the app does not run")
		})

		t.Run("returns ErrUnknownCommand for an unknown command of an app with a handler", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			app.Handle(func(context.Context, *cli.Run[appGlobal, appGlobal]) error { return nil })
			res := execute(t, app, []string{"serv"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrUnknownCommand, "Execute returns ErrUnknownCommand")
		})

		t.Run("replaces the handler of an earlier call", func(t *testing.T) {
			t.Parallel()
			var ran []string
			app := cli.New[appGlobal](appName, appHelp)
			for _, name := range []string{"first", "second"} {
				app.Handle(func(context.Context, *cli.Run[appGlobal, appGlobal]) error {
					ran = append(ran, name)
					return nil
				})
			}
			res := execute(t, app, nil, nil, nil)
			assert.NoError(t, res.err, "Execute runs the handler of the app")
			assert.Equal(t, ran, []string{"second"}, "the handler of the later call runs alone")
		})

		t.Run("returns ErrDeclaration for a nil handler", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			app.Handle(nil)
			res := execute(t, app, []string{command}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: the app srv has a nil handler",
				"the error states the app")
		})
	})

	t.Run("Use", func(t *testing.T) {
		t.Parallel()

		t.Run("calls the hooks of the app around the handler of a command", func(t *testing.T) {
			t.Parallel()
			var events []string
			app := cli.New[appGlobal](appName, appHelp)
			app.Command(command, commandHelp, func(context.Context, *cli.Run[appGlobal, appOptions]) error {
				events = append(events, "handler")
				return nil
			})
			app.Use(
				func(ctx context.Context, _ *cli.Run[appGlobal, appGlobal], next func(context.Context) error) error {
					events = append(events, "start")
					err := next(ctx)
					events = append(events, "end")
					return err
				},
			)
			res := execute(t, app, []string{command}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, events, []string{"start", "handler", "end"}, "the hook runs around the handler")
		})

		t.Run("passes the global options to a hook as the options of its command", func(t *testing.T) {
			t.Parallel()
			var got *cli.Run[appGlobal, appGlobal]
			app, _ := serving[appGlobal, appOptions](t)
			app.Use(
				func(ctx context.Context, run *cli.Run[appGlobal, appGlobal], next func(context.Context) error) error {
					got = run
					return next(ctx)
				},
			)
			res := execute(t, app, []string{command, "-v"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.True(t, got.Config().Verbose, "the hook reads the global options")
			assert.Equal(t, got.Config(), got.Global(), "Config returns the global options", assert.ByIdentity())
		})

		t.Run("calls the hooks of the app around the handler of the app", func(t *testing.T) {
			t.Parallel()
			var events []string
			app := cli.New[appGlobal](appName, appHelp)
			app.Handle(func(context.Context, *cli.Run[appGlobal, appGlobal]) error {
				events = append(events, "handler")
				return nil
			})
			app.Use(
				func(ctx context.Context, _ *cli.Run[appGlobal, appGlobal], next func(context.Context) error) error {
					events = append(events, "hook")
					return next(ctx)
				},
			)
			res := execute(t, app, nil, nil, nil)
			assert.NoError(t, res.err, "Execute runs the handler of the app")
			assert.Equal(t, events, []string{"hook", "handler"}, "the hook runs before the handler")
		})

		t.Run("returns ErrDeclaration for a nil hook", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			app.Use(nil)
			res := execute(t, app, []string{command}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: the app srv has a nil hook",
				"the error states the app")
		})
	})

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil or an *Error for any command line of a valid app", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, executeContract, executeBody)
		})

		t.Run("writes the help to Stdout for the flag of the help", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute returns nil")
			assert.HasPrefix(t, res.stdout, "Usage: srv <command> [options]\n", "Stdout receives the help")
			assert.Empty(t, res.stderr, "Stderr receives nothing")
		})

		t.Run("runs no handler for the flag of the help", func(t *testing.T) {
			t.Parallel()
			ran := false
			app := cli.New[appGlobal](appName, appHelp)
			app.Command(command, commandHelp, func(context.Context, *cli.Run[appGlobal, appOptions]) error {
				ran = true
				return nil
			})
			res := execute(t, app, []string{command, "--help"}, nil, nil)
			assert.NoError(t, res.err, "Execute returns nil")
			assert.False(t, ran, "Execute runs no handler")
		})

		t.Run("writes the help to Stderr for a command line without a command", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			res := execute(t, app, []string{"-v"}, nil, nil)
			assert.HasPrefix(t, res.stderr, "Usage: srv <command> [options]\n", "Stderr receives the help")
			assert.Empty(t, res.stdout, "Stdout receives nothing")
		})

		t.Run("writes the help of a command without a handler to Stderr for a path that ends at it",
			func(t *testing.T) {
				t.Parallel()
				app := cli.New[appGlobal](appName, appHelp)
				cache := app.Command[struct{}]("cache", "", nil)
				cache.Command("purge", "", func(context.Context, *cli.Run[appGlobal, struct{}]) error { return nil })
				res := execute(t, app, []string{"cache"}, nil, nil)
				assert.HasPrefix(t, res.stderr, "Usage: srv cache <command> [options]\n",
					"Stderr receives the help of the command")
			})

		t.Run("returns ErrMissing with the path of a command without a handler", func(t *testing.T) {
			t.Parallel()
			app := cli.New[appGlobal](appName, appHelp)
			cache := app.Command[struct{}]("cache", "", nil)
			cache.Command("purge", "", func(context.Context, *cli.Run[appGlobal, struct{}]) error { return nil })
			res := execute(t, app, []string{"cache"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrMissing, "Execute returns ErrMissing")
			assert.Equal(t, res.err.Error(), "cli: cache: missing command", "the error states the command")
		})

		t.Run("writes a blank line after the help of a command line without a command", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			res := execute(t, app, nil, nil, nil)
			assert.HasSuffix(t, res.stderr, "\n\n", "a blank line follows the help")
		})

		t.Run("returns ErrMissing for a command line without a command", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			res := execute(t, app, nil, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrMissing, "Execute returns ErrMissing")
			assert.Equal(t, res.err.Error(), "cli: missing command", "the error states the missing command")
		})

		t.Run("returns the error of Stderr when the blank line after the help fails to write", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			app.Render(&recorder{})
			var stdout bytes.Buffer
			err := app.Execute(t.Context(), &cli.Env{Stdout: &stdout, Stderr: failingWriter{}})
			assert.ErrorIs(t, err, errWrite, "Execute returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the help: disk full", "Execute wraps the error in a message")
		})

		t.Run("returns the error of Stdout when the help fails to write", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			var stderr bytes.Buffer
			err := app.Execute(t.Context(), &cli.Env{Stdout: failingWriter{}, Stderr: &stderr, Args: []string{"-h"}})
			assert.ErrorIs(t, err, errWrite, "Execute returns the error of the writer")
		})

		t.Run("returns the error of Stderr when the help of a command line without a command fails to write",
			func(t *testing.T) {
				t.Parallel()
				app, _ := serving[appGlobal, appOptions](t)
				var stdout bytes.Buffer
				err := app.Execute(t.Context(), &cli.Env{Stdout: &stdout, Stderr: failingWriter{}})
				assert.ErrorIs(t, err, errWrite, "Execute returns the error of the writer")
			})

		t.Run("returns ErrDeclaration before it reads the command line", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			app.Command("migrate", "", func(context.Context, *cli.Run[appGlobal, chan int]) error { return nil })
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Empty(t, res.stdout, "Execute writes no help")
		})

		t.Run("returns the options of its own command line to each of concurrent runs", func(t *testing.T) {
			t.Parallel()
			app := cli.New[appGlobal](appName, appHelp)
			app.Command(command, commandHelp, func(_ context.Context, run *cli.Run[appGlobal, appOptions]) error {
				_, err := run.Stdout().Write([]byte(strconv.Itoa(run.Config().Port)))
				return err
			})
			h := history.New()
			outcomes := history.Concurrently(4, time.Minute, func(client int) (any, error) {
				for i := range 25 {
					port := 100*client + i
					call := h.Invoke(client, "execute", []any{port})
					var out bytes.Buffer
					args := []string{command, "--port", strconv.Itoa(port)}
					if err := app.Execute(t.Context(), &cli.Env{Stdout: &out, Stderr: &out, Args: args}); err != nil {
						call.OK(err.Error())
						continue
					}
					call.OK(out.String())
				}
				return client, nil
			})
			for _, o := range outcomes {
				assert.True(t, o.Finished, "each client finishes its runs")
			}
			history.Linearizable(t, h, history.Spec[int]{
				Initial: func() int { return 0 },
				Next: func(state int, op history.Operation) []int {
					port, _ := op.Args[0].(int)
					if op.Returned(strconv.Itoa(port)) {
						return []int{state + 1}
					}
					return nil
				},
			}, "each run returns the port of its own command line")
		})
	})

	t.Run("Main", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ExitOK for a run without an error", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			var out bytes.Buffer
			status := app.Main(t.Context(), &cli.Env{Stdout: &out, Stderr: &out, Args: []string{command}})
			assert.Equal(t, status, cli.ExitOK, "Main returns ExitOK")
			assert.Empty(t, out.String(), "Main writes nothing")
		})

		t.Run("returns ExitUsage for an invalid command line", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			var stdout, stderr bytes.Buffer
			args := []string{command, "--prot", "8"}
			status := app.Main(t.Context(), &cli.Env{Stdout: &stdout, Stderr: &stderr, Args: args})
			assert.Equal(t, status, cli.ExitUsage, "Main returns ExitUsage")
			assert.Equal(t, stderr.String(), "srv: serve: unknown flag --prot (did you mean --port?)\n",
				"Main writes the message after the name of the app, without the package name")
		})

		t.Run("returns ExitFailure for an error of the handler", func(t *testing.T) {
			t.Parallel()
			app := cli.New[appGlobal](appName, appHelp)
			app.Command(command, commandHelp, func(context.Context, *cli.Run[appGlobal, appOptions]) error {
				return errHandler
			})
			var stdout, stderr bytes.Buffer
			status := app.Main(t.Context(), &cli.Env{Stdout: &stdout, Stderr: &stderr, Args: []string{command}})
			assert.Equal(t, status, cli.ExitFailure, "Main returns ExitFailure")
			assert.Equal(
				t,
				stderr.String(),
				"srv: main: the store is down\n",
				"Main writes the error after the name of the app",
			)
		})

		t.Run("writes the whole message of an error of the handler that wraps an *Error", func(t *testing.T) {
			t.Parallel()
			app := cli.New[appGlobal](appName, appHelp)
			app.Command(command, commandHelp, func(context.Context, *cli.Run[appGlobal, appOptions]) error {
				return errors.Join(errHandler, &cli.Error{Err: cli.ErrMissing, Command: "migrate"})
			})
			var stdout, stderr bytes.Buffer
			status := app.Main(t.Context(), &cli.Env{Stdout: &stdout, Stderr: &stderr, Args: []string{command}})
			assert.Equal(t, status, cli.ExitUsage, "Main returns ExitUsage")
			assert.Equal(t, stderr.String(), "srv: main: the store is down\ncli: migrate: \n",
				"Main writes the message of the whole error")
		})

		t.Run("returns ExitFailure for a nil Env", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			assert.Equal(t, app.Main(t.Context(), nil), cli.ExitFailure, "Main returns ExitFailure")
		})

		t.Run("writes no message to an Env without Stderr", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			var stdout bytes.Buffer
			status := app.Main(t.Context(), &cli.Env{Stdout: &stdout, Args: []string{command}})
			assert.Equal(t, status, cli.ExitFailure, "Main returns ExitFailure")
			assert.Empty(t, stdout.String(), "Main writes nothing")
		})
	})
}

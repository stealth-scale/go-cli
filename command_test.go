// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// hookContract is the contract of the property of the order of the hooks.
const hookContract = "the hooks of a run start in the order of the path and end in the reverse order"

// Store contains the options of the command db in the tests of commands, and the connection that a
// hook of db opens. Its type is exported, so the options of a subcommand embed a pointer to it.
type Store struct {
	DSN  string `help:"Data source name of the database." short:"d"`
	conn string // the connection of a hook of db
}

// The options of the app and of the subcommand migrate in the tests of commands.
type (
	commandGlobal struct {
		Verbose bool `short:"v"`
	}
	stepsOptions struct {
		*Store
		Steps int `help:"Number of migrations to apply."`
	}
)

// hookOrder registers generated numbers of hooks on the app and on each command of a path of a
// generated depth, and runs the last command of the path. It checks that the hooks start in the
// order of the path and of their registration, and end in the reverse order.
func hookOrder(c *prop.Case) {
	counts := c.Draw(prop.List(prop.Integer(0, 2), prop.MinSize(2), prop.MaxSize(4)), "hooks")
	var events, labels, args []string
	hook := func(label string) cli.Hook[struct{}, struct{}] {
		labels = append(labels, label)
		return func(ctx context.Context, _ *cli.Run[struct{}, struct{}], next func(context.Context) error) error {
			events = append(events, "start "+label)
			err := next(ctx)
			events = append(events, "end "+label)
			return err
		}
	}
	app := cli.New[struct{}](appName, appHelp)
	for i := range counts[0] {
		app.Use(hook("app " + strconv.Itoa(i)))
	}
	var cmd *cli.Command[struct{}, struct{}]
	for level := 1; level < len(counts); level++ {
		name := "c" + strconv.Itoa(level)
		var handle func(context.Context, *cli.Run[struct{}, struct{}]) error
		if level == len(counts)-1 {
			handle = func(context.Context, *cli.Run[struct{}, struct{}]) error {
				events = append(events, "handler")
				return nil
			}
		}
		if cmd == nil {
			cmd = app.Command(name, "", handle)
		} else {
			cmd = cmd.Command(name, "", handle)
		}
		for i := range counts[level] {
			cmd.Use(hook(name + " " + strconv.Itoa(i)))
		}
		args = append(args, name)
	}
	want := make([]string, 0, 2*len(labels)+1)
	for _, label := range labels {
		want = append(want, "start "+label)
	}
	want = append(want, "handler")
	for _, label := range slices.Backward(labels) {
		want = append(want, "end "+label)
	}
	res := execute(c, app, args, nil, nil)
	assert.NoError(c, res.err, "Execute runs the last command of the path")
	assert.Equal(c, events, want, "the hooks start in the order of the path and end in the reverse order")
}

// TestCommand checks the subcommands of a command and the hooks that run around the handler of a
// command.
func TestCommand(t *testing.T) {
	t.Parallel()

	t.Run("Command", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the handler of a subcommand", func(t *testing.T) {
			t.Parallel()
			app, db, migrate := nesting[commandGlobal, Store, stepsOptions](t)
			res := execute(t, app, []string{parent, subcommand, "--steps", "3"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.Equal(t, migrate.Config().Steps, 3, "the handler of the subcommand receives its options")
			assert.Nil(t, db.Config(), "the handler of the command does not run")
		})

		t.Run("runs the handler of a command for a path that ends at the command", func(t *testing.T) {
			t.Parallel()
			app, db, migrate := nesting[commandGlobal, Store, stepsOptions](t)
			res := execute(t, app, []string{parent, "--dsn", "pg"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, db.Config().DSN, "pg", "the handler of the command receives its options")
			assert.Nil(t, migrate.Config(), "the handler of the subcommand does not run")
		})

		t.Run("runs the handler of a subcommand two levels down", func(t *testing.T) {
			t.Parallel()
			var ran []string
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command[struct{}](parent, parentHelp, nil)
			backup := db.Command[struct{}]("backup", "", nil)
			backup.Command("create", "", func(_ context.Context, run *cli.Run[commandGlobal, struct{}]) error {
				ran = append(ran, run.Command())
				return nil
			})
			res := execute(t, app, []string{parent, "backup", "create"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.Equal(t, ran, []string{"db backup create"}, "the handler of the subcommand runs once")
		})
	})

	t.Run("Use", func(t *testing.T) {
		t.Parallel()

		t.Run("calls the hooks of the path from the app down around the handler", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, hookContract, hookOrder)
		})

		t.Run("passes the options of its command to a hook", func(t *testing.T) {
			t.Parallel()
			var got string
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command[Store](parent, parentHelp, nil)
			db.Use(
				func(ctx context.Context, run *cli.Run[commandGlobal, Store], next func(context.Context) error) error {
					got = run.Config().DSN
					return next(ctx)
				},
			)
			db.Command(subcommand, subcommandHelp, func(context.Context, *cli.Run[commandGlobal, struct{}]) error {
				return nil
			})
			res := execute(t, app, []string{parent, subcommand, "--dsn", "pg"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.Equal(t, got, "pg", "the hook reads the option of its command")
		})

		t.Run("passes the path of the command that runs to a hook", func(t *testing.T) {
			t.Parallel()
			var got string
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command[Store](parent, parentHelp, nil)
			db.Use(
				func(ctx context.Context, run *cli.Run[commandGlobal, Store], next func(context.Context) error) error {
					got = run.Command()
					return next(ctx)
				},
			)
			db.Command(subcommand, subcommandHelp, func(context.Context, *cli.Run[commandGlobal, struct{}]) error {
				return nil
			})
			res := execute(t, app, []string{parent, subcommand}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.Equal(t, got, parent+" "+subcommand, "the hook reads the path of the subcommand")
		})

		t.Run("passes a value that a hook sets in the options of its command to a subcommand", func(t *testing.T) {
			t.Parallel()
			var got string
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command[Store](parent, parentHelp, nil)
			db.Use(
				func(ctx context.Context, run *cli.Run[commandGlobal, Store], next func(context.Context) error) error {
					run.Config().conn = "open " + run.Config().DSN
					return next(ctx)
				},
			)
			db.Command(
				subcommand,
				subcommandHelp,
				func(_ context.Context, run *cli.Run[commandGlobal, stepsOptions]) error {
					got = run.Config().conn
					return nil
				},
			)
			res := execute(t, app, []string{parent, "-d", "pg", subcommand}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.Equal(t, got, "open pg", "the subcommand reads the connection that the hook opens")
		})

		t.Run("passes the context of a hook to the handler", func(t *testing.T) {
			t.Parallel()
			var got any
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command(parent, parentHelp, func(ctx context.Context, _ *cli.Run[commandGlobal, Store]) error {
				got = ctx.Value(contextKey{})
				return nil
			})
			db.Use(func(ctx context.Context, _ *cli.Run[commandGlobal, Store], next func(context.Context) error) error {
				return next(context.WithValue(ctx, contextKey{}, parent))
			})
			res := execute(t, app, []string{parent}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got, any(parent), "the handler reads the value of the context of the hook")
		})

		t.Run("ends the run with the result of a hook that does not call next", func(t *testing.T) {
			t.Parallel()
			ran := false
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command(parent, parentHelp, func(context.Context, *cli.Run[commandGlobal, Store]) error {
				ran = true
				return nil
			})
			db.Use(func(context.Context, *cli.Run[commandGlobal, Store], func(context.Context) error) error {
				return errHandler
			})
			res := execute(t, app, []string{parent}, nil, nil)
			assert.Equal(t, res.err, errHandler, "Execute returns the error of the hook", assert.ByIdentity())
			assert.False(t, ran, "the handler does not run")
		})

		t.Run("returns the error of a hook that wraps the error of the handler", func(t *testing.T) {
			t.Parallel()
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command(parent, parentHelp, func(context.Context, *cli.Run[commandGlobal, Store]) error {
				return errHandler
			})
			db.Use(func(ctx context.Context, _ *cli.Run[commandGlobal, Store], next func(context.Context) error) error {
				return fmt.Errorf("main: db: %w", next(ctx))
			})
			res := execute(t, app, []string{parent}, nil, nil)
			assert.ErrorIs(t, res.err, errHandler, "the error wraps the error of the handler")
			assert.Equal(
				t,
				res.err.Error(),
				"main: db: main: the store is down",
				"Execute returns the error of the hook",
			)
		})

		t.Run("runs the code of a hook after next when the handler fails", func(t *testing.T) {
			t.Parallel()
			var events []string
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command(parent, parentHelp, func(context.Context, *cli.Run[commandGlobal, Store]) error {
				events = append(events, "handler")
				return errHandler
			})
			db.Use(func(ctx context.Context, _ *cli.Run[commandGlobal, Store], next func(context.Context) error) error {
				err := next(ctx)
				events = append(events, "after next")
				return err
			})
			res := execute(t, app, []string{parent}, nil, nil)
			assert.Equal(t, res.err, errHandler, "Execute returns the error of the handler", assert.ByIdentity())
			assert.Equal(t, events, []string{"handler", "after next"}, "the hook runs its code after the handler")
		})

		t.Run("runs the remaining hooks and the handler once per call of next", func(t *testing.T) {
			t.Parallel()
			var events []string
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command(parent, parentHelp, func(context.Context, *cli.Run[commandGlobal, Store]) error {
				events = append(events, "handler")
				return nil
			})
			db.Use(func(ctx context.Context, _ *cli.Run[commandGlobal, Store], next func(context.Context) error) error {
				if err := next(ctx); err != nil {
					return err
				}
				return next(ctx)
			})
			db.Use(func(ctx context.Context, _ *cli.Run[commandGlobal, Store], next func(context.Context) error) error {
				events = append(events, "hook")
				return next(ctx)
			})
			res := execute(t, app, []string{parent}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, events, []string{"hook", "handler", "hook", "handler"},
				"each call of next runs the second hook and the handler")
		})

		runs := []struct {
			name     string
			giveArgs []string
		}{
			{name: "does not call a hook for the flag of the help", giveArgs: []string{"cache", "-h"}},
			{name: "does not call a hook for an invalid command line", giveArgs: []string{"cache", "purge", "--port"}},
			{
				name:     "does not call a hook for a path that ends at a command without a handler",
				giveArgs: []string{"cache"},
			},
		}
		for _, tt := range runs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				called := false
				app := cli.New[commandGlobal](appName, appHelp)
				cache := app.Command[struct{}]("cache", "", nil)
				cache.Use(
					func(ctx context.Context, _ *cli.Run[commandGlobal, struct{}], next func(context.Context) error) error {
						called = true
						return next(ctx)
					},
				)
				cache.Command(
					"purge",
					"",
					func(context.Context, *cli.Run[commandGlobal, struct{}]) error { return nil },
				)
				_ = execute(t, app, tt.giveArgs, nil, nil)
				assert.False(t, called, "the run does not call the hook")
			})
		}

		t.Run("returns ErrDeclaration for a nil hook", func(t *testing.T) {
			t.Parallel()
			app := cli.New[commandGlobal](appName, appHelp)
			db := app.Command[struct{}](parent, parentHelp, nil)
			db.Use(nil)
			db.Command(subcommand, subcommandHelp, func(context.Context, *cli.Run[commandGlobal, struct{}]) error {
				return nil
			})
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: the command db has a nil hook",
				"the error states the command")
		})
	})
}

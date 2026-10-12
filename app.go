// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// The exit statuses that [App.Main] returns.
const (
	// ExitOK is the status of a run whose handler succeeds, and of a run that writes the help.
	ExitOK = 0
	// ExitFailure is the status of a run whose handler fails, whose declarations are invalid, whose
	// Env is incomplete, or whose help or --flag-debug fails to write.
	ExitFailure = 1
	// ExitUsage is the status of a run with an invalid command line, environment variable or config
	// file.
	ExitUsage = 2
)

// App is a command-line program with the global options G. The app is the command at the root of
// the tree of its commands, so the global options are its options. [New] returns one, and
// [App.Command] adds each command. [App.Handle] sets the handler of the app, and [App.Use] adds a
// hook that every run calls.
//
// # Concurrency
//
// Execute and Main are safe for concurrent use once the last registration has returned. The
// registrations are the calls of Command, Convert, Format, Handle, Render and Use of the app, and of
// Command and Use of its commands. They are not safe for concurrent use.
type App[G any] struct {
	renderer   Renderer                                        // the renderer of the help, the failures and --flag-debug
	converters map[reflect.Type]converter                      // the conversions of Convert, and the built-in ones
	formats    map[string]Format                               // the formats of config files by their extension in lower case
	handle     func(ctx context.Context, run *Run[G, G]) error // the handler of the app, or nil
	name       string
	help       string
	commands   []*command[G] // the commands in the order of their registration
	hooks      []Hook[G, G]  // the hooks of the app in the order of their registration
	errs       []error       // the invalid registrations, which Execute returns
}

// New returns an app named name, with the help text help. The name starts every environment
// variable of the app and every message that [App.Main] writes. The app reads config files with the
// extension .json as [JSON] and .ini as [INI], and converts url.URL, time.Location, net.IPNet and
// fs.FileMode, which have no text method of their own. It writes its output with [Plain].
func New[G any](name, help string) *App[G] {
	return &App[G]{
		renderer:   Plain{},
		converters: builtins(),
		formats:    map[string]Format{jsonExt: JSON, iniExt: INI},
		name:       name,
		help:       help,
	}
}

// Command adds the command name with the help text help to the app and returns it. The exported
// fields of C are the options of the command. A run of the command calls handle with the options
// after every source is applied, and returns the error of handle unchanged. A command with a nil
// handle runs only its subcommands.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when name is not a name of lower-case
// words, when the app has a command of that name already, when handle is nil for a command without
// subcommands, and when the command has subcommands and declares a positional argument.
func (a *App[G]) Command[C any](
	name, help string,
	handle func(ctx context.Context, run *Run[G, C]) error,
) *Command[G, C] {
	c := newCommand(a, name, name, help, handle)
	a.commands = append(a.commands, c.cmd)
	return c
}

// Handle sets the handler of the app. A command line without a command calls handle with the
// global options as the options of the run. A later call replaces an earlier one.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when handle is nil.
func (a *App[G]) Handle(handle func(ctx context.Context, run *Run[G, G]) error) {
	if handle == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: the app %s has a nil handler", ErrDeclaration, a.name))
		return
	}
	a.handle = handle
}

// Use adds hook to the app, so every run calls hook before the hooks of its commands. The hooks of
// the app run in the order of the calls of Use. The run of a hook of the app contains the global
// options as the options of its command.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when hook is nil.
func (a *App[G]) Use(hook Hook[G, G]) {
	if hook == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: the app %s has a nil hook", ErrDeclaration, a.name))
		return
	}
	a.hooks = append(a.hooks, hook)
}

// Execute runs the command that env.Args selects, and returns the error of its handler. The hooks of
// the app and of each command of the path run around the handler, the app first. The flag -h or
// --help writes the help of the last command of the path, or of the app, to env.Stdout and returns
// nil. A path that ends at a command without a handler writes the help of that command and a blank
// line to env.Stderr. The blank line separates the help from the failure that [App.Main] writes.
// The global flag --flag-debug writes the value and the source of each option to env.Stderr before
// the hooks run. The renderer of the app writes the help and the output of --flag-debug.
//
// Execute checks the options of every command before it reads the command line, so a run of any
// command reports a declaration that the app cannot run.
//
// Error modes:
//   - an error that wraps [ErrEnv] when env is nil, or has no Stdout or no Stderr.
//   - an error that wraps [ErrDeclaration] for a registration or options that the app cannot run.
//   - an [*Error] for an invalid command line, environment variable or config file, and for a path
//     that ends at a command without a handler.
//   - the error of the renderer, unchanged, when the help or the output of --flag-debug fails to
//     write.
//   - an error that wraps the error of env.Stderr when the blank line after the help fails to
//     write.
//   - the error of the handler, unchanged, or the error that a hook returns in its place.
func (a *App[G]) Execute(ctx context.Context, env *Env) error {
	if err := env.check(); err != nil {
		return err
	}
	d, err := a.declare()
	if err != nil {
		return err
	}
	in := &input[G]{cmd: d.root, list: d.root.options}
	if err := in.scan(env.Args); err != nil {
		return in.locate(err)
	}
	if in.help {
		return a.renderer.Help(env.Stdout, env.environ(), a.helpOf(in))
	}
	if in.cmd.handle == nil {
		if err := a.renderer.Help(env.Stderr, env.environ(), a.helpOf(in)); err != nil {
			return err
		}
		if _, err := io.WriteString(env.Stderr, "\n"); err != nil {
			return fmt.Errorf("cli: write the help: %w", err)
		}
		return in.locate(&Error{Err: ErrMissing, text: "missing command"})
	}
	if err := a.resolve(env, d, in); err != nil {
		return in.locate(err)
	}
	if d.debug.value.Bool() {
		if err := a.renderer.Debug(env.Stderr, env.environ(), debugRows(d, in)); err != nil {
			return err
		}
	}
	inv := &invocation[G]{global: d.global, env: env, sources: in.sources(), command: in.cmd.path}
	return in.cmd.around(ctx, inv, func(ctx context.Context) error {
		return in.cmd.handle(ctx, inv)
	})
}

// Main runs [App.Execute] and returns the exit status for os.Exit. The renderer of the app writes
// the [Failure] of an error to env.Stderr. [Plain] writes the name of the app and [Error.Message]
// for an [*Error], such as "srv: serve: unknown flag --adr (did you mean --addr?)". The status is
// [ExitUsage] for an error that wraps an *Error, [ExitFailure] for any other error, and [ExitOK]
// without one. Main does not write the failure to a nil Env or to an Env without Stderr.
func (a *App[G]) Main(ctx context.Context, env *Env) int {
	err := a.Execute(ctx, env)
	if err == nil {
		return ExitOK
	}
	status := ExitFailure
	f := &Failure{Err: err, App: a.name, Message: err.Error()}
	if e, ok := errors.AsType[*Error](err); ok {
		status, f.Usage = ExitUsage, true
		if e.Error() == f.Message {
			f.Command, f.Message, f.Suggestion = e.Command, e.text, e.Suggestion
		}
	}
	if env != nil && env.Stderr != nil {
		// The status reports the error also when the failure fails to write.
		_ = a.renderer.Error(env.Stderr, env.environ(), f)
	}
	return status
}

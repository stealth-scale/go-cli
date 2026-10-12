// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"errors"
	"fmt"
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

// App is a command-line program with the global options G and its commands. [New] returns one, and
// [App.Command] adds each command.
//
// # Concurrency
//
// Execute and Main are safe for concurrent use once the last call of Command, Convert, Format and
// Render has returned. Command, Convert, Format and Render are not safe for concurrent use.
type App[G any] struct {
	renderer   Renderer                   // the renderer of the help, the failures and --flag-debug
	converters map[reflect.Type]converter // the conversions of Convert, and the built-in ones
	formats    map[string]Format          // the formats of config files by their extension in lower case
	name       string
	help       string
	commands   []*command[G] // the commands in the order of their registration
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

// Command adds the command name with the help text help. The exported fields of C are the options
// of the command. A run of the command calls handle with the options after every source is
// applied, and returns the error of handle unchanged.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when handle is nil, when name is not a
// name of lower-case words, and when two commands have one name.
func (a *App[G]) Command[C any](name, help string, handle func(ctx context.Context, run *Run[G, C]) error) {
	if handle == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: the command %s has no handler", ErrDeclaration, name))
		return
	}
	a.commands = append(a.commands, &command[G]{
		name: name,
		help: help,
		bind: func(s scope) ([]*option, handler[G], error) {
			config := new(C)
			s.path = reflect.TypeFor[C]().String()
			list, err := a.options(nil, reflect.ValueOf(config).Elem(), s)
			call := func(ctx context.Context, global *G, env *Env, sources map[any]Source) error {
				return handle(ctx, &Run[G, C]{global: global, config: config, env: env, sources: sources})
			}
			return list, call, err
		},
	})
}

// Execute runs the command that env.Args selects, and returns the error of its handler. The flag -h
// or --help writes the help of the command, or of the app before a command, to env.Stdout and
// returns nil. A command line without a command writes the help of the app to env.Stderr. The
// global flag --flag-debug writes the value and the source of each option to env.Stderr before the
// handler runs. The renderer of the app writes the help and the output of --flag-debug.
//
// Execute checks the options of every command before it reads the command line, so a run of any
// command reports a declaration that the app cannot run.
//
// Error modes:
//   - an error that wraps [ErrEnv] when env is nil, or has no Stdout or no Stderr.
//   - an error that wraps [ErrDeclaration] for a registration or options that the app cannot run.
//   - an [*Error] for an invalid command line, environment variable or config file, and for a
//     command line without a command.
//   - the error of the renderer, unchanged, when the help or the output of --flag-debug fails to
//     write.
//   - the error of the handler, unchanged.
func (a *App[G]) Execute(ctx context.Context, env *Env) error {
	if err := env.check(); err != nil {
		return err
	}
	d, err := a.declare()
	if err != nil {
		return err
	}
	in := &input[G]{list: d.globals}
	if err := in.scan(d, env.Args); err != nil {
		return in.locate(err)
	}
	if in.help {
		return a.renderer.Help(env.Stdout, env.environ(), a.helpOf(d, in))
	}
	if in.cmd == nil {
		if err := a.renderer.Help(env.Stderr, env.environ(), a.helpOf(d, in)); err != nil {
			return err
		}
		return &Error{Err: ErrMissing, text: "missing command"}
	}
	if err := a.resolve(env, d, in); err != nil {
		return in.locate(err)
	}
	if d.debug.value.Bool() {
		if err := a.renderer.Debug(env.Stderr, env.environ(), debugRows(d, in)); err != nil {
			return err
		}
	}
	return in.cmd.call(ctx, d.global, env, in.sources())
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

// handler calls the handler of a command with the global options, the Env and the sources of a run.
type handler[G any] func(ctx context.Context, global *G, env *Env, sources map[any]Source) error

// command is a command of an app with the global options G.
type command[G any] struct {
	// bind returns the options of a new value of the options of the command in the scope s, and the
	// handler of the command over that value.
	bind func(s scope) ([]*option, handler[G], error)
	name string
	help string
}

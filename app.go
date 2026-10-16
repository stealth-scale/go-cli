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
	// ExitOK is the status of a run whose handler succeeds, and of a run that writes the help or the
	// version.
	ExitOK = 0
	// ExitFailure is the status of a run whose handler fails, whose declarations are invalid, whose
	// Env is incomplete, or whose help, version or --flag-debug fails to write.
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
// registrations are the calls of Command, Complete, Convert, Format, Handle, IgnoreKeys,
// LocateConfig, Render, RenderFor, Use and Version of the app, and of Command and Use of its
// commands. They are not safe for concurrent use.
type App[G any] struct {
	renderer   Renderer                                        // the renderer of Render, or nil after RenderFor
	converters conversions                                     // the conversions of Convert, and the built-in ones
	completers map[reflect.Type]completer[G]                   // the functions of Complete by their type
	formats    map[string]Format                               // the formats of config files by their extension in lower case
	handle     func(ctx context.Context, run *Run[G, G]) error // the handler of the app, or nil
	render     func(global *G) Renderer                        // the function of RenderFor, or nil after Render
	locate     func(global *G, env *Env) (string, error)       // the function of LocateConfig, or nil
	name       string
	help       string
	version    string        // the text of Version, or empty for an app without a version
	commands   []*command[G] // the commands in the order of their registration
	hooks      []Hook[G, G]  // the hooks of the app in the order of their registration
	ignored    []string      // the config keys of IgnoreKeys, which the app decodes itself
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
		completers: map[reflect.Type]completer[G]{},
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
// nil. The flag --version of an app with [App.Version] writes the version to env.Stdout and returns
// nil. The command completion writes a completion script or the candidates of a command line to
// env.Stdout, as [App.Complete] describes. A path that ends at a command without a handler writes
// the help of that command and a blank line to env.Stderr. The blank line separates the help from
// the failure that [App.Main] writes. The global flag --flag-debug writes the value and the source
// of each option to env.Stderr before the hooks run. The renderer of the run writes the help and the
// output of --flag-debug. [App.RenderFor] describes how a run selects its renderer.
//
// Execute checks the options of every command before it reads the command line, so a run of any
// command reports a declaration that the app cannot run. A run applies every valid value of its
// sources and returns the first error in the order of the sources.
//
// Error modes:
//   - an error that wraps [ErrEnv] when env is nil, or has no Stdout or no Stderr.
//   - an error that wraps [ErrDeclaration] for a registration or options that the app cannot run.
//   - an [*Error] for an invalid command line, environment variable or config file, and for a path
//     that ends at a command without a handler.
//   - the error of the renderer, unchanged, when the help, the version or the output of
//     --flag-debug fails to write.
//   - an error that wraps the error of env.Stderr when the blank line after the help fails to
//     write.
//   - an error that wraps the error of env.Stdout when the command completion fails to write.
//   - the error of a function of [App.Complete], unchanged.
//   - the error of the handler, unchanged, or the error that a hook returns in its place.
func (a *App[G]) Execute(ctx context.Context, env *Env) error {
	_, err := a.ExecuteRenderer(ctx, env)
	return err
}

// ExecuteRenderer runs the command line of env as [App.Execute] describes, and also returns the
// renderer of the run. The renderer has written the help and the table of --flag-debug of the run.
// A caller that writes the failure of the run itself passes [App.Failure] of the error to the method
// Error of the renderer, as [App.Main] does. ExecuteRenderer returns the renderer of every run,
// also of a run whose Env is incomplete or whose declarations are invalid.
//
// A run of an app with [App.RenderFor] applies its sources also when the command line does not
// scan, has a flag of the help or of the version, or ends at a command without a handler, so the
// renderer of such a run comes from the valid values of its sources.
//
// Error modes:
//   - the error modes of [App.Execute].
func (a *App[G]) ExecuteRenderer(ctx context.Context, env *Env) (Renderer, error) {
	if err := env.check(); err != nil {
		return a.rendererOf(new(G)), err
	}
	d, err := a.declare()
	if err != nil {
		return a.rendererOf(new(G)), err
	}
	in := &input[G]{cmd: d.root, list: d.root.options, versioned: a.version != ""}
	err = in.scan(env.Args)
	completing := in.cmd.path == completionName
	switch {
	case err == nil && !in.help && !in.version && completing:
		// The command completion checks its own arguments alone, so the options of the app do not
		// fail it.
		err = a.resolve(env, d, &input[G]{cmd: in.cmd, list: in.cmd.options, args: in.args})
	case err == nil && !in.help && !in.version && in.cmd.handle != nil:
		err = a.resolve(env, d, in)
	case a.render != nil:
		// Like a run of an app with Render, this run reports the error of its scan, its help, its
		// version or its missing command. The errors of this resolution come after those, so the run
		// ignores them.
		_ = a.resolve(env, d, in)
	}
	r := a.rendererOf(d.global)
	switch {
	case err != nil:
		return r, in.locate(err)
	case in.help:
		return r, r.Help(env.Stdout, env.environ(), a.helpOf(in))
	case in.version:
		return r, r.Version(env.Stdout, env.environ(), &Version{App: a.name, Text: a.version})
	case completing:
		return r, in.cmd.handle(ctx, &invocation[G]{global: d.global, env: env, command: in.cmd.path})
	case in.cmd.handle == nil:
		if err := r.Help(env.Stderr, env.environ(), a.helpOf(in)); err != nil {
			return r, err
		}
		if _, err := io.WriteString(env.Stderr, "\n"); err != nil {
			return r, fmt.Errorf("cli: write the help: %w", err)
		}
		return r, in.locate(&Error{Err: ErrMissing, text: "missing command"})
	}
	if d.debug.value.Bool() {
		if err := r.Debug(env.Stderr, env.environ(), debugRows(d, in)); err != nil {
			return r, err
		}
	}
	inv := &invocation[G]{global: d.global, env: env, sources: in.sources(), command: in.cmd.path}
	return r, in.cmd.around(ctx, inv, func(ctx context.Context) error {
		return in.cmd.handle(ctx, inv)
	})
}

// Main runs [App.ExecuteRenderer] and returns the exit status for os.Exit. The renderer of the run
// writes the failure that [App.Failure] returns for an error to env.Stderr. [Plain] writes the name
// of the app and [Error.Message] for an [*Error], such as
// "srv: serve: unknown flag --adr (did you mean --addr?)". The status is [ExitUsage] for an error
// that wraps an *Error, [ExitFailure] for any other error, and [ExitOK] without one. Main does not
// write the failure to a nil Env or to an Env without Stderr.
func (a *App[G]) Main(ctx context.Context, env *Env) int {
	r, err := a.ExecuteRenderer(ctx, env)
	f := a.Failure(err)
	if f == nil {
		return ExitOK
	}
	if env != nil && env.Stderr != nil {
		// The status reports the error also when the failure fails to write.
		_ = r.Error(env.Stderr, env.environ(), f)
	}
	if f.Usage {
		return ExitUsage
	}
	return ExitFailure
}

// Failure returns the failure that [App.Main] writes for the error err of a run, or nil for a nil
// error. Usage is true for an error that wraps an [*Error]. For an *Error itself, Command, Message
// and Suggestion come from the *Error. For any other error, Message is the text of the error.
func (a *App[G]) Failure(err error) *Failure {
	if err == nil {
		return nil
	}
	f := &Failure{Err: err, App: a.name, Message: err.Error()}
	if e, ok := errors.AsType[*Error](err); ok {
		f.Usage = true
		if e.Error() == f.Message {
			f.Command, f.Message, f.Suggestion = e.Command, e.text, e.Suggestion
		}
	}
	return f
}

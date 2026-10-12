// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"fmt"
	"reflect"
)

// Hook is a function that a run calls before its handler. run contains the options of the command
// that uses the hook. The hook calls next to run the remaining hooks and the handler. It returns
// the error of next, an error that wraps it, or an error of its own. A hook that does not call
// next ends the run with its own result. Each call of next runs the remaining hooks and the handler
// once.
//
// # Concurrency
//
// A run calls its hooks on the goroutine of [App.Execute]. Concurrent runs of one app call each of
// its hooks concurrently, so a hook guards the state that the runs share.
type Hook[G, C any] func(ctx context.Context, run *Run[G, C], next func(ctx context.Context) error) error

// command is a command in the tree of an app with the global options G. A run reads a command and
// does not change it, so concurrent runs share it.
type command[G any] struct {
	// bind returns the binding of a new value of the options of the command in the scope s, and sets
	// the Go path of s to the type of the options.
	bind     func(s *scope) (*binding[G], error)
	name     string
	path     string // the names of the path of the command joined with spaces, such as db migrate
	help     string
	children []*command[G] // the subcommands in the order of their registration
}

// Command is a command of an app with the global options G and the options C. A command has a
// handler, subcommands or both. Its options and its hooks apply to it and to its descendants.
// [App.Command] and Command.Command return one.
//
// # Concurrency
//
// Command and Use are not safe for concurrent use, as the registrations of an [App] are not.
type Command[G, C any] struct {
	app   *App[G]      // the app, which records the invalid registrations
	cmd   *command[G]  // the command in the tree of the app
	hooks []Hook[G, C] // the hooks of the command in the order of their registration
}

// newCommand returns the command name of the app a at the path path, with the help text help and
// the handler handle. The bind of the command reads the hooks of the command when a run declares
// the tree, so every call of Use before a run applies to it.
func newCommand[G, C any](
	a *App[G],
	path, name, help string,
	handle func(ctx context.Context, run *Run[G, C]) error,
) *Command[G, C] {
	c := &Command[G, C]{app: a}
	c.cmd = &command[G]{name: name, path: path, help: help, bind: func(s *scope) (*binding[G], error) {
		config := new(C)
		s.path = reflect.TypeFor[C]().String()
		opts, err := a.options(nil, reflect.ValueOf(config).Elem(), s)
		b := newBinding(config, c.hooks, handle)
		b.options = opts
		return b, err
	}}
	return c
}

// Command adds the subcommand name with the help text help to c and returns it. A subcommand follows
// the rules of [App.Command].
func (c *Command[G, C]) Command[D any](
	name, help string,
	handle func(ctx context.Context, run *Run[G, D]) error,
) *Command[G, D] {
	sub := newCommand(c.app, c.cmd.path+" "+name, name, help, handle)
	c.cmd.children = append(c.cmd.children, sub.cmd)
	return sub
}

// Use adds hook to c, so a run of c or of a descendant of c calls hook after the hooks of the
// ancestors of c. The hooks of c run in the order of the calls of Use.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when hook is nil.
func (c *Command[G, C]) Use(hook Hook[G, C]) {
	if hook == nil {
		c.app.errs = append(c.app.errs, fmt.Errorf("%w: the command %s has a nil hook", ErrDeclaration, c.cmd.path))
		return
	}
	c.hooks = append(c.hooks, hook)
}

// chain calls the first hook of hooks with run and a next that calls the remaining hooks. It calls
// next after the last hook, and calls next alone when hooks is empty. It returns the error of the
// first hook, or of next when hooks is empty.
func chain[G, C any](ctx context.Context, run *Run[G, C], hooks []Hook[G, C], next func(context.Context) error) error {
	if len(hooks) == 0 {
		return next(ctx)
	}
	return hooks[0](ctx, run, func(ctx context.Context) error {
		return chain(ctx, run, hooks[1:], next)
	})
}

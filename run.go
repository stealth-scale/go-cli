// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"io"
	"strings"
)

// Run is one call of a handler or of a hook with the global options G and the options C of its
// command. It returns the options after every source is applied, the standard streams of the run,
// the source of each option and the path of the command of the run.
//
// # Concurrency
//
// The methods of a Run are safe for concurrent use. The options that Global and Config return are
// the handler's own, so a handler that starts goroutines guards what they change.
type Run[G, C any] struct {
	config *C
	invocation[G]
}

// invocation is the part of a run that the hooks and the handler of every command of its path
// share. It contains the global options, the Env, the source of each option and the path of the
// command of the run.
type invocation[G any] struct {
	global  *G
	env     *Env
	sources map[any]Source // the source of each option, by a pointer to its field
	command string         // the path of the command of the run, or empty for the handler of the app
}

// NewRun returns a run of global and config with the streams of env, for a test that calls a
// handler without [App.Execute]. Its options have no source.
//
// Error modes:
//   - an error that wraps [ErrEnv] when env is nil, or has no Stdout or no Stderr.
//   - an error that wraps [ErrDeclaration] when global or config is nil.
func NewRun[G, C any](global *G, config *C, env *Env) (*Run[G, C], error) {
	if err := env.check(); err != nil {
		return nil, err
	}
	if global == nil || config == nil {
		return nil, fmt.Errorf("%w: a run needs the global options and the options of its command", ErrDeclaration)
	}
	return &Run[G, C]{global: global, env: env, config: config}, nil
}

// Global returns the global options of the run.
func (r *Run[G, C]) Global() *G {
	return r.global
}

// Config returns the options of the command of the run.
func (r *Run[G, C]) Config() *C {
	return r.config
}

// Stdin returns the standard input of the run. The input of an [Env] without Stdin is empty.
func (r *Run[G, C]) Stdin() io.Reader {
	if r.env.Stdin == nil {
		return strings.NewReader("")
	}
	return r.env.Stdin
}

// Stdout returns the standard output of the run.
func (r *Run[G, C]) Stdout() io.Writer {
	return r.env.Stdout
}

// Stderr returns the standard error of the run.
func (r *Run[G, C]) Stderr() io.Writer {
	return r.env.Stderr
}

// Environ returns the environment of the run as KEY=value pairs, or nil for an [Env] without
// Environ. A handler passes it to a library that reads the environment, such as a terminal library
// that chooses its colors.
func (r *Run[G, C]) Environ() []string {
	return r.env.environ()
}

// Source returns the source of the option at field, a pointer to the field of an option of the app
// or of a command of the path of the run. It reports false for any other pointer, and for every
// option of a run that [NewRun] returns.
func (r *Run[G, C]) Source(field any) (Source, bool) {
	s, ok := r.sources[field]
	return s, ok
}

// Command returns the path of the command that the run calls, such as "db migrate". It returns the
// empty string in a run of the handler of the app and in a run that [NewRun] returns.
func (r *Run[G, C]) Command() string {
	return r.command
}

// Open opens the input file f of the run for reading. For the path -, Open returns the standard
// input of the run, whose Close does nothing. For any other path, Open returns the file that
// [Env.Open] opens. The caller closes the file. The standard input reads once, so a second open of
// - in one run reads what the first read left.
//
// Error modes:
//   - the error of Env.Open, unchanged.
//   - an *fs.PathError that wraps fs.ErrNotExist for an Env without Open.
func (r *Run[G, C]) Open(f InputFile) (io.ReadCloser, error) {
	if f == stdinPath {
		return io.NopCloser(r.Stdin()), nil
	}
	return r.env.open(string(f))
}

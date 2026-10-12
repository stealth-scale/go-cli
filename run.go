// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"io"
	"strings"
)

// Run is one call of a handler with the global options G and the options C of its command. It
// returns the options after every source is applied, the standard streams of the run and the source
// of each option.
//
// # Concurrency
//
// The methods of a Run are safe for concurrent use. The options that Global and Config return are
// the handler's own, so a handler that starts goroutines guards what they change.
type Run[G, C any] struct {
	global  *G
	config  *C
	env     *Env
	sources map[any]Source // the source of each option, by a pointer to its field
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
	return &Run[G, C]{global: global, config: config, env: env}, nil
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

// Source returns the source of the option at field, a pointer to a field of [Run.Global] or
// [Run.Config]. It reports false for any other pointer, and for every option of a run that
// [NewRun] returns.
func (r *Run[G, C]) Source(field any) (Source, bool) {
	s, ok := r.sources[field]
	return s, ok
}

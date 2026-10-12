// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
)

// Env is the process environment of a run: its arguments, its environment variables, its files and
// its standard streams. A program passes [OS], and a test builds one.
//
// Stdout and Stderr are required. The package reads the environment variables of its options
// through Lookup. Environ returns the environment as KEY=value pairs for a [Renderer] and for the
// libraries that a handler calls. A nil Lookup does not report any environment variable, and
// without Environ a Renderer and [Run.Environ] receive nil. A nil ReadFile does not read a file, as
// if no file existed. A nil Stdin reads as empty.
//
// # Concurrency
//
// A run calls the functions of its Env on the goroutine of [App.Execute]. Runs that share an Env
// share its functions and its streams.
type Env struct {
	Lookup   func(name string) (string, bool)  // reports the value of the environment variable name
	Environ  func() []string                   // returns the environment as KEY=value pairs
	ReadFile func(name string) ([]byte, error) // returns the content of the config file name
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Args     []string // the arguments after the name of the program
}

// OS returns the Env of the current process: the arguments of os.Args after the name of the
// program, os.LookupEnv, os.Environ, os.ReadFile and the standard streams of package os.
func OS() *Env {
	return &Env{
		Lookup:   os.LookupEnv,
		Environ:  os.Environ,
		ReadFile: os.ReadFile,
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Args:     os.Args[min(1, len(os.Args)):],
	}
}

// Vars returns a Lookup function that reports the variables of a copy of vars.
func Vars(vars map[string]string) func(name string) (string, bool) {
	vars = maps.Clone(vars)
	return func(name string) (string, bool) {
		v, ok := vars[name]
		return v, ok
	}
}

// Environ returns an Environ function that returns the variables of a copy of vars as KEY=value
// pairs in the order of their names. Each call returns a list of its own.
func Environ(vars map[string]string) func() []string {
	pairs := make([]string, 0, len(vars))
	for _, name := range slices.Sorted(maps.Keys(vars)) {
		pairs = append(pairs, name+"="+vars[name])
	}
	return func() []string {
		return slices.Clone(pairs)
	}
}

// check returns an error that wraps ErrEnv when e is nil, or has no Stdout or no Stderr.
func (e *Env) check() error {
	switch {
	case e == nil:
		return fmt.Errorf("%w: the Env is nil", ErrEnv)
	case e.Stdout == nil:
		return fmt.Errorf("%w: the Env has no Stdout", ErrEnv)
	case e.Stderr == nil:
		return fmt.Errorf("%w: the Env has no Stderr", ErrEnv)
	default:
		return nil
	}
}

// environ returns the result of the Environ of e, or nil when e has no Environ.
func (e *Env) environ() []string {
	if e.Environ == nil {
		return nil
	}
	return e.Environ()
}

// lookup returns the value of the environment variable name, and reports false when e has no
// Lookup.
func (e *Env) lookup(name string) (string, bool) {
	if e.Lookup == nil {
		return "", false
	}
	return e.Lookup(name)
}

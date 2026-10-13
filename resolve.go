// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import "strconv"

// resolve applies every valid value of the sources of the run in the order of their precedence: the
// defaults, the config file, the environment variables, the flags and the positional arguments, so
// a later source replaces the value of an earlier one. A value that fails leaves its option at the
// value of the earlier sources, and resolve applies the values after it. Each positional argument
// takes one argument in the order of the fields, and a list takes the rest. Then resolve checks
// that every positional argument and every required option has a value. It returns the first error
// in that order.
//
// Error modes:
//   - the errors of applyConfig, applyEnv, applyFlags and assign.
//   - an *Error that wraps ErrUnexpectedArgument for an argument after the last one that the
//     command declares.
//   - the errors of check.
func (a *App[G]) resolve(env *Env, d *declaration[G], in *input[G]) error {
	for _, o := range in.list {
		if o.hasDef {
			o.value.Set(o.def)
			o.set, o.source = true, Source{Kind: Default}
		}
	}
	first := a.applyConfig(env, d, in)
	if err := in.applyEnv(env); first == nil {
		first = err
	}
	if err := in.applyFlags(); first == nil {
		first = err
	}
	args := in.args
	for _, o := range in.list {
		if !o.arg || len(args) == 0 {
			continue
		}
		n := 1
		if o.shape == list {
			n = len(args)
		}
		if err := o.assign(args[:n], Source{Kind: Argument, Name: o.label()}, false); first == nil {
			first = err
		}
		args = args[n:]
	}
	if len(args) > 0 && first == nil {
		first = &Error{
			Err:   ErrUnexpectedArgument,
			Input: args[0],
			text:  "unexpected argument " + strconv.Quote(args[0]),
		}
	}
	for _, o := range in.list {
		if err := o.check(); first == nil {
			first = err
		}
	}
	return first
}

// applyEnv applies the environment variable of each option of in that has one, and returns the
// first error. A value that fails leaves its option at the value of the earlier sources.
//
// Error modes:
//   - the errors of assign.
func (in *input[G]) applyEnv(env *Env) error {
	var first error
	for _, o := range in.list {
		if o.env == "" {
			continue
		}
		s, ok := env.lookup(o.env)
		if !ok {
			continue
		}
		if err := o.assign(o.split(s), Source{Kind: Environment, Name: o.env}, false); first == nil {
			first = err
		}
	}
	return first
}

// applyFlags applies each flag of in and returns the first error. The flags apply in the order of
// the command line. A repeated flag of a list appends to the values of its earlier valid
// occurrences, and of a map adds to their entries. A value that fails leaves its option at the
// value of the earlier sources.
//
// Error modes:
//   - the errors of assign.
func (in *input[G]) applyFlags() error {
	var first error
	repeated := make(map[*option]bool, len(in.flags))
	for _, f := range in.flags {
		err := f.opt.assign(f.opt.split(f.value), Source{Kind: Flag, Name: f.opt.label()}, repeated[f.opt])
		if err == nil {
			repeated[f.opt] = true
		}
		if first == nil {
			first = err
		}
	}
	return first
}

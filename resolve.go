// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import "strconv"

// resolve applies the sources of the run in the order of their precedence: the defaults, the config
// file, the environment variables, the flags and the positional arguments, so a later source
// replaces the value of an earlier one. A repeated flag of a list appends to the values of its
// earlier occurrences, and of a map adds to their entries. Each positional argument takes one
// argument in the order of the fields, and a list takes the rest. Then resolve checks that every
// positional argument and every required option has a value.
//
// Error modes:
//   - the errors of applyConfig and of assign.
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
	if err := a.applyConfig(env, d, in); err != nil {
		return err
	}
	for _, o := range in.list {
		if o.env == "" {
			continue
		}
		if s, ok := env.lookup(o.env); ok {
			if err := o.assign(o.split(s), Source{Kind: Environment, Name: o.env}, false); err != nil {
				return err
			}
		}
	}
	repeated := make(map[*option]bool, len(in.flags))
	for _, f := range in.flags {
		if err := f.opt.assign(
			f.opt.split(f.value),
			Source{Kind: Flag, Name: f.opt.label()},
			repeated[f.opt],
		); err != nil {
			return err
		}
		repeated[f.opt] = true
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
		if err := o.assign(args[:n], Source{Kind: Argument, Name: o.label()}, false); err != nil {
			return err
		}
		args = args[n:]
	}
	if len(args) > 0 {
		return &Error{Err: ErrUnexpectedArgument, Input: args[0], text: "unexpected argument " + strconv.Quote(args[0])}
	}
	for _, o := range in.list {
		if err := o.check(); err != nil {
			return err
		}
	}
	return nil
}

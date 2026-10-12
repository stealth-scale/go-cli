// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"reflect"
	"slices"
)

// The global option of every app that writes the value and the source of each option.
const (
	// debugFlag is the name of the option.
	debugFlag = "flag-debug"
	// debugHelp is the help text of the option.
	debugHelp = "Print the value and the source of each option before the command runs."
)

// debugOption returns the option --flag-debug of the app of the root scope s: a bool with the
// environment variable <APP>_FLAG_DEBUG and no config key. It converts its value by its kind, so
// a conversion of bool that Convert registers does not apply to it.
func debugOption(s *scope) *option {
	t := reflect.TypeFor[bool]()
	o := &option{value: reflect.New(t).Elem(), conv: kindConverter(t), help: debugHelp}
	o.name, o.env, _ = s.names(debugFlag)
	return o
}

// debugRows returns the rows of the table of --flag-debug: the flag or the argument, the value and
// the source of each option of the run of in except --flag-debug. The rows list the options of the
// ancestors of the command and then the options of the command, each in the order of the help. The
// value of a secret option is masked.
func debugRows[G any](d *declaration[G], in *input[G]) []DebugRow {
	var rows []DebugRow
	for _, o := range slices.Concat(ordered(in.cmd.inherited), ordered(in.cmd.options)) {
		if o != d.debug {
			rows = append(rows, DebugRow{Option: o.label(), Value: o.display(), Source: o.source})
		}
	}
	return rows
}

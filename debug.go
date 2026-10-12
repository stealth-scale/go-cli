// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
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
func debugOption(s scope) *option {
	t := reflect.TypeFor[bool]()
	o := &option{value: reflect.New(t).Elem(), conv: kindConverter(t), help: debugHelp}
	o.name, o.env, _ = s.names(debugFlag)
	return o
}

// writeDebug writes to w a table of the options of the run of in except --flag-debug: the flag or
// the argument of each, its value, and its source. The table lists the global options and then the
// options of the command, each in the order of the help, and masks the value of a secret option.
//
// Error modes:
//   - an error that wraps the error of w.
func writeDebug[G any](w io.Writer, d *declaration[G], in *input[G]) error {
	rows := [][3]string{{"OPTION", "VALUE", "SOURCE"}}
	for _, o := range slices.Concat(ordered(d.globals), ordered(in.cmd.options)) {
		if o != d.debug {
			rows = append(rows, [3]string{o.label(), o.display(), o.source.String()})
		}
	}
	var b strings.Builder
	writeRows(&b, rows)
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("cli: write the output of %s%s: %w", flagPrefix, debugFlag, err)
	}
	return nil
}

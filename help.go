// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The text of the help.
const (
	// indent starts each row of a section, and a flag without a short flag has a second indent.
	indent = "  "
	// columnGap separates the columns of a row.
	columnGap = "  "
	// noShort takes the place of the short flag of an option without one, so the long flags align.
	noShort = "    "
	// ellipsis follows the placeholder of a list and of a positional argument that takes the rest.
	ellipsis = "..."
	// helpText is the help text of the flags of the help.
	helpText = "Show the help."
)

// writeHelp writes the help of the command of in to w, or the help of the app when in has no
// command. The help lists the commands of the app, or the positional arguments and the options of
// the command, and then the global options with the flags of the help among them. The row of an
// option includes its default, its environment variable and its config key.
//
// Error modes:
//   - an error that wraps the error of w.
func (a *App[G]) writeHelp(w io.Writer, d *declaration[G], in *input[G]) error {
	var b strings.Builder
	if in.cmd == nil {
		b.WriteString("Usage: " + a.name + " [global options] <command> [options]\n")
		if a.help != "" {
			b.WriteString("\n" + a.help + "\n")
		}
		rows := make([][3]string, len(d.commands))
		for i, c := range d.commands {
			rows[i] = [3]string{c.cmd.name, c.cmd.help}
		}
		writeSection(&b, "Commands", rows)
	} else {
		usage := []string{a.name, in.cmd.cmd.name, "[options]"}
		var options, args [][3]string
		for _, o := range ordered(in.cmd.options) {
			if o.arg {
				usage = append(usage, o.usage())
				args = append(args, [3]string{noShort + o.label(), o.help, o.notes()})
			} else {
				options = append(options, o.row())
			}
		}
		b.WriteString("Usage: " + strings.Join(usage, " ") + "\n")
		if in.cmd.cmd.help != "" {
			b.WriteString("\n" + in.cmd.cmd.help + "\n")
		}
		writeSection(&b, "Arguments", args)
		writeSection(&b, "Options", options)
	}
	help := &option{value: reflect.New(reflect.TypeFor[bool]()).Elem(), name: helpName, help: helpText, short: helpRune}
	globals := ordered(slices.Concat(d.globals, []*option{help}))
	rows := make([][3]string, len(globals))
	for i, o := range globals {
		rows[i] = o.row()
	}
	writeSection(&b, "Global options", rows)
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("cli: write the help: %w", err)
	}
	return nil
}

// usage returns the positional argument o in the usage line of the help: its label, with an
// ellipsis for a list, in brackets when it has a default.
func (o *option) usage() string {
	s := o.label()
	if o.shape == list {
		s += ellipsis
	}
	if o.hasDef {
		s = "[" + s + "]"
	}
	return s
}

// row returns the row of the option o in the help: the flags and the placeholder of the value, the
// help text, and the notes.
func (o *option) row() [3]string {
	flags := noShort + o.label()
	if o.short != 0 {
		flags = shortPrefix + string(o.short) + ", " + o.label()
	}
	return [3]string{flags + o.placeholder(), o.help, o.notes()}
}

// notes returns the last column of the row of o in the help: whether o is required, and its
// default, its environment variable and its config key. A secret option masks its default.
func (o *option) notes() string {
	var notes []string
	if o.required {
		notes = append(notes, "required")
	}
	if o.hasDef {
		def := o.defText
		if o.secret {
			def = secretText
		}
		notes = append(notes, "default "+strconv.Quote(def))
	}
	if o.env != "" {
		notes = append(notes, "env "+o.env)
	}
	if o.key != "" {
		notes = append(notes, "key "+o.key)
	}
	return strings.Join(notes, ", ")
}

// placeholder returns the text after the flag of o in the help: nothing for a bool, the values of
// an enum, and else the name of the type of its values, such as " string,..." for a list of strings.
func (o *option) placeholder() string {
	t := o.value.Type()
	switch {
	case t.Kind() == reflect.Bool:
		return ""
	case o.enum != nil:
		return " " + strings.Join(o.enum, "|")
	case o.shape == list:
		return " " + typeName(t.Elem()) + listSeparator + ellipsis
	case o.shape == dict:
		return " " + typeName(t.Key()) + entrySeparator + typeName(t.Elem()) + listSeparator + ellipsis
	default:
		return " " + typeName(t)
	}
}

// typeName returns the name of the type t in the help: file for a ConfigFile, the name of the
// element for a pointer, the name of a named type in lower case, and else the name of its kind.
func typeName(t reflect.Type) string {
	switch {
	case t == configFileType:
		return "file"
	case t.Kind() == reflect.Pointer:
		return typeName(t.Elem())
	case t.Name() != "":
		return strings.ToLower(t.Name())
	default:
		return t.Kind().String()
	}
}

// writeSection writes the heading and the rows of a section of the help to b, or nothing when rows
// is empty.
func writeSection(b *strings.Builder, heading string, rows [][3]string) {
	if len(rows) == 0 {
		return
	}
	b.WriteString("\n" + heading + ":\n")
	writeRows(b, rows)
}

// writeRows writes rows to b as three columns, each as wide as its widest cell, without the spaces
// at the end of a row.
func writeRows(b *strings.Builder, rows [][3]string) {
	var first, second int
	for _, r := range rows {
		first = max(first, utf8.RuneCountInString(r[0]))
		second = max(second, utf8.RuneCountInString(r[1]))
	}
	for _, r := range rows {
		line := indent + r[0] + strings.Repeat(" ", first-utf8.RuneCountInString(r[0])) + columnGap +
			r[1] + strings.Repeat(" ", second-utf8.RuneCountInString(r[1])) + columnGap + r[2]
		b.WriteString(strings.TrimRight(line, " ") + "\n")
	}
}

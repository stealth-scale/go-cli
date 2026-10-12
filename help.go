// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
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

// helpOf returns the help of the command of in, or the help of the app when in has no command. The
// help lists the commands of the app, or the positional arguments and the options of the command.
// It lists the global options with the flags of the help among them.
func (a *App[G]) helpOf(d *declaration[G], in *input[G]) *Help {
	h := &Help{App: a.name}
	if in.cmd == nil {
		h.Usage = []string{"[global options]", "<command>", "[options]"}
		h.Text = a.help
		h.Commands = make([]HelpCommand, len(d.commands))
		for i, c := range d.commands {
			h.Commands[i] = HelpCommand{Name: c.cmd.name, Text: c.cmd.help}
		}
	} else {
		h.Command = in.cmd.cmd.name
		h.Usage = []string{"[options]"}
		h.Text = in.cmd.cmd.help
		for _, o := range ordered(in.cmd.options) {
			if o.arg {
				h.Usage = append(h.Usage, o.usage())
				h.Arguments = append(h.Arguments, o.helpOption())
			} else {
				h.Options = append(h.Options, o.helpOption())
			}
		}
	}
	help := &option{value: reflect.New(reflect.TypeFor[bool]()).Elem(), name: helpName, help: helpText, short: helpRune}
	for _, o := range ordered(slices.Concat(d.globals, []*option{help})) {
		h.Globals = append(h.Globals, o.helpOption())
	}
	return h
}

// helpOption returns o as an option of the help. The default of a secret option is masked.
func (o *option) helpOption() HelpOption {
	h := HelpOption{Name: o.label(), Text: o.help, Env: o.env, Key: o.key, Required: o.required}
	if o.short != 0 {
		h.Short = shortPrefix + string(o.short)
	}
	if !o.arg {
		h.Placeholder = o.placeholder()
	}
	if o.hasDef {
		def := o.defText
		if o.secret {
			def = secretText
		}
		h.Default = strconv.Quote(def)
	}
	return h
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

// placeholder returns the text after the flag of o in the help: nothing for a bool, the values of
// an enum, and else the name of the type of its values, such as "string,..." for a list of strings.
func (o *option) placeholder() string {
	t := o.value.Type()
	switch {
	case t.Kind() == reflect.Bool:
		return ""
	case o.enum != nil:
		return strings.Join(o.enum, "|")
	case o.shape == list:
		return typeName(t.Elem()) + listSeparator + ellipsis
	case o.shape == dict:
		return typeName(t.Key()) + entrySeparator + typeName(t.Elem()) + listSeparator + ellipsis
	default:
		return typeName(t)
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
	b.WriteString("\n")
	b.WriteString(heading)
	b.WriteString(":\n")
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
		b.WriteString(strings.TrimRight(line, " "))
		b.WriteString("\n")
	}
}

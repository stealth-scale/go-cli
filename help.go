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
	// usageCommand takes the place of the name of a subcommand in the usage line.
	usageCommand = "<command>"
	// usageOptions takes the place of the flags in the usage line.
	usageOptions = "[options]"
)

// helpOf returns the help of the last command of the path of in, which is the app before a command.
// The help lists the subcommands, the positional arguments and the options of the command, and the
// options of its ancestors as the global options. The reserved flags are options of the app. The
// options have their config keys only when an option of the path is a config file, because a run
// without a config file does not read a key.
func (a *App[G]) helpOf(in *input[G]) *Help {
	b := in.cmd
	h := &Help{App: a.name, Command: b.path, Text: b.help, Usage: []string{usageOptions}}
	if len(b.children) > 0 {
		name := usageCommand
		if b.handle != nil {
			name = "[" + usageCommand + "]"
		}
		h.Usage = []string{name, usageOptions}
	}
	for _, c := range b.children {
		h.Commands = append(h.Commands, HelpCommand{Name: c.name, Text: c.help})
	}
	own, inherited := b.options, b.inherited
	if b.parent == nil {
		own = slices.Concat(own, a.reserved())
	} else {
		inherited = slices.Concat(inherited, a.reserved())
	}
	for _, o := range ordered(own) {
		if o.arg {
			h.Usage = append(h.Usage, o.usage())
			h.Arguments = append(h.Arguments, o.helpOption())
		} else {
			h.Options = append(h.Options, o.helpOption())
		}
	}
	for _, o := range ordered(inherited) {
		h.Globals = append(h.Globals, o.helpOption())
	}
	if !slices.ContainsFunc(slices.Concat(b.options, b.inherited), func(o *option) bool { return o.config }) {
		for _, opts := range [...][]HelpOption{h.Options, h.Globals} {
			for i := range opts {
				opts[i].Key = ""
			}
		}
	}
	return h
}

// reserved returns the flags that the package declares on every path of the app as bool options.
// They are the flags of the help and the flag --version of an app with a version. The help lists
// them with the options of the app, and an option of the app cannot declare them.
func (a *App[G]) reserved() []*option {
	t := reflect.TypeFor[bool]()
	opts := []*option{{value: reflect.New(t).Elem(), name: helpName, help: helpText, short: helpRune}}
	if a.version != "" {
		opts = append(opts, &option{value: reflect.New(t).Elem(), name: versionFlag, help: versionHelp})
	}
	return opts
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

// typeName returns the name of the type t in the help: file for a ConfigFile and an InputFile, the
// name of the element for a pointer, the name of a named type in lower case, and else the name of
// its kind.
func typeName(t reflect.Type) string {
	switch {
	case t == configFileType || t == inputFileType:
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

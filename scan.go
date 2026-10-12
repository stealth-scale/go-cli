// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The tokens of a command line.
const (
	// terminator ends the flags of a command line. Every argument after it is positional.
	terminator = "--"
	// flagPrefix starts a long flag, and shortPrefix starts a cluster of short flags.
	flagPrefix  = "--"
	shortPrefix = "-"
	// flagValue separates the name of a long flag from its value in one argument.
	flagValue = "="
	// helpName and helpRune are the long flag and the short flag of the help, which no option
	// declares. A cluster reads helpRune as the short flag of the help.
	helpName = "help"
	helpRune = 'h'
	// helpLong is the long flag of the help on the command line.
	helpLong = flagPrefix + helpName
)

// input is the command line of one run: the path of its command, its flags and its positional
// arguments.
type input[G any] struct {
	cmd   *binding[G]  // the last command of the path, which starts at the app
	list  []*option    // the options of the app and of each command of the path
	flags []occurrence // the flags in the order of the command line
	args  []string     // the positional arguments
	help  bool         // the command line contains a flag of the help
}

// occurrence is a flag of the command line with its value.
type occurrence struct {
	opt   *option
	value string
}

// scan splits args into the path of the command of the run, its flags and its positional arguments,
// by these rules of GNU:
//   - A long flag starts with two dashes. It takes its value after an equals sign or in the next
//     argument.
//   - Short flags combine in a cluster after one dash, such as -vq. A short flag with a value takes
//     the rest of its cluster or the next argument, such as -p8080 or -p 8080.
//   - The flag of a bool takes a value only after an equals sign.
//   - After a command with subcommands, an argument that is not a flag is the name of a subcommand.
//     After a command without subcommands, it is a positional argument.
//   - A flag can come anywhere after the name of the command that declares it, and a flag of the app
//     anywhere.
//   - The argument -- ends the flags. A lone - is a positional argument.
//
// Error modes:
//   - an *Error that wraps ErrUnknownCommand for a name that the command before it does not
//     declare.
//   - an *Error that wraps ErrUnknownFlag for a flag that no option of the path declares.
//   - an *Error that wraps ErrMissing for a flag without its value at the end of args.
func (in *input[G]) scan(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == terminator:
			in.args = append(in.args, args[i+1:]...)
			return nil
		case arg == helpLong:
			in.help = true
		case strings.HasPrefix(arg, flagPrefix):
			next, err := in.long(args, i)
			if err != nil {
				return err
			}
			i = next
		case len(arg) > len(shortPrefix) && strings.HasPrefix(arg, shortPrefix):
			next, err := in.cluster(args, i)
			if err != nil {
				return err
			}
			i = next
		case len(in.cmd.children) > 0:
			j := slices.IndexFunc(in.cmd.children, func(b *binding[G]) bool { return b.name == arg })
			if j < 0 {
				names := make([]string, len(in.cmd.children))
				for k, b := range in.cmd.children {
					names[k] = b.name
				}
				return &Error{
					Err:        ErrUnknownCommand,
					Input:      arg,
					Suggestion: suggest(arg, names, ""),
					text:       "unknown command " + strconv.Quote(arg),
				}
			}
			in.cmd = in.cmd.children[j]
			in.list = slices.Concat(in.list, in.cmd.options)
		default:
			in.args = append(in.args, arg)
		}
	}
	return nil
}

// long scans the long flag at args[i] by the rules of scan, and returns the index of the last
// argument that it reads. The error of an unknown flag suggests the closest long flag of the run.
func (in *input[G]) long(args []string, i int) (int, error) {
	name, value, explicit := strings.Cut(args[i][len(flagPrefix):], flagValue)
	j := slices.IndexFunc(in.list, func(o *option) bool { return !o.arg && o.name == name })
	if j < 0 {
		var names []string
		for _, o := range in.list {
			if !o.arg {
				names = append(names, o.name)
			}
		}
		return i, &Error{
			Err:        ErrUnknownFlag,
			Input:      flagPrefix + name,
			Suggestion: suggest(name, names, flagPrefix),
			text:       "unknown flag " + flagPrefix + name,
		}
	}
	o := in.list[j]
	switch {
	case explicit:
	case o.value.Kind() == reflect.Bool:
		value = "true"
	case i+1 == len(args):
		return i, &Error{Err: ErrMissing, Input: o.label(), text: "flag " + o.label() + " needs a value"}
	default:
		i++
		value = args[i]
	}
	in.flags = append(in.flags, occurrence{opt: o, value: value})
	return i, nil
}

// cluster scans the cluster of short flags at args[i] by the rules of scan, and returns the index
// of the last argument that it reads. A cluster whose text is the name of a long flag, such as
// -addr for --addr, fails instead of reading -a with the value ddr.
func (in *input[G]) cluster(args []string, i int) (int, error) {
	text := args[i][len(shortPrefix):]
	longName := slices.ContainsFunc(in.list, func(o *option) bool { return !o.arg && o.name == text })
	if longName && utf8.RuneCountInString(text) > 1 {
		return i, &Error{
			Err:        ErrUnknownFlag,
			Input:      args[i],
			Suggestion: flagPrefix + text,
			text:       "unknown flag " + args[i],
		}
	}
	for text != "" {
		r, size := utf8.DecodeRuneInString(text)
		text = text[size:]
		if r == helpRune {
			in.help = true
			continue
		}
		j := slices.IndexFunc(in.list, func(o *option) bool { return o.short != 0 && o.short == r })
		if j < 0 {
			flag := shortPrefix + string(r)
			return i, &Error{Err: ErrUnknownFlag, Input: flag, text: "unknown flag " + flag}
		}
		o := in.list[j]
		switch {
		case o.value.Kind() == reflect.Bool:
			in.flags = append(in.flags, occurrence{opt: o, value: "true"})
		case text != "":
			in.flags = append(in.flags, occurrence{opt: o, value: text})
			return i, nil
		case i+1 < len(args):
			in.flags = append(in.flags, occurrence{opt: o, value: args[i+1]})
			return i + 1, nil
		default:
			return i, &Error{Err: ErrMissing, Input: o.label(), text: "flag " + o.label() + " needs a value"}
		}
	}
	return i, nil
}

// locate sets the command of err to the path of the last command of in when err is an *Error, and
// returns err. The path of the app is empty.
func (in *input[G]) locate(err error) error {
	if e, ok := errors.AsType[*Error](err); ok {
		e.Command = in.cmd.path
	}
	return err
}

// sources returns the source of each option of in, by a pointer to its field.
func (in *input[G]) sources() map[any]Source {
	sources := make(map[any]Source, len(in.list))
	for _, o := range in.list {
		sources[o.value.Addr().Interface()] = o.source
	}
	return sources
}

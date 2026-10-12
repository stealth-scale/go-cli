// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"encoding"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// The separators and the placeholders of the values of an option.
const (
	// listSeparator separates the values of a list, the entries of a map and the values of an enum
	// in one text.
	listSeparator = ","
	// entrySeparator separates the key of an entry of a map from its value.
	entrySeparator = "="
	// secretText replaces the value of a secret option in the help and in the output of
	// --flag-debug.
	secretText = "***"
	// nilText is the value of a pointer that no source has set in the output of --flag-debug.
	nilText = "<nil>"
)

// shape is the way that an option takes its values.
type shape uint8

// The shapes of an option.
const (
	// scalar takes one value.
	scalar shape = 0
	// list is a slice, which takes a list of values.
	list shape = 1
	// dict is a map, which takes entries of the form key=value.
	dict shape = 2
	// pointer is a pointer, which is nil until a source sets it.
	pointer shape = 3
)

// option is a field of an options struct, with the names and the rules that its tags declare.
type option struct {
	value    reflect.Value // the field, which the run sets
	def      reflect.Value // the value of the default, when hasDef
	conv     converter     // converts a scalar, an element of a list or a value of a map
	keyConv  converter     // converts a key of a map
	name     string        // the flag without its dashes
	env      string        // the environment variable, or empty for none
	key      string        // the config key, or empty for none
	help     string
	defText  string   // the text of the default, when hasDef
	enum     []string // the texts of the values that the option takes, or nil for every text
	source   Source   // the source that set the value last
	short    rune     // the short flag, or 0 for none
	shape    shape
	hasDef   bool
	required bool
	arg      bool // the field is a positional argument
	secret   bool // the help and --flag-debug mask the value
	config   bool // the field is a ConfigFile
	set      bool // a source has set the value
}

// label returns the name of the option in a message: the flag, or the name of a positional
// argument in upper case with underscores, such as DRY_RUN.
func (o *option) label() string {
	if o.arg {
		return strings.ToUpper(strings.ReplaceAll(o.name, wordSeparator, envSeparator))
	}
	return flagPrefix + o.name
}

// split returns the values of the text s: the texts that commas separate for a list or a map, none
// for an empty text of a list or a map, and s itself for any other shape.
func (o *option) split(s string) []string {
	switch {
	case o.shape != list && o.shape != dict:
		return []string{s}
	case s == "":
		return nil
	default:
		return strings.Split(s, listSeparator)
	}
}

// parse returns the value of the field for the texts vals. It returns the text that fails and the
// reason when a text does not convert, when the enum excludes it, when an entry of a map has no
// equals sign, and when a field that takes one value receives two or more.
func (o *option) parse(vals []string) (reflect.Value, string, string) {
	t := o.value.Type()
	switch o.shape {
	case list:
		values := reflect.MakeSlice(t, 0, len(vals))
		for _, s := range vals {
			v, reason := o.element(s)
			if reason != "" {
				return reflect.Value{}, s, reason
			}
			values = reflect.Append(values, v)
		}
		return values, "", ""
	case dict:
		entries := reflect.MakeMapWithSize(t, len(vals))
		for _, s := range vals {
			k, text, ok := strings.Cut(s, entrySeparator)
			if !ok {
				return reflect.Value{}, s, "want key" + entrySeparator + "value"
			}
			key, reason := o.keyConv(k)
			if reason != "" {
				return reflect.Value{}, k, reason
			}
			v, reason := o.element(text)
			if reason != "" {
				return reflect.Value{}, text, reason
			}
			entries.SetMapIndex(key, v)
		}
		return entries, "", ""
	default:
		if len(vals) != 1 {
			return reflect.Value{}, strings.Join(vals, listSeparator), "the option takes one value"
		}
		v, reason := o.element(vals[0])
		if reason != "" {
			return reflect.Value{}, vals[0], reason
		}
		if o.shape != pointer {
			return v, "", ""
		}
		p := reflect.New(t.Elem())
		p.Elem().Set(v)
		return p, "", ""
	}
}

// element returns the value of the text s, or the reason when s does not convert or the enum of
// the option excludes it. The enum compares the text of the converted value, so an enum of
// durations lists them as time.Duration.String writes them. The reason of a value outside the enum
// lists the values of the enum, such as "want none, gzip or zstd".
func (o *option) element(s string) (reflect.Value, string) {
	v, reason := o.conv(s)
	if reason == "" && o.enum != nil && !slices.Contains(o.enum, fmt.Sprint(v.Interface())) {
		last := len(o.enum) - 1
		reason = "want " + o.enum[last]
		if last > 0 {
			reason = "want " + strings.Join(o.enum[:last], ", ") + " or " + o.enum[last]
		}
	}
	return v, reason
}

// assign sets the option to the texts vals from the source src. With add, a list appends the values
// to its own and a map adds the entries to its own.
//
// Error modes:
//   - an *Error that wraps ErrInvalidValue for a text that parse refuses.
func (o *option) assign(vals []string, src Source, add bool) error {
	value, bad, reason := o.parse(vals)
	if reason != "" {
		return &Error{
			Err:    ErrInvalidValue,
			Input:  o.label(),
			Value:  bad,
			Source: src,
			text:   "invalid " + o.label() + " " + strconv.Quote(bad) + " from " + src.String() + ": " + reason,
		}
	}
	switch {
	case add && o.shape == list:
		value = reflect.AppendSlice(o.value, value)
	case add && o.shape == dict:
		for k, v := range value.Seq2() {
			o.value.SetMapIndex(k, v)
		}
		value = o.value
	}
	o.value.Set(value)
	o.set, o.source = true, src
	return nil
}

// check returns an *Error that wraps ErrMissing for a positional argument and a required option
// that no source has set.
func (o *option) check() error {
	switch {
	case o.set:
		return nil
	case o.arg:
		return &Error{Err: ErrMissing, Input: o.label(), text: "missing argument " + o.label()}
	case o.required:
		return &Error{Err: ErrMissing, Input: o.label(), text: "missing " + o.label()}
	default:
		return nil
	}
}

// display returns the value of the option for the output of --flag-debug: *** for a secret, <nil>
// for a nil pointer, and the text that printable gives fmt for any other value.
func (o *option) display() string {
	v := o.value
	switch {
	case o.secret:
		return secretText
	case v.Kind() == reflect.Pointer && v.IsNil():
		return nilText
	case o.shape == pointer:
		v = v.Elem()
	}
	return fmt.Sprint(printable(v))
}

// printable returns v in a form that fmt prints for --flag-debug. A value whose pointer has
// MarshalText prints as its text, so a time.Time prints in RFC 3339, as a flag takes it. Any other
// value whose pointer has String prints as its result, so a net.IPNet prints as 10.0.0.0/8, and a
// string prints in quotes. printable converts the elements of a slice and the keys and the values
// of a map in the same way, and returns any other value unchanged.
func printable(v reflect.Value) any {
	p := reflect.New(v.Type())
	p.Elem().Set(v)
	if m, ok := reflect.TypeAssert[encoding.TextMarshaler](p); ok {
		if text, err := m.MarshalText(); err == nil {
			return string(text)
		}
	}
	if s, ok := reflect.TypeAssert[fmt.Stringer](p); ok {
		return s.String()
	}
	switch v.Kind() {
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Slice:
		items := make([]any, v.Len())
		for i := range items {
			items[i] = printable(v.Index(i))
		}
		return items
	case reflect.Map:
		entries := make(map[any]any, v.Len())
		for k, e := range v.Seq2() {
			entries[printable(k)] = printable(e)
		}
		return entries
	default:
		return v.Interface()
	}
}

// ordered returns the options of opts in the order of the help and of --flag-debug: the flags by
// name, then the positional arguments in the order of their fields.
func ordered(opts []*option) []*option {
	flags, args := make([]*option, 0, len(opts)), []*option(nil)
	for _, o := range opts {
		if o.arg {
			args = append(args, o)
		} else {
			flags = append(flags, o)
		}
	}
	slices.SortFunc(flags, func(x, y *option) int { return strings.Compare(x.name, y.name) })
	return append(flags, args...)
}

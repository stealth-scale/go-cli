// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// tag is the name of a struct tag of an options field.
type tag string

// The tags of an options field.
const (
	tagHelp     tag = "help"
	tagDefault  tag = "default"
	tagEnum     tag = "enum"
	tagRequired tag = "required"
	tagArg      tag = "arg"
	tagName     tag = "name"
	tagShort    tag = "short"
	tagEnv      tag = "env"
	tagKey      tag = "key"
	tagSecret   tag = "secret"
	tagPrefix   tag = "prefix"
)

// optionTags are the tags of an option. A group takes none of them.
var optionTags = [...]tag{
	tagHelp, tagDefault, tagEnum, tagRequired, tagArg, tagName, tagShort, tagEnv, tagKey, tagSecret,
}

// none is the value of the tags env and key that removes the environment variable or the config key.
const none = "-"

// declaration is the options of an app for one run: the global options, the options of each
// command and the config keys of the app.
type declaration[G any] struct {
	global   *G
	debug    *option         // the option --flag-debug, the last global option
	dicts    map[string]bool // the config key of each option of the app, true for the key of a map
	globals  []*option       // the global options in the order of their fields
	commands []*binding[G]   // the commands in the order of their registration
}

// binding is a command of one run with its options and its handler over them.
type binding[G any] struct {
	cmd     *command[G]
	call    handler[G]
	options []*option // the options of the command in the order of their fields
}

// declare returns the declaration of the app for one run.
//
// Error modes:
//   - an error that wraps ErrDeclaration for an invalid registration, an app name that is not a name
//     of lower-case words, and a command name that is not one or that two commands share.
//   - the errors of options for the options of the app and of each command.
//   - the errors of check for the global options and for the options of each command.
//   - the errors of keys for the config keys of the app.
func (a *App[G]) declare() (*declaration[G], error) {
	if err := errors.Join(a.errs...); err != nil {
		return nil, err
	}
	if !namePattern.MatchString(a.name) {
		return nil, fmt.Errorf("%w: the app name %q is not a name of lower-case words", ErrDeclaration, a.name)
	}
	d := &declaration[G]{global: new(G), dicts: map[string]bool{}}
	root := scope{app: a.name, path: reflect.TypeFor[G]().String()}
	globals, err := a.options(nil, reflect.ValueOf(d.global).Elem(), root)
	if err != nil {
		return nil, err
	}
	d.debug = debugOption(root)
	d.globals = slices.Concat(globals, []*option{d.debug})
	if err := check(d.globals, ""); err != nil {
		return nil, err
	}
	all := slices.Clone(d.globals)
	for _, c := range a.commands {
		if !namePattern.MatchString(c.name) {
			return nil, fmt.Errorf("%w: the command name %q is not a name of lower-case words", ErrDeclaration, c.name)
		}
		if slices.ContainsFunc(d.commands, func(b *binding[G]) bool { return b.cmd.name == c.name }) {
			return nil, fmt.Errorf("%w: two commands have the name %s", ErrDeclaration, c.name)
		}
		options, call, err := c.bind(scope{app: a.name, command: c.name})
		if err != nil {
			return nil, err
		}
		if err := check(slices.Concat(d.globals, options), c.name); err != nil {
			return nil, err
		}
		d.commands = append(d.commands, &binding[G]{cmd: c, call: call, options: options})
		all = append(all, options...)
	}
	if err := d.keys(all); err != nil {
		return nil, err
	}
	return d, nil
}

// keys records the config key of each option of all in d.dicts.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a key that one command declares for a map and another
//     for a value of another shape, and for a key inside the key of a map, because the entries of
//     the map take every key inside it.
func (d *declaration[G]) keys(all []*option) error {
	for _, o := range all {
		if o.key == "" {
			continue
		}
		isDict, seen := d.dicts[o.key]
		if seen && isDict != (o.shape == dict) {
			return fmt.Errorf("%w: the config key %s is a map in one command and not in another", ErrDeclaration, o.key)
		}
		d.dicts[o.key] = o.shape == dict
	}
	keys := slices.Sorted(maps.Keys(d.dicts))
	for _, key := range keys {
		for _, other := range keys {
			if d.dicts[key] && strings.HasPrefix(other, key+keySeparator) {
				return fmt.Errorf("%w: the config key %s is inside the key %s of a map", ErrDeclaration, other, key)
			}
		}
	}
	return nil
}

// options appends to opts an option for each exported field of the struct v in the scope s, in the
// order of the fields, and returns opts. A struct field without a conversion is a group, whose
// fields are options in the scope that nest returns. An embedded group adds the exported fields of
// its type also when its type is unexported.
//
// Error modes:
//   - an error that wraps ErrDeclaration when v is not a struct.
//   - the errors of nest and of newOption.
func (a *App[G]) options(opts []*option, v reflect.Value, s scope) ([]*option, error) {
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: the options type %s is not a struct", ErrDeclaration, v.Type())
	}
	for f, field := range v.Fields() {
		group := f.Type.Kind() == reflect.Struct && a.converter(f.Type) == nil
		if !f.IsExported() && (!group || !f.Anonymous) {
			continue
		}
		if !group {
			o, err := a.newOption(&f, field, s)
			if err != nil {
				return nil, err
			}
			opts = append(opts, o)
			continue
		}
		nested, err := s.nest(&f)
		if err != nil {
			return nil, err
		}
		if opts, err = a.options(opts, field, nested); err != nil {
			return nil, err
		}
	}
	return opts, nil
}

// newOption returns the option of the field f with the value v in the scope s.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a tag prefix, a type without a conversion, a name that
//     is not a name of lower-case words, a short flag that is not one ASCII letter other than h, an
//     empty tag env or key, a config key with an empty name, and for the reasons of argument,
//     configFile and withDefault.
func (a *App[G]) newOption(f *reflect.StructField, v reflect.Value, s scope) (*option, error) {
	field := s.path + "." + f.Name
	if _, ok := f.Tag.Lookup(string(tagPrefix)); ok {
		return nil, fmt.Errorf("%w: the field %s is no group, so it takes no tag prefix", ErrDeclaration, field)
	}
	o := &option{value: v, help: f.Tag.Get(string(tagHelp)), config: f.Type == configFileType}
	if reason := a.setShape(o, f.Type); reason != "" {
		return nil, fmt.Errorf("%w: the field %s %s", ErrDeclaration, field, reason)
	}
	name := kebab(f.Name)
	if tagged, ok := f.Tag.Lookup(string(tagName)); ok {
		name = tagged
	}
	if !namePattern.MatchString(name) {
		return nil, fmt.Errorf("%w: the field %s has the name %q, which is not a name of lower-case words",
			ErrDeclaration, field, name)
	}
	o.name, o.env, o.key = s.names(name)
	_, o.required = f.Tag.Lookup(string(tagRequired))
	_, o.arg = f.Tag.Lookup(string(tagArg))
	_, o.secret = f.Tag.Lookup(string(tagSecret))
	if enum, ok := f.Tag.Lookup(string(tagEnum)); ok {
		o.enum = strings.Split(enum, listSeparator)
	}
	if short, ok := f.Tag.Lookup(string(tagShort)); ok {
		var c byte
		if len(short) == 1 {
			c = short[0]
		}
		if c == helpRune || (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return nil, fmt.Errorf("%w: the field %s has the short flag %q, which is not one ASCII letter other than h",
				ErrDeclaration, field, short)
		}
		o.short = rune(c)
	}
	for _, t := range [...]tag{tagEnv, tagKey} {
		value, ok := f.Tag.Lookup(string(t))
		switch {
		case !ok:
			continue
		case value == "":
			return nil, fmt.Errorf("%w: the field %s has an empty tag %s, and the value %s removes the name",
				ErrDeclaration, field, t, none)
		case value == none:
			value = ""
		case t == tagKey && slices.Contains(strings.Split(value, keySeparator), ""):
			return nil, fmt.Errorf("%w: the field %s has the config key %q, which has an empty name",
				ErrDeclaration, field, value)
		}
		if t == tagEnv {
			o.env = value
		} else {
			o.key = value
		}
	}
	if reason := o.argument(f, s); reason != "" {
		return nil, fmt.Errorf("%w: the field %s %s", ErrDeclaration, field, reason)
	}
	if reason := o.configFile(f); reason != "" {
		return nil, fmt.Errorf("%w: the field %s %s", ErrDeclaration, field, reason)
	}
	if reason := o.withDefault(f); reason != "" {
		return nil, fmt.Errorf("%w: the field %s %s", ErrDeclaration, field, reason)
	}
	return o, nil
}

// setShape sets the shape of o and the conversions of its values for the field type t: a scalar
// for a type with a conversion, and a list, a map or a pointer for a slice, a map or a pointer
// whose elements have a conversion. It returns the reason when t has no shape: a type without a
// conversion, and a list, a map or a pointer of ConfigFile.
func (a *App[G]) setShape(o *option, t reflect.Type) string {
	switch {
	case a.converter(t) != nil:
		o.shape, o.conv = scalar, a.converter(t)
		return ""
	case t.Kind() == reflect.Slice && a.converter(t.Elem()) != nil:
		o.shape, o.conv = list, a.converter(t.Elem())
	case t.Kind() == reflect.Map && a.converter(t.Key()) != nil && a.converter(t.Elem()) != nil:
		o.shape, o.conv, o.keyConv = dict, a.converter(t.Elem()), a.converter(t.Key())
	case t.Kind() == reflect.Pointer && a.converter(t.Elem()) != nil:
		o.shape, o.conv = pointer, a.converter(t.Elem())
	default:
		return "has the type " + t.String() + ", which has no conversion"
	}
	if t.Elem() == configFileType {
		return "has the type " + t.String() + ", and a config file takes one path"
	}
	return ""
}

// argument removes the environment variable and the config key of the option o of a field with
// the tag arg, because only the command line sets a positional argument. It returns the reason
// when o cannot be a positional argument: an argument of the global options, which come before the
// command, an argument of a map, and an argument with a tag short, env or key.
func (o *option) argument(f *reflect.StructField, s scope) string {
	switch {
	case !o.arg:
		return ""
	case s.command == "":
		return "is a positional argument of the global options, which come before the command"
	case o.shape == dict:
		return "is a positional argument of a map, which takes no entries"
	}
	for _, t := range [...]tag{tagShort, tagEnv, tagKey} {
		if _, ok := f.Tag.Lookup(string(t)); ok {
			return "is a positional argument, which takes no tag " + string(t)
		}
	}
	o.env, o.key = "", ""
	return ""
}

// configFile removes the config key of the option o of a field of the type ConfigFile, because the
// config file cannot set its own path. It returns the reason when the field has a tag key or arg.
func (o *option) configFile(f *reflect.StructField) string {
	if !o.config {
		return ""
	}
	for _, t := range [...]tag{tagKey, tagArg} {
		if _, ok := f.Tag.Lookup(string(t)); ok {
			return "is a config file, which takes no tag " + string(t)
		}
	}
	o.key = ""
	return ""
}

// withDefault sets the default of the option o from the tag default of the field f. It returns the
// reason when o is required, because a default makes the tag required void, and when the default
// does not convert or the enum of o excludes it.
func (o *option) withDefault(f *reflect.StructField) string {
	text, ok := f.Tag.Lookup(string(tagDefault))
	if !ok {
		return ""
	}
	if o.required {
		return "is required and has a default, which the tag required makes void"
	}
	value, bad, reason := o.parse(o.split(text))
	if reason != "" {
		return fmt.Sprintf("has the default %q, whose value %q is invalid: %s", text, bad, reason)
	}
	o.def, o.defText, o.hasDef = value, text, true
	return ""
}

// check returns an error that wraps ErrDeclaration when two options of opts declare one flag, one
// short flag, one environment variable or one config key, when an option declares the flag of the
// help, when two options are config files, and when the positional arguments of opts are out of
// order: an argument after a list, or a required argument after an optional one. command is the
// command of opts, or empty for the global options.
func check(opts []*option, command string) error {
	owner := "the global options"
	if command != "" {
		owner = "the command " + command
	}
	seen := make(map[string]bool, 4*len(opts))
	var config, optional, variadic *option
	for _, o := range opts {
		if o.arg {
			switch {
			case variadic != nil:
				return fmt.Errorf("%w: the argument %s of %s follows the list %s",
					ErrDeclaration, o.label(), owner, variadic.label())
			case optional != nil && !o.hasDef:
				return fmt.Errorf("%w: the required argument %s of %s follows the optional argument %s",
					ErrDeclaration, o.label(), owner, optional.label())
			}
			if o.hasDef {
				optional = o
			}
			if o.shape == list {
				variadic = o
			}
			continue
		}
		if o.name == helpName {
			return fmt.Errorf("%w: an option of %s declares the flag %s of the help", ErrDeclaration, owner, o.label())
		}
		if o.config {
			if config != nil {
				return fmt.Errorf("%w: %s and %s of %s are both config files",
					ErrDeclaration, config.label(), o.label(), owner)
			}
			config = o
		}
		names := []string{"the flag " + o.label()}
		if o.short != 0 {
			names = append(names, "the short flag "+shortPrefix+string(o.short))
		}
		if o.env != "" {
			names = append(names, "the environment variable "+o.env)
		}
		if o.key != "" {
			names = append(names, "the config key "+o.key)
		}
		for _, n := range names {
			if seen[n] {
				return fmt.Errorf("%w: two options of %s declare %s", ErrDeclaration, owner, n)
			}
			seen[n] = true
		}
	}
	return nil
}

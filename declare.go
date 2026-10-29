// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
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

// declaration is the options of an app for one run: the tree of its commands with their options,
// and the config keys of the app.
type declaration[G any] struct {
	global *G
	debug  *option         // the option --flag-debug, the last option of the app
	dicts  map[string]bool // the config key of each option of the app, true for the key of a map
	root   *binding[G]     // the app, whose options are the global options
}

// binding is a command of one run. It contains a new value of the options of the command, the
// options of the fields of that value, and the hooks, the handler and the subcommands of the
// command over it.
type binding[G any] struct {
	value  reflect.Value // the pointer to the options of the command
	parent *binding[G]   // the parent command, or nil for the app
	// hooks calls the hooks of the command around next.
	hooks func(ctx context.Context, inv *invocation[G], next func(context.Context) error) error
	// handle calls the handler of the command. It is nil for a command without a handler.
	handle    func(ctx context.Context, inv *invocation[G]) error
	name      string        // the name of the command, or empty for the app
	path      string        // the names of the path of the command joined with spaces, or empty for the app
	help      string        // the help text of the command
	options   []*option     // the options of the command in the order of their fields
	inherited []*option     // the options of the ancestors of the command, the app first
	children  []*binding[G] // the subcommands in the order of their registration
}

// newBinding returns the binding of a command whose options are the value at config, with the
// given hooks and the handler handle of the command. A nil handle gives a binding without a
// handler.
func newBinding[G, C any](
	config *C,
	hooks []Hook[G, C],
	handle func(ctx context.Context, run *Run[G, C]) error,
) *binding[G] {
	b := &binding[G]{value: reflect.ValueOf(config)}
	b.hooks = func(ctx context.Context, inv *invocation[G], next func(context.Context) error) error {
		return chain(ctx, &Run[G, C]{invocation: *inv, config: config}, hooks, next)
	}
	if handle != nil {
		b.handle = func(ctx context.Context, inv *invocation[G]) error {
			return handle(ctx, &Run[G, C]{invocation: *inv, config: config})
		}
	}
	return b
}

// checkTree returns an error that wraps ErrDeclaration when b has neither a handler nor subcommands,
// and when b has both subcommands and a positional argument. owner names b in the message, noun is
// the word for its subcommands, and children is their number.
func (b *binding[G]) checkTree(owner, noun string, children int) error {
	i := slices.IndexFunc(b.options, func(o *option) bool { return o.arg })
	switch {
	case b.handle == nil && children == 0:
		return fmt.Errorf("%w: %s has no handler and no %s", ErrDeclaration, owner, noun)
	case children > 0 && i >= 0:
		return fmt.Errorf("%w: %s has %s and the positional argument %s",
			ErrDeclaration, owner, noun, b.options[i].label())
	}
	return nil
}

// around calls the hooks of the commands of the path of b from the app down, and next inside the
// last hook of b.
func (b *binding[G]) around(ctx context.Context, inv *invocation[G], next func(context.Context) error) error {
	if b.parent == nil {
		return b.hooks(ctx, inv, next)
	}
	return b.parent.around(ctx, inv, func(ctx context.Context) error {
		return b.hooks(ctx, inv, next)
	})
}

// declare returns the declaration of the app for one run. The command completion follows the
// commands of the app when the global options do not declare a positional argument.
//
// Error modes:
//   - an error that wraps ErrDeclaration for an invalid registration, for an app name that is not a
//     name of lower-case words, and for a command named completion.
//   - the errors of options for the global options.
//   - the errors of walk for the app and its commands.
//   - the errors of keys for the config keys of the app.
//   - the errors of checkIgnored for the ignored config keys of the app.
func (a *App[G]) declare() (*declaration[G], error) {
	if err := errors.Join(a.errs...); err != nil {
		return nil, err
	}
	if !namePattern.MatchString(a.name) {
		return nil, fmt.Errorf("%w: the app name %q is not a name of lower-case words", ErrDeclaration, a.name)
	}
	if slices.ContainsFunc(a.commands, func(c *command[G]) bool { return c.name == completionName }) {
		return nil, fmt.Errorf("%w: the command name %s is reserved", ErrDeclaration, completionName)
	}
	d := &declaration[G]{global: new(G), dicts: map[string]bool{}}
	root := &scope{envPrefix: a.envPrefix, path: reflect.TypeFor[G]().String()}
	globals, err := a.options(nil, reflect.ValueOf(d.global).Elem(), root)
	if err != nil {
		return nil, err
	}
	d.debug = debugOption(root)
	d.root = newBinding(d.global, a.hooks, a.handle)
	d.root.options, d.root.help = slices.Concat(globals, []*option{d.debug}), a.help
	all, err := a.walk(d.root, "the app "+a.name, "commands", a.commands, nil)
	if err == nil && !slices.ContainsFunc(globals, func(o *option) bool { return o.arg }) {
		all, err = a.bindAll(d.root, []*command[G]{a.completionCommand(d)}, all)
	}
	if err != nil {
		return nil, err
	}
	if err := d.keys(all); err != nil {
		return nil, err
	}
	if err := a.checkIgnored(d); err != nil {
		return nil, err
	}
	return d, nil
}

// walk checks the options of the path of the command b and the rules of checkTree for b, then binds
// the commands of children as its subcommands. owner names b in a message, and noun is the word for
// its subcommands. walk returns all with the options of b and of each of its descendants.
//
// Error modes:
//   - the errors of check, of checkTree and of bindAll.
func (a *App[G]) walk(b *binding[G], owner, noun string, children []*command[G], all []*option) ([]*option, error) {
	if err := check(slices.Concat(b.inherited, b.options), b.path, a.reserved()); err != nil {
		return nil, err
	}
	if err := b.checkTree(owner, noun, len(children)); err != nil {
		return nil, err
	}
	return a.bindAll(b, children, append(all, b.options...))
}

// bindAll binds each command of cmds as a subcommand of the command parent, and walks each. It
// returns all with the options of every command that it binds.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a command name that is not a name of lower-case words,
//     and for two subcommands of parent with one name.
//   - the errors of bind and of walk for each command.
func (a *App[G]) bindAll(parent *binding[G], cmds []*command[G], all []*option) ([]*option, error) {
	var ancestors []reflect.Value
	for p := parent; p != nil; p = p.parent {
		ancestors = append(ancestors, p.value)
	}
	for _, c := range cmds {
		if !namePattern.MatchString(c.name) {
			return nil, fmt.Errorf("%w: the command name %q is not a name of lower-case words", ErrDeclaration, c.name)
		}
		if slices.ContainsFunc(parent.children, func(b *binding[G]) bool { return b.name == c.name }) {
			return nil, fmt.Errorf("%w: two commands have the name %s", ErrDeclaration, c.path)
		}
		b, err := c.bind(&scope{envPrefix: a.envPrefix, command: strings.Fields(c.path), ancestors: ancestors})
		if err != nil {
			return nil, err
		}
		b.parent, b.name, b.path, b.help = parent, c.name, c.path, c.help
		b.inherited = slices.Concat(parent.inherited, parent.options)
		parent.children = append(parent.children, b)
		if all, err = a.walk(b, "the command "+c.path, "subcommands", c.children, all); err != nil {
			return nil, err
		}
	}
	return all, nil
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
// its type also when its type is unexported. An embedded pointer to a group does not add options,
// and inherit sets it to the options of an ancestor.
//
// Error modes:
//   - an error that wraps ErrDeclaration when v is not a struct.
//   - the errors of inherit, of nest and of newOption.
func (a *App[G]) options(opts []*option, v reflect.Value, s *scope) ([]*option, error) {
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: the options type %s is not a struct", ErrDeclaration, v.Type())
	}
	for f, field := range v.Fields() {
		embedded := f.Anonymous && f.Type.Kind() == reflect.Pointer
		if embedded && f.Type.Elem().Kind() == reflect.Struct && a.converters.converter(f.Type.Elem()) == nil {
			if err := s.inherit(&f, field); err != nil {
				return nil, err
			}
			continue
		}
		group := f.Type.Kind() == reflect.Struct && a.converters.converter(f.Type) == nil
		if !f.IsExported() && (!group || !f.Anonymous) {
			continue
		}
		if !group {
			o, err := a.converters.newOption(&f, field, s)
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
		if opts, err = a.options(opts, field, &nested); err != nil {
			return nil, err
		}
	}
	return opts, nil
}

// newOption returns the option of the field f with the value v in the scope s, with the conversions
// of c.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a tag prefix, a type without a conversion, a short flag
//     that is not one ASCII letter other than h, an empty tag env or key, a config key with an empty
//     name, and for the reasons of fieldName, argument, configFile and withDefault.
func (c conversions) newOption(f *reflect.StructField, v reflect.Value, s *scope) (*option, error) {
	field := s.path + "." + f.Name
	if _, ok := f.Tag.Lookup(string(tagPrefix)); ok {
		return nil, fmt.Errorf("%w: the field %s is no group, so it takes no tag prefix", ErrDeclaration, field)
	}
	o := &option{value: v, help: f.Tag.Get(string(tagHelp)), config: f.Type == configFileType}
	if reason := c.setShape(o, f.Type); reason != "" {
		return nil, fmt.Errorf("%w: the field %s %s", ErrDeclaration, field, reason)
	}
	name, err := fieldName(f, field)
	if err != nil {
		return nil, err
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
	if reason := o.argument(f); reason != "" {
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

// setShape sets the shape of o and the conversions of its values for the field type t, with the
// conversions of c: a scalar for a type with a conversion, and a list, a map or a pointer for a
// slice, a map or a pointer whose elements have a conversion. It returns the reason when t has no
// shape: a type without a conversion, and a list, a map or a pointer of ConfigFile.
func (c conversions) setShape(o *option, t reflect.Type) string {
	switch {
	case c.converter(t) != nil:
		o.shape, o.conv = scalar, c.converter(t)
		return ""
	case t.Kind() == reflect.Slice && c.converter(t.Elem()) != nil:
		o.shape, o.conv = list, c.converter(t.Elem())
	case t.Kind() == reflect.Map && c.converter(t.Key()) != nil && c.converter(t.Elem()) != nil:
		o.shape, o.conv, o.keyConv = dict, c.converter(t.Elem()), c.converter(t.Key())
	case t.Kind() == reflect.Pointer && c.converter(t.Elem()) != nil:
		o.shape, o.conv = pointer, c.converter(t.Elem())
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
// when o cannot be a positional argument: an argument of a map, and an argument with a tag short,
// env or key.
func (o *option) argument(f *reflect.StructField) string {
	switch {
	case !o.arg:
		return ""
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
// short flag, one environment variable or one config key, when an option declares one of the flags
// of reserved, when two options are config files, and when the positional arguments of opts are out
// of order: an argument after a list, or a required argument after an optional one. command is the
// command of opts, or empty for the global options.
func check(opts []*option, command string, reserved []*option) error {
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
		if slices.ContainsFunc(reserved, func(r *option) bool { return r.name == o.name }) {
			return fmt.Errorf("%w: an option of %s declares the reserved flag %s", ErrDeclaration, owner, o.label())
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

// inherit sets the embedded pointer v of the field f to the options of the ancestor whose options
// have the type of the target of f. It leaves v nil when f is unexported and the type is not the
// options type of an ancestor, because the package ignores an unexported field.
//
// Error modes:
//   - an error that wraps ErrDeclaration when the type is not the options type of an ancestor, when
//     it is the options type of two ancestors, and when f is unexported.
func (s *scope) inherit(f *reflect.StructField, v reflect.Value) error {
	field := s.path + "." + f.Name
	var found []reflect.Value
	for _, a := range s.ancestors {
		if a.Type() == f.Type {
			found = append(found, a)
		}
	}
	switch {
	case len(found) == 0 && !f.IsExported():
		return nil
	case len(found) == 0:
		return fmt.Errorf("%w: the field %s embeds %s without an ancestor with options of that type",
			ErrDeclaration, field, f.Type)
	case len(found) > 1:
		return fmt.Errorf("%w: the field %s embeds %s, and two ancestors have options of that type",
			ErrDeclaration, field, f.Type)
	case !f.IsExported():
		return fmt.Errorf("%w: the field %s is unexported, so the package cannot set it to the options of an ancestor",
			ErrDeclaration, field)
	}
	v.Set(found[0])
	return nil
}

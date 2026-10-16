// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// The reasons of an error for a member of a kind that its part does not take.
const (
	wantValue  = "want a value"
	wantObject = "want an object"
	wantArray  = "want an array"
)

// treeLabel is the label of the whole tree in a message, in place of the empty key.
const treeLabel = "tree"

// The tags of an options field that a field of a document does not take.
var (
	// valueRejected are the tags that a value of a document does not take.
	valueRejected = [...]tag{tagArg, tagShort, tagEnv, tagKey, tagSecret}
	// listRejected are the tags that a list or a map of structs does not take. They are the tags of
	// valueRejected, default, enum and prefix.
	listRejected = [...]tag{tagDefault, tagEnum, tagArg, tagShort, tagEnv, tagKey, tagSecret, tagPrefix}
)

// form is the way in which a part of a document takes its member.
type form uint8

// The forms of a part of a document.
const (
	// valueForm is a value, a list or a map of values, or a pointer to a value. Its option converts
	// the member from text.
	valueForm form = 0
	// structForm is a struct. Its member is an object.
	structForm form = 1
	// pointerForm is a pointer to a struct. Its member is an object.
	pointerForm form = 2
	// listForm is a list of structs or of pointers to structs. Its member is an array of objects.
	listForm form = 3
	// mapForm is a map of structs or of pointers to structs. Its member is an object of objects.
	mapForm form = 4
)

// part is the decoding of the type of a field, or of the type of the value of [Decode].
type part struct {
	typ  reflect.Type
	opt  *option    // the option that converts a value of valueForm
	st   *docStruct // the struct, the target of a pointer, or the struct of an element
	key  converter  // the conversion of the keys of mapForm
	form form
	ptr  bool // the elements of listForm and mapForm are pointers to the struct
}

// decode decodes member into v. member is the member key of a tree, and v is a settable value of the
// type of p.
//
// Error modes:
//   - an *Error that wraps ErrInvalidValue for a member of another kind than p takes, and for a
//     key of an entry of mapForm that does not convert.
//   - the errors of value, of element, of docStruct.decode and of docStruct.replace.
func (p *part) decode(key string, member any, v reflect.Value) error {
	switch p.form {
	case structForm, pointerForm:
		obj, ok := member.(map[string]any)
		if !ok {
			return invalid(key, member, wantObject)
		}
		if p.form == structForm {
			return p.st.decode(key, obj, v)
		}
		target, err := p.st.replace(v)
		if err != nil {
			return err
		}
		if err := p.st.decode(key, obj, target.Elem()); err != nil {
			return err
		}
		v.Set(target)
	case listForm:
		items, ok := member.([]any)
		if !ok {
			return invalid(key, member, wantArray)
		}
		elements := reflect.MakeSlice(p.typ, len(items), len(items))
		for i, item := range items {
			if err := p.element(key+"["+strconv.Itoa(i)+"]", item, elements.Index(i)); err != nil {
				return err
			}
		}
		v.Set(elements)
	case mapForm:
		obj, ok := member.(map[string]any)
		if !ok {
			return invalid(key, member, wantObject)
		}
		entries := reflect.MakeMapWithSize(p.typ, len(obj))
		for _, name := range slices.Sorted(maps.Keys(obj)) {
			k, reason := p.key(name)
			if reason != "" {
				return invalid(key, name, reason)
			}
			e := reflect.New(p.typ.Elem()).Elem()
			if err := p.element(keyPrefix(key)+name, obj[name], e); err != nil {
				return err
			}
			entries.SetMapIndex(k, e)
		}
		v.Set(entries)
	default:
		value, err := p.value(key, member)
		if err != nil {
			return err
		}
		v.Set(value)
	}
	return nil
}

// element decodes item into a new struct of p, and sets e to the struct or to a pointer to it. item
// is an element of an array or the value of an entry of an object.
//
// Error modes:
//   - an *Error that wraps ErrInvalidValue for an item that is not an object.
//   - the errors of docStruct.create and of docStruct.decode.
func (p *part) element(key string, item any, e reflect.Value) error {
	obj, ok := item.(map[string]any)
	if !ok {
		return invalid(key, item, wantObject)
	}
	target, err := p.st.create()
	if err != nil {
		return err
	}
	if err := p.st.decode(key, obj, target.Elem()); err != nil {
		return err
	}
	if p.ptr {
		e.Set(target)
	} else {
		e.Set(target.Elem())
	}
	return nil
}

// value converts member with the option of p. It converts the text of a value, each value of an
// array and each entry of an object.
//
// Error modes:
//   - an *Error that wraps ErrInvalidValue for a member of another kind than the option takes, for
//     a key of an entry that does not convert, and for a text that the option refuses.
//   - the errors of scalar.
func (p *part) value(key string, member any) (reflect.Value, error) {
	o := p.opt
	switch m := member.(type) {
	case map[string]any:
		if o.shape != dict {
			return reflect.Value{}, invalid(key, m, o.shape.want())
		}
		entries := reflect.MakeMapWithSize(p.typ, len(m))
		for _, name := range slices.Sorted(maps.Keys(m)) {
			k, reason := o.keyConv(name)
			if reason != "" {
				return reflect.Value{}, invalid(key, name, reason)
			}
			e, err := p.scalar(keyPrefix(key)+name, m[name])
			if err != nil {
				return reflect.Value{}, err
			}
			entries.SetMapIndex(k, e)
		}
		return entries, nil
	case []any:
		if o.shape != list {
			return reflect.Value{}, invalid(key, m, o.shape.want())
		}
		values := reflect.MakeSlice(p.typ, len(m), len(m))
		for i, item := range m {
			e, err := p.scalar(key+"["+strconv.Itoa(i)+"]", item)
			if err != nil {
				return reflect.Value{}, err
			}
			values.Index(i).Set(e)
		}
		return values, nil
	default:
		value, bad, reason := o.parse(o.split(text(m)))
		if reason != "" {
			return reflect.Value{}, invalid(key, bad, reason)
		}
		return value, nil
	}
}

// scalar converts item with the option of p. item is an element of an array or the value of an entry
// of an object.
//
// Error modes:
//   - an *Error that wraps ErrInvalidValue for an item that is nil, an object or an array, and for
//     an item whose text does not convert or that the enum of the option excludes.
func (p *part) scalar(key string, item any) (reflect.Value, error) {
	switch item.(type) {
	case nil, map[string]any, []any:
		return reflect.Value{}, invalid(key, item, wantValue)
	}
	s := text(item)
	v, reason := p.opt.element(s)
	if reason != "" {
		return reflect.Value{}, invalid(key, s, reason)
	}
	return v, nil
}

// docStruct is a struct type of a document. It has the members of its object and the defaults of a
// new struct.
type docStruct struct {
	typ      reflect.Type
	members  map[string]*docMember // the members of the object of the struct by their names
	names    []string              // the names of the members in sorted order, for a suggestion
	required []string              // the names of the members of the required fields, in field order
	inits    []docInit             // the defaults of a new struct
}

// add adds the member name with m to st.
//
// Error modes:
//   - an error that wraps ErrDeclaration when st has a member of that name.
func (st *docStruct) add(name string, m *docMember) error {
	if _, ok := st.members[name]; ok {
		return fmt.Errorf("%w: two fields of %s have the member %s", ErrDeclaration, st.typ, name)
	}
	st.members[name] = m
	return nil
}

// decode decodes obj into v. obj is the object of the member key, and v is an addressable struct of
// st. decode decodes the members in the order of their names, then checks the members of the
// required fields.
//
// Error modes:
//   - an *Error that wraps ErrUnknownKey for a member without a field.
//   - an *Error that wraps ErrMissing for a required field whose member obj lacks or has nil.
//   - the errors of part.decode and of fieldAt.
func (st *docStruct) decode(key string, obj map[string]any, v reflect.Value) error {
	path := keyPrefix(key)
	for _, name := range slices.Sorted(maps.Keys(obj)) {
		m, ok := st.members[name]
		if !ok {
			return &Error{
				Err:        ErrUnknownKey,
				Input:      path + name,
				Suggestion: suggest(name, st.names, path),
				text:       "unknown key " + path + name,
			}
		}
		member := obj[name]
		if member == nil {
			continue
		}
		field, err := fieldAt(v, m.steps)
		if err != nil {
			return err
		}
		if err := m.part.decode(path+name, member, field); err != nil {
			return err
		}
	}
	for _, name := range st.required {
		if obj[name] == nil {
			return &Error{Err: ErrMissing, Input: path + name, text: "missing " + path + name}
		}
	}
	return nil
}

// replace returns a pointer to a copy of the target of the pointer v, or to a new struct of st when
// v is nil.
//
// Error modes:
//   - the errors of create.
func (st *docStruct) replace(v reflect.Value) (reflect.Value, error) {
	if v.IsNil() {
		return st.create()
	}
	target := reflect.New(st.typ)
	target.Elem().Set(v.Elem())
	return target, nil
}

// create returns a pointer to a new struct of st with its zero value and the defaults of its fields
// and of its groups.
//
// Error modes:
//   - the errors of setDefaults.
func (st *docStruct) create() (reflect.Value, error) {
	target := reflect.New(st.typ)
	if err := st.setDefaults(target.Elem()); err != nil {
		return reflect.Value{}, err
	}
	return target, nil
}

// setDefaults sets each field of v that has a default to a new value of the default. It sets the
// fields of the groups of v in the same way. v is a struct of st. setDefaults converts the text of
// each default again, so two new structs do not share a value of a default.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a default that does not convert.
func (st *docStruct) setDefaults(v reflect.Value) error {
	for i := range st.inits {
		d := &st.inits[i]
		field := v.FieldByIndex(d.index)
		if d.group != nil {
			if err := d.group.setDefaults(field); err != nil {
				return err
			}
			continue
		}
		if reason := d.opt.withDefault(&d.field); reason != "" {
			return fmt.Errorf("%w: the field %s %s", ErrDeclaration, d.path, reason)
		}
		field.Set(d.opt.def)
	}
	return nil
}

// docMember is a member of the object of a struct. It has the steps from the struct to the field of
// the member and the part of the type of the field.
type docMember struct {
	part  *part
	steps []step
}

// step is a field on the way from a struct to the field of a member. target is the struct that the
// field points to when the field is a pointer, and nil for any other field.
type step struct {
	target *docStruct
	index  int
}

// docInit is the default of a field of a new struct, or the defaults of a group of the struct.
type docInit struct {
	opt   *option    // the option of a field with a default, or nil for a group
	group *docStruct // the struct of a group, or nil for a field
	path  string     // the Go path of the field in a message, such as main.Section.Main
	index []int      // the index sequence of the field or of the group in the struct
	field reflect.StructField
}

// document is the declaration of the types of one call of [Decode]. It has the conversions of the
// call, the struct of each type that the call declares and the part of each field of those types.
type document struct {
	conv    conversions
	structs map[reflect.Type]*docStruct
	fields  map[fieldKey]*part
}

// fieldKey is a field of a struct type, by the type and the index of the field.
type fieldKey struct {
	typ   reflect.Type
	index int
}

// partOf returns the part of the type t. f is the field of t in the struct of the scope s, or nil
// for the value of Decode.
//
// Error modes:
//   - the errors of structOf, of listOf and of valueOf.
func (d *document) partOf(t reflect.Type, f *reflect.StructField, s *scope) (*part, error) {
	switch target := d.conv.target(t); {
	case target != nil:
		st, err := d.structOf(target)
		if err != nil {
			return nil, err
		}
		p := &part{typ: t, st: st, form: structForm}
		if t.Kind() == reflect.Pointer {
			p.form = pointerForm
		}
		return p, nil
	case t.Kind() == reflect.Slice && d.conv.target(t.Elem()) != nil,
		t.Kind() == reflect.Map && d.conv.converter(t.Key()) != nil && d.conv.target(t.Elem()) != nil:
		return d.listOf(t, f, s)
	default:
		return d.valueOf(t, f, s)
	}
}

// listOf returns the part of t. t is a list or a map of structs or of pointers to structs. f is the
// field of t in the struct of the scope s, or nil for the value of Decode.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a tag of listRejected on f.
//   - the errors of structOf.
func (d *document) listOf(t reflect.Type, f *reflect.StructField, s *scope) (*part, error) {
	if f != nil {
		if err := rejectTags(f, s.path+"."+f.Name, listRejected[:]); err != nil {
			return nil, err
		}
	}
	st, err := d.structOf(d.conv.target(t.Elem()))
	if err != nil {
		return nil, err
	}
	p := &part{typ: t, st: st, form: listForm, ptr: t.Elem().Kind() == reflect.Pointer}
	if t.Kind() == reflect.Map {
		p.form, p.key = mapForm, d.conv.converter(t.Key())
	}
	return p, nil
}

// valueOf returns the part of t. t is a value, a list or a map of values, or a pointer to a value. f
// is the field of t in the struct of the scope s, or nil for the value of Decode.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a tag of valueRejected on f, and for a type without a
//     conversion.
//   - the errors of newOption.
func (d *document) valueOf(t reflect.Type, f *reflect.StructField, s *scope) (*part, error) {
	if f == nil {
		o := &option{value: reflect.New(t).Elem()}
		if reason := d.conv.setShape(o, t); reason != "" {
			return nil, fmt.Errorf("%w: the value of Decode %s", ErrDeclaration, reason)
		}
		return &part{typ: t, opt: o, form: valueForm}, nil
	}
	if err := rejectTags(f, s.path+"."+f.Name, valueRejected[:]); err != nil {
		return nil, err
	}
	o, err := d.conv.newOption(f, reflect.New(t).Elem(), s)
	if err != nil {
		return nil, err
	}
	return &part{typ: t, opt: o, form: valueForm}, nil
}

// fieldPart returns the part of the field f of the struct type t in the scope s. It declares each
// field once in a call of Decode, so the default of a field that two structs contain through an
// embedded pointer converts once.
//
// Error modes:
//   - the errors of partOf.
func (d *document) fieldPart(t reflect.Type, f *reflect.StructField, s *scope) (*part, error) {
	k := fieldKey{typ: t, index: f.Index[0]}
	if p, ok := d.fields[k]; ok {
		return p, nil
	}
	p, err := d.partOf(f.Type, f, s)
	if err != nil {
		return nil, err
	}
	d.fields[k] = p
	return p, nil
}

// structOf returns the struct of the struct type t. It declares each type once in a call of Decode,
// so it also declares a type that refers to itself through a pointer, a list or a map once.
//
// Error modes:
//   - the errors of addFields.
func (d *document) structOf(t reflect.Type) (*docStruct, error) {
	if st, ok := d.structs[t]; ok {
		return st, nil
	}
	st := &docStruct{typ: t, members: map[string]*docMember{}}
	d.structs[t] = st
	if err := d.addFields(st, t, nil, &scope{path: t.String()}); err != nil {
		return nil, err
	}
	st.names = slices.Sorted(maps.Keys(st.members))
	return st, nil
}

// addFields adds a member to st for each exported field of the struct type t in the scope s. st
// contains t at the steps prefix. An embedded struct and a group with an empty prefix add their
// fields in the same way. An exported embedded pointer to a struct adds its fields through a step of
// the pointer. A default of t is a default of st when no step of prefix is a pointer.
//
// Error modes:
//   - an error that wraps ErrDeclaration for an embedded pointer to a struct that contains the
//     pointer, and for a field whose name is not a name of lower-case words.
//   - the errors of nest, of fieldPart, of flatten and of docStruct.add.
func (d *document) addFields(st *docStruct, t reflect.Type, prefix []step, s *scope) error {
	own := !slices.ContainsFunc(prefix, func(p step) bool { return p.target != nil })
	for f := range t.Fields() {
		steps := append(slices.Clone(prefix), step{index: f.Index[0]})
		field := s.path + "." + f.Name
		target := d.conv.target(f.Type)
		pointer := f.Type.Kind() == reflect.Pointer
		if !f.IsExported() && (!f.Anonymous || target == nil || pointer) {
			continue
		}
		name, required := "", false
		if target != nil {
			nested, err := s.nest(&f)
			if err != nil {
				return err
			}
			if len(nested.groups) == 0 {
				if err := d.flatten(st, f.Type, steps, &nested); err != nil {
					return err
				}
				continue
			}
			name = nested.groups[0]
		}
		p, err := d.fieldPart(t, &f, s)
		if err != nil {
			return err
		}
		switch {
		case target != nil:
			if own && !pointer {
				st.inits = append(st.inits, docInit{group: p.st, index: indexes(steps)})
			}
		case p.opt != nil:
			name, required = p.opt.name, p.opt.required
			if own && p.opt.hasDef {
				st.inits = append(st.inits, docInit{opt: p.opt, path: field, index: indexes(steps), field: f})
			}
		default:
			if name, err = fieldName(&f, field); err != nil {
				return err
			}
			_, required = f.Tag.Lookup(string(tagRequired))
		}
		if err := st.add(name, &docMember{part: p, steps: steps}); err != nil {
			return err
		}
		if required {
			st.required = append(st.required, name)
		}
	}
	return nil
}

// flatten adds the fields of t to st at steps in the scope s. t is the type of an embedded struct,
// of an embedded pointer to a struct or of a group with an empty prefix. For a pointer, flatten sets
// the target of the last step to the struct of the pointer.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a pointer to a struct that contains the pointer.
//   - the errors of structOf and of addFields.
func (d *document) flatten(st *docStruct, t reflect.Type, steps []step, s *scope) error {
	if t.Kind() != reflect.Pointer {
		return d.addFields(st, t, steps, s)
	}
	cycle := t.Elem() == st.typ || slices.ContainsFunc(steps, func(p step) bool {
		return p.target != nil && p.target.typ == t.Elem()
	})
	if cycle {
		return fmt.Errorf("%w: the field %s points to the struct %s, which contains the field",
			ErrDeclaration, s.path, t.Elem())
	}
	target, err := d.structOf(t.Elem())
	if err != nil {
		return err
	}
	steps[len(steps)-1].target = target
	return d.addFields(st, t.Elem(), steps, s)
}

// Decode decodes the member key of tree into the value that v points to. tree is a tree that a
// [Format] returns. key joins the names of the objects of the member with dots, as in go.lint. The
// empty key selects the whole tree. A key that tree does not contain and a member with a nil value
// leave the value unchanged.
//
// Decode matches each member of an object to the field with the same name, as a run matches the
// keys of a config file to options. A field without a member keeps its value. An array replaces a
// list and an object replaces a map. Each element of a list of structs and each value of a map of
// structs is an object, also when the list or the map contains pointers to structs. Decode creates
// the struct of each with its zero value and the defaults of its tags. A pointer to a struct takes
// an object. Decode creates the target of a nil pointer in the same way, and decodes into a copy of
// the target of any other pointer. Any other value converts from its text by the rules of options.
// The conversions of [App.Convert] do not apply.
//
// Decode reads the tags name, prefix, default, enum and required of a field. It applies default to
// each struct that it creates and not to the value that v points to.
//
// Decode changes the value only when it returns nil. It does not write into a list, a map or the
// target of a pointer that the value contains. It does not modify tree or keep a map, a slice or a
// pointer of tree in the value, so concurrent calls can share one tree.
//
// Error modes:
//   - an error that wraps [ErrDeclaration] when v is not a non-nil pointer, and for a key with an
//     empty name.
//   - an error that wraps [ErrDeclaration] for a type that Decode cannot decode into, and for a field
//     with the tag arg, short, env, key or secret.
//   - an [*Error] that wraps [ErrUnknownKey] for a member without a field.
//   - an [*Error] that wraps [ErrInvalidValue] for a value that does not convert, a value outside
//     the enum of its field, a nil element of an array or of an object, and a member whose kind
//     the field does not take, such as an array for a field of one value.
//   - an [*Error] that wraps [ErrMissing] for an object without the member of a required field.
func Decode(tree map[string]any, key string, v any) error {
	ptr := reflect.ValueOf(v)
	switch {
	case ptr.Kind() != reflect.Pointer:
		return fmt.Errorf("%w: Decode needs a non-nil pointer, not %T", ErrDeclaration, v)
	case ptr.IsNil():
		return fmt.Errorf("%w: Decode needs a non-nil pointer, not a nil %s", ErrDeclaration, ptr.Type())
	case key != "" && slices.Contains(strings.Split(key, keySeparator), ""):
		return fmt.Errorf("%w: the key %q has an empty name", ErrDeclaration, key)
	}
	d := &document{conv: builtins(), structs: map[reflect.Type]*docStruct{}, fields: map[fieldKey]*part{}}
	p, err := d.partOf(ptr.Type().Elem(), nil, nil)
	if err != nil {
		return err
	}
	member, found, err := lookup(tree, key)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	value := reflect.New(ptr.Type().Elem()).Elem()
	value.Set(ptr.Elem())
	if err := p.decode(key, member, value); err != nil {
		return err
	}
	ptr.Elem().Set(value)
	return nil
}

// lookup returns the member key of tree, and reports false for a key that tree does not contain or
// whose member is nil. It returns tree itself for the empty key.
//
// Error modes:
//   - an *Error that wraps ErrInvalidValue for a name of key whose member is not an object.
func lookup(tree map[string]any, key string) (any, bool, error) {
	var member any = tree
	if key == "" {
		return member, true, nil
	}
	names := strings.Split(key, keySeparator)
	for i, name := range names {
		obj, ok := member.(map[string]any)
		if !ok {
			return nil, false, invalid(strings.Join(names[:i], keySeparator), member, wantObject)
		}
		if member = obj[name]; member == nil {
			return nil, false, nil
		}
	}
	return member, true, nil
}

// invalid returns an *Error that wraps ErrInvalidValue for member with reason. member is the member
// key of a tree. invalid quotes the text of a member that is a value, and it writes tree in place of
// the empty key.
func invalid(key string, member any, reason string) error {
	label := key
	if key == "" {
		label = treeLabel
	}
	e := &Error{Err: ErrInvalidValue, Input: key, text: "invalid " + label + ": " + reason}
	switch member.(type) {
	case nil, map[string]any, []any:
	default:
		e.Value = text(member)
		e.text = "invalid " + label + " " + strconv.Quote(e.Value) + ": " + reason
	}
	return e
}

// keyPrefix returns key followed by a dot, or the empty string for the empty key. The key of a
// member below key is keyPrefix(key) followed by the name of the member.
func keyPrefix(key string) string {
	if key == "" {
		return ""
	}
	return key + keySeparator
}

// fieldAt returns the field at steps in the struct v. It replaces the pointer of each step that is a
// pointer with a new target or with a copy of its target. The field is then never inside a target of
// the caller.
//
// Error modes:
//   - the errors of docStruct.replace.
func fieldAt(v reflect.Value, steps []step) (reflect.Value, error) {
	for _, s := range steps {
		v = v.Field(s.index)
		if s.target == nil {
			continue
		}
		target, err := s.target.replace(v)
		if err != nil {
			return reflect.Value{}, err
		}
		v.Set(target)
		v = target.Elem()
	}
	return v, nil
}

// indexes returns the index sequence of the fields of steps, as reflect.Value.FieldByIndex takes it.
func indexes(steps []step) []int {
	index := make([]int, len(steps))
	for i, s := range steps {
		index[i] = s.index
	}
	return index
}

// rejectTags returns an error that wraps ErrDeclaration for the first tag of rejected that the field
// f has. field is the Go path of f in the message.
func rejectTags(f *reflect.StructField, field string, rejected []tag) error {
	for _, t := range rejected {
		if _, ok := f.Tag.Lookup(string(t)); ok {
			return fmt.Errorf("%w: the field %s takes no tag %s in a document", ErrDeclaration, field, t)
		}
	}
	return nil
}

// target returns the struct type that t is or points to when neither t nor the struct has a
// conversion in c, and nil for any other type.
func (c conversions) target(t reflect.Type) reflect.Type {
	if c.converter(t) != nil {
		return nil
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || c.converter(t) != nil {
		return nil
	}
	return t
}

// want returns the reason of an error for a member of a kind that an option of the shape s does not
// take.
func (s shape) want() string {
	switch s {
	case list:
		return wantArray
	case dict:
		return wantObject
	default:
		return wantValue
	}
}

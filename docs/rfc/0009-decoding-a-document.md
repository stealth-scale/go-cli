---
rfc: 0009
title: Decoding a document into the values of a program
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-16
updated: 2026-10-16
discussion: https://github.com/stealth-scale/go-cli/issues/13
supersedes: none
superseded-by: none
produces-adr: tbd
---

<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0009: Decoding a document into the values of a program

## Summary

`cli.Decode` decodes one member of the tree of a `Format` into a value of the program by the rules of config keys. Decode matches each member of an object to the field with the same name, and a member without a field fails with `ErrUnknownKey`. A field without a member keeps its value, so the value can start at the baseline of the program. An array or an object replaces the list or the map of the value. A list or a map of structs takes an array or an object of objects, and a pointer to a struct takes an object. Decode creates each struct of a list or a map and the target of a nil pointer with the defaults of their tags. Every other value converts from its text by the rules of the options of a command.

## Motivation

ergon keeps the options of each of its producers in a section of `.ergon.yaml`, such as `go` or `license`. It decodes each section into the struct of the producer. The struct contains the baseline values of the producer before the section applies:

- A key without a field is an error.
- A field without a key keeps its baseline value.
- A list or a map of the section replaces the list or the map of the baseline.

At 02cba2f ergon runs its command line on go-cli and decodes `.ergon.yaml` with viper v1.21.0. `Viper.UnmarshalKey` decodes each section through mapstructure v2.5.0. ergon sets six options of that decoder, and four of them decide the rules of a section:

- `TagName` reads the name of each key from the yaml tag of its field.
- `ErrorUnused` fails for a key without a field.
- `ZeroFields` makes a new list, map or pointer in place of the one of the baseline.
- `WeaklyTypedInput` is false, so a value of another kind than its field fails.

go-cli v0.5.1 cannot decode such a section:

- A list of structs has no conversion. ergon has four fields of that kind: `ci.steps` in two types, `go.binaries` and `license.directories`. A command with such an option fails to declare:

  ```text
  cli: invalid declaration: the field main.Section.Binaries has the type []main.Command, which has no conversion
  ```

- A run decodes the options of a command into a new value of their type. Only the tag `default` sets a value before the sources apply, and the baseline of a producer is a value that the producer builds.
- The reading of a config file turns each value into text. An object inside an array becomes the text that `fmt.Sprint` writes for a map.
- `App.IgnoreKeys` skips the keys of a document, so the program decodes them with a decoder of its own.

A decoder of its own gives the program two sets of rules for one file. A probe decoded a section with viper v1.21.0, mapstructure v2.5.0 and the options of the decoder of ergon:

- The step `{env: {GOFLAGS: -mod=mod}, run: [make]}` decoded with the variable `goflags`, because viper lowercases every key of the tree.
- `timeout:` without a value set the baseline `Timeout` of 15 to 0, because `ZeroFields` sets a field to its zero value for a null.
- `timeout: "30"` failed with `expected type 'int', got unconvertible type 'string'`.

The names differ as well. go-cli derives the name of a field from its kebab case. Outside its tests the Go files of ergon at 02cba2f declare 206 struct fields with a yaml name. For 195 of them the yaml name equals the kebab case. Ten of the other 11 are options of producers, and each of the ten needs the tag `name`. One of them is the key `golangci-lint` of the field `GolangCILint`. The kebab case of that field name is `golang-ci-lint`.

The package already derives the name of each field and converts text to the type of a field. It also reports an unknown key with a suggestion. `Decode` applies the same rules to the document of a program, so the program has one set of names and conversions for its config file and its document.

## Detailed design

### Components

| Component | Change | Responsibility |
|---|---|---|
| `Decode` | New | Decodes a member of a tree into a value of the program |
| The fields of a document | New | Derive the member and the conversion of each field of a type by the rules of options |
| The lookup of conversions | Changed | Takes a set of conversions, so `Decode` uses the built-in set without an `App` |
| `Error` and its sentinels | Changed | Report the members of a document as well as the inputs of a run |

### API

```go
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
func Decode(tree map[string]any, key string, v any) error
```

### Example

A program declares the section `go` of its document and decodes it over a baseline value:

```go
type Command struct {
	Name      string `required:""`
	Main      string `default:"."`
	Platforms []string
}

type Fuzz struct {
	Time string
	Args []string
}

type Sign struct {
	Key string `required:""`
}

type Section struct {
	Binaries []Command
	Fuzz     Fuzz
	Sign     *Sign
	Timeout  int
}

tree, err := yaml.Decode(data)
if err != nil {
	return nil, err
}
s := &Section{Fuzz: Fuzz{Time: "30s"}, Timeout: 15} // the baseline
if err := cli.Decode(tree, "go", s); err != nil {
	return nil, err
}
```

For this document Decode returns nil:

```yaml
go:
  binaries:
    - name: ergon
      platforms: [linux/amd64, darwin/arm64]
  fuzz:
    args: [-fuzzminimizetime=2s]
```

- `Binaries` contains the command `ergon` with the default `.` of `Main` and two platforms.
- `Fuzz.Time` keeps `30s`, and `Fuzz.Args` contains one argument.
- `Sign` is nil, because the document has no member `sign`.
- `Timeout` keeps 15.

A misspelled key, a command without a name and a timeout that is not a number fail with these errors:

```text
cli: unknown key go.binaries[0].nme (did you mean go.binaries[0].name?)
cli: missing go.binaries[0].name
cli: invalid go.timeout "soon": invalid syntax
```

A program with a section for each of its producers keeps the options of a producer as a value of an interface. Decode takes the pointer inside that value:

```go
type Options interface {
	Validate() error
}

func decodeSection(tree map[string]any, name string, o Options) error {
	if err := cli.Decode(tree, name, o); err != nil {
		return fmt.Errorf("main: %w", err)
	}
	return o.Validate()
}
```

### The fields of a document

Decode derives the member of a field as a run derives the config key of an option:

- The name of a member is the tag `name` or the name of its field in kebab case.
- A named struct field without a conversion is a group. Its object contains the members of its fields. The tag `prefix` replaces the name of the group, and an empty prefix adds the fields of the group to the object of its parent.
- An embedded struct adds its fields to the object of its parent, also when its type is unexported. An exported embedded pointer to a struct adds its fields in the same way. Decode skips every other unexported field.
- A struct that embeds a pointer to itself, directly or through other structs, fails with `ErrDeclaration`.
- A named field of a pointer to a struct without a conversion is a group behind a pointer. Its member is an object, as for a group.
- Decode creates the target of a nil pointer and copies the target of any other pointer when the object contains a member of the target. A pointer without such a member keeps its value.
- The name of a member follows the rule of every name of the package. It is words of lower-case ASCII letters and digits that single dashes join.
- Decode compares the name of a member exactly. viper and mapstructure compare names without regard to case.

Each field takes a member by its type:

| The type of the field | The member | The result |
|---|---|---|
| A type with a conversion, such as `int`, `time.Duration` or a type with `UnmarshalText` | A value | The text of the value converts to the type |
| A pointer to such a type | A value | A new pointer to the value |
| A list of such a type | An array of values or one text that commas separate | A new list replaces the list |
| A map whose keys and values have a conversion | An object of values or one text of `key=value` entries that commas separate | A new map replaces the map |
| A group | An object | Decode sets the field of each member, and every other field of the group keeps its value |
| A pointer to a struct | An object | Decode decodes the object into a new target or into a copy of the target |
| A list of structs or of pointers to structs | An array of objects | A new list of new structs replaces the list |
| A map of structs or of pointers to structs whose keys have a conversion | An object of objects | A new map of new structs replaces the map |

- Decode skips a member with a nil value, as a run skips such a key of a config file. Its field keeps its value. A nil element of an array and a nil value of an entry fail with `ErrInvalidValue`.
- The text of a value is the result of `MarshalText` for an `encoding.TextMarshaler`, and the result of `fmt.Sprint` for any other value. The text of a JSON number is its digits, and the module yaml returns the source text of each scalar.
- A field of any other type fails with `ErrDeclaration`, such as an interface, a function, a pointer to a pointer, a list of pointers to strings or a map of lists.

Decode reads five tags, ignores `help` and rejects five others:

| Tag | In a document |
|---|---|
| `help` | Decode does not read it. The program can show it in a comment of its document |
| `name` and `prefix` | Rename a member and a group, as for an option |
| `default` | Sets the field in a struct that Decode creates. Decode does not apply it to the value that `v` points to |
| `enum` | Lists the values that the field takes |
| `required` | Makes Decode fail for an object of the struct without the member |
| `arg`, `short`, `env`, `key` and `secret` | Fail with `ErrDeclaration` |

The tags `arg`, `short`, `env`, `key` and `secret` declare a positional argument, a flag, an environment variable, a config key of a run, and a value that the help and `--flag-debug` mask. Documents have none of these, so such a tag is a mistake of the program. A field with both `required` and `default` fails with `ErrDeclaration` in a document and in the options of a command.

A run creates the options of a command and applies their defaults before its sources. Decode applies the defaults of a struct in the same way when it creates the struct. serde fills each missing field of a struct from the `Default` of the struct under its container attribute `default`. Decode does not apply the defaults to the value that `v` points to. Its fields keep the values of the caller.

### Keys in messages

- The key of a member in an error starts at the root of `tree`, such as `go.timeout`.
- An element of an array adds its index from 0 in brackets, such as `go.binaries[1].name`. mapstructure writes the same form.
- An entry of a map adds its name after a dot, such as `plans.api.enabled`.
- The suggestion for an unknown member is the closest member of the same struct within two edits. The suggestion contains the key of the struct before the name.

### The order of a decode

1. Decode checks the type of the value, the types of its groups, and the types of the elements of its lists and maps. A declaration error does not depend on `tree`.
2. Decode finds the member `key` in `tree`. A key that `tree` does not contain and a member with a nil value return nil.
3. Decode decodes the member into a copy of the value. It visits the members of each object in the order of their names and the elements of an array in their order. Then it checks the required fields of the object.
4. At the first error Decode returns the error and leaves the value unchanged. Otherwise it sets the value to the copy and returns nil.

Decode replaces each list, map and pointer that it decodes. It never writes into a list, a map or the target of a pointer that the value contains, so a shallow copy of the value is enough for step 3.

### Invariants

1. Decode changes the value only when it returns nil.
2. Decode does not write into a list, a map or the target of a pointer that the value contains.
3. Decode does not modify `tree`, and the value does not share a map, a slice or a pointer with `tree`.
4. Every member of an object that Decode decodes into a struct matches a field, or Decode fails with `ErrUnknownKey`.
5. A field without a member keeps its value. A struct that Decode creates starts with its zero value and the defaults of its tags, and so does the target of a nil pointer.
6. A value converts from its text with the conversion that an option of the same type uses in an app without `App.Convert`.
7. Decode returns the same error for the same tree, key and value.
8. A run reads its command line, its environment variables and its config file as before. The options of a command do not take a list of structs.
9. The package imports only the standard library.

### Failure handling

| Case | Behaviour |
|---|---|
| `v` is nil, not a pointer, or a nil pointer | `Decode` returns an error that wraps `ErrDeclaration` and does not read `tree` |
| `key` has an empty name, such as `go..lint` | `Decode` returns an error that wraps `ErrDeclaration` |
| A field has a type that Decode does not take, such as `[]*string` | `Decode` returns an error that wraps `ErrDeclaration` |
| A field has the tag `arg`, `short`, `env`, `key` or `secret` | `Decode` returns an error that wraps `ErrDeclaration` |
| `tree` does not contain `key`, or the member has a nil value | `Decode` returns nil and leaves the value unchanged |
| A name of `key` is not an object in `tree` | An `*Error` that wraps `ErrInvalidValue` |
| A member has no field | An `*Error` that wraps `ErrUnknownKey`, with a suggestion |
| A value does not convert or the enum of its field excludes it | An `*Error` that wraps `ErrInvalidValue` |
| A member has a kind that its field does not take, such as a value for a group | An `*Error` that wraps `ErrInvalidValue` |
| An element of an array or the value of an entry is nil | An `*Error` that wraps `ErrInvalidValue` |
| An object lacks the member of a required field | An `*Error` that wraps `ErrMissing` |
| An `UnmarshalText` or a `Set` method panics | The package does not recover the panic, as in a run |
| A handler returns an `*Error` of `Decode` | `App.Main` writes it and returns `ExitUsage`, as for an invalid config file |

### Cost

- Decode visits each member below `key` once, and it declares each field of the types below the value once.
- It allocates the copy of the value, the copy of each target that it decodes into, and each list, map, pointer and struct that it creates. Its conversions also allocate the texts of the values. The number of allocations grows linearly with the number of members.
- A program decodes a document once in a run, so `Decode` has no allocation contract and no benchmark. Nobody has measured its cost.

### Capabilities left out

- **The conversions of `App.Convert`.** Decode has no app. ergon decodes its sections in a library package that has no app, and ergon at 02cba2f does not register a conversion. A document whose type needs a registered conversion reverses this verdict.
- **A null that clears a field.** A nil value in a tree of a `Format` is no value. A null in a document leaves the field unchanged, as a null in a config file leaves an option unchanged. A program writes `[]` or `{}` for an empty list or map. A format whose only empty value is null reverses this verdict.
- **Names without case, and map keys in lower case.** Decode compares names exactly and keeps the case of each map key, so the variable `GOFLAGS` of a step keeps its name. A document whose authors write names in another case reverses this verdict.
- **Lines in messages.** A `Format` returns a tree without lines, so a message contains the key of a member and not its line. A `Format` that returns lines reverses this verdict.
- **Validate methods.** Decode does not call a Validate method. After the decode ergon validates each option and then the rules between the options of a section. Those rules are rules of the producer. A second program that repeats the same walk of Validate methods reverses this verdict.
- **Writing a value into a tree.** ergon writes `.ergon.yaml` through the nodes of yaml.v3 to keep its comments. A program that writes its document through the package reverses this verdict.
- **The tree of the config file of a run.** A handler that decodes the ignored keys of its config file opens the file again through `Run.Open`. ergon has no config file option, because c62ac29 removed its flag `--config`. Programs that decode the ignored keys of their config file in a handler reverse this verdict.

### Changes outside the package

- The example `16-workspace` decodes its plans with `cli.Decode` in place of `encoding/json`. A plan gains a list of steps and the default `true` of its field `Enabled`. The runs of the example show a plan with the default, a step, and a misspelled key of a plan with its suggestion.
- The README gains a section on documents after the section on config files. The documentation of the package gains the section `# Documents`.
- A minor changeset adds `cli.Decode` to the module cli, and a patch changeset covers the example.

### Migration

Programs do not need to change. A program that moves a document from viper's `UnmarshalKey` to `Decode` meets these differences. A probe measured the first four with viper v1.21.0 and mapstructure v2.5.0 and the options `ErrorUnused` and `ZeroFields`:

| The document | viper and mapstructure | Decode |
|---|---|---|
| `env: {GOFLAGS: -mod=mod}` | The key `goflags` | The key `GOFLAGS` |
| `timeout:` over the baseline 15 | 0 | 15 |
| `timeout: "30"` | An error | 30 |
| `Timeout: 30` | 30, because viper lowercases the key | An unknown key with the suggestion `timeout` |
| A field named by its yaml tag | The name of the yaml tag | The tag `name` or the kebab case of the field name |

ergon needs the tag `name` on ten fields of the options of its producers. Its embedded structs do not need a tag, because an embedded struct adds its fields without a group.

### Tests

- `decode_test.go` is the spec of `decode.go`. Its tables cover the names, the tags, the field types, the pointers, the nil values, the kinds of members, the keys that `key` selects, the suggestions and the errors of declarations.
- A property draws a value of a struct with a field of each type that Decode takes, and a document over the value. Decode returns the value with the fields that the document states, and every other field keeps its value. The property also checks that Decode does not write into a list, a map or the target of a pointer of the value.
- A property fails Decode with one fault in a generated document and checks that the value is unchanged.
- `FuzzDecode` decodes generated JSON trees into the struct of the first property. Decode does not panic, and it refuses a tree with an `*Error` and leaves the value unchanged. A test runs the same property.
- The specs of the options pass without a change.
- Coverage and the mutation score are 100%.

## Alternatives considered

### The function of the issue, with a type parameter

`func Decode[T any](tree map[string]any, v *T) error` checks at compile time that `v` is a pointer.

**Why not:** ergon keeps the options of a producer as a value of the interface `language.Options`. Go 1.27.2 infers no `T` from such a value and reports `type Options of o does not match *T (cannot infer T)`. `json.Unmarshal` and `mapstructure.Decode` also take `any`.

### A method of App

`func (a *App[G]) Decode(tree map[string]any, key string, v any) error` applies the conversions of `App.Convert` too.

**Why not:** ergon decodes its sections in a library package that has no app, and ergon does not register a conversion. A method makes every caller pass the app through each package between the command and the document.

### A method of Run

`run.Decode(key, v)` decodes a member of the config file that the run has read. The handler then does not read the file a second time.

**Why not:** ergon reads `.ergon.yaml` without a config file option, because c62ac29 removed its flag `--config`.

### Lists of structs in the options of a command

The options of a command take a list of structs from the config file of the run.

**Why not:** a flag and an environment variable have no form for a list of structs, so such an option takes a value from one source alone. A run also does not give the options a baseline value.

### The options of mapstructure

Decode takes options such as `ErrorUnused`, `ZeroFields` and `WeaklyTypedInput`.

**Why not:** each option doubles the behaviours that a program and the tests of the package have to know. ergon sets six options of mapstructure for its sections.

### Merge semantics

`encoding/json/v2` in Go 1.27.2 merges an object into a map and into a struct. Its documentation calls these semantics similar to JSON Merge Patch. yaml.v3 v3.0.5 and BurntSushi/toml v1.6.0 also merge an object into an existing map. `encoding/json`, BurntSushi/toml and mapstructure without `ZeroFields` decode an array into the elements of an existing list. yaml.v3 makes a new list for each sequence, as Decode does.

**Why not:** a merge cannot remove an entry of the baseline map or an element of the baseline list. JSON Merge Patch removes a member with null, and a nil value in a tree of a `Format` is no value. A merge into the elements of a list also mixes two elements. In a probe with Go 1.27.2 `encoding/json` decoded `{"steps": [{"name": "b"}]}` over the step `{Name: a, Run: [x]}` into `{Name: b, Run: [x]}`.

### A decoder of the format beside the package

The program decodes its document with yaml.v3 or `encoding/json` and its config file with the package.

**Why not:** the program then has the names of yaml tags and the kinds of YAML in its document, and the names and the conversions of the package in its config file. In the earlier probe the two decoders differ in the case of a key, in a null and in the kind of a number.

## Drawbacks

- The package gains the function `Decode`. The documentation of `Error`, `ErrUnknownKey`, `ErrInvalidValue`, `ErrMissing` and `ErrDeclaration` grows to cover a document.
- A field of a document can be a list of structs, a map of structs or a pointer to a struct, and an option of a command cannot. The package has two rules for one field type.
- A program that keeps the old target of a pointer does not see the decoded values, because Decode decodes into a copy of the target.
- A type that `App.Convert` registers converts in a flag and fails with `ErrDeclaration` in a document.
- The tags `arg`, `short`, `env`, `key` and `secret` fail in a document, so a struct that uses them cannot be both the options of a command and a part of a document.
- `default` sets the fields of a struct that Decode creates and not the fields of the value that `v` points to. A reader of a struct has to know which of the two Decode received.
- A value converts from its text, so a document can write a number as a string. Decode cannot check the kind of a scalar.
- A failed Decode returns no part of the value, while a run applies every valid value of its sources.
- A message contains the key of a member and not its line.
- The functions that turn the fields of a struct into options take a set of conversions in place of the app. The change touches the unexported methods `options`, `newOption`, `setShape` and `converter` of `App`.

## Open questions

None.

## Unresolved and future work

- This RFC does not decode the ignored keys of the config file of a run without a second read of the file.
- This RFC does not write a value into a tree.

## References

| What | Where |
|---|---|
| Issue 13: a program cannot decode a section of its document into a struct with lists of structs | <https://github.com/stealth-scale/go-cli/issues/13> |
| ergon at 02cba2f: `UnmarshalKey` and the options of mapstructure | <https://github.com/dokimasia/ergon/blob/02cba2fd4238/ergon-service/baseline/options/resolve.go#L140-L182> |
| ergon at 02cba2f: `language.Options` and `language.Configurable` | <https://github.com/dokimasia/ergon/blob/02cba2fd4238/ergon-core/language/init.go#L117-L131> |
| ergon at 02cba2f: the four lists of structs | <https://github.com/dokimasia/ergon/blob/02cba2fd4238/ergon-core/option/ci.go#L64>, <https://github.com/dokimasia/ergon/blob/02cba2fd4238/ergon-core/option/ci.go#L102>, <https://github.com/dokimasia/ergon/blob/02cba2fd4238/ergon-lang-go/baseline/options.go#L86>, <https://github.com/dokimasia/ergon/blob/02cba2fd4238/ergon-service/licenses/config.go#L83> |
| viper v1.21.0: `UnmarshalKey` and the lowercasing of every key | <https://github.com/spf13/viper/blob/v1.21.0/viper.go#L929-L931>, <https://github.com/spf13/viper/blob/v1.21.0/util.go#L73-L100> |
| mapstructure v2.5.0: `ErrorUnused`, `ZeroFields`, `MatchName`, and the decoding of nil, maps, pointers and slices | <https://github.com/go-viper/mapstructure/blob/v2.5.0/mapstructure.go#L265-L285>, <https://github.com/go-viper/mapstructure/blob/v2.5.0/mapstructure.go#L347-L353>, <https://github.com/go-viper/mapstructure/blob/v2.5.0/mapstructure.go#L564-L577>, <https://github.com/go-viper/mapstructure/blob/v2.5.0/mapstructure.go#L1024-L1029>, <https://github.com/go-viper/mapstructure/blob/v2.5.0/mapstructure.go#L1330-L1344>, <https://github.com/go-viper/mapstructure/blob/v2.5.0/mapstructure.go#L1411-L1428> |
| Go 1.27.2: `encoding/json` decodes an array into existing elements and an object into an existing map | <https://github.com/golang/go/blob/go1.27.2/src/encoding/json/v2_decode.go#L71-L104> |
| Go 1.27.2: `encoding/json/v2` merges objects and resets slices | <https://github.com/golang/go/blob/go1.27.2/src/encoding/json/v2/arshal.go#L332-L385> |
| go-yaml v3.0.5: the decoding of sequences and of mappings into maps | <https://github.com/yaml/go-yaml/blob/v3.0.5/decode.go#L729-L760>, <https://github.com/yaml/go-yaml/blob/v3.0.5/decode.go#L822-L825> |
| BurntSushi/toml v1.6.0: the decoding of maps and slices | <https://github.com/BurntSushi/toml/blob/v1.6.0/decode.go#L343-L425> |
| serde: the container attributes `deny_unknown_fields` and `default` | <https://serde.rs/container-attrs.html> |
| RFC 7396: JSON Merge Patch | <https://www.rfc-editor.org/rfc/rfc7396> |
| The probes of viper, of `encoding/json`, of the yaml names of ergon and of the samples of this RFC against a stub of `Decode`, under Go 1.27.2 | Run on 2026-10-16 |

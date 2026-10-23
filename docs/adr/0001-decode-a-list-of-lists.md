---
adr: 0001
title: Decode takes a list of lists of values
status: Accepted
date: 2026-10-23
supersedes: RFC-0009, in part
superseded-by: none
rfc: RFC-0009
---

<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# ADR-0001: Decode takes a list of lists of values

## Status

Accepted

## Context

In a document a list of values takes an array of values, and a list of structs takes an array of objects. `Decode` fails with `ErrDeclaration` for a field of any other type, which includes a list of lists such as `[][]string`.

ergon's step `generate` needs a list of command lines. Each line is a list of words:

```yaml
go:
  generate:
    commands:
      - [go, run, go.dokimi.dev/dokimi/cmd/dokimi@main, run]
      - [go, generate, -run, go tool kanon]
```

go-cli v0.6.0 refuses the field of that list before it reads the document:

```text
cli: invalid declaration: the field main.generate.Commands has the type [][]string, which has no conversion
```

The YAML format of go-cli returns the list as an array of arrays of strings. yaml.v3 v3.0.5 and `encoding/json` of Go 1.27.2 decode the same arrays into a `[][]string`. Both refuse a string in place of an inner list. The INI format returns only texts. A flag and an environment variable also take one text, so none of the three has a form for a list of lists.

## Decision

We will let `Decode` take a list of lists of values from an array of arrays, because a list of command lines then does not need a key in each line.

- A list of lists of values is a slice whose elements are lists of values, such as `[][]string`.
- Each element of the array is an array of values. Each value converts by the rules of the values of a list field.
- `Decode` fails with `ErrInvalidValue` and the reason `want an array` for an element of another kind, such as nil, an object or a text. A list field also takes one text that commas separate, but an element of a list of lists does not.
- The key of a value in a message has the index of its list and its own index, such as `go.generate.commands[1][0]`.
- A list of lists takes the tags `name` and `required`. `Decode` fails with `ErrDeclaration` for the tags `default`, `enum` and `prefix`, as it does for a list of structs.
- `Decode` fails with `ErrDeclaration` for a list whose lists have no conversion, such as `[][]chan int` and `[][][]string`. It also fails for a map of lists and for a list of maps.
- The options of a command refuse a list of lists, as they refuse a list of structs.

## Alternatives Considered

### A struct for each list

The program declares each command line as a struct with one list, such as `[]struct{ Run []string }`. `Decode` takes such a field as a list of structs, so go-cli does not change. It lost because each line of the document then repeats the key `run`, and the key does not add information to the line.

### A text of values in place of an inner array

An element of the array also takes one text that commas separate, as a list field does. The issue that asked for lists of lists proposed this rule. It lost because no format needs a text inside an array. The text form also decodes the flat array `[go, generate]` into two command lines of one word each. Without it `Decode` refuses that array with `want an array`, as yaml.v3 and `encoding/json` refuse it.

## Consequences

**Positive:**

- ergon decodes its step `generate` into a `[][]string`, and each line of the document is the list of its words.
- `Decode` refuses a flat array for a list of lists, and its message contains the key of the first element.

**Negative:**

- A list of lists is a third kind of list beside a list of values and a list of structs. Its elements take an array alone, while a list of values also takes a text.
- A list of lists is one more type that a field of a document takes and an option of a command refuses.
- A list of lists does not take a default, so a program sets the baseline of such a list in its own code.
- A document with a map of lists needs a struct for each entry.

## References

| What | Where |
|---|---|
| Issue 17: a program cannot decode a list of lists, such as a list of command lines | <https://github.com/stealth-scale/go-cli/issues/17> |
| ergon issue 53: the step generate runs one command | <https://github.com/dokimasia/ergon/issues/53> |
| go-yaml v3.0.5: the decoding of a sequence into a slice | <https://github.com/yaml/go-yaml/blob/v3.0.5/decode.go#L729-L765> |
| The probe of yaml.v3 v3.0.5, `encoding/json` and the YAML and INI formats of go-cli under Go 1.27.2 | Run on 2026-10-23 |

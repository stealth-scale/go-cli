// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

//go:generate go run golang.org/x/tools/cmd/stringer@v0.52.0 -type=Kind -linecomment -output=source.string_gen.go

// Kind is the kind of source that set the value of an option.
//
// # Concurrency
//
// A Kind is a value, which any number of goroutines may read.
type Kind uint8

// The kinds of source, from the lowest precedence to the highest. A positional argument has a kind
// of its own, because only the command line sets one.
const (
	// Default is the tag default of a field, or the zero value of a field without one.
	Default Kind = 0 // default
	// File is a key of a config file.
	File Kind = 1 // file
	// Environment is an environment variable.
	Environment Kind = 2 // environment
	// Flag is a flag of the command line.
	Flag Kind = 3 // flag
	// Argument is a positional argument of the command line.
	Argument Kind = 4 // argument
)

// Valid reports whether k is one of the declared kinds.
func (k Kind) Valid() bool {
	return k <= Argument
}

// Source is the origin of the value of an option. The zero Source is the default of a field.
//
// # Concurrency
//
// A Source is a value, which any number of goroutines may read.
type Source struct {
	Name string // the config key, the environment variable, the flag or the argument, or empty
	File string // the path of the config file of a File source, or empty
	Kind Kind
}

// String returns the source as text, such as "env SRV_SERVE_ADDR", "flag --addr" or
// "key serve.addr of srv.json". A Default source and a source of an invalid kind return the text of
// their kind.
func (s Source) String() string {
	switch s.Kind {
	case File:
		return "key " + s.Name + " of " + s.File
	case Environment:
		return "env " + s.Name
	case Flag:
		return "flag " + s.Name
	case Argument:
		return "argument " + s.Name
	default:
		return s.Kind.String()
	}
}

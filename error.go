// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import "errors"

// prefix starts the text of every error of the package.
const prefix = "cli: "

// The sentinels of the errors of [App.Execute]. An [*Error] wraps one of the first seven, and an
// error that wraps one of the last two reports a defect of the program.
var (
	// ErrUnknownCommand reports a command that the app or the command before it does not declare.
	ErrUnknownCommand = errors.New("cli: unknown command")
	// ErrUnknownFlag reports a flag that the app and the commands of the path do not declare.
	ErrUnknownFlag = errors.New("cli: unknown flag")
	// ErrUnknownKey reports a key of a config file that no option of the app declares.
	ErrUnknownKey = errors.New("cli: unknown config key")
	// ErrMissing reports a missing command, a flag without its value, a missing positional argument
	// and a required option without a value.
	ErrMissing = errors.New("cli: missing input")
	// ErrInvalidValue reports a value that does not convert to the type of its field, a value that the
	// enum of its option excludes, and two or more values for an option that takes one.
	ErrInvalidValue = errors.New("cli: invalid value")
	// ErrUnexpectedArgument reports a positional argument that the command does not declare.
	ErrUnexpectedArgument = errors.New("cli: unexpected argument")
	// ErrConfigFile reports a config file without a format, a config file that the run cannot read,
	// and a config file that its format cannot decode.
	ErrConfigFile = errors.New("cli: unreadable config file")
	// ErrDeclaration reports a registration or options that the app cannot run. [App.Execute]
	// returns it before it reads the command line, for every command of the app.
	ErrDeclaration = errors.New("cli: invalid declaration")
	// ErrEnv reports a nil [Env], and an Env without Stdout or Stderr.
	ErrEnv = errors.New("cli: incomplete env")
)

// Error is an invalid input of a run: an argument, an environment variable or a config file. Err is
// one of [ErrUnknownCommand], [ErrUnknownFlag], [ErrUnknownKey], [ErrMissing], [ErrInvalidValue],
// [ErrUnexpectedArgument] and [ErrConfigFile], and errors.Is matches it.
//
// # Concurrency
//
// An Error does not change after [App.Execute] returns it, so any number of goroutines may read it.
type Error struct {
	Err        error  // the sentinel
	Command    string // the path of the command of the input, such as db migrate, or empty for the app
	Input      string // the command, the flag, the config key, the argument or the config file at fault
	Value      string // the value at fault, or empty
	Suggestion string // the name that the package suggests in place of Input, or empty
	text       string // the message without the package name, the command and the suggestion
	Source     Source // the source of Value, or the zero Source
}

// Error returns [Error.Message] after the package name, such as
// "cli: serve: unknown flag --adr (did you mean --addr?)".
func (e *Error) Error() string {
	return prefix + e.Message()
}

// Unwrap returns the sentinel of e.
func (e *Error) Unwrap() error {
	return e.Err
}

// Message returns the message of e after its command and before its suggestion, such as
// "serve: unknown flag --adr (did you mean --addr?)". The message does not contain the package
// name. [App.Main] writes it after the name of the app.
func (e *Error) Message() string {
	return message(e.Command, e.text, e.Suggestion)
}

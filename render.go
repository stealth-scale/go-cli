// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"io"
	"strings"
)

// Help contains the help of an app or of one of its commands for a [Renderer].
type Help struct {
	App     string // the name of the app
	Command string // the path of the command, such as db migrate, or empty for the help of the app
	Text    string // the help text of the app or of the command
	// Usage contains the parts of the usage line after the names, such as "[options]" and "[NAME]".
	Usage []string
	// Commands contains the commands of the app or the subcommands of the command in the order of
	// their registration.
	Commands []HelpCommand
	// Arguments contains the positional arguments of the command in the order of their fields.
	Arguments []HelpOption
	// Options contains the options of the command in the order of their flags. In the help of the
	// app, Options contains the global options, the flags of the help and the flag --version of an
	// app with a version.
	Options []HelpOption
	// Globals contains the options of the ancestors of the command in the order of their flags. The
	// flags of the help and the flag --version of an app with a version are among them. Globals is
	// empty in the help of the app.
	Globals []HelpOption
}

// HelpCommand is a command or a subcommand in the help.
type HelpCommand struct {
	Name string // the name of the command, such as serve
	Text string // the first paragraph of the help text of the command, on one line
}

// HelpOption is an option or a positional argument in the help.
type HelpOption struct {
	Name        string // the flag such as --addr, or the label of an argument such as NAME
	Short       string // the short flag such as -a, or empty
	Placeholder string // the type or the values such as string or dev|prod, or empty for a bool
	Text        string // the help text
	Default     string // the default in quotes such as ":8080" or "***", or empty without one
	Env         string // the environment variable, or empty
	Key         string // the config key, or empty without one or on a path without a config file
	Required    bool   // a run fails when no source sets the option
}

// Version contains the version of an app for a [Renderer].
type Version struct {
	App  string // the name of the app
	Text string // the text that App.Version registers
}

// DebugRow is a row of the table of --flag-debug.
type DebugRow struct {
	Option string // the flag, or the label of a positional argument
	Value  string // the value as the table prints it, with *** for a secret option
	Source Source // the source of the value
}

// Failure is the error of a run as [App.Main] writes it.
type Failure struct {
	Err        error  // the error of the run
	App        string // the name of the app
	Command    string // the path of the command of the run, such as db migrate, or empty
	Message    string // the message without the names and the suggestion
	Suggestion string // the name that the package suggests in place of the input, or empty
	Usage      bool   // the error is an invalid command line, environment variable or config file
}

// Renderer writes the help, the version, the message of a failure and the table of --flag-debug in
// place of the package. environ is the environment of the run as KEY=value pairs, which
// [Env.Environ] returns.
//
// The package calls a Renderer on the goroutine of [App.Execute] or [App.Main], and returns the
// error of Help, of Version and of Debug unchanged. Concurrent runs of one app share its Renderer,
// so a Renderer must be safe for concurrent use.
type Renderer interface {
	// Help writes the help h to w.
	Help(w io.Writer, environ []string, h *Help) error
	// Version writes the version v to w.
	Version(w io.Writer, environ []string, v *Version) error
	// Error writes the failure f to w.
	Error(w io.Writer, environ []string, f *Failure) error
	// Debug writes the rows of the table of --flag-debug to w.
	Debug(w io.Writer, environ []string, rows []DebugRow) error
}

// Plain is the [Renderer] of an app that registers none. It writes text without colors or styles
// in the columns of the help. A Renderer can embed Plain to replace only some of its methods.
type Plain struct{}

// Help writes the usage line, the text and the sections of h to w. It writes each further line of
// the text of a command or of an option below its row, and indents the line to the column of the
// text.
//
// Error modes:
//   - an error that wraps the error of w.
func (Plain) Help(w io.Writer, _ []string, h *Help) error {
	var b strings.Builder
	b.WriteString("Usage: ")
	b.WriteString(h.App)
	if h.Command != "" {
		b.WriteString(" ")
		b.WriteString(h.Command)
	}
	for _, part := range h.Usage {
		b.WriteString(" ")
		b.WriteString(part)
	}
	b.WriteString("\n")
	if h.Text != "" {
		b.WriteString("\n")
		b.WriteString(h.Text)
		b.WriteString("\n")
	}
	commands := make([][3]string, len(h.Commands))
	for i, c := range h.Commands {
		commands[i] = [3]string{c.Name, c.Text}
	}
	writeSection(&b, "Commands", commands)
	writeSection(&b, "Arguments", plainRows(h.Arguments))
	writeSection(&b, "Options", plainRows(h.Options))
	writeSection(&b, "Global options", plainRows(h.Globals))
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("cli: write the help: %w", err)
	}
	return nil
}

// Version writes the text of v to w, followed by a newline.
//
// Error modes:
//   - an error that wraps the error of w.
func (Plain) Version(w io.Writer, _ []string, v *Version) error {
	if _, err := io.WriteString(w, v.Text+"\n"); err != nil {
		return fmt.Errorf("cli: write the version: %w", err)
	}
	return nil
}

// Error writes the name of the app, the command, the message and the suggestion of f to w on one
// line.
//
// Error modes:
//   - an error that wraps the error of w.
func (Plain) Error(w io.Writer, _ []string, f *Failure) error {
	if _, err := io.WriteString(w, f.App+": "+message(f.Command, f.Message, f.Suggestion)+"\n"); err != nil {
		return fmt.Errorf("cli: write the failure: %w", err)
	}
	return nil
}

// Debug writes the option, the value and the source of each row to w in columns below a header.
//
// Error modes:
//   - an error that wraps the error of w.
func (Plain) Debug(w io.Writer, _ []string, rows []DebugRow) error {
	table := make([][3]string, 0, len(rows)+1)
	table = append(table, [3]string{"OPTION", "VALUE", "SOURCE"})
	for _, r := range rows {
		table = append(table, [3]string{r.Option, r.Value, r.Source.String()})
	}
	var b strings.Builder
	writeRows(&b, table)
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("cli: write the output of %s%s: %w", flagPrefix, debugFlag, err)
	}
	return nil
}

// Render registers r as the renderer of every run of the app. A later call of Render or
// [App.RenderFor] replaces an earlier one. An app without a call uses [Plain].
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when r is nil.
func (a *App[G]) Render(r Renderer) {
	if r == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: the renderer is nil", ErrDeclaration))
		return
	}
	a.renderer, a.render = r, nil
}

// RenderFor registers render as the function that selects the renderer of each run. A run calls
// render once with its global options. The renderer that render returns writes the help of the
// run, its version, its table of --flag-debug and the failure that [App.Main] writes. A later call
// of RenderFor or [App.Render] replaces an earlier one.
//
// The run applies every valid value of its sources before it calls render, also when its command
// line does not scan, has a flag of the help or of the version, or ends at a command without a
// handler. The flags count up to the first fault of the command line, and a value that does not
// convert leaves its option at the value of the earlier sources. A run whose declarations are
// invalid or whose Env is incomplete passes the zero value of G. [Plain] writes the output of a run
// whose render returns nil.
//
// render runs on the goroutine of [App.Execute] or [App.Main]. Runs of the app can call it
// concurrently, so render must be safe for concurrent use. It must not keep or change the options.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when render is nil.
func (a *App[G]) RenderFor(render func(global *G) Renderer) {
	if render == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: the function of the renderer is nil", ErrDeclaration))
		return
	}
	a.renderer, a.render = nil, render
}

// rendererOf returns the renderer of a run with the global options global: the renderer of Render,
// or the renderer that the function of RenderFor returns. It returns Plain when that function
// returns nil.
func (a *App[G]) rendererOf(global *G) Renderer {
	if a.render == nil {
		return a.renderer
	}
	if r := a.render(global); r != nil {
		return r
	}
	return Plain{}
}

// plainRows returns the rows of opts in the columns of the help: the flags with the placeholder,
// the help text and the notes.
func plainRows(opts []HelpOption) [][3]string {
	rows := make([][3]string, len(opts))
	for i := range opts {
		o := &opts[i]
		flags := noShort + o.Name
		if o.Short != "" {
			flags = o.Short + ", " + o.Name
		}
		if o.Placeholder != "" {
			flags += " " + o.Placeholder
		}
		rows[i] = [3]string{flags, o.Text, notes(o)}
	}
	return rows
}

// notes returns the last column of the row of o in the help: whether o is required, and its
// default, its environment variable and its config key.
func notes(o *HelpOption) string {
	var parts []string
	if o.Required {
		parts = append(parts, "required")
	}
	if o.Default != "" {
		parts = append(parts, "default "+o.Default)
	}
	if o.Env != "" {
		parts = append(parts, "env "+o.Env)
	}
	if o.Key != "" {
		parts = append(parts, "key "+o.Key)
	}
	return strings.Join(parts, ", ")
}

// message returns text after the command and before the suggestion, such as
// "serve: unknown flag --adr (did you mean --addr?)". It leaves out an empty command and an empty
// suggestion.
func message(command, text, suggestion string) string {
	if command != "" {
		text = command + ": " + text
	}
	if suggestion != "" {
		text += " (did you mean " + suggestion + "?)"
	}
	return text
}

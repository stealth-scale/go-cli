// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// The command completion of an app whose global options do not declare a positional argument.
const (
	// completionName is the name of the command. A command of the program cannot take it.
	completionName = "completion"
	// completionHelp is the help text of the command.
	completionHelp = "Write the completion script of a shell."
)

// The lines that the command completion writes for the word at the cursor.
const (
	// valuesDirective ends the candidates, and makes the shell offer them alone.
	valuesDirective = ":values"
	// filesDirective makes the shell offer files in place of candidates. The extensions of the
	// files follow it, separated by spaces.
	filesDirective = ":files"
	// textSeparator separates the value of a candidate from its text.
	textSeparator = "\t"
	// breaks are the characters that end a line of the output, and the separator of the text.
	breaks = "\t\r\n"
)

// Candidate is a value that the command completion offers for the word at the cursor.
type Candidate struct {
	Value string // the word that the shell inserts
	Text  string // the description that zsh, fish and PowerShell show, or empty
}

// completer returns the candidates for a value of one type, as [App.Complete] registers it.
type completer[G any] func(ctx context.Context, run *Run[G, G], prefix string) ([]Candidate, error)

// completionArgs contains the positional arguments of the command completion.
type completionArgs struct {
	Shell shell  `help:"Shell of the script, one of bash, zsh, fish and powershell."           arg:"" enum:"bash,zsh,fish,powershell"`
	Words []word `help:"Words of the command line to complete. A script passes them after --." arg:""                                 default:""`
}

// word is a word of the command line that the command completion completes. A conversion of string
// that App.Convert registers does not apply to it.
type word string

// offer is the answer of the command completion for the word at the cursor: candidates, or the
// files of the shell.
type offer struct {
	candidates []Candidate
	exts       []string // the extensions of the files without their dot, or nil for every file
	files      bool     // the shell offers files in place of candidates
}

// Complete registers complete as the source of the candidates for a value of the type T. It applies
// to an option and to a positional argument of the type T, *T or []T. A later registration of T
// replaces an earlier one.
//
// The command completion calls complete with the global options after the valid values of their
// sources, the path of the command that the words select and prefix, the word at the cursor. It
// writes the candidates that start with prefix. The run of complete has an empty standard input and
// discards its standard output. The command returns the error of complete unchanged.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when complete is nil.
func (a *App[G]) Complete[T any](
	complete func(ctx context.Context, run *Run[G, G], prefix string) ([]Candidate, error),
) {
	t := reflect.TypeFor[T]()
	if complete == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: the completion of %s is nil", ErrDeclaration, t))
		return
	}
	a.completers[t] = complete
}

// completionCommand returns the command completion of the declaration d. Its handler writes the
// completion script of a shell, or the candidates for the last of its words.
func (a *App[G]) completionCommand(d *declaration[G]) *command[G] {
	return newCommand(a, completionName, completionName, completionHelp,
		func(ctx context.Context, run *Run[G, completionArgs]) error {
			return a.complete(ctx, d, run)
		}).cmd
}

// complete writes the completion script of the shell of run when run has no words. Otherwise it
// scans the words before the last one against d, and writes the offer for the last word. Words that
// do not scan have no candidates, unless they end with a flag without its value. The scan reads the
// flag --version as an unknown flag, so words with it have no candidates either.
//
// Error modes:
//   - the errors of writeScript and of write.
//   - the error of a function of Complete, unchanged.
func (a *App[G]) complete(ctx context.Context, d *declaration[G], run *Run[G, completionArgs]) error {
	args := run.Config()
	if len(args.Words) == 0 {
		return writeScript(run.Stdout(), args.Shell, a.name)
	}
	words := make([]string, len(args.Words))
	for i, w := range args.Words {
		words[i] = string(w)
	}
	last := len(words) - 1
	in := &input[G]{cmd: d.root, list: d.root.options}
	var o offer
	if scanned := in.scan(words[:last]); scanned == nil || in.pending != nil {
		var err error
		if o, err = a.offer(ctx, d, in, words[last], run.env); err != nil {
			return err
		}
	}
	return o.write(run.Stdout(), args.Shell, words[last])
}

// offer returns the offer for prefix after the words of in. Words with a flag of the help have no
// candidates. A flag at the end of the words without its value takes prefix as its value. After --,
// prefix is a positional argument. A prefix --name=value is a value of the option name. Any other
// prefix that starts with a dash is a flag. After a command with subcommands, prefix is a
// subcommand, and else a positional argument.
//
// Error modes:
//   - the errors of values.
func (a *App[G]) offer(ctx context.Context, d *declaration[G], in *input[G], prefix string, env *Env) (offer, error) {
	name, value, assigned := strings.Cut(strings.TrimPrefix(prefix, flagPrefix), flagValue)
	switch {
	case in.help:
		return offer{}, nil
	case in.pending != nil:
		return a.values(ctx, d, in, in.pending, prefix, env)
	case in.terminated:
		return a.argument(ctx, d, in, prefix, env)
	case strings.HasPrefix(prefix, flagPrefix) && assigned:
		return a.assigned(ctx, d, in, name, value, env)
	case strings.HasPrefix(prefix, shortPrefix):
		return flags(in, a.reserved()), nil
	case len(in.cmd.children) > 0:
		candidates := make([]Candidate, len(in.cmd.children))
		for i, b := range in.cmd.children {
			candidates[i] = Candidate{Value: b.name, Text: b.help}
		}
		return offer{candidates: candidates}, nil
	default:
		return a.argument(ctx, d, in, prefix, env)
	}
}

// assigned returns the offer for the value of the long flag name after its equals sign, each
// candidate after --name=. It returns an empty offer for a name that the flags of in do not declare.
//
// Error modes:
//   - the errors of values.
func (a *App[G]) assigned(
	ctx context.Context,
	d *declaration[G],
	in *input[G],
	name, value string,
	env *Env,
) (offer, error) {
	i := slices.IndexFunc(in.list, func(o *option) bool { return !o.arg && o.name == name })
	if i < 0 {
		return offer{}, nil
	}
	o, err := a.values(ctx, d, in, in.list[i], value, env)
	candidates := make([]Candidate, len(o.candidates))
	for j, c := range o.candidates {
		candidates[j] = Candidate{Value: flagPrefix + name + flagValue + c.Value, Text: c.Text}
	}
	o.candidates = candidates
	return o, err
}

// argument returns the offer for prefix as the positional argument of the command of in at the
// position after the positional arguments of in. A list takes every later position. A position
// after the last argument has no candidates.
//
// Error modes:
//   - the errors of values.
func (a *App[G]) argument(
	ctx context.Context,
	d *declaration[G],
	in *input[G],
	prefix string,
	env *Env,
) (offer, error) {
	n := len(in.args)
	for _, o := range in.cmd.options {
		if !o.arg {
			continue
		}
		if n == 0 || o.shape == list {
			return a.values(ctx, d, in, o, prefix, env)
		}
		n--
	}
	return offer{}, nil
}

// values returns the offer for prefix as a value of the option o, by the first rule that applies:
// no candidates for a map, the candidates of the function of Complete for its type, no candidates
// for a secret option, the values of its enum, true and false for a bool, the files with the
// extensions of the formats of the app for a config file, every file for any other value of the
// kind string such as an input file, and no candidates for any other type.
//
// Error modes:
//   - the error of the function of Complete, unchanged.
func (a *App[G]) values(
	ctx context.Context,
	d *declaration[G],
	in *input[G],
	o *option,
	prefix string,
	env *Env,
) (offer, error) {
	t := o.value.Type()
	if o.shape == list || o.shape == pointer {
		t = t.Elem()
	}
	complete, registered := a.completers[t]
	switch {
	case o.shape == dict:
		return offer{}, nil
	case registered:
		// The words are a partial command line, so the function receives the valid values of their
		// sources and the errors do not count.
		_ = a.resolve(env, d, in)
		quiet := *env
		quiet.Stdin, quiet.Stdout = nil, io.Discard
		inv := invocation[G]{global: d.global, env: &quiet, sources: in.sources(), command: in.cmd.path}
		candidates, err := complete(ctx, &Run[G, G]{invocation: inv, config: d.global}, prefix)
		return offer{candidates: candidates}, err
	case o.secret:
		return offer{}, nil
	case o.enum != nil:
		candidates := make([]Candidate, len(o.enum))
		for i, v := range o.enum {
			candidates[i] = Candidate{Value: v}
		}
		return offer{candidates: candidates}, nil
	case t.Kind() == reflect.Bool:
		return offer{
			candidates: []Candidate{{Value: strconv.FormatBool(true)}, {Value: strconv.FormatBool(false)}},
		}, nil
	case t == configFileType:
		exts := slices.Sorted(maps.Keys(a.formats))
		for i, ext := range exts {
			exts[i] = strings.TrimPrefix(ext, ".")
		}
		return offer{files: true, exts: exts}, nil
	case t.Kind() == reflect.String:
		return offer{files: true}, nil
	default:
		return offer{}, nil
	}
}

// flags returns the offer of the long flags and the short flags of the options of in and of
// reserved, with the help text of each option. It leaves out an option that the words set already,
// unless it takes a list or a map.
func flags[G any](in *input[G], reserved []*option) offer {
	var candidates []Candidate
	for _, o := range ordered(slices.Concat(in.list, reserved)) {
		if o.arg {
			continue
		}
		set := slices.ContainsFunc(in.flags, func(f occurrence) bool { return f.opt == o })
		if set && o.shape != list && o.shape != dict {
			continue
		}
		candidates = append(candidates, Candidate{Value: flagPrefix + o.name, Text: o.help})
		if o.short != 0 {
			candidates = append(candidates, Candidate{Value: shortPrefix + string(o.short), Text: o.help})
		}
	}
	return offer{candidates: candidates}
}

// write writes o for prefix to w as the scripts read it. It writes a line for each candidate that
// starts with prefix and then the line of valuesDirective, or the line of filesDirective with the
// extensions. A candidate line has the first line of the text of the candidate after a tab, except
// for bash. write leaves out a candidate whose value contains a tab or a line break.
//
// Error modes:
//   - an error that wraps the error of w.
func (o offer) write(w io.Writer, sh shell, prefix string) error {
	var b strings.Builder
	if o.files {
		b.WriteString(strings.Join(slices.Concat([]string{filesDirective}, o.exts), " "))
	} else {
		for _, c := range o.candidates {
			if !strings.HasPrefix(c.Value, prefix) || strings.ContainsAny(c.Value, breaks) {
				continue
			}
			b.WriteString(c.Value)
			text, _, _ := strings.Cut(strings.ReplaceAll(c.Text, "\r", "\n"), "\n")
			if text != "" && sh != bash {
				b.WriteString(textSeparator + strings.ReplaceAll(text, textSeparator, " "))
			}
			b.WriteString("\n")
		}
		b.WriteString(valuesDirective)
	}
	b.WriteString("\n")
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("cli: write the completion: %w", err)
	}
	return nil
}

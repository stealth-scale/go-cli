// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package transcript

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"go.stealthscale.io/cli"
)

// safe contains the characters that a shell reads as themselves. A word of only these characters
// does not need quotes.
const safe = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_@%+=:,./-"

// fullDevice is the device of the redirect in the command line of a run with Full. Every write to the
// device fails.
const fullDevice = "/dev/full"

// printfEscapes escapes the text of a standard input for the format of printf in single quotes.
var printfEscapes = strings.NewReplacer(`\`, `\\`, "%", "%%", "\n", `\n`, "'", `'\''`)

// errFull is the error of every write to the standard output of a run with Full.
var errFull = errors.New("transcript: no space left on device")

// Run is one run of a program.
type Run struct {
	Vars  map[string]string // the environment variables of the run
	Stdin string            // the standard input of the run
	Args  []string          // the arguments after the name of the program
	Full  bool              // every write to the standard output fails like a write to /dev/full
}

// full is the standard output of a run with Full. Every write to full fails with errFull.
type full struct{}

// Write returns errFull.
func (full) Write([]byte) (int, error) {
	return 0, errFull
}

// Write runs program once for each run of runs and returns the transcript of the runs. The transcript
// of a run starts with "$ " and the command line. The text that the run writes to its standard output
// and its standard error follows. The last line contains the exit status in brackets. A blank line
// separates two runs.
//
// The environment variables of a run come before name in the order of their names. The command line
// of a run with a standard input pipes the output of printf to the program. The command line of a run
// with Full ends with a redirect to /dev/full. Every write of such a run to its standard output fails.
//
// Each run reads its config files with os.ReadFile. The path of a config file is relative to the
// working directory.
func Write(ctx context.Context, name string, runs []Run, program func(context.Context, *cli.Env) int) []byte {
	var b bytes.Buffer
	for i, r := range runs {
		if i > 0 {
			b.WriteString("\n")
		}
		words := make([]string, 0, len(r.Vars)+len(r.Args)+2)
		for _, k := range slices.Sorted(maps.Keys(r.Vars)) {
			words = append(words, k+"="+quote(r.Vars[k]))
		}
		words = append(words, name)
		for _, arg := range r.Args {
			words = append(words, quote(arg))
		}
		stdout := io.Writer(&b)
		if r.Full {
			words, stdout = append(words, ">"+fullDevice), full{}
		}
		line := strings.Join(words, " ")
		if r.Stdin != "" {
			line = "printf '" + printfEscapes.Replace(r.Stdin) + "' | " + line
		}
		b.WriteString("$ " + line + "\n")
		env := &cli.Env{
			Lookup:   cli.Vars(r.Vars),
			ReadFile: os.ReadFile,
			Stdin:    strings.NewReader(r.Stdin),
			Stdout:   stdout,
			Stderr:   &b,
			Args:     r.Args,
		}
		status := program(ctx, env)
		b.WriteString("[exit " + strconv.Itoa(status) + "]\n")
	}
	return b.Bytes()
}

// quote returns word in a form that a shell reads as word. It returns a nonempty word of only safe
// characters unchanged. It puts any other word in single quotes and escapes each single quote of the
// word for the shell.
func quote(word string) string {
	if word != "" && strings.Trim(word, safe) == "" {
		return word
	}
	return "'" + strings.ReplaceAll(word, "'", `'\''`) + "'"
}

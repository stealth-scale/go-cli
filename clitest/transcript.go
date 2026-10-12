// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package clitest

import (
	"bytes"
	"context"
	"maps"
	"slices"
	"strconv"
	"strings"

	"go.stealthscale.io/cli"
)

// safe contains the characters that a shell reads as themselves. A word of only these characters
// does not need quotes.
const safe = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_@%+=:,./-"

// printfEscapes escapes the text of a standard input for the format of printf in single quotes.
var printfEscapes = strings.NewReplacer(`\`, `\\`, "%", "%%", "\n", `\n`, "'", `'\''`)

// Transcript calls program once for each input of ins, and returns the transcript of the runs as a
// terminal shows them. name is the name of the program in the command lines of the transcript. A
// blank line separates two runs, and the transcript of a run contains these lines:
//   - each line of the comment of the run, after "# ".
//   - each config file that the run read, in the order of the first reads: "$ cat" and the path,
//     then the content of the file with a final newline.
//   - "$ " and the command line of the run: the environment variables in the order of their names,
//     name and the arguments, with quotes where a shell needs them. A standard input precedes the
//     command line as the output of printf. The command line of a run with Input.FailStdin ends with
//     "<&-", and the command line of a run with Input.FailStdout with ">/dev/full".
//   - the standard output and the standard error of the run in the order of the writes.
//   - the exit status in brackets, such as "[exit 2]". After an output without a final newline, the
//     status is on the last line of the output.
//
// Transcript cancels the context of each run when program returns.
func Transcript(ctx context.Context, name string, program func(context.Context, *cli.Env) int, ins ...Input) []byte {
	var b bytes.Buffer
	for i := range ins {
		in := &ins[i]
		if i > 0 {
			b.WriteString("\n")
		}
		env, out := NewEnv(in)
		status := run(ctx, program, env)
		files, output := out.snapshot()
		for line := range strings.Lines(in.Comment) {
			b.WriteString(strings.TrimRight("# "+strings.TrimSuffix(line, "\n"), " "))
			b.WriteString("\n")
		}
		for _, f := range files {
			b.WriteString("$ cat ")
			b.WriteString(quote(f.name))
			b.WriteString("\n")
			b.Write(f.data)
			if len(f.data) > 0 && f.data[len(f.data)-1] != '\n' {
				b.WriteString("\n")
			}
		}
		b.WriteString("$ ")
		b.WriteString(commandLine(name, in))
		b.WriteString("\n")
		b.WriteString(output)
		b.WriteString("[exit ")
		b.WriteString(strconv.Itoa(status))
		b.WriteString("]\n")
	}
	return b.Bytes()
}

// commandLine returns the command line of a run of in by the program name, in the form that a shell
// reads.
func commandLine(name string, in *Input) string {
	words := make([]string, 0, len(in.Vars)+len(in.Args)+3)
	for _, k := range slices.Sorted(maps.Keys(in.Vars)) {
		words = append(words, k+"="+quote(in.Vars[k]))
	}
	words = append(words, name)
	for _, arg := range in.Args {
		words = append(words, quote(arg))
	}
	if in.FailStdin {
		words = append(words, "<&-")
	}
	if in.FailStdout {
		words = append(words, ">/dev/full")
	}
	line := strings.Join(words, " ")
	if in.Stdin != "" && !in.FailStdin {
		line = "printf '" + printfEscapes.Replace(in.Stdin) + "' | " + line
	}
	return line
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

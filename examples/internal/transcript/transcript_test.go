// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package transcript_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/examples/internal/transcript"
)

// program is the name of the program that the tests run.
const program = "say"

// echoOptions contains the options of the command echo.
type echoOptions struct {
	Words []string `arg:"" default:""`
	Extra []string
	Fail  bool
}

// say returns the program say with the command echo. echo writes its standard input to the standard
// error. It then writes each word of its arguments and of the option extra to a line of the standard
// output in upper case.
func say(t *testing.T) func(context.Context, *cli.Env) int {
	t.Helper()
	app := cli.New[struct{ Config cli.ConfigFile }](program, "")
	app.Command("echo", "", func(_ context.Context, run *cli.Run[struct{ Config cli.ConfigFile }, echoOptions]) error {
		in, err := io.ReadAll(run.Stdin())
		if err != nil {
			return err
		}
		if _, err := run.Stderr().Write(in); err != nil {
			return err
		}
		for _, w := range append(run.Config().Words, run.Config().Extra...) {
			if _, err := run.Stdout().Write([]byte(strings.ToUpper(w) + "\n")); err != nil {
				return err
			}
		}
		return nil
	})
	return app.Main
}

// TestTranscript checks the transcript of the runs of a program.
func TestTranscript(t *testing.T) {
	t.Parallel()

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []transcript.Run
			want string
		}{
			{
				name: "writes the command line, the output and the exit status of a run",
				give: []transcript.Run{{Args: []string{"echo", "a", "b"}}},
				want: "$ say echo a b\nA\nB\n[exit 0]\n",
			},
			{
				name: "writes the message and the exit status of a run that fails",
				give: []transcript.Run{{Args: []string{"echo", "--fial"}}},
				want: "$ say echo --fial\nsay: echo: unknown flag --fial (did you mean --fail?)\n[exit 2]\n",
			},
			{
				name: "separates two runs with a blank line",
				give: []transcript.Run{{Args: []string{"echo", "a"}}, {Args: []string{"echo", "b"}}},
				want: "$ say echo a\nA\n[exit 0]\n\n$ say echo b\nB\n[exit 0]\n",
			},
			{
				name: "writes the environment variables before the command in the order of their names",
				give: []transcript.Run{{Vars: map[string]string{"B": "2", "A": "1"}, Args: []string{"echo"}}},
				want: "$ A=1 B=2 say echo\n[exit 0]\n",
			},
			{
				name: "writes the standard input as the format string of printf",
				give: []transcript.Run{{Stdin: "it's 100%\\\nok", Args: []string{"echo"}}},
				want: "$ printf 'it'\\''s 100%%\\\\\\nok' | say echo\nit's 100%\\\nok[exit 0]\n",
			},
			{
				name: "quotes a word with a character that a shell does not read as itself",
				give: []transcript.Run{{Vars: map[string]string{"A": "x y"}, Args: []string{"echo", "it's", "*.log"}}},
				want: "$ A='x y' say echo 'it'\\''s' '*.log'\nIT'S\n*.LOG\n[exit 0]\n",
			},
			{
				name: "quotes an empty word",
				give: []transcript.Run{{Args: []string{"echo", ""}}},
				want: "$ say echo ''\n\n[exit 0]\n",
			},
			{
				name: "fails each write to the standard output of a run with Full",
				give: []transcript.Run{{Args: []string{"echo", "a"}, Full: true}},
				want: "$ say echo a >/dev/full\nsay: transcript: no space left on device\n[exit 1]\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := transcript.Write(t.Context(), program, tt.give, say(t))
				assert.Equal(t, string(got), tt.want, "Write returns the transcript of the runs")
			})
		}

		t.Run("passes the environment variables of a run to Environ", func(t *testing.T) {
			t.Parallel()
			environ := func(_ context.Context, env *cli.Env) int {
				_, err := io.WriteString(env.Stdout, strings.Join(env.Environ(), " ")+"\n")
				assert.NoError(t, err, "the program writes its environment")
				return 0
			}
			runs := []transcript.Run{{Vars: map[string]string{"B": "2", "A": "1"}}}
			got := transcript.Write(t.Context(), program, runs, environ)
			assert.Equal(t, string(got), "$ A=1 B=2 say\nA=1 B=2\n[exit 0]\n", "the program receives the variables")
		})

		t.Run("reads a config file with os.ReadFile", func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "say.json")
			assert.NoError(t, os.WriteFile(path, []byte(`{"echo": {"extra": ["from", "file"]}}`), 0o600),
				"os.WriteFile writes the config file")
			got := transcript.Write(t.Context(), program, []transcript.Run{{Args: []string{"--config", path, "echo"}}},
				say(t))
			assert.HasSuffix(t, string(got), "\nFROM\nFILE\n[exit 0]\n", "the run reads the words of the file")
		})
	})
}

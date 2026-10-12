// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package clitest_test

import (
	"context"
	"io"
	"testing"
	"time"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/clitest"
)

// TestTranscript checks the transcript of the runs of a program.
func TestTranscript(t *testing.T) {
	t.Parallel()

	t.Run("Transcript", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []clitest.Input
			want string
		}{
			{
				name: "writes the command line, the output and the exit status of a run",
				give: []clitest.Input{{Args: []string{"echo", "a", "b"}}},
				want: "$ say echo a b\nA\nB\n[exit 0]\n",
			},
			{
				name: "writes the message and the exit status of a run that fails",
				give: []clitest.Input{{Args: []string{"echo", "--fial"}}},
				want: "$ say echo --fial\nsay: echo: unknown flag --fial (did you mean --fail?)\n[exit 2]\n",
			},
			{
				name: "separates two runs with a blank line",
				give: []clitest.Input{{Args: []string{"echo", "a"}}, {Args: []string{"echo", "b"}}},
				want: "$ say echo a\nA\n[exit 0]\n\n$ say echo b\nB\n[exit 0]\n",
			},
			{
				name: "writes the environment variables before the command in the order of their names",
				give: []clitest.Input{{Vars: map[string]string{"B": "2", "A": "1"}, Args: []string{"echo"}}},
				want: "$ A=1 B=2 say echo\n[exit 0]\n",
			},
			{
				name: "writes the standard input as the format string of printf",
				give: []clitest.Input{{Stdin: "it's 100%\\\nok", Args: []string{"echo"}}},
				want: "$ printf 'it'\\''s 100%%\\\\\\nok' | say echo\nit's 100%\\\nok[exit 0]\n",
			},
			{
				name: "quotes a word with a character that a shell does not read as itself",
				give: []clitest.Input{{Vars: map[string]string{"A": "x y"}, Args: []string{"echo", "it's", "*.log"}}},
				want: "$ A='x y' say echo 'it'\\''s' '*.log'\nIT'S\n*.LOG\n[exit 0]\n",
			},
			{
				name: "quotes an empty word",
				give: []clitest.Input{{Args: []string{"echo", ""}}},
				want: "$ say echo ''\n\n[exit 0]\n",
			},
			{
				name: "fails each write to the standard output of a run with FailStdout",
				give: []clitest.Input{{Args: []string{"echo", "a"}, FailStdout: true}},
				want: "$ say echo a >/dev/full\nsay: clitest: no space left on device\n[exit 1]\n",
			},
			{
				name: "fails each read of the standard input of a run with FailStdin",
				give: []clitest.Input{{Stdin: "in\n", Args: []string{"echo"}, FailStdin: true}},
				want: "$ say echo <&-\nsay: clitest: bad file descriptor\n[exit 1]\n",
			},
			{
				name: "writes each config file that a run read before its command line",
				give: []clitest.Input{{
					Args:  []string{"--config", "say.json", "echo"},
					Files: clitest.Files{"say.json": `{"echo": {"extra": ["from", "file"]}}` + "\n"},
				}},
				want: "$ cat say.json\n{\"echo\": {\"extra\": [\"from\", \"file\"]}}\n$ say --config say.json echo\nFROM\nFILE\n" +
					"[exit 0]\n",
			},
			{
				name: "ends a config file without a final newline with a newline",
				give: []clitest.Input{{
					Args:  []string{"--config", "say.json", "echo"},
					Files: clitest.Files{"say.json": `{"echo": {"extra": ["x"]}}`},
				}},
				want: "$ cat say.json\n{\"echo\": {\"extra\": [\"x\"]}}\n$ say --config say.json echo\nX\n[exit 0]\n",
			},
			{
				name: "writes an empty config file without a line",
				give: []clitest.Input{
					{Args: []string{"--config", "say.json", "echo"}, Files: clitest.Files{"say.json": ""}},
				},
				want: "$ cat say.json\n$ say --config say.json echo\nsay: echo: config file say.json: cli: decode JSON: EOF\n[exit 2]\n",
			},
			{
				name: "writes the lines of the comment of a run above the run",
				give: []clitest.Input{{Comment: "The first line.\n\nThe third line.", Args: []string{"echo"}}},
				want: "# The first line.\n#\n# The third line.\n$ say echo\n[exit 0]\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := clitest.Transcript(t.Context(), program, say(), tt.give...)
				assert.Equal(t, string(got), tt.want, "Transcript returns the transcript of the runs")
			})
		}

		t.Run("writes each config file that a run read once in the order of the first reads", func(t *testing.T) {
			t.Parallel()
			program := func(_ context.Context, env *cli.Env) int {
				for _, name := range []string{"b.json", "a.json", "b.json"} {
					if _, err := env.ReadFile(name); err != nil {
						return cli.ExitFailure
					}
				}
				return cli.ExitOK
			}
			in := clitest.Input{Files: clitest.Files{"a.json": "{}\n", "b.json": "[]\n"}}
			got := clitest.Transcript(t.Context(), "srv", program, in)
			assert.Equal(t, string(got), "$ cat b.json\n[]\n$ cat a.json\n{}\n$ srv\n[exit 0]\n",
				"the transcript writes each file once")
		})

		t.Run("lets a goroutine of a run write after Transcript returns", func(t *testing.T) {
			t.Parallel()
			release, wrote := make(chan struct{}), make(chan error, 1)
			program := func(_ context.Context, env *cli.Env) int {
				go func() {
					<-release
					_, err := io.WriteString(env.Stdout, "late\n")
					wrote <- err
				}()
				return cli.ExitOK
			}
			got := clitest.Transcript(t.Context(), "srv", program, clitest.Input{})
			var err error
			assert.CompletesWithin(t, time.Minute, func(context.Context) error {
				close(release)
				err = <-wrote
				return nil
			}, "the goroutine writes")
			assert.NoError(t, err, "the write of the goroutine succeeds")
			assert.Equal(t, string(got), "$ srv\n[exit 0]\n", "the transcript leaves out the late write")
		})

		t.Run("cancels the context of each run when the program returns", func(t *testing.T) {
			t.Parallel()
			var ended []func() error
			program := func(ctx context.Context, _ *cli.Env) int {
				ended = append(ended, ctx.Err)
				return cli.ExitOK
			}
			clitest.Transcript(t.Context(), "srv", program, clitest.Input{}, clitest.Input{})
			assert.Length(t, ended, 2, "the program runs twice")
			for _, err := range ended {
				assert.ErrorIs(t, err(), context.Canceled, "the context of the run has ended")
			}
		})
	})
}

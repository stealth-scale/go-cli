// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli/clitest"
)

// The config files of the runs, and the other files of the workspace of the shells.
const (
	rolloutJSON = `{"plans": {"api": {"description": "The API."}, "web": {"description": "The website."}}}`
	otherJSON   = `{"plans": {"db": {"description": "The database."}}}`
)

// The runs of the shells.
const (
	// shellTimeout limits a run of a shell, so a script that waits does not stop the test.
	shellTimeout = time.Minute
	// oldBash is the exit status of the driver of bash for a bash older than 4.1.
	oldBash = 3
)

// files are the files of the runs of the transcript.
var files = clitest.Files{"rollout.json": rolloutJSON, "other.json": otherJSON, "notes.txt": ""}

// runs are the command lines of the transcript of the example.
var runs = []clitest.Input{
	{Args: []string{"apply", "web", "--region", "us-east"}, Files: files},
	{Args: []string{"apply", "nope"}, Files: files},
	{Comment: "The commands for the first word.", Args: []string{"completion", "zsh", "--", ""}, Files: files},
	{
		Comment: "The plans of rollout.json, from the function of App.Complete.",
		Args:    []string{"completion", "zsh", "--", "apply", ""},
		Files:   files,
	},
	{
		Comment: "The plans of the config file of the words.",
		Args:    []string{"completion", "zsh", "--", "-c", "other.json", "apply", ""},
		Files:   files,
	},
	{Args: []string{"completion", "zsh", "--", "apply", "web", "--region", ""}, Files: files},
	{
		Comment: "A value after the equals sign of its flag.",
		Args:    []string{"completion", "zsh", "--", "apply", "web", "--region=u"},
		Files:   files,
	},
	{
		Comment: "bash receives the commands without their help texts.",
		Args:    []string{"completion", "bash", "--", ""},
		Files:   files,
	},
	{
		Comment: "The shell offers the files of the formats for a config file.",
		Args:    []string{"completion", "zsh", "--", "-c", ""},
		Files:   files,
	},
	{Args: []string{"completion", "zsh", "--", "logs", ""}, Files: files},
}

// shells are the drivers of the scripts. A driver loads the script of its shell and writes the
// values that the shell offers for the command line in the variable LINE, one on each line.
var shells = []struct {
	name    string
	program string
	args    []string
}{
	{
		name:    "bash",
		program: "bash",
		args: []string{"-c", `((BASH_VERSINFO[0] > 4 || BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] >= 1)) || exit 3
source <(rollout completion bash)
COMP_LINE=$LINE
COMP_POINT=${#COMP_LINE}
__rollout_complete
printf '%s\n' "${COMPREPLY[@]}"`},
	},
	{
		name:    "zsh",
		program: "zsh",
		args: []string{"-f", "-c", `compdef() { : }
compset() { : }
_files() { print -r -- "_files $*" }
_describe() { local c; for c in "${(@P)4}"; do print -r -- "${c%%:*}"; done }
source <(rollout completion zsh)
words=(${(z)LINE})
if [[ $LINE == *' ' ]]; then words+=(''); fi
CURRENT=${#words}
PREFIX=${words[-1]}
_rollout`},
	},
	{
		name:    "fish",
		program: "fish",
		args: []string{"-c", `rollout completion fish | source
complete -C"$LINE"`},
	},
	{
		name:    "powershell",
		program: "pwsh",
		args: []string{
			"-NoProfile",
			"-NonInteractive",
			"-Command",
			`rollout completion powershell | Out-String | Invoke-Expression
(TabExpansion2 -inputScript $env:LINE -cursorColumn $env:LINE.Length).CompletionMatches |
    ForEach-Object { $_.CompletionText }`,
		},
	},
}

// lines are the command lines that the shells complete, with the values that each shell offers.
// A shell without values in want does not complete the line, because it completes the files of
// the line in a form of its own.
var lines = []struct {
	line string
	want map[string][]string
}{
	{
		line: "rollout ",
		want: map[string][]string{
			"bash": {"apply", "logs", "completion"}, "zsh": {"apply", "logs", "completion"},
			"fish": {"apply", "logs", "completion"}, "powershell": {"apply", "logs", "completion"},
		},
	},
	{
		line: "rollout apply ",
		want: map[string][]string{
			"bash": {"api", "web"}, "zsh": {"api", "web"}, "fish": {"api", "web"}, "powershell": {"api", "web"},
		},
	},
	{
		line: "rollout apply web --region=u",
		want: map[string][]string{
			"bash": {"us-east"}, "zsh": {"--region=us-east"}, "fish": {"--region=us-east"},
			"powershell": {"--region=us-east"},
		},
	},
	{
		line: "rollout -c ",
		want: map[string][]string{
			"bash": {"rollout.json", "other.json", "sub"}, "zsh": {"_files -g *.(ini|json)"},
			"fish": {"rollout.json", "other.json", "sub/"},
		},
	},
	{
		line: "rollout --config=o",
		want: map[string][]string{
			"bash": {"other.json"}, "zsh": {"_files -g *.(ini|json)"}, "fish": {"--config=other.json"},
			"powershell": {"--config=other.json"},
		},
	},
	{
		line: "rollout completion ",
		want: map[string][]string{
			"bash": {"bash", "zsh", "fish", "powershell"}, "zsh": {"bash", "zsh", "fish", "powershell"},
			"fish": {"bash", "zsh", "fish", "powershell"}, "powershell": {"bash", "zsh", "fish", "powershell"},
		},
	},
}

// TestCompletion checks the transcript of the runs of the example and the scripts in their shells.
func TestCompletion(t *testing.T) {
	t.Parallel()

	t.Run("app", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the transcript of the runs", func(t *testing.T) {
			t.Parallel()
			got := clitest.Transcript(t.Context(), "rollout", app().Main, runs...)
			golden.Match(t, "transcript.txt", got, golden.ShouldUpdate())
		})

		t.Run("completes the command lines in each installed shell", func(t *testing.T) {
			t.Parallel()
			bin, ws := t.TempDir(), t.TempDir()
			name := "rollout"
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			build := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(bin, name), ".")
			out, err := build.CombinedOutput()
			assert.NoError(t, err, "go build builds the example: "+string(out))
			for path, data := range map[string]string{"rollout.json": rolloutJSON, "other.json": otherJSON, "notes.txt": ""} {
				assert.NoError(
					t,
					os.WriteFile(filepath.Join(ws, path), []byte(data), 0o600),
					"the workspace receives a file",
				)
			}
			assert.NoError(t, os.Mkdir(filepath.Join(ws, "sub"), 0o700), "the workspace receives a directory")
			home := t.TempDir()
			env := append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"HOME="+home, "XDG_CONFIG_HOME="+home, "XDG_DATA_HOME="+home)
			for _, sh := range shells {
				t.Run(sh.name, func(t *testing.T) {
					t.Parallel()
					if _, err := exec.LookPath(sh.program); err != nil {
						t.Skip(sh.program + " is not installed")
					}
					if runtime.GOOS == "windows" && sh.name != "powershell" {
						t.Skip("the test runs " + sh.program + " on Windows without the paths of the system")
					}
					for _, tt := range lines {
						want, ok := tt.want[sh.name]
						if !ok {
							continue
						}
						ctx, cancel := context.WithTimeout(t.Context(), shellTimeout)
						cmd := exec.CommandContext(ctx, sh.program, sh.args...)
						cmd.Dir, cmd.Env = ws, slices.Concat(env, []string{"LINE=" + tt.line})
						out, err := cmd.Output()
						cancel()
						if exit, isExit := errors.AsType[*exec.ExitError](err); isExit && exit.ExitCode() == oldBash {
							t.Skip("bash is older than 4.1")
						}
						assert.NoError(t, err, sh.name+" completes the command line")
						var got []string
						for line := range strings.SplitSeq(strings.ReplaceAll(string(out), "\r", ""), "\n") {
							if value, _, _ := strings.Cut(line, "\t"); value != "" {
								got = append(got, value)
							}
						}
						expect.Permutation(t, got, want, sh.name+" offers the values of the command line "+tt.line)
					}
				})
			}
		})
	})
}

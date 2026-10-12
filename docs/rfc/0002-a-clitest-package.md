---
rfc: 0002
title: A clitest package to test the programs of the package cli
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-12
updated: 2026-10-12
discussion: none
supersedes: none
superseded-by: none
produces-adr: tbd
---

<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0002: A clitest package to test the programs of the package cli

## Summary

The package `go.stealthscale.io/cli/clitest` runs a program of the package `cli` in a test. `Main` runs the program once and returns its exit status and its output. `Transcript` runs the program once for each input of a list and returns the runs as a terminal shows them, with the config files that each run read. `NewEnv` returns the `Env` of an input for a test that calls `App.Execute` or a handler itself. Each run sees only the files, the environment variables and the standard input of its input, and leaves the state of the process unchanged. Tests that call the package can run in parallel, and their results do not depend on the machine. The package imports only the standard library and the package `cli`.

## Motivation

A test of a program builds a `cli.Env` from the arguments, the environment variables, the config files, the standard input and two buffers. It then reads the buffers and the exit status. The test in the section Testing of the README takes 15 lines for one run, and every program repeats them. A test that passes `os.ReadFile` as the `ReadFile` of its `Env` also reads the config file at the default path of the program. Its result then depends on the files of the machine that runs it.

The tests of the examples of this repository run a list of command lines and compare a transcript of the runs with a golden file. The harness `examples/internal/transcript` builds that transcript in 112 lines, with 137 lines of tests. A program outside the repository cannot import it. Its transcripts show the command lines and the output of the runs, and leave out the config files that the runs read.

The libraries of the same kind leave the tests to the program:

- cobra v1.10.2 has the methods `SetArgs`, `SetOut`, `SetErr` and `SetIn` on a command.
- urfave/cli v3.14.0 has the fields `Reader`, `Writer` and `ErrWriter` on a command. It exits through the package variable `OsExiter`.
- kong v1.16.1 has the options `Writers` and `Exit`.
- None of the three has a package for tests.

go-cmdtest and testscript test command-line programs through scripts:

- go-cmdtest v0.4.0 reads files of `$` command lines and their output. Its `InProcessProgram` sets `os.Args`, `os.Stdout` and `os.Stderr` for each command, so its tests in process cannot run in parallel.
- testscript of go-internal v1.16.0 runs scripts in the txtar format. Its `Main` installs each command as a copy of the test binary and runs it as a process.

A program of the package `cli` receives its arguments, its environment, its config files and its streams through `cli.Env`, so a test can run it in process and in parallel without either mechanism. The package `clitest` provides that run as API.

## Detailed design

### Components

| Component | Responsibility |
|---|---|
| `Input` | The arguments, the environment variables, the config files and the standard input of one run, the failures of its streams, and a comment for its transcript |
| `Files` | Text files by their paths, as a file system for `Input.Files` |
| `NewEnv`, `Output` | The `Env` of an input, and what its runs write |
| `Main`, `Result` | One run of a program, and its exit status and output |
| `Transcript` | The runs of a list of inputs as a terminal shows them |
| `ErrStdin`, `ErrStdout` | The errors of the reads and the writes that an input makes fail |

### API

```go
// Package clitest runs the programs of the package go.stealthscale.io/cli in tests.
package clitest

// Input is the input of one run of a program.
type Input struct {
	Vars       map[string]string // the environment variables of Env.Lookup and Env.Environ
	Files      fs.FS             // the file system of the config files, or nil for a run without files
	Comment    string            // the comment above the run in a transcript, or empty
	Stdin      string            // the standard input
	Args       []string          // the arguments after the name of the program
	FailStdin  bool              // every read of the standard input fails with ErrStdin
	FailStdout bool              // every write to the standard output fails with ErrStdout
}

// Files is a file system of text files by their paths, such as srv.json or /etc/srv.json.
type Files map[string]string

// Open opens the file name.
func (f Files) Open(name string) (fs.File, error)

// ErrStdin is the error of every read of the standard input of a run with FailStdin.
var ErrStdin = errors.New("clitest: bad file descriptor")

// ErrStdout is the error of every write to the standard output of a run with FailStdout.
var ErrStdout = errors.New("clitest: no space left on device")

// Output is what the runs of an Env write. Its methods are safe for concurrent use.
type Output struct {
	// contains unexported fields
}

// Stdout returns the text that the runs wrote to the standard output.
func (o *Output) Stdout() string

// Stderr returns the text that the runs wrote to the standard error.
func (o *Output) Stderr() string

// NewEnv returns an Env of in, and the Output that the runs of the Env write.
func NewEnv(in *Input) (*cli.Env, *Output)

// Result is the outcome of one run of a program.
type Result struct {
	Stdout string // the text that the run wrote to its standard output
	Stderr string // the text that the run wrote to its standard error
	Status int    // the exit status that the program returned
}

// Main calls program once with an Env of in, and returns the exit status and the output of the
// run. program is the method App.Main of an app, or a function of its signature. Main cancels the
// context of the run when program returns.
func Main(ctx context.Context, program func(context.Context, *cli.Env) int, in *Input) Result

// Transcript calls program once for each input of ins, and returns the transcript of the runs.
// name is the name of the program in the command lines of the transcript.
func Transcript(ctx context.Context, name string, program func(context.Context, *cli.Env) int,
	ins ...Input) []byte
```

`NewEnv` and `Main` take a pointer, because an `Input` has 88 bytes and gocritic rejects a parameter of 80 bytes or more by value. The fields of each struct follow the order that the check fieldalignment of govet requires.

### The Env of an input

| Field of `Env` | Value |
|---|---|
| `Args` | `in.Args` |
| `Lookup` | `cli.Vars(in.Vars)` |
| `Environ` | `cli.Environ(in.Vars)` |
| `ReadFile` | The file of `in.Files` at the path of the read, after `filepath.ToSlash` and `path.Clean`, and without a leading slash |
| `Stdin` | A reader of `in.Stdin`, or a reader that returns `ErrStdin` when `in.FailStdin` is true |
| `Stdout` | The `Output`, or a writer that returns `ErrStdout` when `in.FailStdout` is true |
| `Stderr` | The `Output` |

- **Files.** A run reads only the files of `in.Files`. A read of any other path returns an error that wraps `fs.ErrNotExist`, so a run skips a config file at its default path. `in.Files` is the root of the file system of the run, so `/etc/srv.json` and `srv.json` name the files `etc/srv.json` and `srv.json` of `in.Files`. A test passes `clitest.Files` for files in the test, `os.DirFS("testdata")` for files on disk, or an `embed.FS`.
- **Streams.** The buffers of an `Output` are safe for concurrent use, as the methods of `os.File` are. A handler that writes from two or more goroutines does not fail its test under the race detector.
- **Context.** `Main` and `Transcript` cancel the context of a run when the program returns, as the end of a process stops its goroutines.

### Tests with clitest

A test of one run reads the fields of the `Result`:

```go
res := clitest.Main(t.Context(), app().Main, &clitest.Input{
	Args:  []string{"serve"},
	Vars:  map[string]string{"SRV_SERVE_TIMEOUT": "5s"},
	Files: clitest.Files{"srv.json": `{"serve": {"addr": ":9000"}}`},
})
assert.Equal(t, res.Status, cli.ExitOK, "Main returns ExitOK")
assert.Equal(t, res.Stdout, "listen on :9000 with a timeout of 5s for []\n", "serve writes the address")
```

To check the typed error of a run, or to call a handler, a test builds the `Env` itself:

```go
env, out := clitest.NewEnv(&clitest.Input{Args: []string{"serve", "--adr", ":9000"}})
err := app().Execute(t.Context(), env)
e := assert.ErrorAs[*cli.Error](t, err, "Execute returns an *Error")
assert.Equal(t, e.Suggestion, "--addr", "the error suggests the flag --addr")
assert.Empty(t, out.Stdout(), "the run writes nothing to its standard output")
```

For a list of runs, a test passes the transcript to its golden assertion:

```go
runs := []clitest.Input{
	{Comment: "The help of the command serve.", Args: []string{"serve", "--help"}},
	{Comment: "A config file sets the address.", Args: []string{"serve"}, Files: files},
	{Comment: "An unknown flag suggests a known one.", Args: []string{"serve", "--adr", ":9000"}},
}
golden.Match(t, "transcript.txt", clitest.Transcript(t.Context(), "srv", app().Main, runs...), golden.ShouldUpdate())
```

### The transcript

The second run of the list in the preceding example writes this part of the transcript:

```console
# A config file sets the address.
$ cat srv.json
{"serve": {"addr": ":9000"}}
$ srv serve
listen on :9000 with a timeout of 30s for []
[exit 0]
```

The transcript follows these rules:

- A run with a comment starts with the lines of its comment, each after `# `.
- Each config file that the run read follows, in the order of the first reads. The line `$ cat` and the path precede the content of the file. A file without a final newline gets one, so the next line of the transcript starts on a line of its own.
- The command line of the run follows the prompt `$`. It lists the environment variables of the run in the order of their names, then the name of the program and the arguments. A word with a character that a shell reads as special is in single quotes.
- A run with a standard input pipes the output of `printf` to the program. The command line of a run with `FailStdin` ends with `<&-`, and the command line of a run with `FailStdout` ends with `>/dev/full`.
- The output of the run follows its command line. The output contains the standard output and the standard error in the order of the writes.
- A run ends with its exit status in brackets, such as `[exit 2]`, on a line of its own. After an output without a final newline, the status is on the last line of the output.
- A blank line separates two runs.

The rules for the command lines, the output and the exit status are the rules of the harness of the examples.

### Capabilities left out

- **Assertions and golden files.** A test compares the `Result` and the transcript with the assertion library of its choice. The package `golden` of assert-go registers the flag `-update` on the flag set of the process, and the flag package panics with `flag redefined` on a second flag of that name. `clitest` does not register a flag for that reason.
- **A script language.** `Transcript` runs the inputs of a Go slice, which the compiler checks. testscript covers scripts that run programs as processes, and a program of the package `cli` runs there with `cli.OS()`.
- **A capture of the options of a run.** A transcript of runs with `--flag-debug` shows the value and the source of each option, and a test of a handler builds its options for `cli.NewRun`.
- **A standard error that fails.** `App.Main` ignores the error of the failure that it writes, so such a run tests nothing that a test of `App.Execute` does not.
- **Interactive programs.** A handler that runs a Bubble Tea program tests its model with teatest: `NewTestModel`, `WaitFor` and `RequireEqualOutput`.

### Changes outside the package

- The tests of the 14 examples call `clitest.Transcript`. The examples that read config files pass `os.DirFS(".")`, so their transcripts show the files. The harness `examples/internal/transcript` is deleted, with its 267 lines of code and tests.
- The example `14-testing` shows `clitest.Main`, and `clitest.NewEnv` with `cli.NewRun`.
- The section Testing of the README shows `clitest.Main`, `clitest.NewEnv` and `clitest.Transcript`.
- The depguard rule `main` already admits the imports of `clitest/`, so the lint configuration does not change.
- A changeset adds the package to the changelog of the module `cli`.

### Migration

The package adds API to the module and changes none of the existing API. Each example test replaces `transcript.Run` with `clitest.Input`, the field `Full` with `FailStdout`, and `transcript.Write(ctx, name, runs, app().Main)` with `clitest.Transcript(ctx, name, app().Main, runs...)`. The golden files of the examples gain the config files that their runs read, and the text of the error of `FailStdout` changes.

### Tests

- `NewEnv` passes each field of an `Input` to the `Env`. One case covers each field of the table of the `Env`.
- A read of a path outside `in.Files` returns an error that wraps `fs.ErrNotExist`, for a nil `in.Files` too. A read of `/etc/srv.json` returns the file `etc/srv.json`.
- `Main` returns the exit status and the two outputs, and cancels the context of the run when the program returns.
- The 10 cases of the tests of the harness move to the tests of `Transcript`. New cases cover the comments, the files that a run read, and `FailStdin`.
- A case writes from two or more goroutines to both buffers under the race detector of the gate.
- Coverage and the mutation score are 100%.

## Alternatives considered

### The files of the system by default

A nil `Input.Files` could read the files of the system through `os.ReadFile`, as the harness of the examples does.

**Why not:** a program reads the config file at its default path when the file exists. A test of a program with a default path such as `/etc/srv/config.json` would then pass or fail with the files of the machine. A test that needs files on disk passes `os.DirFS`, and its transcript shows the files that it read.

### Inputs as shell lines

A function such as `clitest.Line("SRV_PORT=8 srv serve --addr :9000")` would parse the syntax of the transcript into an `Input`, so a test would read as its transcript.

**Why not:** the parser would be a second language with errors of its own at run time. A struct literal of `Input` is checked by the compiler.

### Scripts as in go-cmdtest or testscript

go-cmdtest compares files of command lines with their output. testscript runs scripts in the txtar format.

**Why not:** `InProcessProgram` of go-cmdtest sets `os.Args`, `os.Stdout` and `os.Stderr` for each command, so its tests do not run in parallel. Its format merges the two outputs and cannot express the trailing blank lines of an output. testscript starts a process for each command, and its scripts are a second language beside Go. The `Env` of the package `cli` makes a run in process safe without either tool.

### Test helpers as methods of cli.App

A method such as `App.Test(t, args...)` would not need a package of its own.

**Why not:** the standard library keeps test helpers out of the production API, in packages such as `net/http/httptest` and `testing/fstest`. A method of `cli.App` would put them into the API of every program.

### Assertions in clitest

A function such as `clitest.Golden(t, name, runs...)` would compare the transcript itself.

**Why not:** the comparison needs a flag to update the golden file, and that flag collides with the flag of the package `golden`. The choice of an assertion library belongs to the program.

### A recorder of the renderer

A `cli.Renderer` in `clitest` could record the `Help`, the `Failure` and the rows that an app passes to its renderer, so a test would check a failure by its fields.

**Why not:** `App.Execute` returns the `*cli.Error` of a failure with the same fields, and `NewEnv` gives such a test its `Env`. A test of a renderer builds `cli.Help` values directly, as the tests of the module charm do.

### A module of its own

`clitest` could be a module beside `charm`, `toml` and `yaml`.

**Why not:** a module of its own needs a release of its own for each release of `cli`, and it isolates a dependency only when the package has one. `clitest` imports only the standard library and the package `cli`.

### A parameter of the type cli.App

`Main[G any](ctx context.Context, app *cli.App[G], in Input) Result` would take the app itself.

**Why not:** the example `12-errors` wraps `App.Execute` in a main function of its own, and a parameter of the type `cli.App` could not run that function. A function parameter with the signature of `App.Main` runs both kinds of program without a type parameter.

## Drawbacks

- The module `cli` gains a package with 4 types, 3 methods, 3 functions and 2 errors.
- The format of the transcript becomes API. A change of the format changes the golden files of every program that uses `Transcript`.
- A test that relies on files on disk passes `os.DirFS`, because a run does not read the files of the system by default.
- A transcript shows each config file that a run read, so a large fixture makes a long golden file.
- The transcript merges the standard output and the standard error, as a terminal shows them. A test that needs the two apart calls `Main`.
- We estimate the package at 250 lines of code. The harness that it replaces has 112. We have not measured the package, because it does not exist yet.

## Open questions

None.

## Unresolved and future work

- `cli.Env` reads config files and has no file system for writes. A handler that writes files tests them with its own abstraction and a temporary directory.
- This RFC does not propose a runner of script files.
- This RFC does not propose helpers for interactive programs.

## References

| What | Where |
|---|---|
| go-cmdtest v0.4.0: the format of its files and `InProcessProgram` | <https://github.com/google/go-cmdtest/blob/v0.4.0/cmdtest.go> |
| testscript of go-internal v1.16.0: `Main` | <https://github.com/rogpeppe/go-internal/blob/v1.16.0/testscript/exe.go> |
| cobra v1.10.2: `SetArgs`, `SetOut`, `SetErr` and `SetIn` | <https://github.com/spf13/cobra/blob/v1.10.2/command.go> |
| urfave/cli v3.14.0: `Reader`, `Writer`, `ErrWriter` and `OsExiter` | <https://github.com/urfave/cli/blob/v3.14.0/command.go> |
| kong v1.16.1: `Writers` and `Exit` | <https://github.com/alecthomas/kong/blob/v1.16.1/options.go> |
| teatest v2 at the pseudo-version 20261012121918-43427fdbad12 | <https://github.com/charmbracelet/x/tree/main/exp/teatest/v2> |
| The flag `-update` of the package `golden` of assert-go | <https://github.com/dokimasia/assert-go/blob/main/golden/match.go> |
| Go 1.27.2: the panic `flag redefined` and the concurrency of `os.File` | `flag/flag.go:1024` and `os/types.go:17` |

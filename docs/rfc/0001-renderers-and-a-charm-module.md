---
rfc: 0001
title: Renderers for the output of the package cli and a Charm module
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

# RFC-0001: Renderers for the output of the package cli and a Charm module

## Summary

The package `cli` passes its help, the failure of a run and the table of `--flag-debug` to a `Renderer` as data. The renderer `Plain` writes the plain text of the package without a change. `App.Render` registers another renderer. The optional module `go.stealthscale.io/cli/charm` renders the same data with Lip Gloss in a layout of its own. Its help wraps at the width of the output and puts the notes of an option below its text. In a terminal its failure has a badge, the suggestion on a line of its own and a hint to the help. `Env` and `Run` also return the environment of a run as `KEY=value` pairs, because the Charm libraries choose colors from such pairs. The package `cli` still imports only the standard library.

## Motivation

A handler can use the Charm libraries without this RFC. We built a handler that passes `run.Stdin()`, `run.Stdout()` and its context to Bubble Tea v2.1.0, and its test passes. `lipgloss.Fprintln` removed the colors when the handler wrote to a buffer. A handler cannot change two parts of a run:

- **The output of the package.** The package builds the help, the messages of `App.Main` and the table of `--flag-debug`. Without a hook a program cannot change them. Fang v2.0.1 styles this output for Cobra programs. Its entry point is `fang.Execute(ctx, *cobra.Command)`, and its module requires spf13/cobra.
- **The environment of a run.** Without `Run.Environ` a handler reads only the standard streams of its `Env`. The `Fprint` functions of Lip Gloss read `os.Environ`. Bubble Tea takes a list of variables through `WithEnvironment`. Over SSH or in a test the variables of the `Env` differ from those of the process, so the libraries choose colors from the wrong variables.

The hooks belong in the package `cli` because only the package builds the help, the messages and the table. The styles belong in a module of their own. colorprofile v0.4.3 chooses a color profile from terminfo, from a terminal check of the writer and from nine environment variables. `NO_COLOR`, `CLICOLOR_FORCE` and `TERM` are three of them. On Windows it reads three more variables. The package `cli` does not import a module outside the standard library, so it cannot use colorprofile.

The plain help also does not fit a narrow terminal. It puts the help text and the notes of an option on one line. The widest line of the help of the example `cli/tags` has 150 columns, and 7 of its 16 lines are wider than 80 columns. A terminal of 80 columns breaks those lines at its edge.

## Detailed design

### Components

| Component | Module | Responsibility |
|---|---|---|
| `Help`, `HelpCommand`, `HelpOption` | `cli` | The help of an app or of a command as data |
| `DebugRow` | `cli` | One row of the table of `--flag-debug` |
| `Failure` | `cli` | The error of a run as `App.Main` writes it, with its command, its suggestion and its kind |
| `Error.Suggestion` | `cli` | The suggestion of an invalid input in a field of its own |
| `Renderer` | `cli` | Writes the help, a failure and the table of `--flag-debug` |
| `Plain` | `cli` | The renderer of an app without `App.Render`. It writes the plain text of the package |
| `App.Render` | `cli` | Registers the renderer of an app |
| `Env.Environ`, `Run.Environ`, `Environ` | `cli` | The environment of a run as `KEY=value` pairs |
| `Theme`, `DefaultTheme` | `cli/charm` | The Lip Gloss styles of the twelve parts of the output |
| `Renderer`, `New`, `Register` | `cli/charm` | Render the output in the layout of the module with the styles of a theme |

### The output as data

```go
// Help contains the help of an app or of one of its commands for a Renderer.
type Help struct {
	App     string // the name of the app
	Command string // the name of the command, or empty for the help of the app
	Text    string // the help text of the app or of the command
	// Usage contains the parts of the usage line after the names, such as "[options]" and "[NAME]".
	Usage []string
	// Commands contains the commands of the app in the order of their registration. The help of a
	// command does not list commands.
	Commands []HelpCommand
	// Arguments contains the positional arguments of the command in the order of their fields.
	Arguments []HelpOption
	// Options contains the options of the command in the order of their flags.
	Options []HelpOption
	// Globals contains the global options in the order of their flags. -h and --help are among them.
	Globals []HelpOption
}

// HelpCommand is a command in the help of an app.
type HelpCommand struct {
	Name string // the name of the command, such as serve
	Text string // the help text of the command
}

// HelpOption is an option or a positional argument in the help.
type HelpOption struct {
	Name        string // the flag such as --addr, or the label of an argument such as NAME
	Short       string // the short flag such as -a, or empty
	Placeholder string // the type or the values such as string or dev|prod, or empty for a bool
	Text        string // the help text
	Default     string // the default in quotes such as ":8080" or "***", or empty without one
	Env         string // the environment variable, or empty
	Key         string // the config key, or empty
	Required    bool   // a run fails when no source sets the option
}

// DebugRow is a row of the table of --flag-debug.
type DebugRow struct {
	Option string // the flag, or the label of a positional argument
	Value  string // the value as the table prints it, with *** for a secret option
	Source Source // the source of the value
}

// Failure is the error of a run as App.Main writes it.
type Failure struct {
	Err        error  // the error of the run
	App        string // the name of the app
	Command    string // the command of the run, or empty
	Message    string // the message without the names and the suggestion
	Suggestion string // the name that the package suggests in place of the input, or empty
	Usage      bool   // the error is an invalid command line, environment variable or config file
}
```

`App.Main` fills a `Failure` from the error of the run:

- For an `*Error` it sets `Usage` and takes `Command`, `Message` and `Suggestion` from the `*Error`.
- For an error that wraps an `*Error` it sets `Usage`. `Message` is the text of the whole error.
- For any other error `Message` is the text of the error.

`Error` keeps its suggestion in a field of its own. The text of `Error.Message` and of `Error.Error` contains the suggestion, such as `serve: unknown flag --adr (did you mean --addr?)`.

```go
type Error struct {
	Err        error  // the sentinel
	Command    string // the command of the run, or empty for an input before the command
	Input      string // the command, the flag, the config key, the argument or the config file at fault
	Value      string // the value at fault, or empty
	Suggestion string // the name that the package suggests in place of Input, or empty
	Source     Source // the source of Value, or the zero Source
	// contains unexported fields
}
```

### The renderer

```go
// Renderer writes the help, the message of a failure and the table of --flag-debug in place of the
// package. environ is the environment of the run as KEY=value pairs, which Env.Environ returns.
//
// The package calls a Renderer on the goroutine of App.Execute or App.Main, and returns the error
// of Help and of Debug unchanged. Concurrent runs of one app share its Renderer, so a Renderer must
// be safe for concurrent use.
type Renderer interface {
	// Help writes the help h to w.
	Help(w io.Writer, environ []string, h *Help) error
	// Error writes the failure f to w.
	Error(w io.Writer, environ []string, f *Failure) error
	// Debug writes the rows of the table of --flag-debug to w.
	Debug(w io.Writer, environ []string, rows []DebugRow) error
}

// Plain is the Renderer of an app that registers none. It writes text without colors or styles in
// the columns of the help. A Renderer can embed Plain to replace only some of its methods.
type Plain struct{}

func (Plain) Help(w io.Writer, environ []string, h *Help) error
func (Plain) Error(w io.Writer, environ []string, f *Failure) error
func (Plain) Debug(w io.Writer, environ []string, rows []DebugRow) error

// Render registers r as the renderer of the output of the app. A later call replaces an earlier
// one. An app without a call uses Plain.
//
// App.Execute returns an error that wraps ErrDeclaration when r is nil.
func (a *App[G]) Render(r Renderer)
```

`Render` follows the rules of `Command`, `Convert` and `Format`. A program calls it before the first run. It is not safe for concurrent use. `App.Execute` and `App.Main` are safe for concurrent use once the last registration has returned.

### The environment

```go
type Env struct {
	Lookup   func(name string) (string, bool)  // reports the value of the environment variable name
	Environ  func() []string                   // returns the environment as KEY=value pairs
	ReadFile func(name string) ([]byte, error) // returns the content of the config file name
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Args     []string // the arguments after the name of the program
}

// Environ returns an Environ function that returns the variables of a copy of vars as KEY=value
// pairs in the order of their names. Each call returns a list of its own.
func Environ(vars map[string]string) func() []string

// Environ returns the environment of the run as KEY=value pairs, or nil for an Env without
// Environ.
func (r *Run[G, C]) Environ() []string
```

`OS` sets `Environ` to `os.Environ`. The package itself reads its variables through `Lookup`. A test that sets the variables of its options and the variables of a terminal library builds both functions from one map:

```go
vars := map[string]string{"SRV_SERVE_TIMEOUT": "5s", "NO_COLOR": "1"}
env := &cli.Env{Lookup: cli.Vars(vars), Environ: cli.Environ(vars), Stdout: &stdout, Stderr: &stderr}
```

### The calls of the renderer

| Output | Method | Writer | When |
|---|---|---|---|
| The help of the app or of a command | `Help` | `env.Stdout` | The command line has `-h` or `--help` |
| The help of the app | `Help` | `env.Stderr` | The command line does not have a command. The failure follows |
| The table of `--flag-debug` | `Debug` | `env.Stderr` | Before the handler runs, when `--flag-debug` or `<APP>_FLAG_DEBUG` is true |
| A failure | `Error` | `env.Stderr` | `App.Main` receives an error from `App.Execute` |

### Failure handling

| Case | Behaviour |
|---|---|
| `Help` or `Debug` returns an error | `App.Execute` returns the error unchanged. `App.Main` then returns `ExitFailure` |
| `Error` returns an error | `App.Main` ignores it and returns the exit status of the run |
| `App.Render(nil)` | `App.Execute` returns an error that wraps `ErrDeclaration` for every command line |
| A renderer panics | `App.Execute` does not recover the panic, as it does not recover the panic of a handler |
| `Env.Environ` is nil | The renderer and `Run.Environ` receive nil |

### The module charm

```go
// Theme contains the styles of the parts of the output of a Renderer.
type Theme struct {
	Heading    lipgloss.Style // a heading of the help, such as USAGE and OPTIONS
	Program    lipgloss.Style // the name of the app
	Command    lipgloss.Style // the name of a command
	Argument   lipgloss.Style // a part of the usage line after the names, and the placeholder of a value
	Flag       lipgloss.Style // a flag, or the label of a positional argument
	Text       lipgloss.Style // a help text
	Note       lipgloss.Style // the default, the environment variable and the config key of an option
	Required   lipgloss.Style // the word required in the notes of a required option
	ErrorBadge lipgloss.Style // the badge ERROR before the message of a failure
	ErrorText  lipgloss.Style // the message of a failure
	Suggestion lipgloss.Style // the name that a failure suggests
	Hint       lipgloss.Style // the words around the suggested name and around the help command
}

// DefaultTheme returns the theme of Register. Its colors come from the 16-color palette of the
// terminal, so the terminal decides each shade on a light and on a dark background.
func DefaultTheme() Theme

// Renderer is a cli.Renderer that writes the output of an app in the styles of a Theme.
type Renderer struct {
	// contains unexported fields
}

// New returns a Renderer with a copy of the theme t.
func New(t *Theme) *Renderer

// Register registers a Renderer with DefaultTheme as the renderer of app.
func Register[G any](app *cli.App[G])

func (r *Renderer) Help(w io.Writer, environ []string, h *cli.Help) error
func (r *Renderer) Error(w io.Writer, environ []string, f *cli.Failure) error
func (r *Renderer) Debug(w io.Writer, environ []string, rows []cli.DebugRow) error
```

`DefaultTheme` styles the parts as follows:

| Part | Style |
|---|---|
| `Heading`, `Program` | Bold |
| `Command` | Cyan |
| `Flag` | Magenta |
| `Argument`, `Note`, `Hint` | Faint |
| `Required` | Yellow |
| `Text`, `ErrorText` | No style |
| `ErrorBadge` | Bold bright white on red, with a space on each side |
| `Suggestion` | Bold green |

A program registers the default theme with one call, or changes a style and passes the theme to `New`:

```go
a := cli.New[Global]("srv", "Serve the store.")
charm.Register(a)
```

```go
t := charm.DefaultTheme()
t.Flag = t.Flag.Foreground(lipgloss.Green)
a.Render(charm.New(&t))
```

### The help of the module charm

The example `charm/help` writes this help on a pipe. A terminal receives the same text with the styles of the theme.

```console
$ srv serve --help | cat
Run the server of the store.

USAGE
  srv serve [options] [STORE]

ARGUMENTS
  STORE               Name of the store.
                      default "main"

OPTIONS
  -a, --addr string   Address to listen on.
                      default ":8080" · env SRV_SERVE_ADDR · key serve.addr
      --client-certificate-authority string
                      Certificate authority of the clients.
                      env SRV_SERVE_CLIENT_CERTIFICATE_AUTHORITY
                      key serve.client-certificate-authority
  -q, --quiet         Do not log the requests.
                      env SRV_SERVE_QUIET · key serve.quiet
      --token string  Bearer token of the requests.
                      required · env SRV_SERVE_TOKEN · key serve.token

GLOBAL OPTIONS
  -c, --config file   Config file in JSON.
                      env SRV_CONFIG
      --flag-debug    Print the value and the source of each option before the
                      command runs.
                      env SRV_FLAG_DEBUG
  -h, --help          Show the help.
```

The layout follows these rules:

- The help text comes first. The usage line follows under the heading `USAGE`. The sections `COMMANDS`, `ARGUMENTS`, `OPTIONS` and `GLOBAL OPTIONS` follow in that order. The help leaves out a section without entries.
- The texts of all sections start at one column. The column is the indent of 2, the widest key of at most 32 columns and a gap of 2.
- The text of a key wider than 32 columns starts on the line below the key, at the same column.
- A section with a short flag aligns its long flags. An option without a short flag gets 4 spaces before its long flag. A section without short flags starts each key at the indent.
- The notes of an option follow its text on the lines below it, in the order required, default, variable and config key. A separator ` · ` joins the notes of a line. A note does not break across two lines, so a note that does not fit a line starts the next one.
- The text wraps at the width of the output minus the column, and at no less than 20 columns. The width is the value of the last valid `COLUMNS` of the environment, at most 100. Without a valid `COLUMNS` the width is 80.
- No line ends in a space.

### The failures and the table of the module charm

A terminal receives a failure in this form, with the badge in red:

```console
$ srv serve --adr :9000
 ERROR  unknown flag --adr
        Did you mean --addr?
        Run srv serve --help for the usage.
```

The renderer writes the line of the suggestion when the failure has a suggestion. It writes the hint to the help when `Failure.Usage` is true. The message wraps at the width of the output, and its other lines start below its first. A pipe or a file receives the line of `Plain`:

```console
$ srv serve --adr :9000 2>&1 | cat
srv: serve: unknown flag --adr (did you mean --addr?)
```

A terminal receives the table of `--flag-debug` in the columns of `Plain`. The header has the style `Heading`, the options `Flag`, the values `Text` and the sources `Note`. A pipe or a file receives the table of `Plain`.

### Color profiles

Each method writes through `colorprofile.NewWriter(w, environ)`. `Error` and `Debug` call `Plain` when the profile is NoTTY. colorprofile chooses the profile from the writer and the variables:

- A writer that is not a terminal does not get styles unless `CLICOLOR_FORCE` is true.
- `TERM=dumb` turns the styles off unless `CLICOLOR_FORCE` is true.
- `NO_COLOR` on a terminal removes the colors and keeps bold and faint text.

### Invariants

1. `Plain` writes the same bytes as the package without renderers. The golden files of the package and of the examples contain those bytes.
2. For the profile NoTTY, `Error` and `Debug` of the module charm write the same bytes as `Plain`. Its `Help` does not write an escape sequence or a line that ends in a space. Property tests check both over generated values.
3. The module `go.stealthscale.io/cli` imports only the standard library. The production code of the module charm imports only Lip Gloss, colorprofile and the module `cli`. depguard rejects any other import.
4. A renderer does not decide an exit status. `App.Main` returns the status of the run.
5. A `charm.Renderer` does not change after `New`, so concurrent runs can share it.

### Changes outside the two modules

- `go.work` uses `./charm`.
- The depguard rule `charm` admits `charm.land/lipgloss/v2` and `github.com/charmbracelet/colorprofile` in the production code of `charm/`. The rule `charm-tests` admits Lip Gloss and assert-go in its tests.
- wrapcheck accepts the errors of `cli.Renderer` and of the methods of `Plain` without a wrap. The package returns the error of a renderer unchanged, as it returns the error of a handler. `Plain` wraps the error of its writer itself.
- The module charm gets a changeset. The package `cli` has no release, so its changeset for the whole package covers the new API.
- The examples get `examples/charm/help` with a golden transcript. The harness of the transcripts passes the variables of a run to `Env.Environ` too, so a run with `COLUMNS=60` wraps its help at 60 columns.
- The README of the repository and the documentation of the package `cli` get a section on the output.

### Migration

A program does not need to change. An app without `App.Render` uses `Plain`, and `Plain` writes the plain text of the package. The package builds the help, the table and the failure as data and passes them to the renderer. A program that registers the module charm changes its help on every writer, so the golden files of its help change too.

### Tests

- The golden files of the help and of `--flag-debug` in the package and in the examples match the output of `Plain` without an update.
- `App.Render(nil)` makes `App.Execute` return `ErrDeclaration`.
- `App.Execute` returns the error of `Help` or of `Debug` unchanged.
- `Run.Environ` and the `environ` of each renderer call equal the result of `Env.Environ`.
- `Environ` returns the pairs of a map in the order of their names.
- Golden files of the module charm contain the help, the failures and the table with `CLICOLOR_FORCE=1`, the help without styles, the help at 60 columns and a failure with `NO_COLOR`.
- Exact cases check the rules of the layout for keys of 32 and of 33 columns, and for notes that fill the text column or do not fit it. Two more cases check an option without text and a help without text.
- Property tests check invariant 2, and the fuzz target `FuzzHelp` runs the property of the help.
- Coverage is 100% in both modules. dokimi-mutate-go detects 165 of 165 mutants of the changed files of the package `cli` and 163 of 163 mutants of the module charm.

## Alternatives considered

### Templates

Cobra v1.10.2 takes a help template through `SetHelpTemplate(string)`. urfave/cli v3.14.0 renders its help from `text/template` templates. A template does not need new types. `text/template` is in the standard library.

**Why not:** a template cannot measure the columns of the help without template functions. The package would have to export those functions too. A typo in a template fails only when a program writes its help. A template also does not cover the messages of `App.Main`.

### A function that receives the internal model

Cobra's `SetHelpFunc(func(*Command, []string))` and kong v1.16.1's `HelpPrinter func(options HelpOptions, ctx *Context) error` give the help function the model of the parser. A renderer then gets every detail without new data types.

**Why not:** the declarations of the package `cli` are unexported. Exporting them makes the model of the parser part of the API. The data types of this RFC contain only what the help shows.

### A renderer in a package variable

urfave/cli v3.14.0 exports `var HelpPrinter HelpPrinterFunc`. A program replaces the variable with its own function.

**Why not:** every app in a process shares a package variable. Tests that run in parallel race on it. The package `cli` does not keep package state.

### Colors in the package cli without a dependency

The package could write ANSI codes itself and read `NO_COLOR` and `TERM` through `Env.Lookup`.

**Why not:** colorprofile v0.4.3 decides the profile from terminfo, a terminal check of the writer and nine environment variables. It reads three more variables on Windows. The package would need a second copy of those rules.

### Fang on Cobra

A program could declare its commands with Cobra and run them with `fang.Execute`.

**Why not:** a Cobra program declares each flag, variable and config key by hand. The package `cli` exists to derive them from the fields of structs.

### Run.Lookup in place of Run.Environ

A handler could read single variables through the `Lookup` of its `Env`.

**Why not:** colorprofile and Bubble Tea take the environment as a list of `KEY=value` pairs in `colorprofile.NewWriter(w, environ)` and `tea.WithEnvironment(env)`. A lookup function cannot produce that list.

### Env.Environ in place of Env.Lookup

One field for the environment would replace two.

**Why not:** on Windows `os.LookupEnv` asks the system through `GetEnvironmentVariable`, so the rules of the system for names apply. A scan of a list of pairs needs its own copy of those rules.

### The columns of Plain in the module charm

The module charm could write the help in the columns of `Plain` with styles. One property test would then compare both renderers for every profile.

**Why not:** the columns of `Plain` put the text and the notes of an option on one line. The widest line of the help of the example `cli/tags` has 150 columns. Styles do not make such a line readable in a terminal of 80 columns.

### The help of Plain on a pipe

The module charm could write the help of `Plain` to a writer that is not a terminal, as it does for a failure and for the table.

**Why not:** a user who pipes the help to a pager or to `grep` reads the help of the terminal. The layout of the module charm without styles is that help. A failure on a pipe goes to a script or a log, which reads one line. The table keeps the columns of `Plain` on every writer.

### The width of the terminal

POSIX defines `COLUMNS` as the preferred width of the user. Without it the implementation determines a width that fits the terminal. Fang v2.0.1 calls `term.GetSize(os.Stdout.Fd())`.

**Why not:** the renderer receives an `io.Writer`, which is a terminal only in a run of `cli.OS`. A test of the query needs a pseudo-terminal on Linux, macOS and Windows, the three systems of the CI. The width that `COLUMNS` sets makes the output depend only on the writer and the `Env` of the run. A user who wants another width sets `COLUMNS`.

### The background query and the styled usage of Fang

Fang prints the usage in a styled block. It picks its colors for a light or a dark background and fits the help to the width of the terminal.

**Why not:** Fang reads `os.Stdin` and `os.Stdout` directly for the background query and the width. `lipgloss.HasDarkBackground(in, out)` reads the reply of the terminal from its input. In a run of the package `cli` that input belongs to the handler. The colors of the 16-color palette fit both backgrounds because the terminal decides each shade.

## Drawbacks

- The package `cli` gains 7 types, 5 methods, 1 function and 2 fields. Every field of `HelpOption` is API, so a new note in the help needs a new field.
- The production code of the package grows by 197 lines, doc comments included. The change adds 342 lines and removes 145.
- A program that imports the module charm links 19 more packages from 17 more modules and 51 more packages of the standard library. The binary of the example `charm/help` grows by 41.5% from 3,313,927 to 4,690,183 bytes.
- `Env` has two views of the environment: `Lookup` and `Environ`. A test that needs both sets both. The function `Environ` only helps it build the second view from the same map.
- The help of the module charm differs from the help of `Plain` on every writer. A program that switches renderers updates the golden files of its help.
- The table of the module charm repeats the columns of `Plain` with styles. No test compares the styled table with the table of `Plain`, so a change of the columns of `Plain` needs the same change in the module charm.
- A shell that does not export `COLUMNS` leaves the help at 80 columns in a narrower terminal. The terminal then breaks the long lines at its edge.

## Open questions

None.

## Unresolved and future work

- This RFC does not propose the flag `--version`, the shell completion and the man pages that Fang adds to Cobra programs.
- This RFC does not propose a prompt of Huh for a missing required option.
- The renderer takes the width of the output from `COLUMNS` and does not query the terminal.

## References

| What | Where |
|---|---|
| Fang v2.0.1: `fang.Execute`, its background query and its width | <https://github.com/charmbracelet/fang/tree/v2.0.1> |
| Lip Gloss v2.0.6: `Fprint`, `HasDarkBackground`, `Wrap` and the 16 colors | <https://github.com/charmbracelet/lipgloss/tree/v2.0.6> |
| colorprofile v0.4.3: `Detect` and `NewWriter` | <https://github.com/charmbracelet/colorprofile/blob/v0.4.3/env.go> |
| Bubble Tea v2.1.0: `WithEnvironment` | <https://github.com/charmbracelet/bubbletea/blob/v2.1.0/options.go> |
| Cobra v1.10.2: `SetHelpFunc` and `SetHelpTemplate` | <https://github.com/spf13/cobra/blob/v1.10.2/command.go> |
| kong v1.16.1: `HelpPrinter` | <https://github.com/alecthomas/kong/blob/v1.16.1/help.go> |
| urfave/cli v3.14.0: `HelpPrinter` and its templates | <https://github.com/urfave/cli/blob/v3.14.0/help.go> |
| POSIX.1-2024, section 8.3: `COLUMNS` | <https://pubs.opengroup.org/onlinepubs/9799919799/basedefs/V1_chap08.html> |
| NO_COLOR | <https://no-color.org/> |
| The example `charm/help` with and without `charm.Register`, built with `-trimpath -ldflags='-s -w'` under Go 1.27.2 on linux/amd64 | Measured on 2026-10-12 |

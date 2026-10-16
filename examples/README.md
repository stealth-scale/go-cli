<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# examples/

Each example is a runnable program in a directory of its own. The [README](../README.md) of the
repository describes the features that the examples show. The module
`go.stealthscale.io/cli/examples` contains every example. It is not released.

| Example | Shows |
|---|---|
| [`01-quickstart`](01-quickstart) | the smallest program: a global option, a command with an optional argument and two flags, the help, and the exit status of an error |
| [`02-tags`](02-tags) | every tag of an option: `help`, `default`, `enum`, `required`, `name`, `short`, `env`, `key` and `secret` |
| [`03-commands`](03-commands) | three commands that share embedded options, global flags before and after the command, a required enum, the error of a handler, the suggestion for a command, a command with a handler, a hook and two subcommands, a hook of the app that writes `Run.Command`, a help text with a description after a blank line, and an option text of two lines |
| [`04-arguments`](04-arguments) | positional arguments: a required argument, an optional one and a list, the argument `--`, a lone `-`, and input files that `Run.Open` opens, with `-` for the standard input |
| [`05-groups`](05-groups) | groups of options: nested structs, embedded structs, and the tag `prefix` |
| [`06-types`](06-types) | an option of each type that the package converts without a registration, and the error of each conversion |
| [`07-conversions`](07-conversions) | types of the program: a `flag.Value`, an `encoding.TextUnmarshaler`, the registrations of `App.Convert`, and a registration that replaces a built-in conversion |
| [`08-config`](08-config) | config files in JSON and INI, a format for another extension, a key that two commands share, and an unknown key |
| [`09-yaml`](09-yaml) | a YAML config file through `yaml.Register`, with a map whose keys are integers |
| [`10-toml`](10-toml) | a TOML config file through `toml.Register`, with a date |
| [`11-sources`](11-sources) | the order of the sources of an option, `Run.Source` and the kind of a source, the flag `--flag-debug`, and an empty path of the config file |
| [`12-errors`](12-errors) | a program that inspects the error of `App.Execute` in place of `App.Main`, with a hint for each sentinel that its runs meet |
| [`13-charm`](13-charm) | the help, the failures and the table of `--flag-debug` through `charm.New` with a theme, a help at the width in `COLUMNS`, and `Run.Environ` for a style of a handler |
| [`14-testing`](14-testing) | the tests of a program without commands: runs with an `Env`, input files of `clitest.Files`, an `Env` of `cli.Vars` and `cli.Environ`, the handler of the app through `NewRun`, and a context that has ended |
| [`15-renderer`](15-renderer) | a renderer for each run through `App.RenderFor`: the option `--format` selects `cli.Plain` or a renderer of its own that writes the help, the version, the failures and the rows of `--flag-debug` as JSON, a version of `App.Version` from variables of the linker, and exit statuses of its own through `App.ExecuteRenderer` and `App.Failure` |
| [`16-workspace`](16-workspace) | a workspace file that `App.LocateConfig` finds from `--chdir` upward, with plans that `App.IgnoreKeys` leaves to the handler and `cli.Decode` decodes |
| [`17-completion`](17-completion) | the command `completion` and the candidates of a command line, a function of `App.Complete` that reads the plans of the config file, and a test that runs the script of bash, zsh, fish and PowerShell in its shell |

Each example imports the module `go.stealthscale.io/cli`. The examples `09-yaml`, `10-toml` and
`13-charm` also import the module of their name.

Run an example from the directory `examples/`:

```sh
go run ./01-quickstart hello Ada --times 2
```

The test of each example passes the command lines of its transcript to `clitest.Transcript` of the
package `go.stealthscale.io/cli/clitest`. It compares the transcript with
`testdata/golden/transcript.txt`, so the golden file shows the config files that the runs read and
what the example prints. This command writes the transcripts again after a change:

```sh
go test ./... -update
```

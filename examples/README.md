<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# examples/

Each example is a runnable program in the directory `examples/<module>/<name>`. `<module>` is the
module of this repository that the example uses: `cli`, `yaml` or `toml`. The module
`go.stealthscale.io/cli/examples` contains every example. It is not released.

| Example | Shows |
|---|---|
| [`cli/quickstart`](cli/quickstart) | the smallest program: a global option, a command with an optional argument and two flags, the help, and the exit status of an error |
| [`cli/commands`](cli/commands) | three commands that share embedded options, global flags before and after the command, a required enum, the error of a handler, and the suggestion for a command |
| [`cli/tags`](cli/tags) | every tag of an option: `help`, `default`, `enum`, `required`, `name`, `short`, `env`, `key` and `secret` |
| [`cli/types`](cli/types) | an option of each type that the package converts without a registration, and the error of each conversion |
| [`cli/conversions`](cli/conversions) | types of the program: a `flag.Value`, an `encoding.TextUnmarshaler`, the registrations of `App.Convert`, and a registration that replaces a built-in conversion |
| [`cli/groups`](cli/groups) | groups of options: nested structs, embedded structs, and the tag `prefix` |
| [`cli/arguments`](cli/arguments) | positional arguments: a required argument, an optional one and a list, the argument `--`, and a lone `-` |
| [`cli/sources`](cli/sources) | the order of the sources of an option, `Run.Source`, the flag `--flag-debug`, and an empty path of the config file |
| [`cli/config`](cli/config) | config files in JSON and INI, a format for another extension, a key that two commands share, and an unknown key |
| [`cli/errors`](cli/errors) | a program that inspects the error of `App.Execute` in place of `App.Main` |
| [`cli/testing`](cli/testing) | the tests of a program: runs with an `Env`, a handler through `NewRun`, the standard input, and a context that has ended |
| [`yaml/config`](yaml/config) | a YAML config file through `yaml.Register`, with a map whose keys are integers |
| [`toml/config`](toml/config) | a TOML config file through `toml.Register`, with a date |

Run an example from the directory `examples/`:

```sh
go run ./cli/quickstart hello Ada --times 2
```

The test of each example runs the command lines of its transcript. It compares the output of the
runs with `testdata/golden/transcript.txt`, so the golden file shows what the example prints. This
command writes the transcripts again after a change:

```sh
go test ./cli/... ./yaml/... ./toml/... -update
```

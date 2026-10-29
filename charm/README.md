<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# go.stealthscale.io/cli/charm

[![Go Reference](https://pkg.go.dev/badge/go.stealthscale.io/cli/charm.svg)](https://pkg.go.dev/go.stealthscale.io/cli/charm)
[![CI](https://github.com/stealth-scale/go-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/stealth-scale/go-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/tag/stealth-scale/go-cli?filter=charm/v*&sort=semver&label=release)](CHANGELOG.md)
[![License](https://img.shields.io/github/license/stealth-scale/go-cli)](https://github.com/stealth-scale/go-cli/blob/main/LICENSE)

The package `charm` renders the output of the apps of
[`go.stealthscale.io/cli`](https://github.com/stealth-scale/go-cli) with
[Lip Gloss](https://github.com/charmbracelet/lipgloss). It renders the help, the failures and the
table of `--flag-debug`. It writes the version of an app through `cli.Plain`, so the program
controls the whole text of the version. The package `cli` imports only the standard library, so
the renderer is in a module of its own.

## Install

```sh
go get go.stealthscale.io/cli/charm
```

The module requires Go 1.27.2 or later and imports `charm.land/lipgloss/v2` and
`github.com/charmbracelet/colorprofile`.

## Register the renderer

`charm.Register` registers a renderer with the styles of `charm.DefaultTheme`. A program calls it
before the app runs:

```go
a := cli.New[Global]("srv", "Serve the store.")
charm.Register(a)
a.Command("serve", "Run the server.", serve)
```

## Output

The help puts the notes of each option below the text of the option:

```console
$ srv serve --help | cat
Run the server.

USAGE
  srv serve [options]

OPTIONS
  -a, --addr string         Address to listen on.
                            default ":8080" · env SRV_SERVE_ADDR
                            key serve.addr
      --origins string,...  Origins that may connect.
                            env SRV_SERVE_ORIGINS · key serve.origins
      --timeout duration    Timeout of a request.
                            default "30s" · env SRV_SERVE_TIMEOUT
                            key serve.timeout

GLOBAL OPTIONS
  -c, --config file         Config file in JSON or INI.
                            default "srv.json" · env SRV_CONFIG
      --flag-debug          Print the value and the source of each option before
                            the command runs.
                            env SRV_FLAG_DEBUG
  -h, --help                Show the help.
```

A terminal receives a failure with a red badge, the suggestion and a hint to the help:

```console
$ srv serve --adr :9000
 ERROR  unknown flag --adr
        Did you mean --addr?
        Run srv serve --help for the usage.
```

The form of the output depends on the writer and on the environment of the run:

- The help wraps at the width in the environment variable `COLUMNS`, at most 100 columns. Without
  `COLUMNS` it wraps at 80 columns.
- A terminal receives colors and styles. colorprofile detects whether a writer is a terminal.
- A pipe or a file receives the help without styles. It receives the failures and the table of
  `--flag-debug` of `cli.Plain`.
- `NO_COLOR` removes the colors and keeps bold and faint text. `CLICOLOR_FORCE` writes the styles
  to any writer.
- Every writer receives the version of `cli.Plain`.

## Themes

A program changes the styles of `charm.DefaultTheme` and passes the theme to `charm.New`:

```go
t := charm.DefaultTheme()
t.Heading = t.Heading.Underline(true)
t.Flag = t.Flag.Foreground(lipgloss.Green)
a.Render(charm.New(&t))
```

`charm.Theme` has one `lipgloss.Style` for each part of the output:

| Field | Part of the output |
|---|---|
| `Heading` | A heading of the help, such as `USAGE` and `OPTIONS` |
| `Program` | The name of the app |
| `Command` | The name of a command |
| `Argument` | A part of the usage line after the names, and the placeholder of a value |
| `Flag` | A flag, or the label of a positional argument |
| `Text` | A help text |
| `Note` | The default, the environment variable and the config key of an option |
| `Required` | The word `required` in the notes of a required option |
| `ErrorBadge` | The badge `ERROR` before the message of a failure |
| `ErrorText` | The message of a failure |
| `Suggestion` | The name that a failure suggests |
| `Hint` | The words around the suggested name and around the help command |

`DefaultTheme` takes its colors from the 16-color palette of the terminal, so the terminal decides
each shade on a light and on a dark background. `charm.New` copies the theme. A renderer does not
change after `New`, so concurrent runs can share it.

A handler styles its own output with the same environment. `Run.Environ` returns the environment of
the run for colorprofile, as the example
[`13-charm`](https://github.com/stealth-scale/go-cli/tree/main/examples/13-charm) shows.

## Versions

The module has its own version, its own tags and its own changelog. Its tags are `charm/vX.Y.Z`,
and [CHANGELOG.md](CHANGELOG.md) lists the changes of each release.

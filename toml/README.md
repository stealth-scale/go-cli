<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# go.stealthscale.io/cli/toml

[![Go Reference](https://pkg.go.dev/badge/go.stealthscale.io/cli/toml.svg)](https://pkg.go.dev/go.stealthscale.io/cli/toml)
[![CI](https://github.com/stealth-scale/go-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/stealth-scale/go-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/tag/stealth-scale/go-cli?filter=toml/v*&sort=semver&label=release)](CHANGELOG.md)
[![License](https://img.shields.io/github/license/stealth-scale/go-cli)](https://github.com/stealth-scale/go-cli/blob/main/LICENSE)

The package `toml` reads the TOML config files of the apps of
[`go.stealthscale.io/cli`](https://github.com/stealth-scale/go-cli). The package `cli` imports only
the standard library, so the TOML parser is in a module of its own.

## Install

```sh
go get go.stealthscale.io/cli/toml
```

The module requires Go 1.27.2 or later and imports `github.com/BurntSushi/toml`.

## Register the format

`toml.Register` registers the format for the extension `.toml`. A program calls it before the app
runs:

```go
a := cli.New[Global]("schedule", "Schedule backups.")
toml.Register(a)
a.Command("backup", "Print the schedule of the backups.", backup)
```

`toml.Decode` is the format itself. `App.Format` registers it for another extension:

```go
a.Format(toml.Decode, ".tml")
```

## Read a file

A table contains the keys of a command or of a group, and an array contains the values of a slice.
The program of the example
[`10-toml`](https://github.com/stealth-scale/go-cli/tree/main/examples/10-toml) reads this file:

```toml
[backup]
start = 2026-10-12T02:00:00Z
every = "12h"
targets = ["database", "uploads"]

[backup.retention]
days = 30
```

```console
$ schedule backup
back up ["database" "uploads"] from 2026-10-12T02:00:00Z every 12h0m0s, keep 30 days
```

## Rules

- A table is a `map[string]any`, and an array is a `[]any`. A date and a time are a `time.Time`, and
  any other value is its Go value.
- An option converts each value through its text. A `time.Time` gives the text of `MarshalText`,
  and any other value the text of `fmt.Sprint`. The option `--start` of the type `time.Time` takes
  the date `2026-10-12T02:00:00Z` of the file.
- The run fails with an error that wraps `cli.ErrInvalidValue` for a value that does not convert to
  the type of its option:

  ```console
  $ schedule -c broken.toml backup
  schedule: backup: invalid --every "12" from key backup.every of broken.toml: time: missing unit in duration "12"
  ```

- The run fails with an error that wraps `cli.ErrConfigFile` for data that is not TOML.

`cli.Decode` decodes a document of the program from the tree that `toml.Decode` returns. The
[README of go-cli](https://github.com/stealth-scale/go-cli#documents) describes its rules.

## Versions

The module has its own version, its own tags and its own changelog. Its tags are `toml/vX.Y.Z`, and
[CHANGELOG.md](CHANGELOG.md) lists the changes of each release.

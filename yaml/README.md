<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# go.stealthscale.io/cli/yaml

[![Go Reference](https://pkg.go.dev/badge/go.stealthscale.io/cli/yaml.svg)](https://pkg.go.dev/go.stealthscale.io/cli/yaml)
[![CI](https://github.com/stealth-scale/go-cli/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/stealth-scale/go-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/tag/stealth-scale/go-cli?filter=yaml/v*&sort=semver&label=release)](CHANGELOG.md)
[![License](https://img.shields.io/github/license/stealth-scale/go-cli)](https://github.com/stealth-scale/go-cli/blob/main/LICENSE)

The package `yaml` reads the YAML config files of the apps of
[`go.stealthscale.io/cli`](https://github.com/stealth-scale/go-cli). The package `cli` imports only
the standard library, so the YAML parser is in a module of its own.

## Install

```sh
go get go.stealthscale.io/cli/yaml
```

The module requires Go 1.27.2 or later and imports `go.yaml.in/yaml/v3`.

## Register the format

`yaml.Register` registers the format for the extensions `.yaml` and `.yml`. A program calls it
before the app runs:

```go
a := cli.New[Global]("notify", "Send notifications.")
yaml.Register(a)
a.Command("send", "Print the options of the notifications.", send)
```

`yaml.Decode` is the format itself. `App.Format` registers it for another extension:

```go
a.Format(yaml.Decode, ".conf")
```

## Read a file

A mapping contains the keys of a command or of a group, or the entries of a map. A sequence
contains the values of a slice. The program of the example
[`09-yaml`](https://github.com/stealth-scale/go-cli/tree/main/examples/09-yaml) reads this file:

```yaml
send:
  channels: [email, chat]
  window: 15m
  retries: 3
  smtp:
    host: smtp.example.com
    port: 587
  code:
    404: page not found
    500: server error
```

```console
$ notify send
send to ["email" "chat"] through smtp.example.com:587, every 15m0s, retries 3
code 404: page not found
code 500: server error
```

## Rules

- `Decode` reads the first document of the data. An empty document is an empty tree.
- A mapping is a `map[string]any`, a sequence is a `[]any`, and a null is `nil`. A member with a
  null value does not set an option.
- Every other scalar is its source text. The option that reads the scalar converts the text as it
  converts a flag, so `0644` is the file mode 0644 and `1.10` is the string `1.10`.
- `Decode` applies anchors, aliases and merge keys.
- The run fails with an error that wraps `cli.ErrConfigFile` for data that is not YAML, a document
  that is not a mapping, a duplicate key and an alias that contains itself.

```console
$ notify -c broken.yaml send
notify: send: config file broken.yaml: yaml: decode: yaml: unmarshal errors:
  line 6: cannot unmarshal !!seq into map[string]interface {}
```

`cli.Decode` decodes a document of the program from the tree that `yaml.Decode` returns. The
[README of go-cli](https://github.com/stealth-scale/go-cli#documents) describes its rules.

## Versions

The module has its own version, its own tags and its own changelog. Its tags are `yaml/vX.Y.Z`, and
[CHANGELOG.md](CHANGELOG.md) lists the changes of each release.

<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# Contributing to go-cli

## Tools

- Go 1.27.2 or later.
- GNU make.
- ergon at the release that `.ergon/init.lock` names. The
  [releases of ergon](https://github.com/dokimasia/ergon/releases) contain its archives. The targets
  of the Makefile run every other tool through `ergon tool run`. It installs the version of each
  tool that `.ergon.yaml` pins.
- pre-commit for the hooks of `.pre-commit-config.yaml`.

## The gate

`make check` runs the gate of the repository. It lints the Markdown files and the workflows. In
every module it runs golangci-lint, the tests, the tests under the race detector and govulncheck.
CI runs the same targets on every pull request, so a change that passes `make check` locally
passes the gate in CI. `make help` lists every target.

`pre-commit install` installs the hooks. Each commit runs `make lint` and `make test` and checks
its message with commitlint, and each push runs `make check`.

## Tests

Each source file `x.go` other than `doc.go` has a black-box test file `x_test.go`. The tests of the
rendered output compare it with the golden files in `testdata/golden/` of their module, and the
test of each example compares its transcript with `testdata/golden/transcript.txt`. After a change
of the output, write the golden files of a module again from its directory:

```sh
go test ./... -update
```

## Design records

[docs/README.md](docs/README.md) describes when to write an RFC, an ADR or a research document.

## Changes

- Open a pull request against `main`.
- Add a changeset to `.changeset/` for each change that a release contains.
  [.changeset/README.md](.changeset/README.md) describes the format.
- Write each commit message as a Conventional Commit: `type(scope): subject`. `.commitlint.yaml`
  lists the types.

## Security

Report a vulnerability as [SECURITY.md](SECURITY.md) describes. Do not open a public issue for it.

## Conduct

Every contributor follows the [Code of Conduct](CODE_OF_CONDUCT.md).

// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Package yaml reads the YAML config files of the apps of go.stealthscale.io/cli.
//
// The package cli does not import a package outside the standard library, so the YAML parser is
// in a module of its own. A program registers the format once, before the app runs:
//
//	app := cli.New[Global]("srv", "Serve the store.")
//	yaml.Register(app)
//
// [Register] registers [Decode] for the extensions .yaml and .yml. A program that registers the
// format for other extensions passes Decode to cli.App.Format.
//
// [Decode] returns each scalar as its source text. The option that reads a scalar converts the
// text as it converts a flag, so 0644 is the file mode 0644 and 1.10 is the string 1.10.
//
// # Dependency position
//
// The package imports fmt, go.yaml.in/yaml/v3 and go.stealthscale.io/cli.
package yaml

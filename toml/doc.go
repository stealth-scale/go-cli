// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Package toml reads the TOML config files of the apps of go.stealthscale.io/cli.
//
// The package cli does not import a package outside the standard library, so the TOML parser is
// in a module of its own. A program registers the format once, before the app runs:
//
//	app := cli.New[Global]("srv", "Serve the store.")
//	toml.Register(app)
//
// [Register] registers [Decode] for the extension .toml. A program that registers the format for
// other extensions passes Decode to cli.App.Format.
//
// # Dependency position
//
// The package imports fmt, github.com/BurntSushi/toml and go.stealthscale.io/cli.
package toml

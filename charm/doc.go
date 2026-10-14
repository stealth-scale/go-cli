// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Package charm renders the output of a go.stealthscale.io/cli app with Lip Gloss. It renders the
// help, the failures and the table of --flag-debug. It writes the version of an app through
// cli.Plain.
//
// The package cli imports only the standard library, so the renderer is in a module of its own. A
// program registers the renderer before the app runs:
//
//	app := cli.New[Global]("srv", "Serve the store.")
//	charm.Register(app)
//
// A program with its own theme changes the styles of [DefaultTheme] and passes the theme to [New]:
//
//	t := charm.DefaultTheme()
//	t.Flag = t.Flag.Foreground(lipgloss.Green)
//	app.Render(charm.New(&t))
//
// The app passes the environment of the run to the renderer. colorprofile decides from the writer
// and from that environment whether the output contains styles. A pipe or a file receives the help
// without styles, and the failures and the table of --flag-debug of cli.Plain.
//
// # Dependency position
//
// The package imports fmt, io, slices, strconv, strings, charm.land/lipgloss/v2,
// github.com/charmbracelet/colorprofile and go.stealthscale.io/cli.
package charm

// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import "fmt"

// The global flag --version of an app with a version.
const (
	// versionFlag is the name of the flag.
	versionFlag = "version"
	// versionLong is the flag on the command line.
	versionLong = flagPrefix + versionFlag
	// versionHelp is the help text of the flag.
	versionHelp = "Show the version."
)

// Version registers text as the version of the app and declares the global flag --version. The
// program builds the whole text, such as "srv 1.4.2 (f70f638, 2026-10-13)" from variables that the
// linker sets. A run with the flag writes the text with the renderer of the run to env.Stdout and
// returns nil. [Plain] writes the text followed by a newline.
//
// The flag can come anywhere on the command line, like the flags of the help. It does not take a
// value, and it does not have a short flag, an environment variable or a config key. A run with the
// flag does not check the other options and the positional arguments. It does not run a hook or a
// handler. A command line that does not scan fails with its error. The flags of the help take
// precedence over the flag of the version. A later call replaces the text of an earlier one.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when text is empty, and when an option
// of the app declares the flag --version.
func (a *App[G]) Version(text string) {
	if text == "" {
		a.errs = append(a.errs, fmt.Errorf("%w: the version of the app %s is empty", ErrDeclaration, a.name))
		return
	}
	a.version = text
}

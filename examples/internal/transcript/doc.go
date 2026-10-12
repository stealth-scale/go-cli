// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Package transcript runs a program of go.stealthscale.io/cli with a list of command lines and
// returns the transcript of the runs as a terminal shows it.
//
// The test of each example compares the transcript of its runs with a golden file. The golden file
// shows what the example prints:
//
//	$ greet hello --times twice
//	greet: hello: invalid --times "twice" from flag --times: invalid syntax
//	[exit 2]
//
// # Dependency position
//
// The package imports bytes, context, errors, io, maps, os, slices, strconv, strings and
// go.stealthscale.io/cli.
package transcript

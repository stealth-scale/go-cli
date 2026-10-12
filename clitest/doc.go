// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Package clitest runs the programs of the package go.stealthscale.io/cli in tests.
//
// [Main] runs a program once and returns its exit status and its output. [Transcript] runs a
// program once for each [Input] of a list and returns the runs as a terminal shows them, with the
// config files that each run read. [NewEnv] returns the [cli.Env] of an Input for a test that calls
// App.Execute, or a handler through cli.NewRun:
//
//	res := clitest.Main(t.Context(), app().Main, &clitest.Input{
//		Args:  []string{"serve"},
//		Files: clitest.Files{"srv.json": `{"serve": {"addr": ":9000"}}`},
//	})
//
// A program is the method App.Main of an app, or a function of its signature. A program returns
// its exit status and does not call os.Exit, which would end the test.
//
// # Isolation
//
// A run reads only the arguments, the environment variables, the config files and the standard
// input of its Input. A read of a config file outside [Input.Files] returns an error that wraps
// fs.ErrNotExist, so a run skips the config file at the default path of the program. A test that
// needs files on disk passes os.DirFS as Input.Files. The package leaves the state of the process
// unchanged, so tests that call it can run in parallel.
//
// # Transcripts
//
// A test compares a transcript with a golden file of its choice. The package does not register a
// flag, so it does not collide with the flag -update of a package of golden files.
//
// # Dependency position
//
// The package imports bytes, context, errors, io/fs, maps, path, path/filepath, slices, strconv,
// strings, sync, testing/fstest and go.stealthscale.io/cli.
package clitest

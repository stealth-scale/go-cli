// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package clitest_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"

	"go.stealthscale.io/cli"
)

// program is the name of the program that the tests run.
const program = "say"

// errDisk is the error of every file of brokenFS.
var errDisk = errors.New("disk gone")

// The options of the program say.
type (
	// sayGlobal contains the global options of say.
	sayGlobal struct {
		Config cli.ConfigFile
	}
	// sayOptions contains the options of the command echo.
	sayOptions struct {
		Words []string `arg:"" default:""`
		Extra []string
		Fail  bool
	}
)

// brokenFS is a file system whose every open fails with errDisk, an error that is not an
// *fs.PathError.
type brokenFS struct{}

// Open returns errDisk.
func (brokenFS) Open(string) (fs.File, error) {
	return nil, errDisk
}

// say returns the program say with the command echo. echo writes its standard input to the standard
// error. It then writes each word of its arguments and of the option extra to a line of the
// standard output in upper case.
func say() func(context.Context, *cli.Env) int {
	app := cli.New[sayGlobal](program, "")
	app.Command("echo", "", func(_ context.Context, run *cli.Run[sayGlobal, sayOptions]) error {
		in, err := io.ReadAll(run.Stdin())
		if err != nil {
			return err
		}
		if _, err := run.Stderr().Write(in); err != nil {
			return err
		}
		for _, w := range append(run.Config().Words, run.Config().Extra...) {
			if _, err := io.WriteString(run.Stdout(), strings.ToUpper(w)+"\n"); err != nil {
				return err
			}
		}
		return nil
	})
	return app.Main
}

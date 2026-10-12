// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command arguments is a program whose commands take positional arguments.
//
//	go run ./04-arguments add logs app.log db.log
//
// A field with the tag arg is a positional argument. Only the command line sets a positional
// argument. The package assigns the arguments of the command line to the fields in the order of the
// fields:
//   - An argument without a default is required.
//   - An argument with a default is optional. It must come after the required arguments.
//   - A slice takes the rest of the command line. It must be the last argument. A slice without a
//     default takes at least one argument.
//
// Flags can come between the arguments. Every argument after the argument -- is positional. A lone -
// is an argument too.
//
// The files of add are input files. The handler opens each with Run.Open, which returns the standard
// input for the path -:
//
//	printf 'GET /\n' | go run ./04-arguments add logs -
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"go.stealthscale.io/cli"
)

// Create contains the options of the command create.
type Create struct {
	Name    string `help:"Name of the archive."    arg:""`
	Comment string `help:"Comment of the archive." arg:"" default:"none"`
}

// Add contains the options of the command add.
type Add struct {
	Archive string          `help:"Archive that receives the files."           arg:""`
	Files   []cli.InputFile `help:"Files to add, or - for the standard input." arg:""`
	Level   int             `help:"Level of the compression."                         default:"6" short:"l"`
}

// List contains the options of the command list.
type List struct {
	Patterns []string `help:"Patterns of the names to list." arg:"" default:""`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program pack with the commands create, add and list.
func app() *cli.App[struct{}] {
	a := cli.New[struct{}]("pack", "Pack files into archives.")
	a.Command("create", "Create an archive.", create)
	a.Command("add", "Add files to an archive.", add)
	a.Command("list", "List the files of the archives.", list)
	return a
}

// create writes the name and the comment of the archive.
func create(_ context.Context, run *cli.Run[struct{}, Create]) error {
	c := run.Config()
	return write(run, fmt.Appendf(nil, "create %q with the comment %q\n", c.Name, c.Comment))
}

// add writes a line with the number of bytes of each file. It opens one file at a time.
func add(_ context.Context, run *cli.Run[struct{}, Add]) error {
	a := run.Config()
	var out []byte
	for _, name := range a.Files {
		n, err := size(run, name)
		if err != nil {
			return err
		}
		out = fmt.Appendf(out, "add %q of %d bytes to %q at level %d\n", name, n, a.Archive, a.Level)
	}
	return write(run, out)
}

// size returns the number of bytes of the input file name of run, and closes the file.
func size(run *cli.Run[struct{}, Add], name cli.InputFile) (n int64, err error) {
	f, err := run.Open(name)
	if err != nil {
		return 0, fmt.Errorf("main: add: %w", err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	n, err = io.Copy(io.Discard, f)
	if err != nil {
		return 0, fmt.Errorf("main: read %s: %w", name, err)
	}
	return n, nil
}

// list writes the patterns. It writes "list every file" when the command line does not have a
// pattern.
func list(_ context.Context, run *cli.Run[struct{}, List]) error {
	p := run.Config().Patterns
	if len(p) == 0 {
		return write(run, []byte("list every file\n"))
	}
	return write(run, fmt.Appendf(nil, "list the files that match %q\n", p))
}

// write writes out to the standard output of run.
func write[C any](run *cli.Run[struct{}, C], out []byte) error {
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the result: %w", err)
	}
	return nil
}

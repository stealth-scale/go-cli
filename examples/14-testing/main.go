// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command testing is a program that counts the lines, the words and the bytes of its input files.
// The tests in main_test.go show how to test a program of the cli package with the package clitest.
//
//	printf 'one two\nthree\n' | go run ./14-testing --words
//
// The program has no commands, so the handler of the app runs every command line. Its positional
// arguments are input files with the default -, so a run without a file counts the standard input.
//
// clitest.Transcript runs a list of command lines and returns the transcript that the golden file
// contains. The transcript shows each input file of clitest.Files that a run opens. clitest.Main
// runs one command line. clitest.NewEnv returns the Env of a test that calls App.Execute, and the Env
// of a test that calls the handler through cli.NewRun. A test can also build its Env with cli.Vars
// and cli.Environ. One test of the handler checks that it honours the cancellation of its context.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"go.stealthscale.io/cli"
)

// Count contains the options of the program wc. Without a flag, wc writes every number.
type Count struct {
	Files []cli.InputFile `help:"Files to count, or - for the standard input." arg:"" default:"-"`
	Lines bool            `help:"Write the number of lines."                                      short:"l"`
	Words bool            `help:"Write the number of words."                                      short:"w"`
	Bytes bool            `help:"Write the number of bytes."                                      short:"c"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program wc, whose handler is count.
func app() *cli.App[Count] {
	a := cli.New[Count]("wc", "Count the lines, the words and the bytes of files.")
	a.Handle(count)
	return a
}

// count reads the input files of run in their order. It writes the numbers that its flags select for
// the files together on one line.
//
// Error modes:
//   - the error of ctx when ctx has ended before the first read.
//   - the error of the open, the read and the close of an input file, and of the standard output.
func count(ctx context.Context, run *cli.Run[Count, Count]) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("main: count the input: %w", err)
	}
	c := run.Config()
	var data []byte
	for _, name := range c.Files {
		content, err := read(run, name)
		if err != nil {
			return err
		}
		data = append(data, content...)
	}
	every := !c.Lines && !c.Words && !c.Bytes
	var numbers []string
	for _, n := range []struct {
		selected bool
		value    int
	}{
		{every || c.Lines, strings.Count(string(data), "\n")},
		{every || c.Words, len(strings.Fields(string(data)))},
		{every || c.Bytes, len(data)},
	} {
		if n.selected {
			numbers = append(numbers, strconv.Itoa(n.value))
		}
	}
	if _, err := io.WriteString(run.Stdout(), strings.Join(numbers, " ")+"\n"); err != nil {
		return fmt.Errorf("main: write the numbers: %w", err)
	}
	return nil
}

// read returns the content of the input file name of run, and closes the file.
func read(run *cli.Run[Count, Count], name cli.InputFile) (data []byte, err error) {
	f, err := run.Open(name)
	if err != nil {
		return nil, fmt.Errorf("main: count: %w", err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	data, err = io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("main: read %q: %w", name, err)
	}
	return data, nil
}

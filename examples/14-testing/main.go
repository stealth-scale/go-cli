// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command testing is a program that reads its standard input. The tests in main_test.go show how to
// test a program of the cli package with the package clitest.
//
//	printf 'one two\nthree\n' | go run ./14-testing --words
//
// The program has no commands, so the handler of the app runs every command line. clitest.Transcript
// runs a list of command lines and returns the transcript that the golden file contains.
// clitest.Main runs one command line. clitest.NewEnv returns the Env of a test that calls
// App.Execute, and the Env of a test that calls the handler through cli.NewRun. One test of the
// handler checks that it honours the cancellation of its context.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"go.stealthscale.io/cli"
)

// Count contains the options of the program wc. Without an option, wc writes every number.
type Count struct {
	Lines bool `help:"Write the number of lines." short:"l"`
	Words bool `help:"Write the number of words." short:"w"`
	Bytes bool `help:"Write the number of bytes." short:"c"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program wc, whose handler is count.
func app() *cli.App[Count] {
	a := cli.New[Count]("wc", "Count the lines, the words and the bytes of the standard input.")
	a.Handle(count)
	return a
}

// count reads the standard input of run. It writes the numbers that its options select on one line.
//
// Error modes:
//   - the error of ctx when ctx has ended before the read.
//   - the error of the standard input and the error of the standard output.
func count(ctx context.Context, run *cli.Run[Count, Count]) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("main: count the standard input: %w", err)
	}
	data, err := io.ReadAll(run.Stdin())
	if err != nil {
		return fmt.Errorf("main: read the standard input: %w", err)
	}
	c := *run.Config()
	if !c.Lines && !c.Words && !c.Bytes {
		c = Count{Lines: true, Words: true, Bytes: true}
	}
	var numbers []string
	for _, n := range []struct {
		selected bool
		value    int
	}{
		{c.Lines, strings.Count(string(data), "\n")},
		{c.Words, len(strings.Fields(string(data)))},
		{c.Bytes, len(data)},
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

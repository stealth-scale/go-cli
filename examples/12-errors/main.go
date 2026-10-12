// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command errors is a program that handles the error of a run in place of App.Main.
//
//	go run ./12-errors run --strikt src
//
// The function run calls App.Execute and inspects its error:
//   - A *cli.Error reports an invalid command line, environment variable or config file. run tests
//     the error against each sentinel with errors.Is and selects a hint. It writes Error.Message and
//     the hint. The hint of an invalid value contains the input, the value and the source of the
//     value. The program exits with cli.ExitUsage.
//   - Any other error comes from a handler. The program exits with cli.ExitFailure.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"go.stealthscale.io/cli"
)

// legacyDir is the directory whose files the command run refuses in a strict run.
const legacyDir = "legacy/"

// errLegacy is the error of a strict run over a file of legacyDir.
var errLegacy = errors.New("main: a strict run refuses the files of " + legacyDir)

// Global contains the options of every command.
type Global struct {
	Config cli.ConfigFile `help:"Config file in JSON." short:"c"`
}

// Run contains the options of the command run.
type Run struct {
	Format    string   `help:"Format of the report."          default:"text" enum:"text,json"`
	Paths     []string `help:"Files to lint."                                                 arg:""`
	MaxIssues int      `help:"Most issues to report."         default:"10"`
	Strict    bool     `help:"Fail for the files of legacy/."                                        short:"s"`
}

func main() {
	os.Exit(run(context.Background(), cli.OS()))
}

// app returns the program lint with the command run.
func app() *cli.App[Global] {
	a := cli.New[Global]("lint", "Lint files.")
	a.Command("run", "Lint the files of the paths.", lint)
	return a
}

// run runs the program with env and returns its exit status. It writes the message of an error to
// env.Stderr. For a *cli.Error it also writes a hint.
func run(ctx context.Context, env *cli.Env) int {
	err := app().Execute(ctx, env)
	if err == nil {
		return cli.ExitOK
	}
	e, ok := errors.AsType[*cli.Error](err)
	if !ok {
		// The exit status reports the error when the write of the message fails.
		_, _ = fmt.Fprintf(env.Stderr, "lint: %v\n", err)
		return cli.ExitFailure
	}
	var hint string
	switch {
	case errors.Is(e, cli.ErrUnknownCommand):
		hint = "lint --help lists the commands"
	case errors.Is(e, cli.ErrUnknownFlag):
		hint = "lint " + e.Command + " --help lists the flags of the command"
	case errors.Is(e, cli.ErrMissing):
		hint = "the command needs " + e.Input
	case errors.Is(e, cli.ErrInvalidValue):
		hint = fmt.Sprintf("the value %q of %s comes from %s", e.Value, e.Input, e.Source)
	default:
		hint = "lint --help lists the options"
	}
	// The exit status reports the error when the write of the message fails.
	_, _ = fmt.Fprintf(env.Stderr, "lint: %s\nhint: %s\n", e.Message(), hint)
	return cli.ExitUsage
}

// lint writes a line for each path. A strict run fails for a path in legacy/.
func lint(_ context.Context, run *cli.Run[Global, Run]) error {
	r := run.Config()
	var out []byte
	for _, p := range r.Paths {
		if r.Strict && strings.HasPrefix(p, legacyDir) {
			return errLegacy
		}
		out = fmt.Appendf(out, "%s: 0 issues of at most %d, in %s\n", p, r.MaxIssues, r.Format)
	}
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the report: %w", err)
	}
	return nil
}

// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package clitest

import (
	"context"

	"go.stealthscale.io/cli"
)

// Result is the outcome of one run of a program.
type Result struct {
	Stdout string // the text that the run wrote to its standard output
	Stderr string // the text that the run wrote to its standard error
	Status int    // the exit status that the program returned
}

// Main calls program once with an Env of in, and returns the exit status and the output of the
// run. program is the method App.Main of an app, or a function of its signature. Main cancels the
// context of the run when program returns.
func Main(ctx context.Context, program func(context.Context, *cli.Env) int, in *Input) Result {
	env, out := NewEnv(in)
	status := run(ctx, program, env)
	return Result{Status: status, Stdout: out.Stdout(), Stderr: out.Stderr()}
}

// run calls program with env and a context of ctx, and cancels that context when program returns,
// as the end of a process stops its goroutines.
func run(ctx context.Context, program func(context.Context, *cli.Env) int, env *cli.Env) int {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return program(ctx, env)
}

// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command commands is a program with three commands that share the options of an embedded struct.
//
//	go run ./03-commands plan --env prod
//
// A global flag can come before or after the command. Each command embeds Target, so every command
// has the flags --env and --region without a group in their names. A handler that returns an error
// makes the program exit with the status 1.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"go.stealthscale.io/cli"
)

// errUnconfirmed is the error of an apply without the flag --yes.
var errUnconfirmed = errors.New("main: apply changes prod only with --yes")

// Global contains the options of every command.
type Global struct {
	Verbose bool `help:"Print each step." short:"v"`
}

// Target contains the options of the environment that a command acts on.
type Target struct {
	Env    string `help:"Environment to act on."     required:"" enum:"dev,prod" short:"e"`
	Region string `help:"Region of the environment."                                       default:"eu-west-1"`
}

// Plan contains the options of the command plan.
type Plan struct {
	Target
}

// Apply contains the options of the command apply.
type Apply struct {
	Target
	Yes bool `help:"Apply the changes to prod." short:"y"`
}

// Rollback contains the options of the command rollback.
type Rollback struct {
	Target
	Release int `help:"Release to roll back to." arg:""`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program deploy with the commands plan, apply and rollback.
func app() *cli.App[Global] {
	a := cli.New[Global]("deploy", "Deploy the service.")
	a.Command("plan", "Print the changes of a deployment.", plan)
	a.Command("apply", "Apply the changes of a deployment.", apply)
	a.Command("rollback", "Roll back to an earlier release.", rollback)
	return a
}

// plan writes the changes that a deployment to the target would make.
func plan(_ context.Context, run *cli.Run[Global, Plan]) error {
	return write(run.Stdout(), run.Global(), "plan", &run.Config().Target, "2 changes")
}

// apply writes the changes that it applies to the target. It refuses prod without --yes.
func apply(_ context.Context, run *cli.Run[Global, Apply]) error {
	a := run.Config()
	if a.Env == "prod" && !a.Yes {
		return errUnconfirmed
	}
	return write(run.Stdout(), run.Global(), "apply", &a.Target, "2 changes applied")
}

// rollback writes the release that it restores on the target.
func rollback(_ context.Context, run *cli.Run[Global, Rollback]) error {
	r := run.Config()
	return write(run.Stdout(), run.Global(), "rollback", &r.Target, fmt.Sprintf("release %d restored", r.Release))
}

// write writes the result of the command name on target to w. It first writes the target when g is
// verbose.
func write(w io.Writer, g *Global, name string, target *Target, result string) error {
	var out []byte
	if g.Verbose {
		out = fmt.Appendf(out, "%s: target %s in %s\n", name, target.Env, target.Region)
	}
	out = fmt.Appendf(out, "%s: %s\n", name, result)
	if _, err := w.Write(out); err != nil {
		return fmt.Errorf("main: write the result of %s: %w", name, err)
	}
	return nil
}

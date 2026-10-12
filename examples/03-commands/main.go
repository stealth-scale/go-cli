// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command commands is a program with three commands that share the options of an embedded struct,
// and a command with a hook and two subcommands.
//
//	go run ./03-commands plan --env prod
//	go run ./03-commands -v registry push app:v2
//
// A global flag can come before or after the command. Each of plan, apply and rollback embeds
// Target, so every command has the flags --env and --region without a group in their names. A
// handler that returns an error makes the program exit with the status 1.
//
// The subcommands push and prune of registry embed a pointer to Registry, the options of
// registry. Their flags --url and --user come before or after the name of the subcommand, and
// their names are DEPLOY_REGISTRY_URL and registry.url. The hook signIn opens a session before
// each handler of registry runs, and the subcommands read the session through the pointer.
//
// The hook trace of the app runs before every handler. With the flag --verbose it writes the path
// of the command of the run to the standard error.
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

// Registry contains the options of the command registry and the session that the hook signIn
// opens.
type Registry struct {
	URL     string `help:"URL of the registry."  default:"registry.example.com"`
	User    string `help:"User of the registry." default:"deploy"`
	session string // the session of the user, which signIn opens
}

// Push contains the options of the command registry push.
type Push struct {
	*Registry
	Image string `help:"Image to push." arg:""`
}

// Prune contains the options of the command registry prune.
type Prune struct {
	*Registry
	Keep int `help:"Number of images to keep." default:"3"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program deploy with the commands plan, apply, rollback and registry, and the
// subcommands push and prune of registry.
func app() *cli.App[Global] {
	a := cli.New[Global]("deploy", "Deploy the service.")
	a.Use(trace)
	a.Command("plan", "Print the changes of a deployment.", plan)
	a.Command("apply", "Apply the changes of a deployment.", apply)
	a.Command("rollback", "Roll back to an earlier release.", rollback)
	registry := a.Command("registry", "Count the images of the registry.", images)
	registry.Use(signIn)
	registry.Command("push", "Push an image to the registry.", push)
	registry.Command("prune", "Remove the oldest images of the registry.", prune)
	return a
}

// trace writes the path of the command of the run to the standard error for the flag --verbose,
// and then runs the hooks below it and the handler.
func trace(ctx context.Context, run *cli.Run[Global, Global], next func(context.Context) error) error {
	if run.Global().Verbose {
		if _, err := fmt.Fprintf(run.Stderr(), "deploy: run %s\n", run.Command()); err != nil {
			return fmt.Errorf("main: write the trace: %w", err)
		}
	}
	return next(ctx)
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

// signIn opens the session of the user before the handler of registry or of one of its subcommands
// runs, and closes it after the handler returns. It writes both steps for the flag --verbose.
func signIn(ctx context.Context, run *cli.Run[Global, Registry], next func(context.Context) error) error {
	r := run.Config()
	r.session = r.User + "@" + r.URL
	verbose := run.Global().Verbose
	if verbose {
		if err := say(run.Stdout(), "registry", "sign in as "+r.session); err != nil {
			return err
		}
	}
	err := next(ctx)
	if verbose {
		err = errors.Join(err, say(run.Stdout(), "registry", "sign out"))
	}
	return err
}

// images writes the number of images in the registry.
func images(_ context.Context, run *cli.Run[Global, Registry]) error {
	return say(run.Stdout(), "registry", "4 images in "+run.Config().URL)
}

// push writes the image that it pushes in the session of signIn.
func push(_ context.Context, run *cli.Run[Global, Push]) error {
	p := run.Config()
	return say(run.Stdout(), "push", p.Image+" as "+p.session)
}

// prune writes the number of images that it keeps in the registry.
func prune(_ context.Context, run *cli.Run[Global, Prune]) error {
	p := run.Config()
	return say(run.Stdout(), "prune", fmt.Sprintf("%d images kept in %s", p.Keep, p.URL))
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

// say writes the line text of the command name to w.
func say(w io.Writer, name, text string) error {
	if _, err := fmt.Fprintf(w, "%s: %s\n", name, text); err != nil {
		return fmt.Errorf("main: write the result of %s: %w", name, err)
	}
	return nil
}

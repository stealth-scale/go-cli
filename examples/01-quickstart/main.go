// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command quickstart is the smallest program of the cli package. It has one global option and one
// command with an optional positional argument and two flags.
//
//	go run ./01-quickstart hello Ada --times 2
//
// The fields of Global are the global options. The fields of Hello are the options of the command
// hello. Each field has a flag, an environment variable and a config key. The help lists them. The
// program cancels the context of the run on an interrupt.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"go.stealthscale.io/cli"
)

// Global contains the options of every command.
type Global struct {
	Verbose bool `help:"Print the options of the greeting." short:"v"`
}

// Hello contains the options of the command hello.
type Hello struct {
	Name     string `help:"Name to greet."        arg:"" default:"world"`
	Greeting string `help:"Word of the greeting."        default:"Hello" short:"g"`
	Times    int    `help:"Number of greetings."         default:"1"     short:"n"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	status := app().Main(ctx, cli.OS())
	stop()
	os.Exit(status)
}

// app returns the program greet with the command hello.
func app() *cli.App[Global] {
	a := cli.New[Global]("greet", "Greet someone.")
	a.Command("hello", "Print a greeting.", hello)
	return a
}

// hello writes the greeting as many times as the value of the option times. It first writes the
// options when the global option verbose is set.
func hello(_ context.Context, run *cli.Run[Global, Hello]) error {
	h := run.Config()
	var out []byte
	if run.Global().Verbose {
		out = fmt.Appendf(out, "name=%q greeting=%q times=%d\n", h.Name, h.Greeting, h.Times)
	}
	for range h.Times {
		out = fmt.Appendf(out, "%s, %s!\n", h.Greeting, h.Name)
	}
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the greeting: %w", err)
	}
	return nil
}

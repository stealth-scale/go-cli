// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command charm is a program that writes its help, its failures and the table of --flag-debug with
// the renderer of the module go.stealthscale.io/cli/charm.
//
//	go run ./13-charm serve --help
//
// charm.Register registers the renderer. A terminal receives the output with colors and styles. A
// pipe receives the same help without styles, and a failure as the one line of cli.Plain. The help
// wraps at the width in the environment variable COLUMNS.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/charm"
)

// Global contains the options of every command.
type Global struct {
	Config cli.ConfigFile `help:"Config file in JSON." short:"c"`
}

// Serve contains the options of the command serve.
type Serve struct {
	Store                      string `help:"Name of the store."                    arg:"" default:"main"`
	Addr                       string `help:"Address to listen on."                        default:":8080" short:"a"`
	ClientCertificateAuthority string `help:"Certificate authority of the clients."`
	Token                      string `help:"Bearer token of the requests."                                          required:"" secret:""`
	Quiet                      bool   `help:"Do not log the requests."                                     short:"q"`
}

// Migrate contains the options of the command migrate.
type Migrate struct {
	DryRun bool `help:"Print the plan without a migration." short:"n"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program srv with the commands serve and migrate. The renderer of the module charm
// writes its output.
func app() *cli.App[Global] {
	a := cli.New[Global]("srv", "Serve the store.")
	charm.Register(a)
	a.Command("serve", "Run the server of the store.", serve)
	a.Command("migrate", "Migrate the store to the current schema.", migrate)
	return a
}

// serve writes the store and the address of the server.
func serve(_ context.Context, run *cli.Run[Global, Serve]) error {
	s := run.Config()
	return write(run.Stdout(), fmt.Sprintf("serve %s on %s\n", s.Store, s.Addr))
}

// migrate writes the plan of the migration.
func migrate(_ context.Context, run *cli.Run[Global, Migrate]) error {
	if run.Config().DryRun {
		return write(run.Stdout(), "plan: no migration\n")
	}
	return write(run.Stdout(), "migrate the store\n")
}

// write writes text to w.
func write(w io.Writer, text string) error {
	if _, err := io.WriteString(w, text); err != nil {
		return fmt.Errorf("main: write the result: %w", err)
	}
	return nil
}

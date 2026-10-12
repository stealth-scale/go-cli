// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command charm is a program that writes its help, its failures and the table of --flag-debug with
// the renderer of the module go.stealthscale.io/cli/charm.
//
//	go run ./13-charm serve --help
//
// charm.New returns the renderer for a theme, and App.Render registers it. The theme is
// charm.DefaultTheme with underlined headings. charm.Register registers the renderer of the default
// theme in one call. The renderer writes colors and styles to a terminal. It writes the help to a
// pipe without styles. A failure on a pipe is the one line of cli.Plain. The help wraps at the width
// in the environment variable COLUMNS.
//
// The handler of serve writes the name of the store in bold. It passes Run.Environ to colorprofile,
// which removes the styles for a pipe unless the variable CLICOLOR_FORCE is set.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/charm"
)

// bold is the style of the name of the store in the output of serve.
var bold = lipgloss.NewStyle().Bold(true)

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
// writes its output with underlined headings.
func app() *cli.App[Global] {
	a := cli.New[Global]("srv", "Serve the store.")
	theme := charm.DefaultTheme()
	theme.Heading = theme.Heading.Underline(true)
	a.Render(charm.New(&theme))
	a.Command("serve", "Run the server of the store.", serve)
	a.Command("migrate", "Migrate the store to the current schema.", migrate)
	return a
}

// serve writes the store in bold and the address of the server. The writer of colorprofile reads
// the environment of the run and removes the styles that the output cannot show.
func serve(_ context.Context, run *cli.Run[Global, Serve]) error {
	s := run.Config()
	w := colorprofile.NewWriter(run.Stdout(), run.Environ())
	return write(w, "serve "+bold.Render(s.Store)+" on "+s.Addr+"\n")
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

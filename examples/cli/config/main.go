// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command config is a program that reads its options from config files.
//
//	go run ./cli/config -c cli/config/relay.ini serve
//
// The option Config of the type cli.ConfigFile is the path of the config file. Its default is
// relay.json. The extension of the path selects the format of the file:
//   - cli.New registers JSON for .json and INI for .ini.
//   - app registers INI for .conf with App.Format.
//
// An object contains the keys of a command or of a group, or the entries of a map. An array contains
// the values of a slice. The commands serve and migrate share the config key database.url through
// the tag key. A run fails on a key that does not match an option.
package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"

	"go.stealthscale.io/cli"
)

// Global contains the options of every command.
type Global struct {
	Config cli.ConfigFile `help:"Config file in JSON or INI." default:"relay.json" short:"c"`
}

// Database contains the option of the database. Serve and Migrate embed it. The tag key replaces the
// whole config key, so both commands read the key database.url.
type Database struct {
	URL string `help:"URL of the database." default:"postgres://localhost/relay" key:"database.url"`
}

// TLS contains the options of the certificate of the server.
type TLS struct {
	Cert string `help:"Certificate file."`
}

// Serve contains the options of the command serve.
type Serve struct {
	Database
	Routes map[string]string `help:"Upstream of each path."    name:"route"`
	TLS    TLS
	Addr   string   `help:"Address to listen on."                  default:":8080"`
	Allow  []string `help:"Origins that may connect."`
}

// Migrate contains the options of the command migrate.
type Migrate struct {
	Database
	Steps int `help:"Number of migrations to apply." default:"1"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program relay with the commands serve and migrate. The program reads a config file
// with the extension .conf as INI.
func app() *cli.App[Global] {
	a := cli.New[Global]("relay", "Relay requests to upstreams.")
	a.Format(cli.INI, ".conf")
	a.Command("serve", "Print the options of the server.", serve)
	a.Command("migrate", "Print the options of the migration.", migrate)
	return a
}

// serve writes the options of the server. It then writes the routes in the order of their paths.
func serve(_ context.Context, run *cli.Run[Global, Serve]) error {
	s := run.Config()
	out := fmt.Appendf(nil, "listen on %s with %q\ndatabase %s\nallow %q\n", s.Addr, s.TLS.Cert, s.URL, s.Allow)
	for _, path := range slices.Sorted(maps.Keys(s.Routes)) {
		out = fmt.Appendf(out, "route %s to %s\n", path, s.Routes[path])
	}
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the options of the server: %w", err)
	}
	return nil
}

// migrate writes the options of the migration.
func migrate(_ context.Context, run *cli.Run[Global, Migrate]) error {
	m := run.Config()
	if _, err := fmt.Fprintf(run.Stdout(), "migrate %s, steps %d\n", m.URL, m.Steps); err != nil {
		return fmt.Errorf("main: write the options of the migration: %w", err)
	}
	return nil
}

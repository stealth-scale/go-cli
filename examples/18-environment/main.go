// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command environment is a program whose environment variables start with a prefix of its own.
//
//	STORE_SERVE_PORT=9000 go run ./18-environment serve
//
// The program is named storectl, so its variables would start with STORECTL. App.EnvPrefix replaces
// that start with STORE. The option Port of the command serve reads STORE_SERVE_PORT, the global
// option Config reads STORE_CONFIG, and the flag --flag-debug reads STORE_FLAG_DEBUG. The tag env of
// the option DatabaseURL replaces its whole variable, so the prefix does not apply to DATABASE_URL.
// The empty prefix of App.EnvPrefix leaves the start out, as in SERVE_PORT.
package main

import (
	"context"
	"fmt"
	"os"

	"go.stealthscale.io/cli"
)

// Global contains the options of every command.
type Global struct {
	// The variable STORE_CONFIG sets Config.
	Config cli.ConfigFile `help:"Config file in JSON." short:"c"`
}

// Serve contains the options of the command serve.
type Serve struct {
	// The variable DATABASE_URL sets DatabaseURL. --flag-debug prints *** in place of its value.
	DatabaseURL string `help:"URL of the database." env:"DATABASE_URL" secret:""`
	// The variable STORE_SERVE_PORT sets Port.
	Port int `help:"Port to listen on." default:"8080" short:"p"`
	// The variable STORE_SERVE_WORKERS sets Workers.
	Workers int `help:"Number of workers." default:"4"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program storectl with the command serve. The environment variables of the
// program start with STORE.
func app() *cli.App[Global] {
	a := cli.New[Global]("storectl", "Control the store.")
	a.EnvPrefix("STORE")
	a.Command("serve", "Run the server of the store.", serve)
	return a
}

// serve writes the options of the server. It writes whether the database URL is set and does not
// write the URL.
func serve(_ context.Context, run *cli.Run[Global, Serve]) error {
	s := run.Config()
	database := "unset"
	if s.DatabaseURL != "" {
		database = "set"
	}
	out := fmt.Appendf(nil, "serve on port %d with %d workers, database URL %s\n", s.Port, s.Workers, database)
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the options: %w", err)
	}
	return nil
}

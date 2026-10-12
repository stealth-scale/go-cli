// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command config is a program that reads its options from a TOML file.
//
//	go run ./toml/config -c toml/config/schedule.toml backup
//
// toml.Register registers the TOML format of the module go.stealthscale.io/cli/toml for the extension
// .toml. A table contains the keys of a command or of a group. An array contains the values of a
// slice. A TOML date converts to a time.Time.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/toml"
)

// Global contains the options of every command.
type Global struct {
	Config cli.ConfigFile `help:"Config file in TOML." default:"schedule.toml" short:"c"`
}

// Retention contains the options of the archives that a backup keeps.
type Retention struct {
	Days int `help:"Days to keep an archive." default:"7"`
}

// Backup contains the options of the command backup.
type Backup struct {
	Start     time.Time     `help:"Time of the first backup, in RFC 3339."`
	Targets   []string      `help:"Targets to back up."`
	Every     time.Duration `help:"Time between two backups."              default:"24h"`
	Retention Retention
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program schedule with the command backup. The program reads TOML config files.
func app() *cli.App[Global] {
	a := cli.New[Global]("schedule", "Schedule backups.")
	toml.Register(a)
	a.Command("backup", "Print the schedule of the backups.", backup)
	return a
}

// backup writes the schedule of the backups.
func backup(_ context.Context, run *cli.Run[Global, Backup]) error {
	b := run.Config()
	out := fmt.Appendf(nil, "back up %q from %s every %s, keep %d days\n",
		b.Targets, b.Start.Format(time.RFC3339), b.Every, b.Retention.Days)
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the schedule: %w", err)
	}
	return nil
}

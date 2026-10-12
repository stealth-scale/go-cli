// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command tags is a program whose options use the tags of the cli package.
//
//	go run ./cli/tags run --bucket nightly
//
// The fields of Backup use the tags help, default, enum, required, name, short, env, key and secret.
// The comment of each field describes the effect of its tags. The examples arguments and groups show
// the tags arg and prefix.
package main

import (
	"context"
	"fmt"
	"os"

	"go.stealthscale.io/cli"
)

// Backup contains the options of the command run.
type Backup struct {
	// A run fails when no source sets Bucket.
	Bucket string `help:"Bucket that receives the archive." required:""`
	// Compression takes one of three values. Its short flag is -z.
	Compression string `help:"Compression of the archive." default:"zstd" enum:"none,gzip,zstd" short:"z"`
	// The variable BACKUP_TOKEN sets Token. --flag-debug prints *** in place of its value.
	Token string `help:"Access token of the bucket." env:"BACKUP_TOKEN" secret:""`
	// Endpoint has the config key storage.endpoint in place of run.endpoint. Other commands can
	// declare the same key.
	Endpoint string `help:"Endpoint of the object store." default:"https://s3.example.com" key:"storage.endpoint"`
	// Only the flag --schedule sets Schedule. It does not have a variable or a config key.
	Schedule string `help:"Cron schedule of the next backup." env:"-" key:"-"`
	// Keep has the flag --retain, the variable BACKUP_RUN_RETAIN and the key run.retain.
	Keep int `help:"Number of archives to keep." default:"7" name:"retain"`
	// The package derives the flag --dry-run from the name DryRun. The tag short adds -n.
	DryRun bool `help:"Print the plan without a backup." short:"n"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program backup with the command run.
func app() *cli.App[struct{}] {
	a := cli.New[struct{}]("backup", "Back up the store.")
	a.Command("run", "Write an archive of the store to a bucket.", backup)
	return a
}

// backup writes the plan of the backup. It writes whether a token is set and does not write the
// token.
func backup(_ context.Context, run *cli.Run[struct{}, Backup]) error {
	b := run.Config()
	out := fmt.Appendf(nil, "archive %s with %s, keep %d, endpoint %s\n", b.Bucket, b.Compression, b.Keep, b.Endpoint)
	if b.Token != "" {
		out = append(out, "authenticate with the token\n"...)
	}
	if b.Schedule != "" {
		out = fmt.Appendf(out, "schedule the next backup at %q\n", b.Schedule)
	}
	if b.DryRun {
		out = append(out, "dry run: nothing written\n"...)
	}
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the plan: %w", err)
	}
	return nil
}

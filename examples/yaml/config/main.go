// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command config is a program that reads its options from a YAML file.
//
//	go run ./yaml/config -c yaml/config/notify.yaml send
//
// yaml.Register registers the YAML format of the module go.stealthscale.io/cli/yaml for the
// extensions .yaml and .yml. A mapping contains the keys of a command or of a group, or the entries of
// a map. A sequence contains the values of a slice. The keys of the map Codes are numbers.
package main

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"time"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/yaml"
)

// Global contains the options of every command.
type Global struct {
	Config cli.ConfigFile `help:"Config file in YAML." default:"notify.yaml" short:"c"`
}

// SMTP contains the options of the mail server.
type SMTP struct {
	Host string `help:"Host of the mail server." default:"localhost"`
	Port int    `help:"Port of the mail server." default:"25"`
}

// Send contains the options of the command send.
type Send struct {
	Codes    map[int]string `help:"Message of each status code." name:"code"`
	SMTP     SMTP
	Channels []string      `help:"Channels to notify."                      enum:"email,chat,pager"`
	Window   time.Duration `help:"Window of a batch."                                               default:"5m"`
	Retries  int           `help:"Number of retries."                                               default:"1"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program notify with the command send. The program reads YAML config files.
func app() *cli.App[Global] {
	a := cli.New[Global]("notify", "Send notifications.")
	yaml.Register(a)
	a.Command("send", "Print the options of the notifications.", send)
	return a
}

// send writes the options. It then writes the messages in the order of their codes.
func send(_ context.Context, run *cli.Run[Global, Send]) error {
	s := run.Config()
	out := fmt.Appendf(nil, "send to %q through %s:%d, every %s, retries %d\n",
		s.Channels, s.SMTP.Host, s.SMTP.Port, s.Window, s.Retries)
	for _, code := range slices.Sorted(maps.Keys(s.Codes)) {
		out = fmt.Appendf(out, "code %d: %s\n", code, s.Codes[code])
	}
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the options: %w", err)
	}
	return nil
}

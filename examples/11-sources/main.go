// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command sources is a program whose handler writes the source of each option.
//
//	go run ./11-sources -c 11-sources/fetch.json get --timeout 2m
//
// A run applies the sources of each option in this order:
//   - the tag default
//   - the config file
//   - the environment variable
//   - the flag
//
// A later source replaces the value of an earlier one. Run.Source returns the source of an option.
//
// With the global flag --flag-debug or the variable FETCH_FLAG_DEBUG, a run writes the value and the
// source of every option to the standard error before the handler runs. It masks the value of a
// secret option. A run with an empty path of the config file does not read a file.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.stealthscale.io/cli"
)

// Global contains the options of every command.
type Global struct {
	Config cli.ConfigFile `help:"Config file in JSON." default:"fetch.json" short:"c"`
}

// Get contains the options of the command get.
type Get struct {
	URL     string        `help:"URL to fetch."              default:"https://example.com"`
	Agent   string        `help:"User agent of a request."   default:"fetch/1.0"`
	Token   string        `help:"Bearer token of a request."                               secret:""`
	Timeout time.Duration `help:"Timeout of a request."      default:"10s"`
	Retries int           `help:"Number of retries."         default:"3"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program fetch with the command get.
func app() *cli.App[Global] {
	a := cli.New[Global]("fetch", "Fetch a URL.")
	a.Command("get", "Print the options of the request and the source of each.", get)
	return a
}

// get writes each option of the request with its source. It writes whether the token is set and
// does not write the token.
func get(_ context.Context, run *cli.Run[Global, Get]) error {
	g := run.Config()
	token := "unset"
	if g.Token != "" {
		token = "set"
	}
	var out []byte
	for _, o := range []struct {
		field any
		name  string
		value string
	}{
		{&g.URL, "url", g.URL},
		{&g.Agent, "agent", g.Agent},
		{&g.Token, "token", token},
		{&g.Timeout, "timeout", g.Timeout.String()},
		{&g.Retries, "retries", strconv.Itoa(g.Retries)},
	} {
		src, _ := run.Source(o.field)
		out = fmt.Appendf(out, "%-8s %-20s %s\n", o.name, o.value, src)
	}
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the options: %w", err)
	}
	return nil
}

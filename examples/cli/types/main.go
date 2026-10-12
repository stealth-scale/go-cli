// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command types is a program with an option of each type that the cli package converts without a
// registration.
//
//	go run ./cli/types show --port 8080 --zone Europe/Amsterdam --tags a,b
//
// The kinds string, bool, int, uint and float convert through strconv. time.Duration converts through
// time.ParseDuration. A type whose pointer has the method UnmarshalText converts through that method.
// time.Time, netip.Addr and slog.Level are such types. url.URL, time.Location, net.IPNet and
// fs.FileMode have conversions of their own.
//
// A pointer is nil until a source sets it. A slice takes values that commas separate. A map takes
// entries of the form key=value. A repeated flag of a slice or a map adds to its earlier values.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"time"
	_ "time/tzdata" // The program reads time zones on a system without a zone database.

	"go.stealthscale.io/cli"
)

// nilText is the text of a pointer that a source has not set.
const nilText = "<nil>"

// Show contains an option of each type that the program converts.
type Show struct {
	Since    time.Time      `help:"A time.Time in RFC 3339."`
	Upstream *url.URL       `help:"A *url.URL."`
	Zone     *time.Location `help:"A *time.Location of the zone database."`
	Peer     *net.IPNet     `help:"A *net.IPNet in CIDR notation."`
	Workers  *int           `help:"A *int, nil until a source sets it."`
	Limits   map[string]int `help:"A map of entries of the form key=value."`
	Mirror   url.URL        `help:"A url.URL."`
	Network  net.IPNet      `help:"A net.IPNet in CIDR notation."`
	Addr     netip.Addr     `help:"A netip.Addr."`
	Name     string         `help:"A string."`
	Tags     []string       `help:"A slice of strings that commas separate."`
	Ports    []uint16       `help:"A slice of uint16."`
	Timeout  time.Duration  `help:"A time.Duration, such as 1m30s."`
	Ratio    float64        `help:"A float64."`
	Log      slog.Level     `help:"A slog.Level, such as debug or warn."`
	Mode     fs.FileMode    `help:"An fs.FileMode in octal, such as 0640."`
	Port     uint16         `help:"A uint16."`
	Level    int8           `help:"An int8."`
	Enabled  bool           `help:"A bool."`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program types with the command show.
func app() *cli.App[struct{}] {
	a := cli.New[struct{}]("types", "Convert an option of each type.")
	a.Command("show", "Print the value of each option.", show)
	return a
}

// show writes the value of each option on a line of its own. It writes <nil> for a nil pointer.
func show(_ context.Context, run *cli.Run[struct{}, Show]) error {
	s := run.Config()
	upstream, zone, peer, workers := nilText, nilText, nilText, nilText
	if s.Upstream != nil {
		upstream = s.Upstream.String()
	}
	if s.Zone != nil {
		zone = s.Zone.String()
	}
	if s.Peer != nil {
		peer = s.Peer.String()
	}
	if s.Workers != nil {
		workers = strconv.Itoa(*s.Workers)
	}
	out := fmt.Appendf(nil, "name     %q\nenabled  %t\nlevel    %d\nport     %d\nratio    %g\n",
		s.Name, s.Enabled, s.Level, s.Port, s.Ratio)
	out = fmt.Appendf(out, "timeout  %s\nsince    %s\naddr     %s\nlog      %s\n",
		s.Timeout, s.Since.Format(time.RFC3339), s.Addr, s.Log)
	out = fmt.Appendf(out, "upstream %s\nmirror   %q\nzone     %s\nnetwork  %s\npeer     %s\n",
		upstream, s.Mirror.String(), zone, s.Network.String(), peer)
	out = fmt.Appendf(out, "mode     %s (%#o)\nworkers  %s\ntags     %q\nports    %v\nlimits   %v\n",
		s.Mode, uint32(s.Mode), workers, s.Tags, s.Ports, s.Limits)
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the options: %w", err)
	}
	return nil
}

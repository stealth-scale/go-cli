// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command groups is a program whose options are in groups. A group is a struct inside the options of
// a command.
//
//	go run ./cli/groups serve --admin-tls-cert admin.pem --backend-timeout 5s
//
// A named struct field is a group. Its name starts the flag, the environment variable and the config
// key of each option inside it. Groups nest, so the field Admin.TLS.Cert has the flag
// --admin-tls-cert, the variable PROXY_SERVE_ADMIN_TLS_CERT and the key serve.admin.tls.cert.
//
// An embedded struct adds its options without a group. The tag prefix replaces the name of a group.
// An empty prefix removes the group. A prefix on an embedded struct makes the struct a group.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.stealthscale.io/cli"
)

// Global contains the options of every command.
type Global struct {
	Config cli.ConfigFile `help:"Config file in JSON." short:"c"`
}

// TLS contains the options of a certificate.
type TLS struct {
	Cert string `help:"Certificate file."`
	Key  string `help:"Key file."`
}

// Listener contains the address and the certificate of a listener of the proxy.
type Listener struct {
	TLS  TLS
	Addr string `help:"Address to listen on."`
}

// Backend contains the options of the server that receives the requests of the proxy.
type Backend struct {
	URL     string        `help:"URL of the server."    default:"http://localhost:9000"`
	Timeout time.Duration `help:"Timeout of a request." default:"30s"`
}

// Logging contains the options of the log. Serve embeds it without a group.
type Logging struct {
	Level string `help:"Level of the log." default:"info" enum:"debug,info,warn"`
}

// Cache contains the options of the cache of responses. Serve embeds it with the prefix cache.
type Cache struct {
	Dir  string `help:"Directory of the cache."`
	Size int    `help:"Size of the cache in megabytes." default:"64"`
}

// Serve contains the options of the command serve.
type Serve struct {
	// Logging adds the flag --level.
	Logging
	// Cache adds the flags --cache-dir and --cache-size.
	Cache `prefix:"cache"`
	// Public adds the flags --addr, --tls-cert and --tls-key.
	Public Listener `prefix:""`
	// Admin adds the flags --admin-addr, --admin-tls-cert and --admin-tls-key.
	Admin Listener
	// Upstream adds the flags --backend-url and --backend-timeout.
	Upstream Backend `prefix:"backend"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program proxy with the command serve.
func app() *cli.App[Global] {
	a := cli.New[Global]("proxy", "Proxy requests to a backend.")
	a.Command("serve", "Print the listeners, the backend, the cache and the log of the proxy.", serve)
	return a
}

// serve writes the options of each group.
func serve(_ context.Context, run *cli.Run[Global, Serve]) error {
	s := run.Config()
	out := fmt.Appendf(nil, "public %q with %q and %q\n", s.Public.Addr, s.Public.TLS.Cert, s.Public.TLS.Key)
	out = fmt.Appendf(out, "admin  %q with %q and %q\n", s.Admin.Addr, s.Admin.TLS.Cert, s.Admin.TLS.Key)
	out = fmt.Appendf(out, "backend %s within %s\n", s.Upstream.URL, s.Upstream.Timeout)
	out = fmt.Appendf(out, "cache %q of %d MB, log %s\n", s.Dir, s.Size, s.Level)
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the options: %w", err)
	}
	return nil
}

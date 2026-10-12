// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"context"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli"
)

// The options of the app in the tests of the help.
type (
	helpGlobal struct {
		Config  cli.ConfigFile `help:"Config file in JSON or INI." short:"c"`
		Verbose bool           `help:"Log more."                   short:"v"`
	}
	helpTLS struct {
		Cert string `help:"Certificate file."`
	}
	helpOptions struct {
		TLS      helpTLS
		Labels   map[string]int `help:"Labels of the server."  name:"label"`
		Workers  *int           `help:"Number of workers."`
		Password string         `help:"Password of the store."              default:"hunter2" secret:""`
		Mode     string         `help:"Run mode."                           default:"dev"               enum:"dev,prod" short:"m"`
		Source   string         `help:"Source of the data."                                                                       arg:""`
		Target   string         `help:"Target of the data."                 default:"."                                           arg:""`
		Files    []string       `help:"Files to copy."                      default:""                                            arg:""`
		Tags     []string       `help:"Tags to attach."                                                                 short:"t"`
		Timeout  time.Duration  `help:"Request timeout."                    default:"30s"`
		Secret   string         `                                                                                                        env:"-" key:"-"`
		Region   string         `help:"Region of the server."                                                                                            required:""`
		Quiet    bool           `help:"Log less."`
	}
	listOnlyOptions struct {
		Files []string `help:"Files to copy." arg:""`
	}
)

// TestHelp checks the help of an app and of its commands.
func TestHelp(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the usage, the commands and the global options of the app", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[helpGlobal, helpOptions](t)
			app.Command("migrate", "Run the migrations.", func(context.Context, *cli.Run[helpGlobal, struct{}]) error {
				return nil
			})
			res := execute(t, app, []string{"--help"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			golden.Match(t, "help/app.txt", []byte(res.stdout), golden.ShouldUpdate())
		})

		t.Run("writes the usage, the arguments, the options and the global options of a command", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[helpGlobal, helpOptions](t)
			res := execute(t, app, []string{command, "-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			golden.Match(t, "help/command.txt", []byte(res.stdout), golden.ShouldUpdate())
		})

		t.Run("leaves out the text of an app without help", func(t *testing.T) {
			t.Parallel()
			app := cli.New[struct{ Config cli.ConfigFile }](appName, "")
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			golden.Match(t, "help/bare-app.txt", []byte(res.stdout), golden.ShouldUpdate())
		})

		t.Run("names the kind of a value of an unnamed type", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, struct{ Pair [2]int }](t)
			app.Convert(func(string) ([2]int, error) { return [2]int{}, nil })
			res := execute(t, app, []string{command, "-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.Contains(t, res.stdout, "      --pair array  ", "the help names the kind of the type")
		})

		t.Run("leaves out the text of a command without help", func(t *testing.T) {
			t.Parallel()
			app := cli.New[struct{}](appName, appHelp)
			app.Command(command, "", func(context.Context, *cli.Run[struct{}, listOnlyOptions]) error { return nil })
			res := execute(t, app, []string{command, "-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			golden.Match(t, "help/bare-command.txt", []byte(res.stdout), golden.ShouldUpdate())
		})
	})
}

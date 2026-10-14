// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.stealthscale.io/cli"
)

// versionOption is the flag of the version in the help of an app with a version.
var versionOption = cli.HelpOption{Name: "--version", Text: "Show the version."}

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
	bareGlobal struct {
		Config cli.ConfigFile
	}
	countGlobal struct {
		Files []string `help:"Files to count."  arg:"" default:""`
		Lines bool     `help:"Count the lines."                   short:"l"`
	}
	// keyGlobal, keyStore and keySteps are the options of an app whose command db declares the config
	// file and whose subcommand db migrate inherits it.
	keyGlobal struct {
		Verbose bool `help:"Log more."`
	}
	keyStore struct {
		Config cli.ConfigFile `help:"Config file in JSON."`
		DSN    string         `help:"Address of the database."`
	}
	keySteps struct {
		Steps int `help:"Number of migrations."`
	}
)

// TestHelp checks the help of an app and of its commands.
func TestHelp(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the usage, the commands and the options of the app", func(t *testing.T) {
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
			app := cli.New[bareGlobal](appName, "")
			app.Handle(func(context.Context, *cli.Run[bareGlobal, bareGlobal]) error { return nil })
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			golden.Match(t, "help/bare-app.txt", []byte(res.stdout), golden.ShouldUpdate())
		})

		t.Run("writes the usage, the arguments and the options of an app with a handler", func(t *testing.T) {
			t.Parallel()
			app := cli.New[countGlobal]("wc", "Count the lines of files.")
			app.Handle(func(context.Context, *cli.Run[countGlobal, countGlobal]) error { return nil })
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			golden.Match(t, "help/handler.txt", []byte(res.stdout), golden.ShouldUpdate())
		})

		t.Run("writes the usage, the subcommands and the options of a command with subcommands", func(t *testing.T) {
			t.Parallel()
			app := cli.New[helpGlobal](appName, appHelp)
			db := app.Command[Store](parent, parentHelp, nil)
			db.Command(subcommand, subcommandHelp, func(context.Context, *cli.Run[helpGlobal, stepsOptions]) error {
				return nil
			})
			db.Command(
				"seed",
				"Load the fixtures into the database.",
				func(context.Context, *cli.Run[helpGlobal, struct{}]) error {
					return nil
				},
			)
			res := execute(t, app, []string{parent, "-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			golden.Match(t, "help/parent.txt", []byte(res.stdout), golden.ShouldUpdate())
		})

		t.Run(
			"writes an optional command in the usage of a command with a handler and subcommands",
			func(t *testing.T) {
				t.Parallel()
				app, _, _ := nesting[helpGlobal, Store, stepsOptions](t)
				res := execute(t, app, []string{parent, "-h"}, nil, nil)
				assert.NoError(t, res.err, "Execute writes the help")
				assert.HasPrefix(
					t,
					res.stdout,
					"Usage: srv db [<command>] [options]\n",
					"the usage marks the command optional",
				)
			},
		)

		t.Run("writes the options of the ancestors of a subcommand as its global options", func(t *testing.T) {
			t.Parallel()
			app, _, _ := nesting[helpGlobal, Store, stepsOptions](t)
			res := execute(t, app, []string{parent, subcommand, "-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			golden.Match(t, "help/subcommand.txt", []byte(res.stdout), golden.ShouldUpdate())
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

		t.Run("lists the flag of the version among the options of the app", func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			app := rendering(r, nil)
			app.Version(versionText)
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.NotNil(t, r.help, "the renderer receives the help")
			assert.Equal(t, r.help.Options, slices.Concat(globalsHelp, []cli.HelpOption{versionOption}),
				"the help of the app lists the flag after the global options")
		})

		t.Run("lists the flag of the version among the global options of a command", func(t *testing.T) {
			t.Parallel()
			r := &recorder{}
			app := rendering(r, nil)
			app.Version(versionText)
			res := execute(t, app, []string{command, "-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.NotNil(t, r.help, "the renderer receives the help")
			assert.Equal(t, r.help.Globals, slices.Concat(globalsHelp, []cli.HelpOption{versionOption}),
				"the help of the command lists the flag after the global options")
		})

		keys := []struct {
			name     string
			giveArgs []string
			want     map[string]string
		}{
			{
				name:     "leaves out the config keys in the help of an app without a config file",
				giveArgs: []string{"-h"},
				want:     map[string]string{"--flag-debug": "", "--help": "", "--verbose": ""},
			},
			{
				name:     "notes the config keys in the help of a command with a config file",
				giveArgs: []string{parent, "-h"},
				want: map[string]string{
					"--config": "", "--dsn": "db.dsn", "--flag-debug": "", "--help": "", "--verbose": "verbose",
				},
			},
			{
				name:     "notes the config keys in the help of a subcommand of a command with a config file",
				giveArgs: []string{parent, subcommand, "-h"},
				want: map[string]string{
					"--config": "", "--dsn": "db.dsn", "--flag-debug": "", "--help": "",
					"--steps": "db.migrate.steps", "--verbose": "verbose",
				},
			},
			{
				name:     "leaves out the config keys in the help of a command without a config file on its path",
				giveArgs: []string{command, "-h"},
				want:     map[string]string{"--addr": "", "--flag-debug": "", "--help": "", "--verbose": ""},
			},
		}
		for _, tt := range keys {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _, _ := nesting[keyGlobal, keyStore, keySteps](t)
				app.Command(
					command,
					commandHelp,
					func(context.Context, *cli.Run[keyGlobal, struct{ Addr string }]) error {
						return nil
					},
				)
				r := &recorder{}
				app.Render(r)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.NoError(t, res.err, "Execute writes the help")
				assert.NotNil(t, r.help, "the renderer receives the help")
				got := map[string]string{}
				for _, o := range slices.Concat(r.help.Options, r.help.Globals) {
					got[o.Name] = o.Key
				}
				assert.Equal(t, got, tt.want, "the help notes the config key of each option")
			})
		}
	})
}

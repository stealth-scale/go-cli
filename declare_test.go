// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"context"
	"net/url"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"

	"go.stealthscale.io/cli"
)

// rank is an unexported type that is not a struct, which an options struct embeds.
type rank int

// The options of the tests of declarations.
type (
	shortOptions struct {
		Port int `short:"ab"`
	}
	tlsOptions struct {
		Cert string
	}
	database struct {
		URL string
	}
	promotedOptions struct {
		database
		rank
		addr string
		tls  tlsOptions
		Port int
	}
	prefixedOptions struct {
		DB  database   `prefix:"db"`
		TLS tlsOptions `prefix:""`
	}
	// Settings contains global options, whose type is exported so that the options of a command
	// embed a pointer to them.
	Settings struct {
		Verbose bool
	}
	settingsOptions struct {
		*Settings
		Port int
	}
	vault struct {
		Token string
	}
	// Session is a type without options that two commands of one path have as their options. Its
	// type is exported, so the options of a command embed a pointer to it.
	Session struct{}
	// Pipe is an exported type without a conversion that is not a struct.
	Pipe        chan int
	invalidTLS  struct{ Cert chan int }
	tlsGroup    struct{ TLS invalidTLS }
	embeddedURL struct{ *url.URL }
)

// declared returns the error of a run of the app srv with the global options G and the command serve
// with the options C.
func declared[G, C any](tb assert.TB) error {
	tb.Helper()
	app, _ := serving[G, C](tb)
	return execute(tb, app, []string{"-h"}, nil, nil).err
}

// nestedDeclared returns the error of a run of the app srv with the global options G, the command db
// with the options P and its subcommand migrate with the options C.
func nestedDeclared[G, P, C any](tb assert.TB) error {
	tb.Helper()
	app, _, _ := nesting[G, P, C](tb)
	return execute(tb, app, []string{"-h"}, nil, nil).err
}

// versionDeclared returns the error of a run of the app srv with the global options G, the version
// versionText and the command serve with the options C.
func versionDeclared[G, C any](tb assert.TB) error {
	tb.Helper()
	app, _ := serving[G, C](tb)
	app.Version(versionText)
	return execute(tb, app, []string{"-h"}, nil, nil).err
}

// TestDeclare checks the options that Execute declares for the fields of the options structs, and
// the declarations that it refuses.
func TestDeclare(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		t.Run("names the type and the field of an invalid option", func(t *testing.T) {
			t.Parallel()
			err := declared[struct{}, shortOptions](t)
			assert.ErrorIs(t, err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, err.Error(), `cli: invalid declaration: the field cli_test.shortOptions.Port has the `+
				`short flag "ab", which is not one ASCII letter other than h`, "the error names the type and the field")
		})

		t.Run("names the group of an invalid option in the path of its field", func(t *testing.T) {
			t.Parallel()
			err := declared[struct{}, tlsGroup](t)
			assert.Equal(t, err.Error(), "cli: invalid declaration: the field cli_test.tlsGroup.TLS.Cert has the type "+
				"chan int, which has no conversion", "the error names the group and the field")
		})

		t.Run("declares an option for an embedded pointer to a type with a conversion", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, embeddedURL](t)
			res := execute(t, app, []string{command, "--url", "https://example.com"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().String(), "https://example.com", "the run sets the embedded pointer")
		})

		tests := []struct {
			name string
			give func(tb assert.TB) error
			want string
		}{
			{
				name: "returns ErrDeclaration for global options of a type that is not a struct",
				give: declared[int, struct{}],
				want: "the options type int is not a struct",
			},
			{
				name: "returns ErrDeclaration for options of a command of a type that is not a struct",
				give: declared[struct{}, []string],
				want: "the options type []string is not a struct",
			},
			{
				name: "returns ErrDeclaration for a field of a type without a conversion",
				give: declared[struct{}, struct{ Done chan int }],
				want: "has the type chan int, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a map with keys of a type without a conversion",
				give: declared[struct{}, struct{ Done map[chan int]string }],
				want: "has the type map[chan int]string, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a map with values of a type without a conversion",
				give: declared[struct{}, struct{ Done map[string]chan int }],
				want: "has the type map[string]chan int, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a slice of a type without a conversion",
				give: declared[struct{}, struct{ Done []chan int }],
				want: "has the type []chan int, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a pointer to a type without a conversion",
				give: declared[struct{}, struct{ Done *chan int }],
				want: "has the type *chan int, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a list of config files",
				give: declared[struct{ Config []cli.ConfigFile }, struct{}],
				want: "has the type []cli.ConfigFile, and a config file takes one path",
			},
			{
				name: "returns ErrDeclaration for a pointer to a config file",
				give: declared[struct{ Config *cli.ConfigFile }, struct{}],
				want: "has the type *cli.ConfigFile, and a config file takes one path",
			},
			{
				name: "returns ErrDeclaration for a tag prefix of an option",
				give: declared[struct{}, struct {
					Addr string `prefix:"x"`
				}],
				want: "is no group, so it takes no tag prefix",
			},
			{
				name: "returns ErrDeclaration for a field name that is not a name of lower-case ASCII words",
				give: declared[struct{}, struct{ Größe int }],
				want: `has the name "größe", which is not a name of lower-case words`,
			},
			{
				name: "returns ErrDeclaration for a tag name that is not a name of lower-case words",
				give: declared[struct{}, struct {
					Addr string `name:"Listen"`
				}],
				want: `has the name "Listen", which is not a name of lower-case words`,
			},
			{
				name: "returns ErrDeclaration for a field name whose words the case of its letters does not separate",
				give: declared[struct{}, struct{ Ab中Cd int }],
				want: `has the name "ab中cd", which is not a name of lower-case words`,
			},
			{
				name: "returns ErrDeclaration for a short flag of a digit",
				give: declared[struct{}, struct {
					Port int `short:"1"`
				}],
				want: `has the short flag "1", which is not one ASCII letter other than h`,
			},
			{
				name: "returns ErrDeclaration for a short flag of the character after z",
				give: declared[struct{}, struct {
					Port int `short:"{"`
				}],
				want: `has the short flag "{", which is not one ASCII letter other than h`,
			},
			{
				name: "returns ErrDeclaration for a short flag of a character between Z and a",
				give: declared[struct{}, struct {
					Port int `short:"_"`
				}],
				want: `has the short flag "_", which is not one ASCII letter other than h`,
			},
			{
				name: "returns ErrDeclaration for the short flag h of the help",
				give: declared[struct{}, struct {
					Port int `short:"h"`
				}],
				want: `has the short flag "h", which is not one ASCII letter other than h`,
			},
			{
				name: "returns ErrDeclaration for an empty short flag",
				give: declared[struct{}, struct {
					Port int `short:""`
				}],
				want: `has the short flag "", which is not one ASCII letter other than h`,
			},
			{
				name: "returns ErrDeclaration for an empty tag env",
				give: declared[struct{}, struct {
					Port int `env:""`
				}],
				want: "has an empty tag env, and the value - removes the name",
			},
			{
				name: "returns ErrDeclaration for an empty tag key",
				give: declared[struct{}, struct {
					Port int `key:""`
				}],
				want: "has an empty tag key, and the value - removes the name",
			},
			{
				name: "returns ErrDeclaration for a config key with an empty name",
				give: declared[struct{}, struct {
					Port int `key:"serve..port"`
				}],
				want: `has the config key "serve..port", which has an empty name`,
			},
			{
				name: "returns ErrDeclaration for a positional argument of the global options of an app with commands",
				give: declared[struct {
					Target string `arg:""`
				}, struct{}],
				want: "the app srv has commands and the positional argument TARGET",
			},
			{
				name: "returns ErrDeclaration for an app without a handler and without commands",
				give: func(tb assert.TB) error {
					tb.Helper()
					return execute(tb, cli.New[struct{}](appName, appHelp), []string{"-h"}, nil, nil).err
				},
				want: "the app srv has no handler and no commands",
			},
			{
				name: "returns ErrDeclaration for a subcommand without a handler and without subcommands",
				give: func(tb assert.TB) error {
					tb.Helper()
					app := cli.New[struct{}](appName, appHelp)
					db := app.Command(parent, parentHelp, func(context.Context, *cli.Run[struct{}, struct{}]) error {
						return nil
					})
					db.Command[struct{}](subcommand, "", nil)
					return execute(tb, app, []string{"-h"}, nil, nil).err
				},
				want: "the command db migrate has no handler and no subcommands",
			},
			{
				name: "returns ErrDeclaration for a command with subcommands and a positional argument",
				give: nestedDeclared[struct{}, struct {
					Name string `arg:""`
				}, struct{}],
				want: "the command db has subcommands and the positional argument NAME",
			},
			{
				name: "returns ErrDeclaration for a subcommand name that is not a name of lower-case words",
				give: func(tb assert.TB) error {
					tb.Helper()
					app := cli.New[struct{}](appName, appHelp)
					db := app.Command[struct{}](parent, parentHelp, nil)
					db.Command("Migrate", "", func(context.Context, *cli.Run[struct{}, struct{}]) error { return nil })
					return execute(tb, app, []string{"-h"}, nil, nil).err
				},
				want: `the command name "Migrate" is not a name of lower-case words`,
			},
			{
				name: "returns ErrDeclaration for two subcommands of one name",
				give: func(tb assert.TB) error {
					tb.Helper()
					app := cli.New[struct{}](appName, appHelp)
					db := app.Command[struct{}](parent, parentHelp, nil)
					for range 2 {
						db.Command(
							subcommand,
							"",
							func(context.Context, *cli.Run[struct{}, struct{}]) error { return nil },
						)
					}
					return execute(tb, app, []string{"-h"}, nil, nil).err
				},
				want: "two commands have the name db migrate",
			},
			{
				name: "returns ErrDeclaration for an option of a subcommand with the flag of an option of its command",
				give: nestedDeclared[struct{}, struct{ Steps int }, struct{ Steps int }],
				want: "two options of the command db migrate declare the flag --steps",
			},
			{
				name: "returns ErrDeclaration for an option of a subcommand with the variable of an option of its command",
				give: nestedDeclared[struct{}, struct{ MigrateSteps int }, struct{ Steps int }],
				want: "two options of the command db migrate declare the environment variable SRV_DB_MIGRATE_STEPS",
			},
			{
				name: "returns ErrDeclaration for an embedded pointer to a type without a conversion that is not a struct",
				give: declared[struct{}, struct{ *Pipe }],
				want: "has the type *cli_test.Pipe, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a named pointer to a group",
				give: declared[struct{}, struct{ DB *Store }],
				want: "has the type *cli_test.Store, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for an embedded pointer without an ancestor of its options type",
				give: declared[struct{}, struct{ *Store }],
				want: "embeds *cli_test.Store without an ancestor with options of that type",
			},
			{
				name: "returns ErrDeclaration for an embedded pointer with two ancestors of its options type",
				give: func(tb assert.TB) error {
					tb.Helper()
					app := cli.New[struct{}](appName, appHelp)
					backup := app.Command[Session](parent, parentHelp, nil).Command[Session]("backup", "", nil)
					backup.Command("create", "", func(context.Context, *cli.Run[struct{}, struct{ *Session }]) error {
						return nil
					})
					return execute(tb, app, []string{"-h"}, nil, nil).err
				},
				want: "embeds *cli_test.Session, and two ancestors have options of that type",
			},
			{
				name: "returns ErrDeclaration for an unexported embedded pointer to the options of an ancestor",
				give: nestedDeclared[struct{}, vault, struct{ *vault }],
				want: "is unexported, so the package cannot set it to the options of an ancestor",
			},
			{
				name: "returns ErrDeclaration for a positional argument of a map",
				give: declared[struct{}, struct {
					Labels map[string]string `arg:""`
				}],
				want: "is a positional argument of a map, which takes no entries",
			},
			{
				name: "returns ErrDeclaration for a positional argument with a short flag",
				give: declared[struct{}, struct {
					Target string `arg:"" short:"t"`
				}],
				want: "is a positional argument, which takes no tag short",
			},
			{
				name: "returns ErrDeclaration for a positional argument with an environment variable",
				give: declared[struct{}, struct {
					Target string `arg:"" env:"TARGET"`
				}],
				want: "is a positional argument, which takes no tag env",
			},
			{
				name: "returns ErrDeclaration for a positional argument with a config key",
				give: declared[struct{}, struct {
					Target string `arg:"" key:"target"`
				}],
				want: "is a positional argument, which takes no tag key",
			},
			{
				name: "returns ErrDeclaration for a config file with a config key",
				give: declared[struct {
					Config cli.ConfigFile `key:"config"`
				}, struct{}],
				want: "is a config file, which takes no tag key",
			},
			{
				name: "returns ErrDeclaration for a config file that is a positional argument",
				give: declared[struct{}, struct {
					Config cli.ConfigFile `arg:""`
				}],
				want: "is a config file, which takes no tag arg",
			},
			{
				name: "returns ErrDeclaration for a required option with a default",
				give: declared[struct{}, struct {
					Port int `required:"" default:"80"`
				}],
				want: "is required and has a default, which the tag required makes void",
			},
			{
				name: "returns ErrDeclaration for a default that does not convert",
				give: declared[struct{}, struct {
					Port int `default:"eighty"`
				}],
				want: `has the default "eighty", whose value "eighty" is invalid: invalid syntax`,
			},
			{
				name: "returns ErrDeclaration for a default that the enum excludes",
				give: declared[struct{}, struct {
					Mode string `default:"test" enum:"dev,prod"`
				}],
				want: `has the default "test", whose value "test" is invalid: want dev or prod`,
			},
			{
				name: "returns ErrDeclaration for a group with a tag of an option",
				give: declared[struct{}, struct {
					TLS tlsOptions `help:"The certificate."`
				}],
				want: "takes no tag help",
			},
			{
				name: "returns ErrDeclaration for a prefix that is not a name of lower-case words",
				give: declared[struct{}, struct {
					TLS tlsOptions `prefix:"TLS"`
				}],
				want: `has the name "TLS", which is not a name of lower-case words`,
			},
			{
				name: "returns ErrDeclaration for a group whose field name is not a name of lower-case ASCII words",
				give: declared[struct{}, struct{ Über tlsOptions }],
				want: `has the name "über", which is not a name of lower-case words`,
			},
			{
				name: "returns ErrDeclaration for an invalid option of a group",
				give: declared[struct{}, struct {
					TLS struct {
						Cert chan int
					}
				}],
				want: "has the type chan int, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for two options of one flag",
				give: declared[struct{}, struct {
					Addr   string
					Listen string `name:"addr"`
				}],
				want: "two options of the command serve declare the flag --addr",
			},
			{
				name: "returns ErrDeclaration for two options of one short flag",
				give: declared[struct{}, struct {
					Addr string `short:"a"`
					All  bool   `short:"a"`
				}],
				want: "two options of the command serve declare the short flag -a",
			},
			{
				name: "returns ErrDeclaration for two options of one environment variable",
				give: declared[struct{}, struct {
					Addr string `env:"ADDR"`
					Host string `env:"ADDR"`
				}],
				want: "two options of the command serve declare the environment variable ADDR",
			},
			{
				name: "returns ErrDeclaration for two options of one config key",
				give: declared[struct{}, struct {
					Addr string `key:"listen"`
					Host string `key:"listen"`
				}],
				want: "two options of the command serve declare the config key listen",
			},
			{
				name: "returns ErrDeclaration for a global option and an option of the command with one flag",
				give: declared[struct{ Verbose bool }, struct {
					Verbose bool `env:"-" key:"-"`
				}],
				want: "two options of the command serve declare the flag --verbose",
			},
			{
				name: "returns ErrDeclaration for two global options of one environment variable",
				give: declared[struct {
					Addr string `env:"ADDR"`
					Host string `env:"ADDR"`
				}, struct{}],
				want: "two options of the global options declare the environment variable ADDR",
			},
			{
				name: "returns ErrDeclaration for a global option with the name of --flag-debug",
				give: declared[struct{ FlagDebug bool }, struct{}],
				want: "two options of the global options declare the flag --flag-debug",
			},
			{
				name: "returns ErrDeclaration for an option with the flag of the help",
				give: declared[struct{}, struct{ Help bool }],
				want: "an option of the command serve declares the reserved flag --help",
			},
			{
				name: "returns ErrDeclaration for a global option with the flag of the version",
				give: versionDeclared[struct{ Version bool }, struct{}],
				want: "an option of the global options declares the reserved flag --version",
			},
			{
				name: "returns ErrDeclaration for an option of a command with the flag of the version",
				give: versionDeclared[struct{}, struct{ Version string }],
				want: "an option of the command serve declares the reserved flag --version",
			},
			{
				name: "returns ErrDeclaration for two config files",
				give: declared[struct{ Config cli.ConfigFile }, struct{ Settings cli.ConfigFile }],
				want: "--config and --settings of the command serve are both config files",
			},
			{
				name: "returns ErrDeclaration for a positional argument after a list",
				give: declared[struct{}, struct {
					Files  []string `arg:""`
					Target string   `arg:""`
				}],
				want: "the argument TARGET of the command serve follows the list FILES",
			},
			{
				name: "returns ErrDeclaration for a required positional argument after an optional one",
				give: declared[struct{}, struct {
					Source string `arg:"" default:"."`
					Target string `arg:""`
				}],
				want: "the required argument TARGET of the command serve follows the optional argument SOURCE",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.give(t)
				assert.ErrorIs(t, err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
				assert.HasSuffix(t, err.Error(), tt.want, "the error states the invalid declaration")
			})
		}

		t.Run("declares an option with the flag of the version in an app without a version", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, struct{ Version string }](t)
			res := execute(t, app, []string{command, "--version", "v2"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Version, "v2", "the option has the value of the flag")
		})

		t.Run("returns ErrDeclaration for an app name that is not a name of lower-case words", func(t *testing.T) {
			t.Parallel()
			app := cli.New[struct{}]("Srv", appHelp)
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.Equal(
				t,
				res.err.Error(),
				`cli: invalid declaration: the app name "Srv" is not a name of lower-case words`,
				"Execute returns ErrDeclaration",
			)
		})

		t.Run("returns ErrDeclaration for a command name that is not a name of lower-case words", func(t *testing.T) {
			t.Parallel()
			app := cli.New[struct{}](appName, appHelp)
			app.Command("Serve", commandHelp, func(context.Context, *cli.Run[struct{}, struct{}]) error { return nil })
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.Equal(t, res.err.Error(),
				`cli: invalid declaration: the command name "Serve" is not a name of lower-case words`,
				"Execute returns ErrDeclaration")
		})

		t.Run("returns ErrDeclaration for two commands of one name", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, struct{}](t)
			app.Command(command, commandHelp, func(context.Context, *cli.Run[struct{}, struct{}]) error { return nil })
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: two commands have the name serve",
				"Execute returns ErrDeclaration")
		})

		t.Run("returns every invalid registration", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, struct{}](t)
			app.Format(nil, ".conf")
			app.Use(nil)
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: a format needs a decoder and an extension\n"+
				"cli: invalid declaration: the app srv has a nil hook", "Execute returns both registrations")
		})

		t.Run("returns ErrDeclaration for a config key of a map in one command and of a value in another",
			func(t *testing.T) {
				t.Parallel()
				app, _ := serving[struct{}, struct {
					Labels map[string]string `key:"labels"`
				}](t)
				app.Command("migrate", "", func(context.Context, *cli.Run[struct{}, struct {
					Labels string `key:"labels"`
				}],
				) error {
					return nil
				})
				res := execute(t, app, []string{"-h"}, nil, nil)
				assert.Equal(t, res.err.Error(),
					"cli: invalid declaration: the config key labels is a map in one command and not in another",
					"Execute returns ErrDeclaration")
			})

		t.Run("returns ErrDeclaration for a config key inside the config key of a map", func(t *testing.T) {
			t.Parallel()
			err := declared[struct {
				Labels map[string]string `key:"labels"`
			}, struct {
				Team string `key:"labels.team"`
			}](t)
			assert.Equal(
				t,
				err.Error(),
				"cli: invalid declaration: the config key labels.team is inside the key labels of a map",
				"Execute returns ErrDeclaration",
			)
		})

		declarations := []struct {
			name     string
			giveArgs []string
			want     promotedOptions
		}{
			{
				name:     "declares the exported fields of an unexported embedded struct",
				giveArgs: []string{command, "--url", "postgres://db"},
				want:     promotedOptions{database: database{URL: "postgres://db"}},
			},
			{
				name:     "declares the exported field of options with unexported fields",
				giveArgs: []string{command, "--port", "8"},
				want:     promotedOptions{Port: 8},
			},
		}
		for _, tt := range declarations {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[struct{}, promotedOptions](t)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.NoError(t, res.err, "Execute runs the command")
				assert.Equal(t, *got.Config(), tt.want, "the run sets the declared option")
			})
		}

		valid := []struct {
			name string
			give func(tb assert.TB) error
		}{
			{
				name: "accepts the short flags a and z",
				give: declared[struct{}, struct {
					All  bool `short:"a"`
					Zone bool `short:"z"`
				}],
			},
			{
				name: "accepts the short flags A and Z",
				give: declared[struct{}, struct {
					All  bool `short:"A"`
					Zone bool `short:"Z"`
				}],
			},
			{
				name: "accepts two options without an environment variable",
				give: declared[struct{}, struct {
					Addr string `env:"-"`
					Port int    `env:"-"`
				}],
			},
			{
				name: "accepts a map and a value without a config key",
				give: declared[struct{}, struct {
					Labels map[string]string `key:"-"`
					Port   int               `key:"-"`
				}],
			},
			{
				name: "accepts a config key inside the config key of a value",
				give: declared[struct{}, struct {
					Addr string `key:"listen"`
					Port int    `key:"listen.port"`
				}],
			},
			{
				name: "accepts an environment variable with dots",
				give: declared[struct{}, struct {
					Addr string `env:"SRV..ADDR"`
				}],
			},
			{
				name: "accepts one config key of a value for two commands",
				give: func(tb assert.TB) error {
					tb.Helper()
					app, _ := serving[struct{}, struct {
						URL string `key:"database.url"`
					}](tb)
					app.Command("migrate", "", func(context.Context, *cli.Run[struct{}, struct {
						URL string `key:"database.url"`
					}],
					) error {
						return nil
					})
					return execute(tb, app, []string{"-h"}, nil, nil).err
				},
			},
			{
				name: "accepts a positional argument of the global options of an app without commands",
				give: func(tb assert.TB) error {
					tb.Helper()
					app := cli.New[filesGlobal](appName, appHelp)
					app.Handle(func(context.Context, *cli.Run[filesGlobal, filesGlobal]) error { return nil })
					return execute(tb, app, []string{"-h"}, nil, nil).err
				},
			},
			{
				name: "accepts one config key of a map for two commands",
				give: func(tb assert.TB) error {
					tb.Helper()
					app, _ := serving[struct{}, struct {
						Labels map[string]string `key:"labels"`
					}](tb)
					app.Command("migrate", "", func(context.Context, *cli.Run[struct{}, struct {
						Labels map[string]int `key:"labels"`
					}],
					) error {
						return nil
					})
					return execute(tb, app, []string{"-h"}, nil, nil).err
				},
			},
		}
		for _, tt := range valid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, tt.give(t), "Execute accepts the declaration")
			})
		}

		for _, flag := range []string{"--addr", "--tls-cert", "--rank"} {
			t.Run("declares no flag "+flag+" for an unexported field", func(t *testing.T) {
				t.Parallel()
				app, _ := serving[struct{}, promotedOptions](t)
				res := execute(t, app, []string{command, flag, "x"}, nil, nil)
				assert.ErrorIs(t, res.err, cli.ErrUnknownFlag, "Execute returns ErrUnknownFlag")
			})
		}

		t.Run("leaves an unexported field at its zero value", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, promotedOptions](t)
			vars := map[string]string{"SRV_SERVE_ADDR": "x", "SRV_SERVE_TLS_CERT": "x", "SRV_SERVE_RANK": "1"}
			res := execute(t, app, []string{command}, vars, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			expect.Equal(t, got.Config().addr, "", "the run does not set an unexported field")
			expect.Equal(t, got.Config().tls, tlsOptions{}, "the run does not set an unexported group")
			expect.Equal(t, got.Config().rank, rank(0), "the run does not set an embedded field of an unexported type")
		})

		t.Run("sets an embedded pointer to the options of an ancestor", func(t *testing.T) {
			t.Parallel()
			app, _, migrate := nesting[struct{}, Store, stepsOptions](t)
			res := execute(t, app, []string{parent, subcommand}, map[string]string{"SRV_DB_DSN": "pg"}, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.NotNil(t, migrate.Config().Store, "the run sets the embedded pointer")
			assert.Equal(t, migrate.Config().DSN, "pg", "the subcommand reads the option of its command")
		})

		t.Run("sets an embedded pointer to the global options", func(t *testing.T) {
			t.Parallel()
			app, got := serving[Settings, settingsOptions](t)
			res := execute(t, app, []string{command, "--verbose"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Settings, got.Global(), "the embedded pointer points to the global options",
				assert.ByIdentity())
		})

		t.Run(
			"leaves an unexported embedded pointer without an ancestor of its options type at nil",
			func(t *testing.T) {
				t.Parallel()
				app, got := serving[struct{}, struct {
					*vault
					Port int
				}](t)
				res := execute(t, app, []string{command, "--port", "8"}, nil, nil)
				assert.NoError(t, res.err, "Execute runs the command")
				assert.Nil(t, got.Config().vault, "the run does not set the pointer")
			},
		)

		t.Run("declares the options of an embedded struct with a prefix in a group", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, prefixedOptions](t)
			res := execute(t, app, []string{command, "--db-url", "postgres://db", "--cert", "srv.pem"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, *got.Config(), prefixedOptions{
				DB:  database{URL: "postgres://db"},
				TLS: tlsOptions{Cert: "srv.pem"},
			}, "the run sets the options of the group db and of the group without a prefix")
		})

		t.Run("declares no environment variable for the value - of the tag env", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, struct {
				Port int `env:"-"`
			}](t)
			res := execute(t, app, []string{command}, map[string]string{"SRV_SERVE_PORT": "8"}, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Port, 0, "the run reads no variable")
		})

		t.Run("declares no config key for the value - of the tag key", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{ Config cli.ConfigFile }, struct {
				Port int `key:"-"`
			}](t)
			files := fstest.MapFS{configPath: {Data: []byte(`{"serve": {"port": 8}}`)}}
			res := execute(t, app, []string{"--config", configPath, command}, nil, files)
			assert.ErrorIs(t, res.err, cli.ErrUnknownKey, "Execute returns ErrUnknownKey for the removed key")
		})

		t.Run("declares no config key for a config file", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{ Config cli.ConfigFile }, struct{}](t)
			files := fstest.MapFS{configPath: {Data: []byte(`{"config": "other.json"}`)}}
			res := execute(t, app, []string{"--config", configPath, command}, nil, files)
			assert.ErrorIs(t, res.err, cli.ErrUnknownKey, "Execute returns ErrUnknownKey for the key of the path")
		})

		t.Run("declares no environment variable for a positional argument", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, struct {
				Target string `arg:""`
			}](t)
			res := execute(t, app, []string{command}, map[string]string{"SRV_SERVE_TARGET": "v2"}, nil)
			assert.ErrorIs(t, res.err, cli.ErrMissing, "the run reads no variable for the argument")
		})
	})
}

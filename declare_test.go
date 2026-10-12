// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"context"
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
)

// declared returns the error of a run of the app srv with the global options G and the command serve
// with the options C.
func declared[G, C any](tb assert.TB) error {
	tb.Helper()
	app, _ := serving[G, C](tb)
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
				name: "returns ErrDeclaration for a positional argument of the global options",
				give: declared[struct {
					Target string `arg:""`
				}, struct{}],
				want: "is a positional argument of the global options, which come before the command",
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
				want: "an option of the command serve declares the flag --help of the help",
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
			app.Command[struct{}]("migrate", "", nil)
			res := execute(t, app, []string{"-h"}, nil, nil)
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: a format needs a decoder and an extension\n"+
				"cli: invalid declaration: the command migrate has no handler", "Execute returns both registrations")
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

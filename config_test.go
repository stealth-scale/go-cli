// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"

	"go.stealthscale.io/cli"
)

// The config files of the tests of config files.
const (
	configPath   = "srv.json"
	otherPath    = "other.json"
	defaultPath  = "default.json"
	unknownPath  = "srv.unknown"
	decodedValue = "decoded"
)

// errPermission is the error of an Env that may not read a file.
var errPermission = errors.New("open srv.json: permission denied")

// The options of the app in the tests of config files.
type (
	configGlobal struct {
		Config  cli.ConfigFile `short:"c"`
		Verbose bool
	}
	defaultGlobal struct {
		Config cli.ConfigFile `default:"default.json"`
	}
	firstGlobal struct {
		Verbose bool
		Config  cli.ConfigFile
	}
	unnamedGlobal struct {
		Config cli.ConfigFile `env:"-"`
	}
	configOptions struct {
		When   time.Time
		Labels map[string]string
		TLS    struct {
			Cert string
		}
		Addr string `default:":8080"`
		Tags []string
		Port int
	}
	migrateOptions struct {
		Target string
	}
)

// decoder returns a format whose tree sets the address of the command serve to value, whatever the
// data.
func decoder(value string) cli.Format {
	return func([]byte) (map[string]any, error) {
		return map[string]any{"serve": map[string]any{"addr": value}}, nil
	}
}

// TestConfig checks the registration of a format and the reading of a config file.
func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("Format", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveExts []string
			givePath string
		}{
			{name: "registers the decoder for an extension", giveExts: []string{".conf"}, givePath: "srv.conf"},
			{
				name:     "registers the decoder for each extension of the call",
				giveExts: []string{".a", ".b"},
				givePath: "srv.b",
			},
			{
				name:     "compares the extension of a path without regard to case",
				giveExts: []string{".conf"},
				givePath: "SRV.CONF",
			},
			{
				name:     "compares a registered extension without regard to case",
				giveExts: []string{".CONF"},
				givePath: "srv.conf",
			},
			{name: "replaces a format that New registers", giveExts: []string{".json"}, givePath: "srv.json"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[configGlobal, configOptions](t)
				app.Format(decoder(decodedValue), tt.giveExts...)
				files := fstest.MapFS{tt.givePath: {Data: []byte("data")}}
				res := execute(t, app, []string{"-c", tt.givePath, command}, nil, files)
				assert.NoError(t, res.err, "Execute reads the config file")
				assert.Equal(t, got.Config().Addr, decodedValue, "the run decodes the file with the registered format")
			})
		}

		t.Run("replaces an earlier registration of an extension", func(t *testing.T) {
			t.Parallel()
			app, got := serving[configGlobal, configOptions](t)
			app.Format(decoder("first"), ".conf")
			app.Format(decoder("second"), ".conf")
			res := execute(t, app, []string{"-c", "srv.conf", command}, nil, fstest.MapFS{"srv.conf": {}})
			assert.NoError(t, res.err, "Execute reads the config file")
			assert.Equal(t, got.Config().Addr, "second", "the run decodes the file with the later format")
		})

		invalid := []struct {
			name       string
			giveDecode cli.Format
			giveExts   []string
			want       string
		}{
			{
				name:     "returns ErrDeclaration for a nil decoder",
				giveExts: []string{".conf"},
				want:     "cli: invalid declaration: a format needs a decoder and an extension",
			},
			{
				name:       "returns ErrDeclaration for no extension",
				giveDecode: decoder(decodedValue),
				want:       "cli: invalid declaration: a format needs a decoder and an extension",
			},
			{
				name:       "returns ErrDeclaration for an extension without a dot",
				giveDecode: decoder(decodedValue),
				giveExts:   []string{"conf"},
				want:       `cli: invalid declaration: the extension "conf" is not a dot followed by a name`,
			},
			{
				name:       "returns ErrDeclaration for a dot without a name",
				giveDecode: decoder(decodedValue),
				giveExts:   []string{".conf", "."},
				want:       `cli: invalid declaration: the extension "." is not a dot followed by a name`,
			},
			{
				name:       "returns ErrDeclaration for an extension of two dots",
				giveDecode: decoder(decodedValue),
				giveExts:   []string{".tar.gz"},
				want:       `cli: invalid declaration: the extension ".tar.gz" is not a dot followed by a name`,
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[configGlobal, configOptions](t)
				app.Format(tt.giveDecode, tt.giveExts...)
				res := execute(t, app, []string{command}, nil, nil)
				assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
				assert.Equal(t, res.err.Error(), tt.want, "the error states the invalid registration")
			})
		}

		t.Run("registers no extension of a call with an invalid extension", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[configGlobal, configOptions](t)
			app.Format(decoder(decodedValue), ".conf", "conf")
			res := execute(t, app, []string{"-c", "srv.conf", command}, nil, fstest.MapFS{"srv.conf": {}})
			assert.Equal(t, res.err.Error(),
				`cli: invalid declaration: the extension "conf" is not a dot followed by a name`,
				"Execute returns the invalid registration and reads no file")
		})

		t.Run("decodes each file with the last registration of its extension under any sequence of calls",
			func(t *testing.T) {
				t.Parallel()
				prop.ForAll(
					t,
					"Execute decodes a config file with the format of the last registration of its extension",
					formatMachine,
				)
			})
	})

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveArgs []string
			giveData string
			want     configOptions
		}{
			{
				name:     "reads the keys of the command",
				giveArgs: []string{"-c", configPath, command},
				giveData: `{"serve": {"addr": ":80", "port": 8}}`,
				want:     configOptions{Addr: ":80", Port: 8},
			},
			{
				name:     "reads the keys of a group under the key of its field",
				giveArgs: []string{"-c", configPath, command},
				giveData: `{"serve": {"tls": {"cert": "srv.pem"}}}`,
				want:     configOptions{Addr: ":8080", TLS: struct{ Cert string }{Cert: "srv.pem"}},
			},
			{
				name:     "skips a key of an option of another command",
				giveArgs: []string{"-c", configPath, command},
				giveData: `{"migrate": {"target": "v2"}, "serve": {"port": 8}}`,
				want:     configOptions{Addr: ":8080", Port: 8},
			},
			{
				name:     "reads a list from an array",
				giveArgs: []string{"-c", configPath, command},
				giveData: `{"serve": {"tags": ["a", "b"]}}`,
				want:     configOptions{Addr: ":8080", Tags: []string{"a", "b"}},
			},
			{
				name:     "reads a list from a text of values that commas separate",
				giveArgs: []string{"-c", configPath, command},
				giveData: `{"serve": {"tags": "a,b"}}`,
				want:     configOptions{Addr: ":8080", Tags: []string{"a", "b"}},
			},
			{
				name:     "reads a map from an object",
				giveArgs: []string{"-c", configPath, command},
				giveData: `{"serve": {"labels": {"team": "core", "env": "prod"}}}`,
				want:     configOptions{Addr: ":8080", Labels: map[string]string{"env": "prod", "team": "core"}},
			},
			{
				name:     "reads a null as no value",
				giveArgs: []string{"-c", configPath, command},
				giveData: `{"serve": {"addr": null}}`,
				want:     configOptions{Addr: ":8080"},
			},
			{
				name:     "reads no file for an empty path",
				giveArgs: []string{"-c", "", command},
				giveData: `{"serve": {"port": 8}}`,
				want:     configOptions{Addr: ":8080"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[configGlobal, configOptions](t)
				app.Command("migrate", "", func(_ context.Context, _ *cli.Run[configGlobal, migrateOptions]) error {
					return nil
				})
				res := execute(t, app, tt.giveArgs, nil, fstest.MapFS{configPath: {Data: []byte(tt.giveData)}})
				assert.NoError(t, res.err, "Execute runs the command")
				assert.Equal(t, *got.Config(), tt.want, "the run reads the keys of the file", assert.EquateEmpty())
			})
		}

		t.Run("reads the key of a global option", func(t *testing.T) {
			t.Parallel()
			app, got := serving[configGlobal, configOptions](t)
			files := fstest.MapFS{configPath: {Data: []byte(`{"verbose": true}`)}}
			res := execute(t, app, []string{"-c", configPath, command}, nil, files)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.True(t, got.Global().Verbose, "the run reads the global key")
		})

		t.Run("reads the key of the first global option", func(t *testing.T) {
			t.Parallel()
			app, got := serving[firstGlobal, configOptions](t)
			files := fstest.MapFS{configPath: {Data: []byte(`{"verbose": true}`)}}
			res := execute(t, app, []string{"--config", configPath, command}, nil, files)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.True(t, got.Global().Verbose, "the run reads the key of the first option")
		})

		t.Run("reads no path from a variable of no name for a config file without a variable", func(t *testing.T) {
			t.Parallel()
			app, got := serving[unnamedGlobal, configOptions](t)
			files := fstest.MapFS{configPath: {Data: []byte(`{"serve": {"addr": ":80"}}`)}}
			res := execute(t, app, []string{command}, map[string]string{"": configPath}, files)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Addr, ":8080", "the run reads no file")
		})

		t.Run("reads the source of a key", func(t *testing.T) {
			t.Parallel()
			app, got := serving[configGlobal, configOptions](t)
			files := fstest.MapFS{configPath: {Data: []byte(`{"serve": {"port": 8}}`)}}
			res := execute(t, app, []string{"-c", configPath, command}, nil, files)
			assert.NoError(t, res.err, "Execute runs the command")
			src, _ := got.Source(&got.Config().Port)
			assert.Equal(t, src, cli.Source{Kind: cli.File, Name: "serve.port", File: configPath},
				"the source of the option is the key of the file")
		})

		t.Run("reads a value through its MarshalText", func(t *testing.T) {
			t.Parallel()
			app, got := serving[configGlobal, configOptions](t)
			when := time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC)
			app.Format(func([]byte) (map[string]any, error) {
				return map[string]any{"serve": map[string]any{"when": when}}, nil
			}, ".conf")
			res := execute(t, app, []string{"-c", "srv.conf", command}, nil, fstest.MapFS{"srv.conf": {}})
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().When, when, "the run reads the text of the time")
		})

		t.Run("reads a value through fmt.Sprint when its MarshalText fails", func(t *testing.T) {
			t.Parallel()
			app, got := serving[configGlobal, configOptions](t)
			app.Format(func([]byte) (map[string]any, error) {
				return map[string]any{"serve": map[string]any{"addr": failingMarshaler(":81")}}, nil
			}, ".conf")
			res := execute(t, app, []string{"-c", "srv.conf", command}, nil, fstest.MapFS{"srv.conf": {}})
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Addr, ":81", "the run reads the value as fmt.Sprint writes it")
		})

		paths := []struct {
			name     string
			giveArgs []string
			giveVars map[string]string
			want     string
		}{
			{
				name:     "reads the file of the last flag of the path",
				giveArgs: []string{"-c", otherPath, "--config", configPath, command},
				want:     ":80",
			},
			{
				name:     "reads the file of the environment variable",
				giveArgs: []string{command},
				giveVars: map[string]string{"SRV_CONFIG": configPath},
				want:     ":80",
			},
			{
				name:     "reads the file of the flag over the file of the environment variable",
				giveArgs: []string{"-c", configPath, command},
				giveVars: map[string]string{"SRV_CONFIG": otherPath},
				want:     ":80",
			},
		}
		for _, tt := range paths {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[configGlobal, configOptions](t)
				files := fstest.MapFS{
					configPath: {Data: []byte(`{"serve": {"addr": ":80"}}`)},
					otherPath:  {Data: []byte(`{"serve": {"addr": ":81"}}`)},
				}
				res := execute(t, app, tt.giveArgs, tt.giveVars, files)
				assert.NoError(t, res.err, "Execute runs the command")
				assert.Equal(t, got.Config().Addr, tt.want, "the run reads the file of the path")
			})
		}

		t.Run("reads the file at the default path", func(t *testing.T) {
			t.Parallel()
			app, got := serving[defaultGlobal, configOptions](t)
			files := fstest.MapFS{defaultPath: {Data: []byte(`{"serve": {"addr": ":80"}}`)}}
			res := execute(t, app, []string{command}, nil, files)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Addr, ":80", "the run reads the file at the default path")
		})

		t.Run("skips a missing file at the default path", func(t *testing.T) {
			t.Parallel()
			app, got := serving[defaultGlobal, configOptions](t)
			res := execute(t, app, []string{command}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Addr, ":8080", "the option keeps its default")
		})

		t.Run("skips the default path for an Env without ReadFile", func(t *testing.T) {
			t.Parallel()
			app, got := serving[defaultGlobal, configOptions](t)
			var out bytes.Buffer
			err := app.Execute(t.Context(), &cli.Env{Stdout: &out, Stderr: &out, Args: []string{command}})
			assert.NoError(t, err, "Execute runs the command")
			assert.Equal(t, got.Config().Addr, ":8080", "the option keeps its default")
		})

		failures := []struct {
			name     string
			giveArgs []string
			giveVars map[string]string
			giveRead func(string) ([]byte, error)
			wantErr  error
			want     string
		}{
			{
				name:     "returns ErrConfigFile for a missing file at the path of a flag",
				giveArgs: []string{"-c", otherPath, command},
				giveRead: fstest.MapFS{}.ReadFile,
				wantErr:  cli.ErrConfigFile,
				want:     "cli: serve: config file other.json: open other.json: file does not exist",
			},
			{
				name:     "returns ErrConfigFile for a missing file at the path of an environment variable",
				giveArgs: []string{command},
				giveVars: map[string]string{"SRV_CONFIG": otherPath},
				giveRead: fstest.MapFS{}.ReadFile,
				wantErr:  cli.ErrConfigFile,
				want:     "cli: serve: config file other.json: open other.json: file does not exist",
			},
			{
				name:     "returns ErrConfigFile for a path of a flag and an Env without ReadFile",
				giveArgs: []string{"-c", otherPath, command},
				wantErr:  cli.ErrConfigFile,
				want:     "cli: serve: config file other.json: read other.json: file does not exist",
			},
			{
				name:     "returns ErrConfigFile for a file that the Env may not read",
				giveArgs: []string{"-c", configPath, command},
				giveRead: func(string) ([]byte, error) { return nil, errPermission },
				wantErr:  cli.ErrConfigFile,
				want:     "cli: serve: config file srv.json: open srv.json: permission denied",
			},
			{
				name:     "returns ErrConfigFile for an extension without a format",
				giveArgs: []string{"-c", unknownPath, command},
				giveRead: fstest.MapFS{unknownPath: {}}.ReadFile,
				wantErr:  cli.ErrConfigFile,
				want:     `cli: serve: config file srv.unknown: no format for ".unknown"`,
			},
			{
				name:     "returns ErrConfigFile for a file that its format cannot decode",
				giveArgs: []string{"-c", configPath, command},
				giveRead: fstest.MapFS{configPath: {Data: []byte(`{"serve": `)}}.ReadFile,
				wantErr:  cli.ErrConfigFile,
				want:     "cli: serve: config file srv.json: cli: decode JSON: unexpected EOF",
			},
			{
				name:     "returns ErrUnknownKey with the closest key of the app",
				giveArgs: []string{"-c", configPath, command},
				giveRead: fstest.MapFS{configPath: {Data: []byte(`{"serve": {"adr": ":80"}}`)}}.ReadFile,
				wantErr:  cli.ErrUnknownKey,
				want:     "cli: serve: unknown key serve.adr in srv.json (did you mean serve.addr?)",
			},
			{
				name:     "returns ErrUnknownKey without a key that is far from every key of the app",
				giveArgs: []string{"-c", configPath, command},
				giveRead: fstest.MapFS{configPath: {Data: []byte(`{"serve": {"listen": ":80"}}`)}}.ReadFile,
				wantErr:  cli.ErrUnknownKey,
				want:     "cli: serve: unknown key serve.listen in srv.json",
			},
			{
				name:     "returns ErrInvalidValue for a value that does not convert",
				giveArgs: []string{"-c", configPath, command},
				giveRead: fstest.MapFS{configPath: {Data: []byte(`{"serve": {"port": "x"}}`)}}.ReadFile,
				wantErr:  cli.ErrInvalidValue,
				want:     `cli: serve: invalid --port "x" from key serve.port of srv.json: invalid syntax`,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[configGlobal, configOptions](t)
				var out bytes.Buffer
				env := &cli.Env{
					Lookup:   cli.Vars(tt.giveVars),
					ReadFile: tt.giveRead,
					Stdout:   &out,
					Stderr:   &out,
					Args:     tt.giveArgs,
				}
				err := app.Execute(t.Context(), env)
				assert.ErrorIs(t, err, tt.wantErr, "Execute returns the error of the config file")
				assert.Equal(t, err.Error(), tt.want, "the error states the file and the cause")
			})
		}

		t.Run("states the key and the file of an unknown key in the *Error", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[configGlobal, configOptions](t)
			files := fstest.MapFS{configPath: {Data: []byte(`{"serve": {"adr": ":80"}}`)}}
			res := execute(t, app, []string{"-c", configPath, command}, nil, files)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error")
			expect.Equal(t, e.Command, command, "the *Error states the command")
			expect.Equal(t, e.Input, "serve.adr", "the *Error states the key")
			expect.Equal(t, e.Value, "", "the *Error states no value")
			expect.Equal(t, e.Source, cli.Source{Kind: cli.File, Name: "serve.adr", File: configPath},
				"the *Error states the key and the file as the source")
		})
	})
}

// failingMarshaler is a value of a config file whose MarshalText fails.
type failingMarshaler string

// MarshalText returns errPermission.
func (failingMarshaler) MarshalText() ([]byte, error) {
	return nil, errPermission
}

// formatMachine registers formats for the extensions .a, .A and .b in a sequence of steps, and
// checks after each run that the run decodes its file with the last registration of its extension.
func formatMachine(c *prop.Case) {
	app, got := serving[configGlobal, configOptions](c)
	stateful.Steps(c, stateful.Machine[map[string]string]{
		Spec: history.Spec[map[string]string]{
			Initial: func() map[string]string { return map[string]string{} },
			Next: func(state map[string]string, op history.Operation) []map[string]string {
				ext, _ := op.Args[0].(string)
				if op.Name == "format" {
					next := maps.Clone(state)
					next[strings.ToLower(ext)], _ = op.Args[1].(string)
					return []map[string]string{next}
				}
				want, ok := state[ext]
				if !ok {
					want = "no format"
				}
				if op.Returned(want) {
					return []map[string]string{state}
				}
				return nil
			},
		},
		Actions: []stateful.Action[map[string]string]{
			{
				Name: "format",
				Input: func(c *prop.Case, _ map[string]string) any {
					return []string{
						c.Draw(prop.SampledFrom(".a", ".A", ".b"), "ext"),
						strconv.Itoa(c.Draw(prop.Integer(1, 3), "decoder")),
					}
				},
				Run: func(c *prop.Case, client int, input any) {
					in, _ := input.([]string)
					call := c.History().Invoke(client, "format", []any{in[0], in[1]})
					app.Format(decoder(in[1]), in[0])
					call.OK(nil)
				},
			},
			{
				Name:  "execute",
				Input: func(c *prop.Case, _ map[string]string) any { return c.Draw(prop.SampledFrom(".a", ".b", ".c"), "ext") },
				Run: func(c *prop.Case, client int, input any) {
					ext, _ := input.(string)
					call := c.History().Invoke(client, "execute", []any{ext})
					path := "srv" + ext
					res := execute(c, app, []string{"-c", path, command}, nil, fstest.MapFS{path: {}})
					switch {
					case errors.Is(res.err, cli.ErrConfigFile):
						call.OK("no format")
					case res.err != nil:
						call.OK(res.err.Error())
					default:
						call.OK(got.Config().Addr)
					}
				},
			},
		},
	})
}

// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"context"
	"strconv"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// The options of the app in the tests of the sources of a run.
type (
	resolveGlobal struct {
		Config cli.ConfigFile
	}
	resolveOptions struct {
		Port int `default:"80"`
	}
	copyOptions struct {
		Source string   `arg:""`
		Target string   `arg:"" default:"."`
		Files  []string `arg:"" default:"a,b"`
	}
	listOptions struct {
		Files []string `arg:""`
	}
	levelGlobal struct {
		Files []string `arg:""`
		Level int
	}
)

// TestResolve checks the order in which a run applies its sources, and the positional arguments.
func TestResolve(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		t.Run("applies the value of the source of the highest precedence", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, "the value and the source of an option are those of its source of the highest precedence",
				func(c *prop.Case) {
					value := prop.Optional(prop.Integer(-1000, 1000))
					file, env, flag := c.Draw(value, "file"), c.Draw(value, "env"), c.Draw(value, "flag")
					args, vars, want := []string{"--config", configPath, command}, map[string]string{}, 80
					wantSource := cli.Source{Kind: cli.Default}
					data := `{}`
					if file != nil {
						data, want = `{"serve": {"port": `+strconv.Itoa(*file)+`}}`, *file
						wantSource = cli.Source{Kind: cli.File, Name: "serve.port", File: configPath}
					}
					if env != nil {
						vars["SRV_SERVE_PORT"], want = strconv.Itoa(*env), *env
						wantSource = cli.Source{Kind: cli.Environment, Name: "SRV_SERVE_PORT"}
					}
					if flag != nil {
						args, want = append(args, "--port", strconv.Itoa(*flag)), *flag
						wantSource = cli.Source{Kind: cli.Flag, Name: "--port"}
					}
					app, got := serving[resolveGlobal, resolveOptions](c)
					res := execute(c, app, args, vars, fstest.MapFS{configPath: {Data: []byte(data)}})
					assert.NoError(c, res.err, "Execute runs the command")
					assert.Equal(
						c,
						got.Config().Port,
						want,
						"the option has the value of the source of the highest precedence",
					)
					src, _ := got.Source(&got.Config().Port)
					assert.Equal(c, src, wantSource, "the source of the option is the source of the highest precedence")
				})
		})

		t.Run("reads no variable for an option without one, also when the Env has a variable of no name",
			func(t *testing.T) {
				t.Parallel()
				app, got := serving[resolveGlobal, struct {
					Port int `default:"80" env:"-"`
				}](t)
				res := execute(t, app, []string{command}, map[string]string{"": "81"}, nil)
				assert.NoError(t, res.err, "Execute runs the command")
				assert.Equal(t, got.Config().Port, 80, "the option keeps its default")
			})

		t.Run("applies the last of repeated flags of a value", func(t *testing.T) {
			t.Parallel()
			app, got := serving[resolveGlobal, resolveOptions](t)
			res := execute(t, app, []string{command, "--port", "81", "--port=82"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Port, 82, "the option has the value of the last flag")
		})

		arguments := []struct {
			name     string
			giveArgs []string
			want     copyOptions
		}{
			{
				name:     "assigns the positional arguments in the order of their fields",
				giveArgs: []string{command, "src", "dst"},
				want:     copyOptions{Source: "src", Target: "dst", Files: []string{"a", "b"}},
			},
			{
				name:     "assigns the rest of the arguments to a list",
				giveArgs: []string{command, "src", "dst", "x", "y,z"},
				want:     copyOptions{Source: "src", Target: "dst", Files: []string{"x", "y,z"}},
			},
			{
				name:     "keeps the defaults of the arguments that the command line leaves out",
				giveArgs: []string{command, "src"},
				want:     copyOptions{Source: "src", Target: ".", Files: []string{"a", "b"}},
			},
		}
		for _, tt := range arguments {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[resolveGlobal, copyOptions](t)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.NoError(t, res.err, "Execute runs the command")
				assert.Equal(t, *got.Config(), tt.want, "the run assigns the positional arguments")
			})
		}

		t.Run("states the source of a positional argument", func(t *testing.T) {
			t.Parallel()
			app, got := serving[resolveGlobal, copyOptions](t)
			res := execute(t, app, []string{command, "src"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			src, _ := got.Source(&got.Config().Source)
			assert.Equal(t, src, cli.Source{Kind: cli.Argument, Name: "SOURCE"}, "the source is the argument")
		})

		t.Run("returns ErrMissing for a list argument without an argument", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[resolveGlobal, listOptions](t)
			res := execute(t, app, []string{command}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrMissing, "Execute returns ErrMissing")
			assert.Equal(t, res.err.Error(), "cli: serve: missing argument FILES", "the error names the argument")
		})

		failures := []struct {
			name     string
			giveArgs []string
			giveVars map[string]string
			wantErr  error
			want     string
		}{
			{
				name:     "returns ErrInvalidValue for an environment variable that does not convert",
				giveArgs: []string{command},
				giveVars: map[string]string{"SRV_SERVE_PORT": "x"},
				wantErr:  cli.ErrInvalidValue,
				want:     `cli: serve: invalid --port "x" from env SRV_SERVE_PORT: invalid syntax`,
			},
			{
				name:     "returns ErrInvalidValue for a flag that does not convert",
				giveArgs: []string{command, "--port", "x"},
				wantErr:  cli.ErrInvalidValue,
				want:     `cli: serve: invalid --port "x" from flag --port: invalid syntax`,
			},
			{
				name:     "returns ErrUnexpectedArgument for an argument that the command does not declare",
				giveArgs: []string{command, "extra"},
				wantErr:  cli.ErrUnexpectedArgument,
				want:     `cli: serve: unexpected argument "extra"`,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[resolveGlobal, resolveOptions](t)
				res := execute(t, app, tt.giveArgs, tt.giveVars, nil)
				assert.ErrorIs(t, res.err, tt.wantErr, "Execute returns the error of the source")
				assert.Equal(t, res.err.Error(), tt.want, "the error states the source")
			})
		}

		t.Run("returns ErrInvalidValue for a positional argument that does not convert", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[resolveGlobal, struct {
				Count int `arg:""`
			}](t)
			res := execute(t, app, []string{command, "x"}, nil, nil)
			assert.Equal(t, res.err.Error(), `cli: serve: invalid COUNT "x" from argument COUNT: invalid syntax`,
				"the error states the argument")
		})

		t.Run("states the first argument that the command does not declare in the *Error", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[resolveGlobal, struct {
				Source string `arg:""`
			}](t)
			res := execute(t, app, []string{command, "src", "dst", "more"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error")
			assert.Equal(t, e.Input, "dst", "the *Error states the first unexpected argument")
		})

		t.Run("applies the flags after an environment variable that does not convert", func(t *testing.T) {
			t.Parallel()
			var got []formatGlobal
			vars := map[string]string{"SRV_DEPTH": "deep"}
			res := execute(t, formats(&recorder{}, &recorder{}, &got), []string{"--format", "json", command}, vars, nil)
			assert.ErrorIs(t, res.err, cli.ErrInvalidValue, "Execute returns ErrInvalidValue")
			assert.Equal(t, got, []formatGlobal{{Format: "json"}}, "the function of RenderFor receives the flag")
		})

		t.Run("returns the error of an environment variable before a later valid variable", func(t *testing.T) {
			t.Parallel()
			var got []formatGlobal
			vars := map[string]string{"SRV_LEVELS": "x", "SRV_FORMAT": "json"}
			res := execute(t, formats(&recorder{}, &recorder{}, &got), []string{command}, vars, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error")
			assert.Equal(t, e.Source, cli.Source{Kind: cli.Environment, Name: "SRV_LEVELS"},
				"the error is the error of the first variable")
		})

		t.Run("returns the error of an environment variable before the error of a flag", func(t *testing.T) {
			t.Parallel()
			var got []formatGlobal
			vars := map[string]string{"SRV_FORMAT": "xml"}
			res := execute(t, formats(&recorder{}, &recorder{}, &got), []string{"--depth", "deep", command}, vars, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error")
			assert.Equal(t, e.Source, cli.Source{Kind: cli.Environment, Name: "SRV_FORMAT"},
				"the error is the error of the variable")
		})

		t.Run("replaces the values of a variable with the next flag after a flag that does not convert",
			func(t *testing.T) {
				t.Parallel()
				var got []formatGlobal
				args := []string{"--levels", "x", "--levels", "3", command}
				vars := map[string]string{"SRV_LEVELS": "1,2"}
				res := execute(t, formats(&recorder{}, &recorder{}, &got), args, vars, nil)
				assert.ErrorIs(t, res.err, cli.ErrInvalidValue, "Execute returns ErrInvalidValue")
				assert.Equal(t, got, []formatGlobal{{Levels: []int{3}, Format: "text"}},
					"the list has the value of the next flag alone")
			})

		t.Run("applies the positional arguments after a flag that does not convert", func(t *testing.T) {
			t.Parallel()
			var got levelGlobal
			app := cli.New[levelGlobal](appName, appHelp)
			app.Handle(func(context.Context, *cli.Run[levelGlobal, levelGlobal]) error { return nil })
			app.RenderFor(func(g *levelGlobal) cli.Renderer {
				got = *g
				return &recorder{}
			})
			res := execute(t, app, []string{"--level", "x", "a.txt"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrInvalidValue, "Execute returns ErrInvalidValue")
			assert.Equal(t, got.Files, []string{"a.txt"}, "the function of RenderFor receives the argument")
		})

		t.Run("returns the error of a flag before the error of an unexpected argument", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[resolveGlobal, struct {
				Source string `arg:""`
				Port   int
			}](t)
			res := execute(t, app, []string{command, "--port", "x", "src", "dst"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrInvalidValue, "Execute returns the error of the flag")
		})

		t.Run("returns the error of a flag before the error of a positional argument", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[resolveGlobal, struct {
				Count int `arg:""`
				Port  int
			}](t)
			res := execute(t, app, []string{command, "--port", "x", "many"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error")
			assert.Equal(t, e.Input, "--port", "the error is the error of the flag")
		})

		t.Run("returns the error of the config file before the error of an environment variable", func(t *testing.T) {
			t.Parallel()
			var got []formatGlobal
			files := fstest.MapFS{configPath: {Data: []byte(`{"depth": "deep"}`)}}
			vars := map[string]string{"SRV_FORMAT": "xml"}
			res := execute(t, formats(&recorder{}, &recorder{}, &got), []string{"-c", configPath, command}, vars, files)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error")
			assert.Equal(t, e.Source, cli.Source{Kind: cli.File, Name: "depth", File: configPath},
				"the error is the error of the key")
		})
	})
}

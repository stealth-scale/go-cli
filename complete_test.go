// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// The words of the command completion in the specs.
const (
	// completionCommand is the name of the command.
	completionCommand = "completion"
	// shellZsh and shellBash are shells of the command.
	shellZsh  = "zsh"
	shellBash = "bash"
	// wordsAfter ends the flags before the words of the command.
	wordsAfter = "--"
	// valuesLine and filesLine are the directive lines of the command.
	valuesLine = ":values\n"
	filesLine  = ":files\n"
)

// region is a value that a function of App.Complete completes in the specs.
type region string

// The options of the apps in the specs of completion.
type (
	completeGlobal struct {
		Config  cli.ConfigFile `help:"Config file."   short:"c"`
		Format  string         `help:"Output format."           default:"text" enum:"text,json"`
		Verbose bool           `help:"Log more."      short:"v"`
	}
	completeServe struct {
		Addr    string            `help:"Address to listen on."     short:"a"`
		Labels  map[string]string `help:"Labels of the server."`
		Origins []string          `help:"Origins that may connect."`
		Port    int               `help:"Port to listen on."        short:"p"`
		Token   string            `help:"Token of the store."                 secret:""`
		Target  string            `help:"Target of the data."                           arg:"" enum:"head,tail"`
		Files   []cli.InputFile   `help:"Files to serve."                               arg:""                  default:""`
	}
	completeMigrate struct {
		Mode string `help:"Direction of the migration." arg:"" enum:"up,down"`
	}
	completeCopy struct {
		From string `arg:"" enum:"a,b"`
		To   string `arg:"" enum:"x,y"`
	}
	regionGlobal struct {
		Region  region   `help:"Region of the store."`
		Regions []region `help:"Regions of the replicas."`
		Home    *region  `help:"Home region."`
		Zone    region   `help:"Zone of the store."       enum:"a,b"`
		Config  cli.ConfigFile
		Format  string `                                           default:"text"`
	}
)

// regionCandidates are the candidates that the function of Complete for region returns.
var regionCandidates = []cli.Candidate{{Value: "eu-west", Text: "Europe West"}, {Value: "us-east", Text: "US East"}}

// completing returns the app srv with the global options completeGlobal, the command serve with the
// options completeServe, and the command db with the subcommand migrate of the options
// completeMigrate.
func completing(tb assert.TB) *cli.App[completeGlobal] {
	tb.Helper()
	app, _ := serving[completeGlobal, completeServe](tb)
	db := app.Command[struct{}](parent, parentHelp, nil)
	db.Command(subcommand, subcommandHelp, func(context.Context, *cli.Run[completeGlobal, completeMigrate]) error {
		return nil
	})
	return app
}

// TestComplete checks the command completion and the functions that Complete registers.
func TestComplete(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		rootFlags := "--config\tConfig file.\n-c\tConfig file.\n" +
			"--flag-debug\tPrint the value and the source of each option before the command runs.\n" +
			"--format\tOutput format.\n--help\tShow the help.\n-h\tShow the help.\n--verbose\tLog more.\n-v\tLog more.\n"
		tests := []struct {
			name     string
			giveArgs []string
			want     string
		}{
			{
				name:     "writes the commands of the app with their help texts",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, ""},
				want: "serve\tRun the server.\ndb\tManage the database.\n" +
					"completion\tWrite the completion script of a shell.\n" + valuesLine,
			},
			{
				name:     "writes the commands without their help texts for bash",
				giveArgs: []string{completionCommand, shellBash, wordsAfter, ""},
				want:     "serve\ndb\ncompletion\n" + valuesLine,
			},
			{
				name:     "writes the commands that start with the prefix",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "s"},
				want:     "serve\tRun the server.\n" + valuesLine,
			},
			{
				name:     "writes the subcommands of a command",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, parent, ""},
				want:     "migrate\tMigrate the database to the current schema.\n" + valuesLine,
			},
			{
				name:     "writes the long flags and the short flags of the app",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "-"},
				want:     rootFlags + valuesLine,
			},
			{
				name:     "writes the flags that start with the prefix",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--f"},
				want: "--flag-debug\tPrint the value and the source of each option before the command runs.\n" +
					"--format\tOutput format.\n" + valuesLine,
			},
			{
				name:     "writes the flags of the path of a command",
				giveArgs: []string{completionCommand, shellBash, wordsAfter, command, "--"},
				want: "--addr\n--config\n--flag-debug\n--format\n--help\n--labels\n--origins\n--port\n--token\n" +
					"--verbose\n" + valuesLine,
			},
			{
				name:     "leaves out a flag that the words set already",
				giveArgs: []string{completionCommand, shellBash, wordsAfter, command, "--port", "8", "--p"},
				want:     valuesLine,
			},
			{
				name:     "writes a flag that the words do not set",
				giveArgs: []string{completionCommand, shellBash, wordsAfter, command, "--port", "8", "--a"},
				want:     "--addr\n" + valuesLine,
			},
			{
				name:     "writes a list flag that the words set already",
				giveArgs: []string{completionCommand, shellBash, wordsAfter, command, "--origins", "a", "--o"},
				want:     "--origins\n" + valuesLine,
			},
			{
				name:     "writes a map flag that the words set already",
				giveArgs: []string{completionCommand, shellBash, wordsAfter, command, "--labels", "a=1", "--l"},
				want:     "--labels\n" + valuesLine,
			},
			{
				name:     "writes the values of the flag before the prefix",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--format", ""},
				want:     "text\njson\n" + valuesLine,
			},
			{
				name:     "writes the values of a short flag at the end of a cluster",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "-vc", ""},
				want:     ":files ini json\n",
			},
			{
				name:     "writes the values of a long flag after its equals sign",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--format=j"},
				want:     "--format=json\n" + valuesLine,
			},
			{
				name:     "writes the files of a long flag after its equals sign",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--config="},
				want:     ":files ini json\n",
			},
			{
				name:     "writes no values after the equals sign of an unknown flag",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--nope="},
				want:     valuesLine,
			},
			{
				name:     "writes no values after the equals sign of a positional argument",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "--target=h"},
				want:     valuesLine,
			},
			{
				name:     "writes files for a positional argument with an equals sign",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "head", "a=b"},
				want:     filesLine,
			},
			{
				name:     "writes true and false for a bool",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--verbose="},
				want:     "--verbose=true\n--verbose=false\n" + valuesLine,
			},
			{
				name:     "writes files with the extensions of the formats for a config file",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--config", ""},
				want:     ":files ini json\n",
			},
			{
				name:     "writes files for an input file",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "head", ""},
				want:     filesLine,
			},
			{
				name:     "writes files for a string",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "--addr", ""},
				want:     filesLine,
			},
			{
				name:     "writes files for a list of strings",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "--origins", ""},
				want:     filesLine,
			},
			{
				name:     "writes no values for an int",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "--port", ""},
				want:     valuesLine,
			},
			{
				name:     "writes no values for a secret option",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "--token", ""},
				want:     valuesLine,
			},
			{
				name:     "writes no values for a map",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "--labels", ""},
				want:     valuesLine,
			},
			{
				name:     "writes the values of the first positional argument",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, ""},
				want:     "head\ntail\n" + valuesLine,
			},
			{
				name:     "writes the values of a list argument at a later position",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "head", "a.txt", ""},
				want:     filesLine,
			},
			{
				name:     "writes no candidates after the last positional argument",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, parent, subcommand, "up", ""},
				want:     valuesLine,
			},
			{
				name:     "writes the values of a positional argument after --",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "--", ""},
				want:     "head\ntail\n" + valuesLine,
			},
			{
				name:     "reads a prefix that starts with a dash after -- as a positional argument",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "--", "-"},
				want:     valuesLine,
			},
			{
				name:     "writes no candidates after a word that does not scan",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "--bogus", ""},
				want:     valuesLine,
			},
			{
				name:     "writes no candidates after a flag of the help",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, command, "-h", ""},
				want:     valuesLine,
			},
			{
				name:     "writes the shells of the command completion",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, completionCommand, ""},
				want:     "bash\nzsh\nfish\npowershell\n" + valuesLine,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				res := execute(t, completing(t), tt.giveArgs, nil, nil)
				assert.NoError(t, res.err, "Execute runs the command completion")
				assert.Equal(t, res.stdout, tt.want, "Stdout receives the candidates and the directive line")
			})
		}

		t.Run("writes the flag of the version of an app with a version", func(t *testing.T) {
			t.Parallel()
			app := completing(t)
			app.Version(versionText)
			res := execute(t, app, []string{completionCommand, shellBash, wordsAfter, "--ver"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "--verbose\n--version\n"+valuesLine, "Stdout receives the flag of the version")
		})

		t.Run("writes no candidates after the flag of the version", func(t *testing.T) {
			t.Parallel()
			app := completing(t)
			app.Version(versionText)
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, "--version", ""}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, valuesLine, "Stdout receives the directive line alone")
		})

		t.Run("writes the values of the second positional argument", func(t *testing.T) {
			t.Parallel()
			app := cli.New[struct{}](appName, appHelp)
			app.Command("copy", "", func(context.Context, *cli.Run[struct{}, completeCopy]) error { return nil })
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, "copy", "a", ""}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "x\ny\n"+valuesLine, "Stdout receives the values of the second argument")
		})

		t.Run("returns ErrUnknownFlag for an unknown flag of the command completion", func(t *testing.T) {
			t.Parallel()
			res := execute(t, completing(t), []string{completionCommand, shellBash, "--bogus"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrUnknownFlag, "Execute returns ErrUnknownFlag")
			assert.Empty(t, res.stdout, "Stdout receives nothing")
		})

		t.Run("writes the help of the command completion", func(t *testing.T) {
			t.Parallel()
			res := execute(t, completing(t), []string{completionCommand, "--help"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.HasPrefix(t, res.stdout, "Usage: srv completion [options] SHELL [WORDS...]\n",
				"Stdout receives the help of the command")
		})

		t.Run("writes the version for the flag of the version after the command completion", func(t *testing.T) {
			t.Parallel()
			app := completing(t)
			app.Version(versionText)
			res := execute(t, app, []string{completionCommand, "--version"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the version")
			assert.Equal(t, res.stdout, versionText+"\n", "Stdout receives the version")
		})

		t.Run("writes the first paragraph of the help text of a command on one line", func(t *testing.T) {
			t.Parallel()
			app := cli.New[struct{}](appName, appHelp)
			app.Command(
				command,
				"Run the server\nof the store.\n\nIt listens on --addr.",
				func(context.Context, *cli.Run[struct{}, struct{}]) error {
					return nil
				},
			)
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, "s"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "serve\tRun the server of the store.\n"+valuesLine,
				"Stdout receives the first paragraph of the help text")
		})

		t.Run("writes the first line of the text of a candidate", func(t *testing.T) {
			t.Parallel()
			app := cli.New[completeGlobal](appName, appHelp)
			app.Command(command, commandHelp, func(context.Context, *cli.Run[completeGlobal, struct {
				Mode string `help:"Run mode.\nIt selects the store." enum:"dev,prod"`
			}],
			) error {
				return nil
			})
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, command, "--m"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "--mode\tRun mode.\n"+valuesLine, "Stdout receives the first line of the text")
		})

		t.Run("returns ErrMissing for a missing shell", func(t *testing.T) {
			t.Parallel()
			res := execute(t, completing(t), []string{completionCommand}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrMissing, "Execute returns ErrMissing")
			assert.Equal(t, res.err.Error(), "cli: completion: missing argument SHELL", "the error states the argument")
		})

		t.Run("returns ErrInvalidValue for an unknown shell", func(t *testing.T) {
			t.Parallel()
			res := execute(t, completing(t), []string{completionCommand, "tcsh"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrInvalidValue, "Execute returns ErrInvalidValue")
			assert.Equal(t, res.err.Error(),
				`cli: completion: invalid SHELL "tcsh" from argument SHELL: want bash, zsh, fish or powershell`,
				"the error states the shells")
		})

		t.Run("does not check the required options of the app", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[requiredGlobal, struct{}](t)
			res := execute(t, app, []string{completionCommand, shellBash, wordsAfter, "s"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "serve\n"+valuesLine, "Stdout receives the candidates")
		})

		t.Run("does not check the environment variables of the app", func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"SRV_VERBOSE": "maybe"}
			res := execute(t, completing(t), []string{completionCommand, shellBash, wordsAfter, "s"}, vars, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "serve\n"+valuesLine, "Stdout receives the candidates")
		})

		t.Run("does not call a hook", func(t *testing.T) {
			t.Parallel()
			ran := false
			app := completing(t)
			app.Use(
				func(ctx context.Context, _ *cli.Run[completeGlobal, completeGlobal], next func(context.Context) error) error {
					ran = true
					return next(ctx)
				},
			)
			res := execute(t, app, []string{completionCommand, shellBash, wordsAfter, ""}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.False(t, ran, "the command calls no hook")
		})

		t.Run("does not open the config file", func(t *testing.T) {
			t.Parallel()
			opened := 0
			files := fstest.MapFS{"srv.json": {Data: []byte(`{"format": "json"}`)}}
			var out bytes.Buffer
			env := &cli.Env{
				Open: func(name string) (fs.File, error) {
					opened++
					return files.Open(name)
				},
				Stdout: &out,
				Stderr: &out,
				Args:   []string{completionCommand, shellBash, wordsAfter, "-c", "srv.json", command, ""},
			}
			assert.NoError(t, completing(t).Execute(t.Context(), env), "Execute runs the command completion")
			assert.Equal(t, opened, 0, "the command opens no file")
		})

		t.Run("returns an error that wraps the error of the standard output", func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			args := []string{completionCommand, shellBash, wordsAfter, ""}
			err := completing(t).Execute(t.Context(), &cli.Env{Stdout: failingWriter{}, Stderr: &stderr, Args: args})
			assert.ErrorIs(t, err, errWrite, "Execute returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the completion: disk full", "the error states the completion")
		})

		t.Run("adds the command completion to an app with a handler and without commands", func(t *testing.T) {
			t.Parallel()
			app := cli.New[appGlobal](appName, appHelp)
			app.Handle(func(context.Context, *cli.Run[appGlobal, appGlobal]) error { return nil })
			res := execute(t, app, []string{completionCommand, shellBash, wordsAfter, ""}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "completion\n"+valuesLine, "Stdout receives the command completion")
		})

		t.Run("adds no command completion to an app whose root takes positional arguments", func(t *testing.T) {
			t.Parallel()
			var got []string
			app := cli.New[filesGlobal](appName, appHelp)
			app.Handle(func(_ context.Context, run *cli.Run[filesGlobal, filesGlobal]) error {
				got = run.Global().Files
				return nil
			})
			res := execute(t, app, []string{completionCommand, shellBash}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the handler of the app")
			assert.Equal(t, got, []string{completionCommand, shellBash}, "the handler receives the words as arguments")
		})

		t.Run("returns ErrDeclaration for a command named completion", func(t *testing.T) {
			t.Parallel()
			app := completing(t)
			app.Command(
				completionCommand,
				"",
				func(context.Context, *cli.Run[completeGlobal, struct{}]) error { return nil },
			)
			res := execute(t, app, []string{command, "head"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: the command name completion is reserved",
				"the error states the name")
		})

		t.Run("runs a subcommand named completion", func(t *testing.T) {
			t.Parallel()
			ran := false
			app := completing(t)
			cache := app.Command[struct{}]("cache", "", nil)
			cache.Command(completionCommand, "", func(context.Context, *cli.Run[completeGlobal, struct{}]) error {
				ran = true
				return nil
			})
			res := execute(t, app, []string{"cache", completionCommand}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.True(t, ran, "the handler of the subcommand runs")
		})

		t.Run("offers flags and commands that scan after the words", func(t *testing.T) {
			t.Parallel()
			token := prop.SampledFrom(command, parent, subcommand, "up", "head", "a.txt", "--format", "json", "-v",
				"-c", "srv.json", "--port", "8", "--origins", "a", "--", "-")
			prop.ForAll(t, "each flag and command of the candidates scans after the words", func(c *prop.Case) {
				words := c.Draw(prop.List(token, prop.MaxSize(5)), "words")
				prefix := c.Draw(prop.SampledFrom("", "-", "--"), "prefix")
				args := slices.Concat([]string{completionCommand, shellBash, wordsAfter}, words, []string{prefix})
				res := execute(c, completing(c), args, nil, nil)
				assert.NoError(c, res.err, "Execute runs the command completion")
				lines := strings.Split(strings.TrimSuffix(res.stdout, "\n"), "\n")
				assert.Total(c, func(value string) error {
					err := execute(c, completing(c), slices.Concat(words, []string{value}), nil, nil).err
					if errors.Is(err, cli.ErrUnknownFlag) || errors.Is(err, cli.ErrUnknownCommand) {
						return err
					}
					return nil
				}, lines[:len(lines)-1], "every candidate scans after the words")
			})
		})
	})

	t.Run("Complete", func(t *testing.T) {
		t.Parallel()

		types := []struct {
			name     string
			giveArgs []string
			want     string
		}{
			{
				name:     "writes the candidates of the function for an option of its type",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--region", ""},
				want:     "eu-west\tEurope West\nus-east\tUS East\n" + valuesLine,
			},
			{
				name:     "writes the candidates of the function for a list of its type",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--regions", ""},
				want:     "eu-west\tEurope West\nus-east\tUS East\n" + valuesLine,
			},
			{
				name:     "writes the candidates of the function for a pointer to its type",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--home", ""},
				want:     "eu-west\tEurope West\nus-east\tUS East\n" + valuesLine,
			},
			{
				name:     "writes the candidates of the function in place of the values of an enum",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--zone", ""},
				want:     "eu-west\tEurope West\nus-east\tUS East\n" + valuesLine,
			},
			{
				name:     "writes the candidates of the function that start with the prefix",
				giveArgs: []string{completionCommand, shellZsh, wordsAfter, "--region", "us"},
				want:     "us-east\tUS East\n" + valuesLine,
			},
		}
		for _, tt := range types {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[regionGlobal, struct{}](t)
				app.Complete[region](
					func(context.Context, *cli.Run[regionGlobal, regionGlobal], string) ([]cli.Candidate, error) {
						return regionCandidates, nil
					},
				)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.NoError(t, res.err, "Execute runs the command completion")
				assert.Equal(t, res.stdout, tt.want, "Stdout receives the candidates of the function")
			})
		}

		t.Run("writes no values for a map of the type of the function", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{ Labels map[string]string }, struct{}](t)
			app.Complete[map[string]string](
				func(context.Context, *cli.Run[struct{ Labels map[string]string }, struct{ Labels map[string]string }], string) ([]cli.Candidate, error) {
					return regionCandidates, nil
				},
			)
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, "--labels", ""}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, valuesLine, "Stdout receives the directive line alone")
		})

		t.Run("passes the prefix to the function", func(t *testing.T) {
			t.Parallel()
			var got string
			app, _ := serving[regionGlobal, struct{}](t)
			app.Complete[region](
				func(_ context.Context, _ *cli.Run[regionGlobal, regionGlobal], prefix string) ([]cli.Candidate, error) {
					got = prefix
					return nil, nil
				},
			)
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, "--region=eu"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, got, "eu", "the function receives the value after the equals sign")
		})

		t.Run("passes the global options of the words to the function", func(t *testing.T) {
			t.Parallel()
			var got regionGlobal
			app, _ := serving[regionGlobal, struct{}](t)
			app.Complete[region](
				func(_ context.Context, run *cli.Run[regionGlobal, regionGlobal], _ string) ([]cli.Candidate, error) {
					got = *run.Global()
					return nil, nil
				},
			)
			files := fstest.MapFS{"srv.json": {Data: []byte(`{"format": "json"}`)}}
			args := []string{
				completionCommand,
				shellZsh,
				wordsAfter,
				"--config",
				"srv.json",
				"--zone",
				"b",
				"--region",
				"",
			}
			res := execute(t, app, args, nil, files)
			assert.NoError(t, res.err, "Execute runs the command completion")
			expect.Equal(t, got.Format, "json", "the function receives the value of the config file")
			expect.Equal(t, got.Zone, region("b"), "the function receives the value of a flag of the words")
		})

		t.Run("passes the path of the command of the words to the function", func(t *testing.T) {
			t.Parallel()
			var got string
			app, _ := serving[regionGlobal, struct{}](t)
			app.Complete[region](
				func(_ context.Context, run *cli.Run[regionGlobal, regionGlobal], _ string) ([]cli.Candidate, error) {
					got = run.Command()
					return nil, nil
				},
			)
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, command, "--region", ""}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, got, command, "the function receives the path of the command")
		})

		t.Run("returns the error of the function unchanged", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[regionGlobal, struct{}](t)
			app.Complete[region](
				func(context.Context, *cli.Run[regionGlobal, regionGlobal], string) ([]cli.Candidate, error) {
					return nil, errHandler
				},
			)
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, "--region", ""}, nil, nil)
			assert.Equal(t, res.err, errHandler, "Execute returns the error of the function", assert.ByIdentity())
		})

		t.Run("discards what the function writes to the standard output", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[regionGlobal, struct{}](t)
			app.Complete[region](
				func(_ context.Context, run *cli.Run[regionGlobal, regionGlobal], _ string) ([]cli.Candidate, error) {
					_, err := io.WriteString(run.Stdout(), "noise\n")
					return regionCandidates[:1], err
				},
			)
			res := execute(t, app, []string{completionCommand, shellBash, wordsAfter, "--region", ""}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "eu-west\n"+valuesLine, "Stdout receives the candidates alone")
		})

		t.Run("gives the function an empty standard input", func(t *testing.T) {
			t.Parallel()
			var got []byte
			app, _ := serving[regionGlobal, struct{}](t)
			app.Complete[region](
				func(_ context.Context, run *cli.Run[regionGlobal, regionGlobal], _ string) ([]cli.Candidate, error) {
					var err error
					got, err = io.ReadAll(run.Stdin())
					return nil, err
				},
			)
			var out bytes.Buffer
			env := &cli.Env{
				Stdin:  strings.NewReader("typed text"),
				Stdout: &out,
				Stderr: &out,
				Args:   []string{completionCommand, shellZsh, wordsAfter, "--region", ""},
			}
			assert.NoError(t, app.Execute(t.Context(), env), "Execute runs the command completion")
			assert.Empty(t, got, "the function reads nothing")
		})

		t.Run("leaves out a candidate whose value contains a tab or a line break", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[regionGlobal, struct{}](t)
			app.Complete[region](
				func(context.Context, *cli.Run[regionGlobal, regionGlobal], string) ([]cli.Candidate, error) {
					return []cli.Candidate{
						{Value: "a\tb"}, {Value: "c\nd"}, {Value: "e\rf"}, {Value: "ok", Text: "one\ttwo\nthree"},
					}, nil
				},
			)
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, "--region", ""}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "ok\tone two\n"+valuesLine,
				"Stdout receives the valid candidate with the first line of its text")
		})

		t.Run("replaces the function of an earlier registration", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[regionGlobal, struct{}](t)
			app.Complete[region](
				func(context.Context, *cli.Run[regionGlobal, regionGlobal], string) ([]cli.Candidate, error) {
					return []cli.Candidate{{Value: "first"}}, nil
				},
			)
			app.Complete[region](
				func(context.Context, *cli.Run[regionGlobal, regionGlobal], string) ([]cli.Candidate, error) {
					return []cli.Candidate{{Value: "last"}}, nil
				},
			)
			res := execute(t, app, []string{completionCommand, shellZsh, wordsAfter, "--region", ""}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command completion")
			assert.Equal(t, res.stdout, "last\n"+valuesLine, "Stdout receives the candidates of the last function")
		})

		t.Run("returns ErrDeclaration for a nil function", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[regionGlobal, struct{}](t)
			app.Complete[region](nil)
			res := execute(t, app, []string{command}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: the completion of cli_test.region is nil",
				"the error names the type")
		})
	})
}

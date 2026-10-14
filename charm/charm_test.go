// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package charm_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/charm"
)

// The contracts of the properties of the renderer.
const (
	helpContract  = "Help writes plain text without trailing spaces to a writer that is not a terminal"
	errorContract = "Error writes the line of cli.Plain to a writer that is not a terminal"
	debugContract = "Debug writes the table of cli.Plain to a writer that is not a terminal"
)

// errWrite is the error of a writer that fails every write.
var errWrite = errors.New("disk full")

// forced is the environment that makes colorprofile write the styles of a theme to any writer.
var forced = []string{"CLICOLOR_FORCE=1", "TERM=xterm-256color"}

// failingWriter is a writer that fails every write with errWrite.
type failingWriter struct{}

// Write returns errWrite.
func (failingWriter) Write([]byte) (int, error) {
	return 0, errWrite
}

// appHelp is the help of the app srv in the tests.
var appHelp = &cli.Help{
	App:   "srv",
	Usage: []string{"[global options]", "<command>", "[options]"},
	Text:  "Serve the store.",
	Commands: []cli.HelpCommand{
		{Name: "serve", Text: "Run the server."},
		{Name: "migrate", Text: "Migrate the store."},
	},
	Globals: []cli.HelpOption{
		{
			Name: "--config", Short: "-c", Placeholder: "file", Text: "Config file in JSON.", Default: `"srv.json"`,
			Env: "SRV_CONFIG",
		},
		{
			Name: "--flag-debug", Text: "Print the value and the source of each option before the command runs.",
			Env: "SRV_FLAG_DEBUG",
		},
		{Name: "--help", Short: "-h", Text: "Show the help."},
	},
}

// serveHelp is the help of the command serve of the app srv in the tests.
var serveHelp = &cli.Help{
	App:       "srv",
	Command:   "serve",
	Usage:     []string{"[options]", "[NAME]"},
	Text:      "Run the server of the store.",
	Arguments: []cli.HelpOption{{Name: "NAME", Text: "Name of the store.", Default: `"main"`}},
	Options: []cli.HelpOption{
		{
			Name: "--addr", Short: "-a", Placeholder: "string", Text: "Address to listen on.", Default: `":8080"`,
			Env: "SRV_SERVE_ADDR", Key: "serve.addr",
		},
		{
			Name: "--client-certificate-authority", Placeholder: "file",
			Text: "Certificate authority of the clients. The server checks the certificate of each client against it.",
			Env:  "SRV_SERVE_CLIENT_CERTIFICATE_AUTHORITY", Key: "serve.client-certificate-authority",
		},
		{Name: "--quiet", Short: "-q"},
		{
			Name: "--token", Placeholder: "string", Text: "Bearer token of the requests.", Default: `"***"`,
			Required: true, Env: "SRV_TOKEN",
		},
	},
	Globals: []cli.HelpOption{{Name: "--help", Short: "-h", Text: "Show the help."}},
}

// usageFailure is the failure of an unknown flag of the command serve.
var usageFailure = &cli.Failure{
	App: "srv", Command: "serve", Message: "unknown flag --adr", Suggestion: "--addr", Usage: true,
}

// version is the version of the app srv in the tests.
var version = &cli.Version{App: "srv", Text: "srv 1.4.2 (f70f638, 2026-10-13)"}

// debugRows are the rows of the table of --flag-debug in the tests.
var debugRows = []cli.DebugRow{
	{Option: "--config", Value: `"srv.json"`},
	{Option: "--addr", Value: `":9000"`, Source: cli.Source{Name: "serve.addr", File: "srv.json", Kind: cli.File}},
	{Option: "--token", Value: "***", Source: cli.Source{Name: "SRV_TOKEN", Kind: cli.Environment}},
}

// printable generates text of printable ASCII characters. A writer without a terminal receives such
// text unchanged.
var printable = prop.StringMatching("[ -~]{0,40}")

// word generates printable ASCII text without spaces, such as a flag, a label or a name.
var word = prop.StringMatching("[!-~]{1,12}")

// maybe generates a word or an empty text.
var maybe = prop.StringMatching("([!-~]{1,12})?")

// option generates the options of a help.
var option = prop.Composite(func(c *prop.Case) cli.HelpOption {
	return cli.HelpOption{
		Name: c.Draw(word, "name"), Short: c.Draw(maybe, "short"), Placeholder: c.Draw(maybe, "placeholder"),
		Text: c.Draw(printable, "text"), Default: c.Draw(maybe, "default"), Env: c.Draw(maybe, "env"),
		Key: c.Draw(maybe, "key"), Required: c.Draw(prop.Boolean(), "required"),
	}
})

// optionsHead is the start of the help that options returns.
const optionsHead = "USAGE\n  srv\n\nOPTIONS\n"

// renderer returns a Renderer with the default theme.
func renderer() *charm.Renderer {
	return charm.New(new(charm.DefaultTheme()))
}

// options returns the help of the app srv with the options opts.
func options(opts ...cli.HelpOption) *cli.Help {
	return &cli.Help{App: "srv", Options: opts}
}

// helpBody renders a generated help without a terminal. It checks the output for escape sequences
// and for spaces at the end of a line.
func helpBody(c *prop.Case) {
	h := &cli.Help{
		App:       c.Draw(word, "app"),
		Command:   c.Draw(maybe, "command"),
		Usage:     c.Draw(prop.List(word, prop.MaxSize(3)), "usage"),
		Text:      c.Draw(printable, "text"),
		Commands:  []cli.HelpCommand{{Name: c.Draw(word, "name"), Text: c.Draw(printable, "help")}},
		Arguments: c.Draw(prop.List(option, prop.MaxSize(2)), "arguments"),
		Options:   c.Draw(prop.List(option, prop.MaxSize(4)), "options"),
	}
	columns := "COLUMNS=" + c.Draw(prop.SampledFrom("20", "80", "300", "x"), "columns")
	var out bytes.Buffer
	assert.NoError(c, renderer().Help(&out, []string{columns}, h), "Help writes the help")
	assert.NotContains(c, out.String(), "\x1b", "the help contains no escape sequence")
	assert.NotContains(c, out.String(), " \n", "no line of the help ends in a space")
}

// errorBody renders a generated failure without a terminal with the renderer and with cli.Plain.
// Both must write the same line.
func errorBody(c *prop.Case) {
	f := &cli.Failure{
		App: c.Draw(printable, "app"), Command: c.Draw(printable, "command"), Message: c.Draw(printable, "message"),
		Suggestion: c.Draw(printable, "suggestion"), Usage: c.Draw(prop.Boolean(), "usage"),
	}
	var got, want bytes.Buffer
	assert.NoError(c, renderer().Error(&got, nil, f), "Error writes the failure")
	assert.NoError(c, cli.Plain{}.Error(&want, nil, f), "Plain writes the failure")
	assert.Equal(c, got.String(), want.String(), "Error writes the line of cli.Plain")
}

// debugBody renders generated rows without a terminal with the renderer and with cli.Plain. Both
// must write the same table.
func debugBody(c *prop.Case) {
	row := prop.Composite(func(c *prop.Case) cli.DebugRow {
		return cli.DebugRow{Option: c.Draw(printable, "option"), Value: c.Draw(printable, "value")}
	})
	rows := c.Draw(prop.List(row, prop.MaxSize(5)), "rows")
	var got, want bytes.Buffer
	assert.NoError(c, renderer().Debug(&got, nil, rows), "Debug writes the table")
	assert.NoError(c, cli.Plain{}.Debug(&want, nil, rows), "Plain writes the table")
	assert.Equal(c, got.String(), want.String(), "Debug writes the table of cli.Plain")
}

// FuzzHelp fuzzes the help that the renderer writes without a terminal.
func FuzzHelp(f *testing.F) {
	prop.Fuzz(f, helpContract, helpBody)
}

// TestCharm checks the registration of the renderer and the help, the failures and the tables that
// it writes.
func TestCharm(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("registers the renderer as the renderer of the app", func(t *testing.T) {
			t.Parallel()
			app := cli.New[struct{}]("srv", "Serve the store.")
			serve := func(context.Context, *cli.Run[struct{}, struct{}]) error { return nil }
			app.Command("serve", "Run the server.", serve)
			charm.Register(app)
			var out bytes.Buffer
			err := app.Execute(t.Context(), &cli.Env{Stdout: &out, Stderr: &out, Args: []string{"--help"}})
			assert.NoError(t, err, "Execute writes the help")
			assert.HasPrefix(t, out.String(), "Serve the store.\n\nUSAGE\n", "the help has the layout of the renderer")
		})
	})

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps a copy of the theme", func(t *testing.T) {
			t.Parallel()
			theme := charm.DefaultTheme()
			r := charm.New(&theme)
			theme.Heading = theme.Heading.Underline(true)
			var got, want bytes.Buffer
			assert.NoError(t, r.Help(&got, forced, appHelp), "Help writes the help")
			assert.NoError(t, renderer().Help(&want, forced, appHelp), "Help writes the help")
			assert.Equal(t, got.String(), want.String(), "a change of the theme after New leaves the renderer intact")
		})
	})

	t.Run("Help", func(t *testing.T) {
		t.Parallel()

		goldens := []struct {
			name        string
			giveHelp    *cli.Help
			giveEnviron []string
			want        string
		}{
			{
				name:     "writes the text, the usage and the sections of an app without styles to a buffer",
				giveHelp: appHelp,
				want:     "help/app.txt",
			},
			{
				name:     "writes the arguments and the options of a command with their notes below the text",
				giveHelp: serveHelp,
				want:     "help/command.txt",
			},
			{
				name:        "writes the styles of the theme when CLICOLOR_FORCE is true",
				giveHelp:    serveHelp,
				giveEnviron: forced,
				want:        "help/command-styled.txt",
			},
			{
				name:        "wraps the text at the width in COLUMNS",
				giveHelp:    serveHelp,
				giveEnviron: []string{"COLUMNS=60"},
				want:        "help/command-narrow.txt",
			},
		}
		for _, tt := range goldens {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var out bytes.Buffer
				assert.NoError(t, renderer().Help(&out, tt.giveEnviron, tt.giveHelp), "Help writes the help")
				golden.Match(t, tt.want, out.Bytes(), golden.ShouldUpdate())
			})
		}

		layouts := []struct {
			name        string
			giveHelp    *cli.Help
			giveEnviron []string
			want        string
		}{
			{
				name:     "starts with the usage for a help without text",
				giveHelp: &cli.Help{App: "srv"},
				want:     "USAGE\n  srv\n",
			},
			{
				name:     "writes the notes of an option without text on the line of its key",
				giveHelp: options(cli.HelpOption{Name: "--workers", Placeholder: "int", Default: "4"}),
				want:     optionsHead + "  --workers int  default 4\n",
			},
			{
				name:     "writes the text of a key of 32 columns on the line of the key",
				giveHelp: options(cli.HelpOption{Name: "--" + strings.Repeat("k", 30), Text: "Text."}),
				want:     optionsHead + "  --" + strings.Repeat("k", 30) + "  Text.\n",
			},
			{
				name: "writes the text of a key of 33 columns on the line below the key",
				giveHelp: options(
					cli.HelpOption{Name: "--" + strings.Repeat("k", 28), Text: "Text."},
					cli.HelpOption{Name: "--" + strings.Repeat("k", 31), Text: "Text."},
				),
				want: optionsHead + "  --" + strings.Repeat("k", 28) + "  Text.\n" +
					"  --" + strings.Repeat("k", 31) + "\n" + strings.Repeat(" ", 34) + "Text.\n",
			},
			{
				name:        "joins the notes in a line as wide as the text column",
				giveHelp:    options(cli.HelpOption{Name: "--a", Default: `"x"`, Env: strings.Repeat("E", 22)}),
				giveEnviron: []string{"COLUMNS=47"},
				want:        optionsHead + "  --a  default \"x\" · env " + strings.Repeat("E", 22) + "\n",
			},
			{
				name:        "starts a line of notes at a note that does not fit the text column",
				giveHelp:    options(cli.HelpOption{Name: "--a", Default: `"x"`, Env: strings.Repeat("E", 23)}),
				giveEnviron: []string{"COLUMNS=47"},
				want:        optionsHead + "  --a  default \"x\"\n       env " + strings.Repeat("E", 23) + "\n",
			},
		}
		for _, tt := range layouts {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var out bytes.Buffer
				assert.NoError(t, renderer().Help(&out, tt.giveEnviron, tt.giveHelp), "Help writes the help")
				assert.Equal(t, out.String(), tt.want, "the help has the layout of the case")
			})
		}

		widths := []struct {
			name        string
			giveEnviron []string
			want        float64
		}{
			{name: "wraps the text at 80 columns without COLUMNS", want: 80},
			{name: "wraps the text at 100 columns at most", giveEnviron: []string{"COLUMNS=300"}, want: 100},
			{name: "ignores a COLUMNS that is not a number", giveEnviron: []string{"COLUMNS=x"}, want: 80},
			{name: "uses the last valid COLUMNS", giveEnviron: []string{"COLUMNS=50", "COLUMNS=0"}, want: 50},
			{
				name:        "ignores a COLUMNS beyond the range of int",
				giveEnviron: []string{"COLUMNS=99999999999999999999"},
				want:        80,
			},
		}
		for _, tt := range widths {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var out bytes.Buffer
				h := &cli.Help{App: "srv", Text: strings.Repeat("word ", 40)}
				assert.NoError(t, renderer().Help(&out, tt.giveEnviron, h), "Help writes the help")
				widest := 0
				for line := range strings.Lines(out.String()) {
					widest = max(widest, len(strings.TrimSuffix(line, "\n")))
				}
				assert.InRange(t, widest, tt.want-5, tt.want, "the widest line of the text fills the width")
			})
		}

		t.Run("writes plain text without trailing spaces to a writer that is not a terminal", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, helpContract, helpBody)
		})

		t.Run("returns an error that wraps the error of the writer", func(t *testing.T) {
			t.Parallel()
			err := renderer().Help(failingWriter{}, nil, appHelp)
			assert.ErrorIs(t, err, errWrite, "Help returns the error of the writer")
			assert.Equal(t, err.Error(), "charm: write the help: disk full", "Help wraps the error in a message")
		})
	})

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		goldens := []struct {
			name        string
			giveFailure *cli.Failure
			giveEnviron []string
			want        string
		}{
			{
				name:        "writes the badge, the message, the suggestion and the hint of an invalid command line",
				giveFailure: usageFailure,
				giveEnviron: forced,
				want:        "error/usage-styled.txt",
			},
			{
				name:        "writes the badge and the message of the failure of a handler",
				giveFailure: &cli.Failure{App: "srv", Command: "serve", Message: "main: the store is down"},
				giveEnviron: forced,
				want:        "error/handler-styled.txt",
			},
			{
				name: "writes the hint to the help of the app for an input before the command",
				giveFailure: &cli.Failure{
					App: "srv", Message: `unknown command "serv"`, Suggestion: "serve", Usage: true,
				},
				giveEnviron: forced,
				want:        "error/app-styled.txt",
			},
			{
				name: "wraps a long message under its first line",
				giveFailure: &cli.Failure{
					App: "srv", Command: "serve",
					Message: "open /var/lib/srv/store.db: the file is locked by another process of the server",
				},
				giveEnviron: []string{"CLICOLOR_FORCE=1", "TERM=xterm-256color", "COLUMNS=50"},
				want:        "error/handler-narrow.txt",
			},
			{
				name:        "writes the badge and the hints without colors when NO_COLOR is true",
				giveFailure: usageFailure,
				giveEnviron: []string{"NO_COLOR=1", "TTY_FORCE=1", "TERM=xterm-256color"},
				want:        "error/usage-no-color.txt",
			},
		}
		for _, tt := range goldens {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var out bytes.Buffer
				assert.NoError(t, renderer().Error(&out, tt.giveEnviron, tt.giveFailure), "Error writes the failure")
				golden.Match(t, tt.want, out.Bytes(), golden.ShouldUpdate())
			})
		}

		t.Run("writes the line of cli.Plain to a writer that is not a terminal", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, errorContract, errorBody)
		})

		t.Run("returns an error that wraps the error of the writer", func(t *testing.T) {
			t.Parallel()
			err := renderer().Error(failingWriter{}, forced, usageFailure)
			assert.ErrorIs(t, err, errWrite, "Error returns the error of the writer")
			assert.Equal(t, err.Error(), "charm: write the failure: disk full", "Error wraps the error in a message")
		})

		t.Run("returns the error of cli.Plain without a terminal", func(t *testing.T) {
			t.Parallel()
			err := renderer().Error(failingWriter{}, nil, usageFailure)
			assert.ErrorIs(t, err, errWrite, "Error returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the failure: disk full", "Error returns the error of cli.Plain")
		})
	})

	t.Run("Debug", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the styles of the theme when CLICOLOR_FORCE is true", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			assert.NoError(t, renderer().Debug(&out, forced, debugRows), "Debug writes the table")
			golden.Match(t, "debug/table-styled.txt", out.Bytes(), golden.ShouldUpdate())
		})

		t.Run("writes the table of cli.Plain to a writer that is not a terminal", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, debugContract, debugBody)
		})

		t.Run("returns an error that wraps the error of the writer", func(t *testing.T) {
			t.Parallel()
			err := renderer().Debug(failingWriter{}, forced, debugRows)
			assert.ErrorIs(t, err, errWrite, "Debug returns the error of the writer")
			assert.Equal(t, err.Error(), "charm: write the output of --flag-debug: disk full",
				"Debug wraps the error in a message")
		})

		t.Run("returns the error of cli.Plain without a terminal", func(t *testing.T) {
			t.Parallel()
			err := renderer().Debug(failingWriter{}, nil, debugRows)
			assert.ErrorIs(t, err, errWrite, "Debug returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the output of --flag-debug: disk full",
				"Debug returns the error of cli.Plain")
		})
	})

	t.Run("Version", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the text of cli.Plain to a terminal", func(t *testing.T) {
			t.Parallel()
			var got, want bytes.Buffer
			assert.NoError(t, renderer().Version(&got, forced, version), "Version writes the version")
			assert.NoError(t, cli.Plain{}.Version(&want, nil, version), "Plain writes the version")
			assert.Equal(t, got.String(), want.String(), "Version writes the text of cli.Plain")
		})

		t.Run("returns the error of cli.Plain", func(t *testing.T) {
			t.Parallel()
			err := renderer().Version(failingWriter{}, forced, version)
			assert.ErrorIs(t, err, errWrite, "Version returns the error of the writer")
			assert.Equal(t, err.Error(), "cli: write the version: disk full", "Version returns the error of cli.Plain")
		})
	})
}

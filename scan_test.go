// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// The options of the app in the tests of the command line.
type (
	scanGlobal struct {
		Verbose bool `short:"v"`
	}
	scanOptions struct {
		Name  string   `short:"n"`
		Port  int      `short:"p"`
		Quiet bool     `short:"q"`
		X     string   `short:"x"`
		Args  []string `          arg:"" default:""`
	}
)

// TestScan checks how Execute reads the command line by the rules of GNU.
func TestScan(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		t.Run("sets an option to the same value through every form of its flag", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, "each form of a flag sets the option to the value", func(c *prop.Case) {
				value := c.Draw(prop.String(prop.MinSize(1), prop.MaxSize(8)), "value")
				forms := [][]string{{"--name=" + value}, {"--name", value}, {"-n", value}, {"-n" + value}}
				form := c.Draw(prop.SampledFrom(forms...), "form")
				app, got := serving[scanGlobal, scanOptions](c)
				res := execute(c, app, append([]string{command}, form...), nil, nil)
				assert.NoError(c, res.err, "Execute runs the command")
				assert.Equal(c, got.Config().Name, value, "the option has the value of the flag")
			})
		})

		tests := []struct {
			name     string
			giveArgs []string
			want     scanOptions
		}{
			{
				name:     "reads the value after the equals sign of a long flag",
				giveArgs: []string{command, "--port=8"},
				want:     scanOptions{Port: 8},
			},
			{
				name:     "reads the next argument as the value of a long flag",
				giveArgs: []string{command, "--port", "8"},
				want:     scanOptions{Port: 8},
			},
			{
				name:     "reads a next argument that starts with a dash as the value of a long flag",
				giveArgs: []string{command, "--name", "-v"},
				want:     scanOptions{Name: "-v"},
			},
			{
				name:     "sets a bool to true for its long flag without a value",
				giveArgs: []string{command, "--quiet"},
				want:     scanOptions{Quiet: true},
			},
			{
				name:     "reads the value after the equals sign of the long flag of a bool",
				giveArgs: []string{command, "--quiet=true", "x"},
				want:     scanOptions{Quiet: true, Args: []string{"x"}},
			},
			{
				name:     "reads the rest of a cluster as the value of a short flag",
				giveArgs: []string{command, "-p8"},
				want:     scanOptions{Port: 8},
			},
			{
				name:     "reads the next argument as the value of a short flag",
				giveArgs: []string{command, "-p", "8"},
				want:     scanOptions{Port: 8},
			},
			{
				name:     "sets a bool of a cluster before a short flag with a value",
				giveArgs: []string{command, "-qp8"},
				want:     scanOptions{Port: 8, Quiet: true},
			},
			{
				name:     "reads a short flag whose letter is the name of its long flag",
				giveArgs: []string{command, "-x", "8"},
				want:     scanOptions{X: "8"},
			},
			{
				name:     "reads every argument after -- as a positional argument",
				giveArgs: []string{command, "--", "--port", "-q"},
				want:     scanOptions{Args: []string{"--port", "-q"}},
			},
			{
				name:     "reads a lone dash as a positional argument",
				giveArgs: []string{command, "-"},
				want:     scanOptions{Args: []string{"-"}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[scanGlobal, scanOptions](t)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.NoError(t, res.err, "Execute runs the command")
				assert.Equal(t, *got.Config(), tt.want, "the run reads the command line", assert.EquateEmpty())
			})
		}

		globals := []struct {
			name     string
			giveArgs []string
		}{
			{name: "reads a global flag before the command", giveArgs: []string{"-v", command}},
			{name: "reads a global flag after the command", giveArgs: []string{command, "--verbose"}},
			{name: "reads a global short flag in a cluster of the command", giveArgs: []string{command, "-qv"}},
		}
		for _, tt := range globals {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[scanGlobal, scanOptions](t)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.NoError(t, res.err, "Execute runs the command")
				assert.True(t, got.Global().Verbose, "the run sets the global option")
			})
		}

		paths := []struct {
			name     string
			giveArgs []string
		}{
			{
				name:     "reads a flag of a command before the name of its subcommand",
				giveArgs: []string{parent, "--dsn", "pg", subcommand},
			},
			{
				name:     "reads a flag of a command after the name of its subcommand",
				giveArgs: []string{parent, subcommand, "--dsn", "pg"},
			},
			{
				name:     "reads a short flag of a command in a cluster after the name of its subcommand",
				giveArgs: []string{parent, subcommand, "-vd", "pg"},
			},
		}
		for _, tt := range paths {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _, migrate := nesting[scanGlobal, Store, stepsOptions](t)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.NoError(t, res.err, "Execute runs the subcommand")
				assert.Equal(t, migrate.Config().DSN, "pg", "the subcommand reads the option of its command")
			})
		}

		t.Run("reads a global flag between the names of a path", func(t *testing.T) {
			t.Parallel()
			app, _, migrate := nesting[scanGlobal, Store, stepsOptions](t)
			res := execute(t, app, []string{parent, "-v", subcommand}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the subcommand")
			assert.True(t, migrate.Global().Verbose, "the run sets the global option")
		})

		t.Run("reads h in a cluster as the flag of the help", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[scanGlobal, scanOptions](t)
			res := execute(t, app, []string{command, "-qh"}, nil, nil)
			assert.NoError(t, res.err, "Execute returns nil")
			assert.HasPrefix(t, res.stdout, "Usage: srv serve [options]", "Stdout receives the help of the command")
		})

		failures := []struct {
			name     string
			giveArgs []string
			wantErr  error
			want     string
		}{
			{
				name:     "returns ErrMissing for a long flag without its value at the end",
				giveArgs: []string{command, "--port"},
				wantErr:  cli.ErrMissing,
				want:     "cli: serve: flag --port needs a value",
			},
			{
				name:     "returns ErrMissing for a short flag without its value at the end",
				giveArgs: []string{command, "-p"},
				wantErr:  cli.ErrMissing,
				want:     "cli: serve: flag --port needs a value",
			},
			{
				name:     "returns ErrUnknownFlag for a flag of the command before the command",
				giveArgs: []string{"--port", "8", command},
				wantErr:  cli.ErrUnknownFlag,
				want:     "cli: unknown flag --port",
			},
			{
				name:     "returns ErrUnknownFlag for a cluster whose text is the name of a long flag",
				giveArgs: []string{command, "-port", "8"},
				wantErr:  cli.ErrUnknownFlag,
				want:     "cli: serve: unknown flag -port (did you mean --port?)",
			},
			{
				name:     "returns ErrUnknownFlag for a cluster whose text is the name of a positional argument",
				giveArgs: []string{command, "-args"},
				wantErr:  cli.ErrUnknownFlag,
				want:     "cli: serve: unknown flag -a",
			},
			{
				name:     "returns ErrUnknownFlag for a long flag with the name of a positional argument",
				giveArgs: []string{command, "--args", "x"},
				wantErr:  cli.ErrUnknownFlag,
				want:     "cli: serve: unknown flag --args",
			},
			{
				name:     "returns ErrUnknownFlag for a NUL character in a cluster",
				giveArgs: []string{command, "-\x00"},
				wantErr:  cli.ErrUnknownFlag,
				want:     "cli: serve: unknown flag -\x00",
			},
			{
				name:     "returns ErrUnknownFlag for an unknown short flag",
				giveArgs: []string{command, "-z"},
				wantErr:  cli.ErrUnknownFlag,
				want:     "cli: serve: unknown flag -z",
			},
			{
				name:     "returns ErrUnknownFlag for an unknown short flag after a bool of its cluster",
				giveArgs: []string{command, "-qz"},
				wantErr:  cli.ErrUnknownFlag,
				want:     "cli: serve: unknown flag -z",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[scanGlobal, scanOptions](t)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.ErrorIs(t, res.err, tt.wantErr, "Execute returns the error of the command line")
				assert.Equal(t, res.err.Error(), tt.want, "the error states the flag")
			})
		}

		nested := []struct {
			name     string
			giveArgs []string
			wantErr  error
			want     string
		}{
			{
				name:     "returns ErrUnknownFlag for a flag of a subcommand before the name of the subcommand",
				giveArgs: []string{parent, "--steps", "3", subcommand},
				wantErr:  cli.ErrUnknownFlag,
				want:     "cli: db: unknown flag --steps",
			},
			{
				name:     "returns ErrUnknownCommand for an unknown subcommand",
				giveArgs: []string{parent, "migrat"},
				wantErr:  cli.ErrUnknownCommand,
				want:     `cli: db: unknown command "migrat" (did you mean migrate?)`,
			},
		}
		for _, tt := range nested {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _, _ := nesting[scanGlobal, Store, stepsOptions](t)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.ErrorIs(t, res.err, tt.wantErr, "Execute returns the error of the command line")
				assert.Equal(t, res.err.Error(), tt.want, "the error states the path and the input")
			})
		}

		t.Run("returns ErrUnknownFlag for the flag of the version with a value", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[scanGlobal, scanOptions](t)
			app.Version(versionText)
			res := execute(t, app, []string{"--version=true"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrUnknownFlag, "Execute returns ErrUnknownFlag")
			assert.Equal(t, res.err.Error(), "cli: unknown flag --version", "the error states the flag")
		})

		t.Run("states the flag of an unknown flag in the *Error", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[scanGlobal, scanOptions](t)
			res := execute(t, app, []string{command, "--prot", "8"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error")
			assert.Equal(t, e.Input, "--prot", "the *Error states the flag")
		})
	})
}

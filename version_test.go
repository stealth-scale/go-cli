// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"context"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"

	"go.stealthscale.io/cli"
)

// itemOptions are the options of a command with a required positional argument.
type itemOptions struct {
	Item string `arg:""`
}

// TestVersion checks the version that App.Version registers and the flag --version.
func TestVersion(t *testing.T) {
	t.Parallel()

	t.Run("Version", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveArgs []string
			giveVars map[string]string
		}{
			{
				name:     "writes the text to Stdout for a command line without a command",
				giveArgs: []string{"--version"},
			},
			{
				name:     "writes the text for a command without its required argument",
				giveArgs: []string{command, "--version"},
			},
			{
				name:     "writes the text for an unexpected argument",
				giveArgs: []string{command, "a", "b", "--version"},
			},
			{
				name:     "writes the text for an invalid environment variable",
				giveArgs: []string{"--version", command},
				giveVars: map[string]string{"SRV_VERBOSE": "maybe"},
			},
			{
				name:     "writes the text for a config file that does not exist",
				giveArgs: []string{"-c", "srv.json", "--version"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[appGlobal, itemOptions](t)
				app.Version(versionText)
				res := execute(t, app, tt.giveArgs, tt.giveVars, nil)
				assert.NoError(t, res.err, "Execute returns nil")
				expect.Equal(t, res.stdout, versionText+"\n", "Stdout receives the text followed by a newline")
				expect.Empty(t, res.stderr, "Stderr receives nothing")
			})
		}

		t.Run("runs no handler for the flag --version", func(t *testing.T) {
			t.Parallel()
			ran := false
			app := cli.New[appGlobal](appName, appHelp)
			app.Command(command, commandHelp, func(context.Context, *cli.Run[appGlobal, appOptions]) error {
				ran = true
				return nil
			})
			app.Version(versionText)
			res := execute(t, app, []string{command, "--version"}, nil, nil)
			assert.NoError(t, res.err, "Execute returns nil")
			assert.False(t, ran, "Execute runs no handler")
		})

		t.Run("writes nothing for a command line without the flag --version", func(t *testing.T) {
			t.Parallel()
			app, got := serving[appGlobal, itemOptions](t)
			app.Version(versionText)
			res := execute(t, app, []string{command, "bolts"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			expect.Equal(t, got.Config().Item, "bolts", "the command receives its argument")
			expect.Empty(t, res.stdout, "Stdout receives nothing")
		})

		t.Run("writes the help for the flag of the help beside the flag --version", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			app.Version(versionText)
			res := execute(t, app, []string{"--version", "--help"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.HasPrefix(t, res.stdout, "Usage: srv <command> [options]\n", "Stdout receives the help")
		})

		t.Run("returns the error of an unknown flag beside the flag --version", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			app.Version(versionText)
			res := execute(t, app, []string{"--version", "--bogus"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrUnknownFlag, "Execute returns ErrUnknownFlag")
			assert.Empty(t, res.stdout, "Stdout receives nothing")
		})

		t.Run("uses the text of the last call", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			app.Version("srv 1.0.0")
			app.Version(versionText)
			res := execute(t, app, []string{"--version"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the version")
			assert.Equal(t, res.stdout, versionText+"\n", "Stdout receives the text of the last call")
		})

		t.Run("returns ErrUnknownFlag for the flag --version of an app without a version", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			res := execute(t, app, []string{"--version"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrUnknownFlag, "Execute returns ErrUnknownFlag")
			assert.Equal(t, res.err.Error(), "cli: unknown flag --version", "the error states the flag")
		})

		t.Run("returns ErrDeclaration for an empty text", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[appGlobal, appOptions](t)
			app.Version("")
			res := execute(t, app, []string{"--version"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: the version of the app srv is empty",
				"the error states the app")
		})
	})
}

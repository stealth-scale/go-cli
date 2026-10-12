// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"

	"go.stealthscale.io/cli"
)

// The options of the app in the tests of options.
type (
	optionGlobal struct {
		Config cli.ConfigFile
	}
	optionOptions struct {
		Limits  map[string]int
		Workers *int
		Mode    string `enum:"dev,prod"`
		Codec   string `enum:"none,gzip,zstd"`
		Stage   string `enum:"prod"`
		Token   string `required:""`
		Tags    []string
		Levels  []string `enum:"low,high"`
		Port    int
	}
	argumentOptions struct {
		DryRun string `arg:""`
	}
)

// TestOption checks how an option takes its values: the shapes, the enum, and the required options.
func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		workers := 4
		tests := []struct {
			name     string
			giveArgs []string
			giveVars map[string]string
			want     optionOptions
		}{
			{
				name:     "splits the text of a list at each comma",
				giveArgs: []string{command, "--token", "t", "--tags", "a,b"},
				want:     optionOptions{Token: "t", Tags: []string{"a", "b"}},
			},
			{
				name:     "appends the values of a repeated flag of a list",
				giveArgs: []string{command, "--token", "t", "--tags", "a,b", "--tags", "c"},
				want:     optionOptions{Token: "t", Tags: []string{"a", "b", "c"}},
			},
			{
				name:     "replaces the list of the environment with the list of a flag",
				giveArgs: []string{command, "--token", "t", "--tags", "c"},
				giveVars: map[string]string{"SRV_SERVE_TAGS": "a,b"},
				want:     optionOptions{Token: "t", Tags: []string{"c"}},
			},
			{
				name:     "splits the text of a map into entries of the form key=value",
				giveArgs: []string{command, "--token", "t", "--limits", "cpu=2,mem=8"},
				want:     optionOptions{Token: "t", Limits: map[string]int{"cpu": 2, "mem": 8}},
			},
			{
				name:     "adds the entries of a repeated flag of a map to the earlier entries",
				giveArgs: []string{command, "--token", "t", "--limits", "cpu=2", "--limits", "mem=8"},
				want:     optionOptions{Token: "t", Limits: map[string]int{"cpu": 2, "mem": 8}},
			},
			{
				name:     "replaces the entry of a key with the entry of a later flag of the map",
				giveArgs: []string{command, "--token", "t", "--limits", "cpu=2,mem=8", "--limits", "cpu=4"},
				want:     optionOptions{Token: "t", Limits: map[string]int{"cpu": 4, "mem": 8}},
			},
			{
				name:     "sets a pointer to the value of a source",
				giveArgs: []string{command, "--token", "t", "--workers", "4"},
				want:     optionOptions{Token: "t", Workers: &workers},
			},
			{
				name:     "leaves a pointer nil without a source",
				giveArgs: []string{command, "--token", "t"},
				want:     optionOptions{Token: "t"},
			},
			{
				name:     "sets an option of an enum to a value of the enum",
				giveArgs: []string{command, "--token", "t", "--mode", "prod"},
				want:     optionOptions{Token: "t", Mode: "prod"},
			},
			{
				name:     "sets each element of a list of an enum to a value of the enum",
				giveArgs: []string{command, "--token", "t", "--levels", "low,high"},
				want:     optionOptions{Token: "t", Levels: []string{"low", "high"}},
			},
			{
				name:     "sets a required option from the environment",
				giveArgs: []string{command},
				giveVars: map[string]string{"SRV_SERVE_TOKEN": "t"},
				want:     optionOptions{Token: "t"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[optionGlobal, optionOptions](t)
				res := execute(t, app, tt.giveArgs, tt.giveVars, nil)
				assert.NoError(t, res.err, "Execute runs the command")
				assert.Equal(t, *got.Config(), tt.want, "the run sets the options")
			})
		}

		invalid := []struct {
			name     string
			giveArgs []string
			giveData string
			wantErr  error
			want     string
		}{
			{
				name:     "returns ErrInvalidValue for an entry of a map without an equals sign",
				giveArgs: []string{command, "--token", "t", "--limits", "cpu"},
				wantErr:  cli.ErrInvalidValue,
				want:     `invalid --limits "cpu" from flag --limits: want key=value`,
			},
			{
				name:     "returns ErrInvalidValue for a value of a map that does not convert",
				giveArgs: []string{command, "--token", "t", "--limits", "cpu=two"},
				wantErr:  cli.ErrInvalidValue,
				want:     `invalid --limits "two" from flag --limits: invalid syntax`,
			},
			{
				name:     "returns ErrInvalidValue for a value that an enum of two values excludes",
				giveArgs: []string{command, "--token", "t", "--mode", "test"},
				wantErr:  cli.ErrInvalidValue,
				want:     `invalid --mode "test" from flag --mode: want dev or prod`,
			},
			{
				name:     "returns ErrInvalidValue for a value that an enum of three values excludes",
				giveArgs: []string{command, "--token", "t", "--codec", "rar"},
				wantErr:  cli.ErrInvalidValue,
				want:     `invalid --codec "rar" from flag --codec: want none, gzip or zstd`,
			},
			{
				name:     "returns ErrInvalidValue for a value that an enum of one value excludes",
				giveArgs: []string{command, "--token", "t", "--stage", "dev"},
				wantErr:  cli.ErrInvalidValue,
				want:     `invalid --stage "dev" from flag --stage: want prod`,
			},
			{
				name:     "returns ErrInvalidValue for an element of a list that the enum excludes",
				giveArgs: []string{command, "--token", "t", "--levels", "low,mid"},
				wantErr:  cli.ErrInvalidValue,
				want:     `invalid --levels "mid" from flag --levels: want low or high`,
			},
			{
				name:     "returns ErrInvalidValue for two values of an option that takes one",
				giveArgs: []string{"--config", configPath, command, "--token", "t"},
				giveData: `{"serve": {"port": [80, 81]}}`,
				wantErr:  cli.ErrInvalidValue,
				want:     `invalid --port "80,81" from key serve.port of srv.json: the option takes one value`,
			},
			{
				name:     "returns ErrMissing for a required option that no source sets",
				giveArgs: []string{command},
				wantErr:  cli.ErrMissing,
				want:     "missing --token",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[optionGlobal, optionOptions](t)
				res := execute(t, app, tt.giveArgs, nil, fstest.MapFS{configPath: {Data: []byte(tt.giveData)}})
				assert.ErrorIs(t, res.err, tt.wantErr, "Execute returns the error of the option")
				assert.Equal(t, res.err.Error(), "cli: serve: "+tt.want, "the error states the option and the reason")
			})
		}

		t.Run("returns the reason of the conversion for a value of an enum that does not convert", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[optionGlobal, struct {
				Level int `enum:"1,2"`
			}](t)
			res := execute(t, app, []string{command, "--level", "x"}, nil, nil)
			assert.Equal(t, res.err.Error(), `cli: serve: invalid --level "x" from flag --level: invalid syntax`,
				"the error states the reason of the conversion")
		})

		t.Run("returns ErrInvalidValue for a key of a map that does not convert", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[optionGlobal, struct{ Ports map[int]string }](t)
			res := execute(t, app, []string{command, "--ports", "http=80"}, nil, nil)
			assert.Equal(t, res.err.Error(), `cli: serve: invalid --ports "http" from flag --ports: invalid syntax`,
				"the error states the key")
		})

		t.Run("states the option, the value and the source of an invalid value in the *Error", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[optionGlobal, optionOptions](t)
			res := execute(t, app, []string{command, "--port", "x"}, map[string]string{"SRV_SERVE_TOKEN": "t"}, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error")
			expect.Equal(t, e.Command, command, "the *Error states the command")
			expect.Equal(t, e.Input, "--port", "the *Error states the flag")
			expect.Equal(t, e.Value, "x", "the *Error states the value")
			expect.Equal(t, e.Source, cli.Source{Kind: cli.Flag, Name: "--port"}, "the *Error states the source")
		})

		t.Run("names a positional argument in upper case with underscores", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[optionGlobal, argumentOptions](t)
			res := execute(t, app, []string{command}, nil, nil)
			assert.Equal(t, res.err.Error(), "cli: serve: missing argument DRY_RUN", "the error names the argument")
		})
	})
}

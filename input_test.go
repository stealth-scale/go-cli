// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// The options of the commands in the tests of input files.
type (
	docOptions struct {
		Doc cli.InputFile `arg:"" default:"-"`
	}
	inputOptions struct {
		Input  cli.InputFile
		Inputs []cli.InputFile
	}
)

// TestInputFile checks the input files that a handler opens, and their help.
func TestInputFile(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			giveArgs  []string
			giveStdin string
			want      string
		}{
			{
				name:     "passes the path of the command line to the handler",
				giveArgs: []string{command, "doc.json"},
				want:     `{"name": "store"}`,
			},
			{
				name:      "passes the standard input to the handler for the path -",
				giveArgs:  []string{command, "-"},
				giveStdin: "piped",
				want:      "piped",
			},
			{
				name:      "passes the standard input to the handler for the default -",
				giveArgs:  []string{command},
				giveStdin: "piped",
				want:      "piped",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got string
				app := cli.New[struct{}](appName, appHelp)
				app.Command(command, commandHelp, func(_ context.Context, run *cli.Run[struct{}, docOptions]) error {
					f, err := run.Open(run.Config().Doc)
					if err != nil {
						return err
					}
					data, err := io.ReadAll(f)
					got = string(data)
					return errors.Join(err, f.Close())
				})
				var out bytes.Buffer
				files := fstest.MapFS{"doc.json": {Data: []byte(`{"name": "store"}`)}}
				err := app.Execute(t.Context(), &cli.Env{
					Open:   files.Open,
					Stdin:  strings.NewReader(tt.giveStdin),
					Stdout: &out,
					Stderr: &out,
					Args:   tt.giveArgs,
				})
				assert.NoError(t, err, "Execute runs the command")
				assert.Equal(t, got, tt.want, "the handler reads the input")
			})
		}

		t.Run("shows the value of an input file as file in the help", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, inputOptions](t)
			res := execute(t, app, []string{command, "-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.Contains(t, res.stdout, "--input file ", "the help shows the value as file")
		})

		t.Run("shows the values of a list of input files as file,... in the help", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, inputOptions](t)
			res := execute(t, app, []string{command, "-h"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.Contains(t, res.stdout, "--inputs file,...", "the help shows the values as file,...")
		})
	})
}

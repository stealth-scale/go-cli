// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// envOptions are the options of the command serve in the tests of Env.
type envOptions struct {
	Addr string `default:":8080"`
}

// TestEnv checks the Env of the process, the variables of a map and the Env that a run requires.
func TestEnv(t *testing.T) {
	t.Parallel()

	t.Run("OS", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the arguments of the process after the name of the program", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, cli.OS().Args, os.Args[1:], "OS returns os.Args without the name of the program")
		})

		t.Run("returns the functions of package os", func(t *testing.T) {
			t.Parallel()
			env := cli.OS()
			assert.Equal(t, env.Lookup, os.LookupEnv, "OS looks up the variables with os.LookupEnv")
			assert.Equal(t, env.Environ, os.Environ, "OS lists the variables with os.Environ")
			assert.Equal(t, env.ReadFile, os.ReadFile, "OS reads the files with os.ReadFile")
		})

		t.Run("returns the standard streams of package os", func(t *testing.T) {
			t.Parallel()
			env := cli.OS()
			assert.Equal(t, env.Stdin, io.Reader(os.Stdin), "OS reads os.Stdin", assert.ByIdentity())
			assert.Equal(t, env.Stdout, io.Writer(os.Stdout), "OS writes os.Stdout", assert.ByIdentity())
			assert.Equal(t, env.Stderr, io.Writer(os.Stderr), "OS writes os.Stderr", assert.ByIdentity())
		})
	})

	t.Run("Vars", func(t *testing.T) {
		t.Parallel()

		t.Run("reports the value of a variable of the map", func(t *testing.T) {
			t.Parallel()
			v, ok := cli.Vars(map[string]string{"SRV_ADDR": ":80"})("SRV_ADDR")
			assert.True(t, ok, "Vars reports the variable")
			assert.Equal(t, v, ":80", "Vars returns the value of the variable")
		})

		t.Run("reports false for a variable that the map lacks", func(t *testing.T) {
			t.Parallel()
			_, ok := cli.Vars(map[string]string{"SRV_ADDR": ":80"})("SRV_PORT")
			assert.False(t, ok, "Vars reports no variable")
		})

		t.Run("reports the variables of the map at the call", func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"SRV_ADDR": ":80"}
			lookup := cli.Vars(vars)
			vars["SRV_ADDR"] = ":81"
			v, _ := lookup("SRV_ADDR")
			assert.Equal(t, v, ":80", "Vars reports the value at the call of Vars")
		})
	})

	t.Run("Environ", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the variables of the map as pairs in the order of their names", func(t *testing.T) {
			t.Parallel()
			got := cli.Environ(map[string]string{"TERM": "xterm", "NO_COLOR": "1", "EMPTY": ""})()
			assert.Equal(t, got, []string{"EMPTY=", "NO_COLOR=1", "TERM=xterm"}, "Environ returns KEY=value pairs")
		})

		t.Run("returns the variables of the map at the call", func(t *testing.T) {
			t.Parallel()
			vars := map[string]string{"TERM": "xterm"}
			environ := cli.Environ(vars)
			vars["TERM"] = "dumb"
			assert.Equal(t, environ(), []string{"TERM=xterm"}, "Environ returns the value at the call of Environ")
		})

		t.Run("returns a list of its own to each caller", func(t *testing.T) {
			t.Parallel()
			environ := cli.Environ(map[string]string{"TERM": "xterm"})
			environ()[0] = "TERM=dumb"
			assert.Equal(t, environ(), []string{"TERM=xterm"}, "a change of one result leaves the next result intact")
		})
	})

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		var stdout, stderr bytes.Buffer
		tests := []struct {
			name string
			give *cli.Env
			want string
		}{
			{
				name: "returns ErrEnv for a nil Env",
				give: nil,
				want: "cli: incomplete env: the Env is nil",
			},
			{
				name: "returns ErrEnv for an Env without Stdout",
				give: &cli.Env{Stderr: &stderr},
				want: "cli: incomplete env: the Env has no Stdout",
			},
			{
				name: "returns ErrEnv for an Env without Stderr",
				give: &cli.Env{Stdout: &stdout},
				want: "cli: incomplete env: the Env has no Stderr",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[struct{}, envOptions](t)
				err := app.Execute(t.Context(), tt.give)
				assert.ErrorIs(t, err, cli.ErrEnv, "Execute returns ErrEnv")
				assert.Equal(t, err.Error(), tt.want, "the error states what the Env lacks")
			})
		}

		t.Run("reports no environment variable for an Env without Lookup", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, envOptions](t)
			var out bytes.Buffer
			err := app.Execute(t.Context(), &cli.Env{Stdout: &out, Stderr: &out, Args: []string{command}})
			assert.NoError(t, err, "Execute runs the command")
			assert.Equal(t, got.Config().Addr, ":8080", "the option keeps its default")
		})
	})
}

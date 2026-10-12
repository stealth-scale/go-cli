// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package toml_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"testing/fstest"
	"time"

	burntsushi "github.com/BurntSushi/toml"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/toml"
)

// decodeContract is the contract of the property that FuzzDecode fuzzes and TestTOML checks.
const decodeContract = "Decode returns a tree or an error for any data"

// options are the options of the command serve of the app that reads a TOML file.
type options struct {
	Since time.Time
	Addr  string
	Tags  []string
}

// decodeBody decodes arbitrary data with Decode, and checks that it returns no tree with its error.
func decodeBody(c *prop.Case) {
	tree, err := toml.Decode(c.Draw(prop.Bytes(prop.MaxSize(64)), "data"))
	if err != nil {
		assert.Nil(c, tree, "Decode returns no tree with an error")
	}
}

// FuzzDecode fuzzes Decode with arbitrary data.
func FuzzDecode(f *testing.F) {
	prop.Fuzz(f, decodeContract, decodeBody)
}

// TestTOML checks the registration of the TOML format and the trees that Decode returns.
func TestTOML(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		t.Run("registers Decode for the extension .toml", func(t *testing.T) {
			t.Parallel()
			var got options
			app := cli.New[struct{ Config cli.ConfigFile }]("srv", "Serve the store.")
			toml.Register(app)
			app.Command(
				"serve",
				"",
				func(_ context.Context, run *cli.Run[struct{ Config cli.ConfigFile }, options]) error {
					got = *run.Config()
					return nil
				},
			)
			var out bytes.Buffer
			files := fstest.MapFS{"srv.toml": {Data: []byte(
				"[serve]\naddr = \":80\"\ntags = [\"a\", \"b\"]\nsince = 2026-10-12T09:30:00Z\n",
			)}}
			env := &cli.Env{
				Open:   files.Open,
				Stdout: &out,
				Stderr: &out,
				Args:   []string{"--config", "srv.toml", "serve"},
			}
			assert.NoError(t, app.Execute(t.Context(), env), "Execute reads the TOML file")
			assert.Equal(t, got, options{
				Since: time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC),
				Addr:  ":80",
				Tags:  []string{"a", "b"},
			}, "the run reads the keys of the file")
		})
	})

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a tree or an error for any data", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, decodeContract, decodeBody)
		})

		t.Run("returns each table as a map and each array as a slice", func(t *testing.T) {
			t.Parallel()
			got, err := toml.Decode([]byte("[serve.tls]\ncert = \"srv.pem\"\nports = [80, 81]\n"))
			assert.NoError(t, err, "Decode decodes the document")
			assert.Equal(t, got, map[string]any{
				"serve": map[string]any{"tls": map[string]any{"cert": "srv.pem", "ports": []any{int64(80), int64(81)}}},
			}, "Decode returns the tree of the document")
		})

		t.Run("returns the error of github.com/BurntSushi/toml for data that is not TOML", func(t *testing.T) {
			t.Parallel()
			data := []byte("serve = \n")
			_, err := toml.Decode(data)
			var tree map[string]any
			want := burntsushi.Unmarshal(data, &tree)
			assert.Equal(t, errors.Unwrap(err), want, "Decode wraps the error of the parser")
			assert.Equal(t, err.Error(), "toml: decode: "+want.Error(), "the error states the error of the parser")
		})
	})
}

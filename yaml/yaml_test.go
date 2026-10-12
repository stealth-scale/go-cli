// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package yaml_test

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
	yamlv3 "go.yaml.in/yaml/v3"

	"go.stealthscale.io/cli"
	"go.stealthscale.io/cli/yaml"
)

// decodeContract is the contract of the property that FuzzDecode fuzzes and TestYAML checks.
const decodeContract = "Decode returns a tree whose objects are maps of strings, or an error, for any data"

// options are the options of the command serve of the app that reads a YAML file.
type options struct {
	Labels map[int]string
	Addr   string
}

// decodeBody decodes arbitrary data with Decode, and checks that each object of a tree without an
// error is a map[string]any.
func decodeBody(c *prop.Case) {
	tree, err := yaml.Decode(c.Draw(prop.Bytes(prop.MaxSize(64)), "data"))
	if err == nil {
		assertObjects(c, tree)
	}
}

// assertObjects checks that each object of v, at any depth, is a map[string]any.
func assertObjects(tb assert.TB, v any) {
	tb.Helper()
	switch v := v.(type) {
	case map[string]any:
		for _, e := range v {
			assertObjects(tb, e)
		}
	case []any:
		for _, e := range v {
			assertObjects(tb, e)
		}
	default:
		_, isMap := v.(map[any]any)
		assert.False(tb, isMap, "no object of the tree is a map[any]any")
	}
}

// FuzzDecode fuzzes Decode with arbitrary data.
func FuzzDecode(f *testing.F) {
	prop.Fuzz(f, decodeContract, decodeBody)
}

// TestYAML checks the registration of the YAML format and the trees that Decode returns.
func TestYAML(t *testing.T) {
	t.Parallel()

	t.Run("Register", func(t *testing.T) {
		t.Parallel()

		for _, path := range []string{"srv.yaml", "srv.yml"} {
			t.Run("registers Decode for the extension of "+path, func(t *testing.T) {
				t.Parallel()
				var got options
				app := cli.New[struct{ Config cli.ConfigFile }]("srv", "Serve the store.")
				yaml.Register(app)
				app.Command(
					"serve",
					"",
					func(_ context.Context, run *cli.Run[struct{ Config cli.ConfigFile }, options]) error {
						got = *run.Config()
						return nil
					},
				)
				var out bytes.Buffer
				files := fstest.MapFS{path: {Data: []byte("serve:\n  addr: ':80'\n  labels: {1: web}\n")}}
				env := &cli.Env{
					ReadFile: files.ReadFile,
					Stdout:   &out,
					Stderr:   &out,
					Args:     []string{"--config", path, "serve"},
				}
				assert.NoError(t, app.Execute(t.Context(), env), "Execute reads the YAML file")
				assert.Equal(
					t,
					got,
					options{Labels: map[int]string{1: "web"}, Addr: ":80"},
					"the run reads the keys of the file",
				)
			})
		}
	})

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a tree whose objects are maps of strings, or an error, for any data", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, decodeContract, decodeBody)
		})

		t.Run("returns the text of each key of a map with keys other than strings", func(t *testing.T) {
			t.Parallel()
			prop.RoundTrip(t,
				func(m map[int]string) ([]byte, error) { return yamlv3.Marshal(map[string]any{"m": m}) },
				func(data []byte) (map[int]string, error) {
					tree, err := yaml.Decode(data)
					object, _ := tree["m"].(map[string]any)
					m := make(map[int]string, len(object))
					for k, v := range object {
						n, aerr := strconv.Atoi(k)
						m[n], _ = v.(string)
						err = errors.Join(err, aerr)
					}
					return m, err
				},
				"Decode returns each integer key as its text", assert.EquateEmpty())
		})

		tests := []struct {
			name string
			give string
			want map[string]any
		}{
			{
				name: "returns each mapping as a map of strings",
				give: "serve:\n  tls:\n    cert: srv.pem\n",
				want: map[string]any{"serve": map[string]any{"tls": map[string]any{"cert": "srv.pem"}}},
			},
			{
				name: "returns the text of each key of a mapping with keys other than strings",
				give: "labels: {1: web, true: yes}\n",
				want: map[string]any{"labels": map[string]any{"1": "web", "true": "yes"}},
			},
			{
				name: "returns the mappings inside a sequence as maps of strings",
				give: "items:\n  - {1: web}\n",
				want: map[string]any{"items": []any{map[string]any{"1": "web"}}},
			},
			{
				name: "returns the Go value of each scalar",
				give: "port: 80\nratio: 0.5\nverbose: true\nname: srv\n",
				want: map[string]any{"port": 80, "ratio": 0.5, "verbose": true, "name": "srv"},
			},
			{
				name: "returns no tree for an empty document",
				give: "",
				want: nil,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := yaml.Decode([]byte(tt.give))
				assert.NoError(t, err, "Decode decodes the document")
				assert.Equal(t, got, tt.want, "Decode returns the tree of the document")
			})
		}

		t.Run("returns the error of go.yaml.in/yaml/v3 for a document that is not a mapping", func(t *testing.T) {
			t.Parallel()
			data := []byte("- serve\n")
			_, err := yaml.Decode(data)
			var tree map[string]any
			want := yamlv3.Unmarshal(data, &tree)
			assert.Equal(t, errors.Unwrap(err), want, "Decode wraps the error of the parser")
			assert.Equal(t, err.Error(), "yaml: decode: "+want.Error(), "the error states the error of the parser")
		})
	})
}

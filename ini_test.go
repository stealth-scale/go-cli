// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// iniContract is the contract of the property that FuzzINI fuzzes and TestINI checks.
const iniContract = "INI returns a tree of strings and maps, or an error, for any data"

// iniBody decodes arbitrary data with INI, and checks that a tree without an error has only strings
// and maps as values.
func iniBody(c *prop.Case) {
	tree, err := cli.INI(c.Draw(prop.Bytes(prop.MaxSize(64)), "data"))
	if err == nil {
		assertTree(c, tree)
	}
}

// assertTree checks that each value of tree is a string or a tree.
func assertTree(tb assert.TB, tree map[string]any) {
	tb.Helper()
	for _, v := range tree {
		if sub, ok := v.(map[string]any); ok {
			assertTree(tb, sub)
			continue
		}
		_, ok := v.(string)
		assert.True(tb, ok, "each value of the tree is a string or a map")
	}
}

// FuzzINI fuzzes INI with arbitrary data.
func FuzzINI(f *testing.F) {
	prop.Fuzz(f, iniContract, iniBody)
}

// TestINI checks the trees that INI decodes and the data that it refuses.
func TestINI(t *testing.T) {
	t.Parallel()

	t.Run("INI", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a tree of strings and maps, or an error, for any data", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, iniContract, iniBody)
		})

		t.Run("returns the keys and the sections of every rendered tree", func(t *testing.T) {
			t.Parallel()
			name := prop.String(prop.Alphabet("ab"), prop.MinSize(1), prop.MaxSize(3))
			section := prop.String(prop.Alphabet("xy"), prop.MinSize(1), prop.MaxSize(3))
			values := prop.Dict(name, prop.String(prop.MaxSize(8)), prop.MaxSize(3))
			prop.ForAll(t, "INI decodes the keys of each section that a renderer writes", func(c *prop.Case) {
				top := c.Draw(values, "top")
				sections := c.Draw(prop.Dict(section, values, prop.MaxSize(3)), "sections")
				var data []byte
				want := map[string]any{}
				for _, k := range slices.Sorted(maps.Keys(top)) {
					data = fmt.Appendf(data, "%s = %q\n", k, top[k])
					want[k] = top[k]
				}
				for _, s := range slices.Sorted(maps.Keys(sections)) {
					data = fmt.Appendf(data, "[%s]\n", s)
					keys := map[string]any{}
					for k, v := range sections[s] {
						data = fmt.Appendf(data, "%s = %q\n", k, v)
						keys[k] = v
					}
					want[s] = keys
				}
				got, err := cli.INI(data)
				assert.NoError(c, err, "INI decodes the rendered data")
				assert.Equal(c, got, want, "INI returns the rendered keys and sections")
			})
		})

		tests := []struct {
			name string
			give string
			want map[string]any
		}{
			{
				name: "returns a key before the first section as a top-level key",
				give: "port = 8\n",
				want: map[string]any{"port": "8"},
			},
			{
				name: "nests the keys of a section under its name",
				give: "[serve]\naddr = :80\n",
				want: map[string]any{"serve": map[string]any{"addr": ":80"}},
			},
			{
				name: "nests the keys of a dotted section under each of its names",
				give: "[serve.tls]\ncert = srv.pem\n",
				want: map[string]any{"serve": map[string]any{"tls": map[string]any{"cert": "srv.pem"}}},
			},
			{
				name: "continues the keys of a section that appears again",
				give: "[a]\nx = 1\n[b]\ny = 2\n[a]\nz = 3\n",
				want: map[string]any{"a": map[string]any{"x": "1", "z": "3"}, "b": map[string]any{"y": "2"}},
			},
			{
				name: "trims the spaces around a key, a value and the names of a section",
				give: "[ serve . tls ]\n  cert   =   srv.pem  \n",
				want: map[string]any{"serve": map[string]any{"tls": map[string]any{"cert": "srv.pem"}}},
			},
			{
				name: "unquotes a value in double quotes",
				give: `name = " a \"b\" "` + "\n",
				want: map[string]any{"name": ` a "b" `},
			},
			{
				name: "keeps a value of one double quote",
				give: `quote = "` + "\n",
				want: map[string]any{"quote": `"`},
			},
			{
				name: "keeps a value that starts with a double quote and ends without one",
				give: `name = "srv` + "\n",
				want: map[string]any{"name": `"srv`},
			},
			{
				name: "keeps a value that ends with a double quote and starts without one",
				give: `name = srv"` + "\n",
				want: map[string]any{"name": `srv"`},
			},
			{
				name: "keeps the text after the first equals sign as the value",
				give: "url = a=b\n",
				want: map[string]any{"url": "a=b"},
			},
			{
				name: "skips the comments, the empty lines and a byte order mark",
				give: "\U0000FEFF; comment\n# comment\n\nport = 8\n",
				want: map[string]any{"port": "8"},
			},
			{
				name: "reads the lines of CRLF line endings",
				give: "[serve]\r\nport = 8\r\n",
				want: map[string]any{"serve": map[string]any{"port": "8"}},
			},
			{
				name: "returns an empty tree for empty data",
				give: "",
				want: map[string]any{},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := cli.INI([]byte(tt.give))
				assert.NoError(t, err, "INI decodes the data")
				assert.Equal(t, got, tt.want, "INI returns the tree of the data")
			})
		}

		invalid := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns an error for a line without an equals sign",
				give: "port\n",
				want: "cli: INI line 1: want a comment, a [section] or key = value",
			},
			{
				name: "returns an error for an empty key",
				give: " = 8\n",
				want: "cli: INI line 1: want a comment, a [section] or key = value",
			},
			{
				name: "returns an error for a key that its section has already",
				give: "[serve]\nport = 8\nport = 9\n",
				want: "cli: INI line 3: the section has the key port already",
			},
			{
				name: "returns an error for a section with an empty name",
				give: "[serve..tls]\n",
				want: "cli: INI line 1: the section [serve..tls] has an empty name",
			},
			{
				name: "returns an error for a section whose name is a key",
				give: "serve = 1\n[serve]\n",
				want: "cli: INI line 2: the name serve of the section [serve] is a key",
			},
			{
				name: "returns an error for a key that is the name of a section",
				give: "[serve.tls]\n[serve]\ntls = 1\n",
				want: "cli: INI line 3: the section has the key tls already",
			},
			{
				name: "returns an error for a quoted value that does not unquote",
				give: "port = 8\nname = \"\\q\"\n",
				want: "cli: INI line 2: invalid syntax",
			},
			{
				name: "returns an error for a section without its closing bracket",
				give: "[serve\n",
				want: "cli: INI line 1: want a comment, a [section] or key = value",
			},
			{
				name: "returns an error for a section without its opening bracket",
				give: "serve]\n",
				want: "cli: INI line 1: want a comment, a [section] or key = value",
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := cli.INI([]byte(tt.give))
				assert.HasError(t, err, "INI returns an error")
				assert.Equal(t, err.Error(), tt.want, "the error states the line and the reason")
			})
		}
	})
}

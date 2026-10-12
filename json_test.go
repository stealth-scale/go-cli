// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// jsonContract is the contract of the property that FuzzJSON fuzzes and TestJSON checks.
const jsonContract = "JSON returns an object or an error for any data"

// jsonBody decodes arbitrary data with JSON, and checks that it returns no tree with its error.
func jsonBody(c *prop.Case) {
	tree, err := cli.JSON(c.Draw(prop.Bytes(prop.MaxSize(64)), "data"))
	if err != nil {
		assert.Nil(c, tree, "JSON returns no tree with an error")
	}
}

// FuzzJSON fuzzes JSON with arbitrary data.
func FuzzJSON(f *testing.F) {
	prop.Fuzz(f, jsonContract, jsonBody)
}

// TestJSON checks the trees that JSON decodes and the data that it refuses.
func TestJSON(t *testing.T) {
	t.Parallel()

	t.Run("JSON", func(t *testing.T) {
		t.Parallel()

		t.Run("returns an object or an error for any data", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, jsonContract, jsonBody)
		})

		t.Run("returns the members of every encoded object of strings", func(t *testing.T) {
			t.Parallel()
			prop.RoundTrip(t,
				func(m map[string]string) ([]byte, error) { return json.Marshal(m) },
				func(data []byte) (map[string]string, error) {
					tree, err := cli.JSON(data)
					m := make(map[string]string, len(tree))
					for k, v := range tree {
						m[k], _ = v.(string)
					}
					return m, err
				},
				"JSON returns the members that json.Marshal encodes", assert.EquateEmpty())
		})

		tests := []struct {
			name string
			give string
			want map[string]any
		}{
			{
				name: "returns the members of an object",
				give: `{"serve": {"addr": ":80", "tags": ["a", true]}}`,
				want: map[string]any{"serve": map[string]any{"addr": ":80", "tags": []any{"a", true}}},
			},
			{
				name: "keeps the text of a number",
				give: `{"id": 9007199254740993, "ratio": 1.50}`,
				want: map[string]any{"id": json.Number("9007199254740993"), "ratio": json.Number("1.50")},
			},
			{
				name: "returns no members for null",
				give: "null",
				want: nil,
			},
			{
				name: "skips the space after the object",
				give: "{}\n\t ",
				want: map[string]any{},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := cli.JSON([]byte(tt.give))
				assert.NoError(t, err, "JSON decodes the data")
				assert.Equal(t, got, tt.want, "JSON returns the tree of the data")
			})
		}

		t.Run("returns the error of encoding/json for data that is not an object", func(t *testing.T) {
			t.Parallel()
			_, err := cli.JSON([]byte(`["serve"]`))
			want := assert.ErrorAs[*json.UnmarshalTypeError](t, err, "JSON wraps the error of encoding/json")
			assert.Equal(
				t,
				err.Error(),
				"cli: decode JSON: "+want.Error(),
				"the error states the error of encoding/json",
			)
		})

		t.Run("returns an error for data after the object", func(t *testing.T) {
			t.Parallel()
			_, err := cli.JSON([]byte(`{"a": 1} {"b": 2}`))
			assert.Equal(t, err.Error(), "cli: decode JSON: data after the object, which ends at offset 8",
				"the error states the end of the object")
		})
	})
}

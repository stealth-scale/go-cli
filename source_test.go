// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// TestSource checks the text of a source and the kinds that Valid reports.
func TestSource(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give cli.Source
			want string
		}{
			{
				name: "returns the key and the file of a File source",
				give: cli.Source{Kind: cli.File, Name: "serve.addr", File: "srv.json"},
				want: "key serve.addr of srv.json",
			},
			{
				name: "returns the variable of an Environment source",
				give: cli.Source{Kind: cli.Environment, Name: "SRV_SERVE_ADDR"},
				want: "env SRV_SERVE_ADDR",
			},
			{
				name: "returns the flag of a Flag source",
				give: cli.Source{Kind: cli.Flag, Name: "--addr"},
				want: "flag --addr",
			},
			{
				name: "returns the argument of an Argument source",
				give: cli.Source{Kind: cli.Argument, Name: "TARGET"},
				want: "argument TARGET",
			},
			{
				name: "returns default for the zero Source",
				give: cli.Source{},
				want: "default",
			},
			{
				name: "returns the text of the kind for a kind past Argument",
				give: cli.Source{Kind: cli.Argument + 1, Name: "x"},
				want: "Kind(5)",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "String returns the source as text")
			})
		}
	})

	t.Run("Kind", func(t *testing.T) {
		t.Parallel()

		t.Run("Valid", func(t *testing.T) {
			t.Parallel()

			t.Run("reports true for each declared kind", func(t *testing.T) {
				t.Parallel()
				prop.True(t, cli.Kind.Valid, "Valid reports true for a declared kind",
					prop.Using(prop.SampledFrom(cli.Default, cli.File, cli.Environment, cli.Flag, cli.Argument)))
			})

			t.Run("reports false for each kind past Argument", func(t *testing.T) {
				t.Parallel()
				prop.False(t, cli.Kind.Valid, "Valid reports false for a kind past Argument",
					prop.Using(prop.Integer(cli.Argument+1, 255)))
			})
		})
	})
}

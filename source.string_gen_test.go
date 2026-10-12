// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// TestKindString pins the spelling of each kind, which the output of --flag-debug writes.
func TestKindString(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give cli.Kind
			want string
		}{
			{name: "returns default for Default", give: cli.Default, want: "default"},
			{name: "returns file for File", give: cli.File, want: "file"},
			{name: "returns environment for Environment", give: cli.Environment, want: "environment"},
			{name: "returns flag for Flag", give: cli.Flag, want: "flag"},
			{name: "returns argument for Argument", give: cli.Argument, want: "argument"},
			{name: "returns the number of a kind past Argument", give: cli.Argument + 1, want: "Kind(5)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "String returns the spelling of the kind")
			})
		}
	})
}

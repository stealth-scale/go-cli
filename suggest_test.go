// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// suggestOptions are the options of the command serve in the tests of suggestions, whose flags
// --port and --post are one edit apart.
type suggestOptions struct {
	Addr string
	Port int
	Post string
}

// TestSuggest checks the name that a message suggests for an unknown name.
func TestSuggest(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			giveArgs []string
			want     string
		}{
			{
				name:     "suggests the flag one insertion away",
				giveArgs: []string{command, "--adr", "x"},
				want:     "cli: serve: unknown flag --adr (did you mean --addr?)",
			},
			{
				name:     "suggests the flag one deletion away",
				giveArgs: []string{command, "--addrr", "x"},
				want:     "cli: serve: unknown flag --addrr (did you mean --addr?)",
			},
			{
				name:     "suggests the flag one substitution away",
				giveArgs: []string{command, "--adds", "x"},
				want:     "cli: serve: unknown flag --adds (did you mean --addr?)",
			},
			{
				name:     "suggests the flag two edits away",
				giveArgs: []string{command, "--ad", "x"},
				want:     "cli: serve: unknown flag --ad (did you mean --addr?)",
			},
			{
				name:     "suggests no flag three edits away",
				giveArgs: []string{command, "--a", "x"},
				want:     "cli: serve: unknown flag --a",
			},
			{
				name:     "suggests no flag three deletions away",
				giveArgs: []string{command, "--zzzaddr", "x"},
				want:     "cli: serve: unknown flag --zzzaddr",
			},
			{
				name:     "suggests the first of two flags at one distance",
				giveArgs: []string{command, "--pot", "x"},
				want:     "cli: serve: unknown flag --pot (did you mean --port?)",
			},
			{
				name:     "suggests the closer of two flags",
				giveArgs: []string{command, "--posts", "x"},
				want:     "cli: serve: unknown flag --posts (did you mean --post?)",
			},
			{
				name:     "suggests a command without a prefix",
				giveArgs: []string{"serv"},
				want:     `cli: unknown command "serv" (did you mean serve?)`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[struct{}, suggestOptions](t)
				res := execute(t, app, tt.giveArgs, nil, nil)
				assert.Equal(t, res.err.Error(), tt.want, "the message suggests the closest name")
			})
		}
	})
}

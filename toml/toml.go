// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package toml

import (
	"fmt"

	burntsushi "github.com/BurntSushi/toml"

	"go.stealthscale.io/cli"
)

// extension is the extension of the config files that Register registers.
const extension = ".toml"

// Register registers [Decode] as the format of the config files of app with the extension .toml.
func Register[G any](app *cli.App[G]) {
	app.Format(Decode, extension)
}

// Decode decodes the TOML document data into a tree. A table is a map[string]any, an array is a
// []any, a date and a time are a time.Time, and any other value is its Go value.
//
// Error modes:
//   - an error that wraps the error of github.com/BurntSushi/toml for data that is not TOML.
func Decode(data []byte) (map[string]any, error) {
	var tree map[string]any
	if err := burntsushi.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("toml: decode: %w", err)
	}
	return tree, nil
}

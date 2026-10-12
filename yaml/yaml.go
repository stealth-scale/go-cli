// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package yaml

import (
	"fmt"

	yamlv3 "go.yaml.in/yaml/v3"

	"go.stealthscale.io/cli"
)

// extensions are the extensions of the config files that Register registers.
var extensions = [...]string{".yaml", ".yml"}

// Register registers [Decode] as the format of the config files of app with the extensions .yaml
// and .yml.
func Register[G any](app *cli.App[G]) {
	app.Format(Decode, extensions[:]...)
}

// Decode decodes the first YAML document of data into a tree. A mapping is a map[string]any whose
// keys are the text of the YAML keys, a sequence is a []any, and a scalar is its Go value. An empty
// document is an empty tree.
//
// Error modes:
//   - an error that wraps the error of go.yaml.in/yaml/v3 for data that is not a YAML mapping.
func Decode(data []byte) (map[string]any, error) {
	var tree map[string]any
	if err := yamlv3.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("yaml: decode: %w", err)
	}
	for k, v := range tree {
		tree[k] = normalize(v)
	}
	return tree, nil
}

// normalize returns v with each mapping that has a key other than a string as a map[string]any,
// whose keys are the text of the YAML keys. go.yaml.in/yaml/v3 decodes such a mapping as a
// map[any]any, which the format of the cli package does not take. normalize changes the maps and
// the slices of v in place.
func normalize(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for k, e := range v {
			v[k] = normalize(e)
		}
		return v
	case map[any]any:
		m := make(map[string]any, len(v))
		for k, e := range v {
			m[fmt.Sprint(k)] = normalize(e)
		}
		return m
	case []any:
		for i, e := range v {
			v[i] = normalize(e)
		}
		return v
	default:
		return v
	}
}

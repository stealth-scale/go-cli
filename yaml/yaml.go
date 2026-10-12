// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package yaml

import (
	"fmt"

	yamlv3 "go.yaml.in/yaml/v3"

	"go.stealthscale.io/cli"
)

// The tags of the scalars that Decode tells apart: the text, the null and the key of a merge.
const (
	strTag   = "!!str"
	nullTag  = "!!null"
	mergeTag = "!!merge"
)

// extensions are the extensions of the config files that Register registers.
var extensions = [...]string{".yaml", ".yml"}

// Register registers [Decode] as the format of the config files of app with the extensions .yaml
// and .yml.
func Register[G any](app *cli.App[G]) {
	app.Format(Decode, extensions[:]...)
}

// Decode decodes the first YAML document of data into a tree. A mapping is a map[string]any, a
// sequence is a []any and a null is nil. Every other scalar is its source text, such as 0644 or
// 1.10, and the option that reads it converts the text. Decode applies anchors, aliases and merge
// keys. An empty document is an empty tree.
//
// Error modes:
//   - an error that wraps the error of go.yaml.in/yaml/v3. The parser reports data that is not
//     YAML, a document that is not a mapping, a duplicate key and an alias that contains itself.
func Decode(data []byte) (map[string]any, error) {
	var doc yamlv3.Node
	if err := yamlv3.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("yaml: decode: %w", err)
	}
	asText(&doc)
	var tree map[string]any
	if err := doc.Decode(&tree); err != nil {
		return nil, fmt.Errorf("yaml: decode: %w", err)
	}
	return tree, nil
}

// asText tags each scalar below n as text, so that a decode returns the source text of the scalar
// and not the value that the YAML schema resolves. A null value and the key << of a merge keep
// their tags. An alias of a scalar becomes a copy of the scalar, so a key is always text.
func asText(n *yamlv3.Node) {
	for i, c := range n.Content {
		if c.Kind == yamlv3.AliasNode && c.Alias.Kind == yamlv3.ScalarNode {
			scalar := *c.Alias
			c = &scalar
			n.Content[i] = c
		}
		key := n.Kind == yamlv3.MappingNode && i%2 == 0
		if c.Kind == yamlv3.ScalarNode && c.ShortTag() != mergeTag && (key || c.ShortTag() != nullTag) {
			c.Tag = strTag
		}
		asText(c)
	}
}

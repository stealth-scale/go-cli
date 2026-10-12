// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"strconv"
	"strings"
)

// The tokens of an INI file.
const (
	// byteOrderMark is the mark that an editor may write at the start of a UTF-8 file.
	byteOrderMark = "\U0000FEFF"
	// semicolonComment and hashComment start a comment line.
	semicolonComment = ";"
	hashComment      = "#"
	// sectionStart and sectionEnd enclose the name of a section.
	sectionStart = "["
	sectionEnd   = "]"
	// quote encloses a quoted value.
	quote = `"`
)

// INI is the [Format] of INI, which [New] registers for the extension .ini. A key before the first
// section is a top-level key. A section [serve] nests its keys under serve, and a section
// [serve.tls] nests them under serve and tls. A section that appears again continues its keys. A
// value is text, so a list is text of values that commas separate. A value in double quotes is
// unquoted by the rules of Go, and a line that starts with ; or # is a comment.
//
// Error modes:
//   - an error with the number of the line for a line that is not a comment, a section or a key
//     with a value, a section with an empty name, a key that its section has already, a name that
//     is a key and a section, and a quoted value that strconv.Unquote refuses.
func INI(data []byte) (map[string]any, error) {
	tree := map[string]any{}
	section := tree
	n := 0
	for line := range strings.Lines(strings.TrimPrefix(string(data), byteOrderMark)) {
		n++
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, semicolonComment) || strings.HasPrefix(line, hashComment):
		case strings.HasPrefix(line, sectionStart) && strings.HasSuffix(line, sectionEnd):
			var reason string
			if section, reason = iniSection(tree, line[len(sectionStart):len(line)-len(sectionEnd)]); reason != "" {
				return nil, fmt.Errorf("cli: INI line %d: %s", n, reason)
			}
		default:
			key, value, ok := strings.Cut(line, entrySeparator)
			key, value = strings.TrimSpace(key), strings.TrimSpace(value)
			if !ok || key == "" {
				return nil, fmt.Errorf("cli: INI line %d: want a comment, a [section] or key = value", n)
			}
			if _, dup := section[key]; dup {
				return nil, fmt.Errorf("cli: INI line %d: the section has the key %s already", n, key)
			}
			if len(value) > 1 && strings.HasPrefix(value, quote) && strings.HasSuffix(value, quote) {
				unquoted, err := strconv.Unquote(value)
				if err != nil {
					return nil, fmt.Errorf("cli: INI line %d: %w", n, err)
				}
				value = unquoted
			}
			section[key] = value
		}
	}
	return tree, nil
}

// iniSection returns the map of the section name in tree, and adds the maps of its names that tree
// does not have. It returns the reason when a name of the section is empty or is a key with a value.
func iniSection(tree map[string]any, name string) (map[string]any, string) {
	section := tree
	for part := range strings.SplitSeq(name, keySeparator) {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, "the section [" + name + "] has an empty name"
		}
		child, ok := section[part]
		if !ok {
			child = map[string]any{}
			section[part] = child
		}
		next, ok := child.(map[string]any)
		if !ok {
			return nil, "the name " + part + " of the section [" + name + "] is a key"
		}
		section = next
	}
	return section, ""
}

// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// JSON is the [Format] of JSON, which [New] registers for the extension .json. The data is one JSON
// object, and JSON keeps the text of each number, so a large integer keeps every digit.
//
// Error modes:
//   - an error that wraps the error of encoding/json for data that does not start with a JSON
//     object.
//   - an error for data after the object.
func JSON(data []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var tree map[string]any
	if err := dec.Decode(&tree); err != nil {
		return nil, fmt.Errorf("cli: decode JSON: %w", err)
	}
	end := dec.InputOffset()
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("cli: decode JSON: data after the object, which ends at offset %d", end)
	}
	return tree, nil
}

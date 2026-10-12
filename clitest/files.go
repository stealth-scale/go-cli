// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package clitest

import (
	"io/fs"
	"path"
	"strings"
	"testing/fstest"
)

// Files is a file system of text files by their paths, for Input.Files. A path is the path that a
// program reads, such as srv.json or /etc/srv.json.
type Files map[string]string

// Open opens the file name of f. f contains each file at the path that path.Clean returns for its
// path, without the leading slash, so the path /etc/srv.json of f is the file etc/srv.json.
//
// Error modes:
//   - an *fs.PathError that wraps fs.ErrNotExist when f does not contain name.
//   - an *fs.PathError that wraps fs.ErrInvalid when name is not a valid path of a file system.
func (f Files) Open(name string) (fs.File, error) {
	files := make(fstest.MapFS, len(f))
	for p, text := range f {
		files[strings.TrimPrefix(path.Clean(p), "/")] = &fstest.MapFile{Data: []byte(text)}
	}
	return files.Open(name)
}

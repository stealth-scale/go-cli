// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package clitest_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli/clitest"
)

// TestFiles checks the file system of text files.
func TestFiles(t *testing.T) {
	t.Parallel()

	t.Run("Open", func(t *testing.T) {
		t.Parallel()

		t.Run("passes the checks of fstest.TestFS", func(t *testing.T) {
			t.Parallel()
			files := clitest.Files{"srv.json": "{}", "/etc/srv/config.json": "{}"}
			assert.NoError(t, fstest.TestFS(files, "srv.json", "etc/srv/config.json"), "Files is a file system")
		})

		tests := []struct {
			name      string
			giveFiles clitest.Files
			givePath  string
			want      string
		}{
			{
				name:      "opens a file by its path",
				giveFiles: clitest.Files{"srv.json": `{"port": 8}`},
				givePath:  "srv.json",
				want:      `{"port": 8}`,
			},
			{
				name:      "opens a file of an absolute path at the path without its leading slash",
				giveFiles: clitest.Files{"/etc/srv.json": `{"port": 8}`},
				givePath:  "etc/srv.json",
				want:      `{"port": 8}`,
			},
			{
				name:      "opens a file at the path that path.Clean returns for its path",
				giveFiles: clitest.Files{"./conf/../srv.json": `{"port": 8}`},
				givePath:  "srv.json",
				want:      `{"port": 8}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := fs.ReadFile(tt.giveFiles, tt.givePath)
				assert.NoError(t, err, "the file system reads the file")
				assert.Equal(t, string(got), tt.want, "the file has the text of its path")
			})
		}

		t.Run("returns fs.ErrNotExist for a path that Files does not contain", func(t *testing.T) {
			t.Parallel()
			_, err := fs.ReadFile(clitest.Files{"srv.json": "{}"}, "srv.yaml")
			assert.ErrorIs(t, err, fs.ErrNotExist, "Open returns fs.ErrNotExist")
		})
	})
}

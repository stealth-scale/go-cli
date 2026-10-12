// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package clitest_test

import (
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"

	"go.stealthscale.io/cli/clitest"
)

// TestEnv checks the Env of an input and the output of its runs.
func TestEnv(t *testing.T) {
	t.Parallel()

	t.Run("NewEnv", func(t *testing.T) {
		t.Parallel()

		t.Run("passes the arguments of the input", func(t *testing.T) {
			t.Parallel()
			env, _ := clitest.NewEnv(&clitest.Input{Args: []string{"serve", "--port", "8"}})
			assert.Equal(t, env.Args, []string{"serve", "--port", "8"}, "the Env has the arguments of the input")
		})

		t.Run("reports the variables of the input through Lookup", func(t *testing.T) {
			t.Parallel()
			env, _ := clitest.NewEnv(&clitest.Input{Vars: map[string]string{"SRV_PORT": "8"}})
			got, ok := env.Lookup("SRV_PORT")
			assert.True(t, ok, "Lookup reports the variable")
			assert.Equal(t, got, "8", "Lookup returns the value of the variable")
		})

		t.Run("lists the variables of the input through Environ", func(t *testing.T) {
			t.Parallel()
			env, _ := clitest.NewEnv(&clitest.Input{Vars: map[string]string{"B": "2", "A": "1"}})
			assert.Equal(t, env.Environ(), []string{"A=1", "B=2"}, "Environ lists the variables by their names")
		})

		t.Run("reads the standard input of the input", func(t *testing.T) {
			t.Parallel()
			env, _ := clitest.NewEnv(&clitest.Input{Stdin: "one two\n"})
			got, err := io.ReadAll(env.Stdin)
			assert.NoError(t, err, "the standard input reads")
			assert.Equal(t, string(got), "one two\n", "the standard input has the text of the input")
		})

		t.Run("returns ErrStdin from every read of the standard input when FailStdin is true", func(t *testing.T) {
			t.Parallel()
			env, _ := clitest.NewEnv(&clitest.Input{Stdin: "one two\n", FailStdin: true})
			_, err := io.ReadAll(env.Stdin)
			assert.ErrorIs(t, err, clitest.ErrStdin, "the read returns ErrStdin")
		})

		t.Run("writes the standard output to Output.Stdout", func(t *testing.T) {
			t.Parallel()
			env, out := clitest.NewEnv(&clitest.Input{})
			n, err := io.WriteString(env.Stdout, "out\n")
			assert.NoError(t, err, "the write succeeds")
			assert.Equal(t, n, 4, "the write returns the length of the text")
			assert.Equal(t, out.Stdout(), "out\n", "Stdout returns the text of the standard output")
			assert.Empty(t, out.Stderr(), "Stderr returns nothing")
		})

		t.Run("writes the standard error to Output.Stderr", func(t *testing.T) {
			t.Parallel()
			env, out := clitest.NewEnv(&clitest.Input{})
			n, err := io.WriteString(env.Stderr, "err\n")
			assert.NoError(t, err, "the write succeeds")
			assert.Equal(t, n, 4, "the write returns the length of the text")
			assert.Equal(t, out.Stderr(), "err\n", "Stderr returns the text of the standard error")
			assert.Empty(t, out.Stdout(), "Stdout returns nothing")
		})

		t.Run("returns ErrStdout from every write to the standard output when FailStdout is true", func(t *testing.T) {
			t.Parallel()
			env, out := clitest.NewEnv(&clitest.Input{FailStdout: true})
			_, err := io.WriteString(env.Stdout, "out\n")
			assert.ErrorIs(t, err, clitest.ErrStdout, "the write returns ErrStdout")
			assert.Empty(t, out.Stdout(), "Stdout returns nothing")
		})

		opens := []struct {
			name      string
			giveFiles fs.FS
			givePath  string
			want      string
		}{
			{
				name:      "opens a file of Files",
				giveFiles: clitest.Files{"srv.json": `{"port": 8}`},
				givePath:  "srv.json",
				want:      `{"port": 8}`,
			},
			{
				name:      "opens the file of an absolute path at the root of Files",
				giveFiles: clitest.Files{"etc/srv.json": `{"port": 8}`},
				givePath:  "/etc/srv.json",
				want:      `{"port": 8}`,
			},
			{
				name:      "opens the file at the path that path.Clean returns for the path of the open",
				giveFiles: clitest.Files{"srv.json": `{"port": 8}`},
				givePath:  "./conf/../srv.json",
				want:      `{"port": 8}`,
			},
			{
				name:      "opens a file of a file system of another type",
				giveFiles: fstest.MapFS{"srv.json": {Data: []byte(`{"port": 8}`)}},
				givePath:  "srv.json",
				want:      `{"port": 8}`,
			},
		}
		for _, tt := range opens {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				env, _ := clitest.NewEnv(&clitest.Input{Files: tt.giveFiles})
				f, err := env.Open(tt.givePath)
				assert.NoError(t, err, "Open opens the file")
				got, err := io.ReadAll(f)
				assert.NoError(t, err, "the file reads")
				assert.NoError(t, f.Close(), "the file closes")
				assert.Equal(t, string(got), tt.want, "the file has the text of Files")
			})
		}

		failures := []struct {
			name      string
			giveFiles fs.FS
			wantErr   error
			want      string
		}{
			{
				name:      "returns an error that wraps fs.ErrNotExist for a path outside Files",
				giveFiles: clitest.Files{"srv.yaml": "port: 8"},
				wantErr:   fs.ErrNotExist,
				want:      "open /etc/srv.json: file does not exist",
			},
			{
				name:    "returns an error that wraps fs.ErrNotExist for every path without Files",
				wantErr: fs.ErrNotExist,
				want:    "open /etc/srv.json: file does not exist",
			},
			{
				name:      "returns an error that wraps an error of Files that is not an *fs.PathError",
				giveFiles: brokenFS{},
				wantErr:   errDisk,
				want:      "open /etc/srv.json: disk gone",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				env, _ := clitest.NewEnv(&clitest.Input{Files: tt.giveFiles})
				_, err := env.Open("/etc/srv.json")
				assert.ErrorIs(t, err, tt.wantErr, "Open returns the error of the file system")
				assert.Equal(t, err.Error(), tt.want, "the error states the path of the open")
			})
		}
	})

	t.Run("Output", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps every write of goroutines that write at the same time", func(t *testing.T) {
			t.Parallel()
			env, out := clitest.NewEnv(&clitest.Input{})
			outcomes := history.Concurrently(8, time.Minute, func(client int) (any, error) {
				if _, err := io.WriteString(env.Stdout, "out\n"); err != nil {
					return client, err
				}
				_, err := io.WriteString(env.Stderr, "err\n")
				return client, err
			})
			for _, o := range outcomes {
				assert.True(t, o.Finished, "each writer finishes")
				assert.NoError(t, o.Error, "each write succeeds")
			}
			assert.Equal(t, out.Stdout(), strings.Repeat("out\n", 8), "Stdout returns every write")
			assert.Equal(t, out.Stderr(), strings.Repeat("err\n", 8), "Stderr returns every write")
		})
	})
}

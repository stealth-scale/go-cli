// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package clitest

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"go.stealthscale.io/cli"
)

// The errors of the streams that an Input makes fail.
var (
	// ErrStdin is the error of every read of the standard input of a run with Input.FailStdin.
	ErrStdin = errors.New("clitest: bad file descriptor")
	// ErrStdout is the error of every write to the standard output of a run with Input.FailStdout.
	ErrStdout = errors.New("clitest: no space left on device")
)

// Input is the input of one run of a program.
type Input struct {
	Vars       map[string]string // the environment variables of Env.Lookup and Env.Environ
	Files      fs.FS             // the file system of the config files, or nil for a run without files
	Comment    string            // the comment above the run in a transcript, or empty
	Stdin      string            // the standard input
	Args       []string          // the arguments after the name of the program
	FailStdin  bool              // every read of the standard input fails with ErrStdin
	FailStdout bool              // every write to the standard output fails with ErrStdout
}

// Output is what the runs of an Env write. It keeps the standard output, the standard error, and
// both streams in the order of the writes. It also records the config files that the runs read.
//
// # Concurrency
//
// The methods of an Output and the writers of its Env are safe for concurrent use, as the methods
// of os.File are.
type Output struct {
	files  []file // the config files that the runs read, in the order of the first reads
	stdout bytes.Buffer
	stderr bytes.Buffer
	both   bytes.Buffer // both streams in the order of the writes
	mu     sync.Mutex   // guards the fields of the Output
}

// Stdout returns the text that the runs wrote to the standard output.
func (o *Output) Stdout() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.stdout.String()
}

// Stderr returns the text that the runs wrote to the standard error.
func (o *Output) Stderr() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.stderr.String()
}

// read records the config file name with the content data, unless the runs read the file before.
func (o *Output) read(name string, data []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !slices.ContainsFunc(o.files, func(f file) bool { return f.name == name }) {
		o.files = append(o.files, file{name: name, data: data})
	}
}

// snapshot returns the config files that the runs read and the text of both streams.
func (o *Output) snapshot() ([]file, string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return slices.Clone(o.files), o.both.String()
}

// file is a config file that a run read.
type file struct {
	name string // the path of the read
	data []byte // the content of the file
}

// stream is a writer of one stream of an Output. It writes to the buffer of the stream and to the
// buffer of both streams.
type stream struct {
	out *Output
	buf *bytes.Buffer // the buffer of the stream
}

// Write appends p to the buffer of the stream and to the buffer of both streams. It returns len(p)
// and a nil error.
func (s stream) Write(p []byte) (int, error) {
	s.out.mu.Lock()
	defer s.out.mu.Unlock()
	s.out.both.Write(p)
	s.buf.Write(p)
	return len(p), nil
}

// closed is a standard input whose every read fails with ErrStdin.
type closed struct{}

// Read returns ErrStdin.
func (closed) Read([]byte) (int, error) {
	return 0, ErrStdin
}

// full is a standard output whose every write fails with ErrStdout.
type full struct{}

// Write returns ErrStdout.
func (full) Write([]byte) (int, error) {
	return 0, ErrStdout
}

// NewEnv returns an Env of in, and the Output that the runs of the Env write.
//
// A run of the Env reads only the config files of in.Files. It reads the file at the path of the
// read after filepath.ToSlash and path.Clean, and without the leading slash, so /etc/srv.json is the
// file etc/srv.json of in.Files. A read of a path outside in.Files, and every read without
// in.Files, returns an error that wraps fs.ErrNotExist. The error of a read is an *fs.PathError with
// the path of the read.
func NewEnv(in *Input) (*cli.Env, *Output) {
	out := new(Output)
	files := in.Files
	if files == nil {
		files = Files{}
	}
	env := &cli.Env{
		Lookup:  cli.Vars(in.Vars),
		Environ: cli.Environ(in.Vars),
		ReadFile: func(name string) ([]byte, error) {
			data, err := fs.ReadFile(files, strings.TrimPrefix(path.Clean(filepath.ToSlash(name)), "/"))
			if err != nil {
				if e, ok := errors.AsType[*fs.PathError](err); ok {
					err = e.Err
				}
				return nil, &fs.PathError{Op: "read", Path: name, Err: err}
			}
			out.read(name, data)
			return data, nil
		},
		Stdin:  strings.NewReader(in.Stdin),
		Stdout: stream{out: out, buf: &out.stdout},
		Stderr: stream{out: out, buf: &out.stderr},
		Args:   in.Args,
	}
	if in.FailStdin {
		env.Stdin = closed{}
	}
	if in.FailStdout {
		env.Stdout = full{}
	}
	return env, out
}

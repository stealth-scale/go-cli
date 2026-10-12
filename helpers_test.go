// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"testing/fstest"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// The app and the command that the specs run.
const (
	appName     = "srv"
	appHelp     = "Serve the store."
	command     = "serve"
	commandHelp = "Run the server."
)

// errWrite is the error of a writer that fails every write.
var errWrite = errors.New("disk full")

// result is the error of one run of an app and what the run wrote.
type result struct {
	err    error
	stdout string
	stderr string
}

// failingWriter is a writer that fails every write with errWrite.
type failingWriter struct{}

// Write returns errWrite.
func (failingWriter) Write([]byte) (int, error) {
	return 0, errWrite
}

// execute runs app with the arguments args, the environment variables vars and the files files, in
// the context of tb, and returns the error of the run and what it wrote.
func execute[G any](tb assert.TB, app *cli.App[G], args []string, vars map[string]string, files fstest.MapFS) result {
	tb.Helper()
	var stdout, stderr bytes.Buffer
	env := &cli.Env{Lookup: cli.Vars(vars), ReadFile: files.ReadFile, Stdout: &stdout, Stderr: &stderr, Args: args}
	err := app.Execute(assert.Context(tb), env)
	return result{err: err, stdout: stdout.String(), stderr: stderr.String()}
}

// serving returns the app srv with the global options G and the command serve with the options C.
// The handler of serve stores its run in the run that serving returns, so a test reads the options
// that the last run received.
func serving[G, C any](tb assert.TB) (*cli.App[G], *cli.Run[G, C]) {
	tb.Helper()
	got := new(cli.Run[G, C])
	app := cli.New[G](appName, appHelp)
	app.Command(command, commandHelp, func(_ context.Context, run *cli.Run[G, C]) error {
		*got = *run
		return nil
	})
	return app, got
}

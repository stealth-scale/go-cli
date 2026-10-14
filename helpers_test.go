// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"testing/fstest"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// The app, its version and the commands that the specs run.
const (
	appName        = "srv"
	appHelp        = "Serve the store."
	versionText    = "srv 1.4.2 (f70f638, 2026-10-13)"
	command        = "serve"
	commandHelp    = "Run the server."
	parent         = "db"
	parentHelp     = "Manage the database."
	subcommand     = "migrate"
	subcommandHelp = "Migrate the database to the current schema."
)

// The errors of the writers and the files that fail.
var (
	// errWrite is the error of a writer that fails every write.
	errWrite = errors.New("disk full")
	// errIO is the error of a file that fails to read or to close.
	errIO = errors.New("input/output error")
)

// formatGlobal contains the global options of the app of formats. Format selects the renderer of a
// run.
type formatGlobal struct {
	Levels []int
	Config cli.ConfigFile `short:"c"`
	Format string         `          default:"text" enum:"text,json"`
	Depth  int
}

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

// brokenFile is an fs.File whose Read returns readErr, or io.EOF without readErr, and whose Close
// returns closeErr.
type brokenFile struct {
	readErr  error
	closeErr error
}

// Stat returns fs.ErrInvalid, because a brokenFile has no information.
func (brokenFile) Stat() (fs.FileInfo, error) {
	return nil, fs.ErrInvalid
}

// Read returns readErr, or io.EOF without readErr.
func (f brokenFile) Read([]byte) (int, error) {
	if f.readErr != nil {
		return 0, f.readErr
	}
	return 0, io.EOF
}

// Close returns closeErr.
func (f brokenFile) Close() error {
	return f.closeErr
}

// recorder is a Renderer that records what the package passes to it, and returns err. It writes
// nothing.
type recorder struct {
	err     error
	help    *cli.Help
	version *cli.Version
	failure *cli.Failure
	environ []string
	rows    []cli.DebugRow
}

// Help records h and environ.
func (r *recorder) Help(_ io.Writer, environ []string, h *cli.Help) error {
	r.help, r.environ = h, environ
	return r.err
}

// Version records v and environ.
func (r *recorder) Version(_ io.Writer, environ []string, v *cli.Version) error {
	r.version, r.environ = v, environ
	return r.err
}

// Error records f and environ.
func (r *recorder) Error(_ io.Writer, environ []string, f *cli.Failure) error {
	r.failure, r.environ = f, environ
	return r.err
}

// Debug records rows and environ.
func (r *recorder) Debug(_ io.Writer, environ []string, rows []cli.DebugRow) error {
	r.rows, r.environ = rows, environ
	return r.err
}

// execute runs app with the arguments args, the environment variables vars and the files files, in
// the context of tb, and returns the error of the run and what it wrote.
func execute[G any](tb assert.TB, app *cli.App[G], args []string, vars map[string]string, files fstest.MapFS) result {
	tb.Helper()
	var stdout, stderr bytes.Buffer
	env := &cli.Env{Lookup: cli.Vars(vars), Open: files.Open, Stdout: &stdout, Stderr: &stderr, Args: args}
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

// formats returns the app srv with the global options formatGlobal, the version versionText, the
// command serve with the options appOptions, and the command db with the subcommand migrate and
// without a handler. The function of RenderFor appends the options of each call to got. It returns
// json for the format json and text for any other format.
func formats(text, json *recorder, got *[]formatGlobal) *cli.App[formatGlobal] {
	app := cli.New[formatGlobal](appName, appHelp)
	app.Version(versionText)
	app.Command(command, commandHelp, func(context.Context, *cli.Run[formatGlobal, appOptions]) error { return nil })
	db := app.Command[struct{}](parent, parentHelp, nil)
	db.Command(subcommand, subcommandHelp, func(context.Context, *cli.Run[formatGlobal, struct{}]) error {
		return nil
	})
	app.RenderFor(func(g *formatGlobal) cli.Renderer {
		*got = append(*got, *g)
		if g.Format == "json" {
			return json
		}
		return text
	})
	return app
}

// nesting returns the app srv with the global options G, the command db with the options P, and the
// subcommand migrate of db with the options C. The handlers of db and migrate store their runs in
// the runs that nesting returns, so a test reads the options that the last run of each received.
func nesting[G, P, C any](tb assert.TB) (*cli.App[G], *cli.Run[G, P], *cli.Run[G, C]) {
	tb.Helper()
	db, migrate := new(cli.Run[G, P]), new(cli.Run[G, C])
	app := cli.New[G](appName, appHelp)
	cmd := app.Command(parent, parentHelp, func(_ context.Context, run *cli.Run[G, P]) error {
		*db = *run
		return nil
	})
	cmd.Command(subcommand, subcommandHelp, func(_ context.Context, run *cli.Run[G, C]) error {
		*migrate = *run
		return nil
	})
	return app, db, migrate
}

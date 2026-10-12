// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"errors"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// The options of the app in the tests of Error.
type (
	errorGlobal struct {
		Config cli.ConfigFile
	}
	errorOptions struct {
		Addr string
	}
)

// TestError checks the text of an *Error and the sentinel that it wraps.
func TestError(t *testing.T) {
	t.Parallel()

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the package name and the command before the message", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, errorOptions](t)
			res := execute(t, app, []string{command, "--adr", ":80"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error for an unknown flag")
			assert.Equal(t, e.Error(), "cli: serve: unknown flag --adr (did you mean --addr?)",
				"Error returns the package name and the command before the message")
		})

		t.Run("returns the package name before the message of an input before the command", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, errorOptions](t)
			res := execute(t, app, []string{"serv"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error for an unknown command")
			assert.Equal(t, e.Error(), `cli: unknown command "serv" (did you mean serve?)`,
				"Error returns the package name before the message")
		})
	})

	t.Run("Unwrap", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the sentinel of the error", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, errorOptions](t)
			res := execute(t, app, []string{command, "--adr", ":80"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error for an unknown flag")
			assert.Equal(t, errors.Unwrap(e), cli.ErrUnknownFlag, "Unwrap returns the sentinel", assert.ByIdentity())
		})
	})

	t.Run("Message", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the command before the message without the package name", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, errorOptions](t)
			res := execute(t, app, []string{command, "--adr", ":80"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error for an unknown flag")
			assert.Equal(t, e.Suggestion, "--addr", "the error suggests the closest flag")
			assert.Equal(t, e.Message(), "serve: unknown flag --adr (did you mean --addr?)",
				"Message returns the command before the message")
		})

		t.Run("suggests the long flag of a long flag with one dash", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, errorOptions](t)
			res := execute(t, app, []string{command, "-addr", ":80"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error for an unknown flag")
			assert.Equal(t, e.Suggestion, "--addr", "the error suggests the long flag")
			assert.Equal(t, e.Message(), "serve: unknown flag -addr (did you mean --addr?)",
				"Message returns the suggestion after the message")
		})

		t.Run("suggests the closest key of an unknown config key", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[errorGlobal, errorOptions](t)
			files := fstest.MapFS{"srv.json": {Data: []byte(`{"serve": {"adr": ":80"}}`)}}
			res := execute(t, app, []string{"--config", "srv.json", command}, nil, files)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error for an unknown key")
			assert.Equal(t, e.Suggestion, "serve.addr", "the error suggests the closest key")
			assert.Equal(t, e.Message(), "serve: unknown key serve.adr in srv.json (did you mean serve.addr?)",
				"Message returns the suggestion after the message")
		})

		t.Run("returns the message without a suggestion when no name is close", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, errorOptions](t)
			res := execute(t, app, []string{command, "--zzzzzz"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error for an unknown flag")
			assert.Empty(t, e.Suggestion, "the error has no suggestion")
			assert.Equal(t, e.Message(), "serve: unknown flag --zzzzzz", "Message returns the message alone")
		})

		t.Run("returns the message alone for an input before the command", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, errorOptions](t)
			res := execute(t, app, []string{"serv"}, nil, nil)
			e := assert.ErrorAs[*cli.Error](t, res.err, "Execute returns an *Error for an unknown command")
			assert.Equal(t, e.Suggestion, command, "the error suggests the closest command")
			assert.Equal(t, e.Message(), `unknown command "serv" (did you mean serve?)`,
				"Message returns the message without a prefix")
		})
	})
}

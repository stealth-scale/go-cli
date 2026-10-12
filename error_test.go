// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"errors"
	"testing"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// errorOptions are the options of the command serve in the tests of Error.
type errorOptions struct {
	Addr string
}

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
}

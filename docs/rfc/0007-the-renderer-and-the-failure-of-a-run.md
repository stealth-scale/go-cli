---
rfc: 0007
title: The renderer and the failure of a run for a caller of Execute
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-13
updated: 2026-10-13
discussion: https://github.com/stealth-scale/go-cli/issues/8
supersedes: none
superseded-by: none
produces-adr: tbd
---

<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0007: The renderer and the failure of a run for a caller of Execute

## Summary

`App.ExecuteRenderer` runs a command line as `App.Execute` does and also returns the renderer of the run. `App.Failure` returns the failure that `App.Main` writes for the error of a run. A program with exit statuses or streams of its own then writes the failure of a run with the renderer that the run selected. `App.Main` becomes these two calls, the write of the failure and the mapping to an exit status.

## Motivation

A program can map the error of each run to exit statuses of its own. `sysexits.h` defines `EX_USAGE` as 64 for a usage error. Under `--format=json` the program writes a usage error as JSON on the standard output, so a script reads one stream. `App.RenderFor` selects the renderer of such a run from its options. It does so also for a run whose command line fails.

`App.Main` writes the failure with that renderer, but it writes to the standard error. It returns 1 for every error of a handler and 2 for every usage error. `App.Execute` returns the error alone, so its caller does not know which renderer the run selected.

A caller also cannot rebuild the failure that a renderer expects. `Main` takes `Failure.Message` from the text of an `*Error` without its command and its suggestion. That text is unexported, and `Error.Message` returns it with the command and the suggestion.

cobra v1.10.2 has the same split. `Command.Execute` returns the error, and `Command.ExecuteC` returns the command that ran beside the error.

## Detailed design

### API

```go
// ExecuteRenderer runs the command line of env as [App.Execute] describes, and also returns the
// renderer of the run. The renderer has written the help and the table of --flag-debug of the run.
// A caller that writes the failure of the run itself passes [App.Failure] of the error to the
// method Error of the renderer, as [App.Main] does.
//
// Error modes:
//   - the error modes of [App.Execute].
func (a *App[G]) ExecuteRenderer(ctx context.Context, env *Env) (Renderer, error)

// Failure returns the failure that [App.Main] writes for the error err of a run, or nil for a nil
// error. Usage is true for an error that wraps an [*Error]. For an *Error itself, Command, Message
// and Suggestion come from the *Error. For any other error, Message is the text of the error.
func (a *App[G]) Failure(err error) *Failure
```

`Main` becomes these two calls:

```go
func (a *App[G]) Main(ctx context.Context, env *Env) int {
	r, err := a.ExecuteRenderer(ctx, env)
	f := a.Failure(err)
	if f == nil {
		return ExitOK
	}
	if env != nil && env.Stderr != nil {
		_ = r.Error(env.Stderr, env.environ(), f)
	}
	if f.Usage {
		return ExitUsage
	}
	return ExitFailure
}
```

A program with exit statuses of its own writes a failure in JSON to its standard output:

```go
r, err := a.ExecuteRenderer(ctx, env)
f := a.Failure(err)
if f == nil {
	return 0
}
w := env.Stderr
if _, ok := r.(jsonRenderer); ok {
	w = env.Stdout
}
_ = r.Error(w, environ, f)
if f.Usage {
	return 64
}
return 1
```

### Rules

- `Execute` returns the error of `ExecuteRenderer` and drops its renderer.
- `ExecuteRenderer` returns the renderer of every run, also of a run whose `Env` is incomplete or whose declarations are invalid. Those runs pass the zero value of the global options to the function of `RenderFor`.
- `Failure` reads only its error, so a caller of `Execute` uses it too.

### Invariants

1. `Execute` and `Main` return the same errors and exit statuses as before, and `Main` writes the same failures.
2. `Main` writes the failure that `Failure` returns, with the renderer that `ExecuteRenderer` returns.
3. The package imports only the standard library.

### Changes outside the package

- The example `15-renderer` calls `ExecuteRenderer` and `Failure` in place of `Main`. It exits with 64 for a usage error and writes a failure in JSON to the standard output.
- The README and the documentation of the package describe both methods.
- The changeset of the package adds both methods as a minor change.

### Migration

A program does not need to change.

### Tests

- `ExecuteRenderer` returns the error of a run beside the renderer of `Render`, or beside the renderer that the function of `RenderFor` selects.
- `ExecuteRenderer` returns `Plain` when the function of `RenderFor` returns nil.
- `Failure` returns nil for a nil error, the parts of an `*Error`, the text of an error that wraps an `*Error`, and the text of any other error.
- The specs of `Execute` and `Main` pass without a change.
- Coverage and the mutation score are 100%.

## Alternatives considered

### One method for the renderer and the failure

`ExecuteRenderer` could return the failure in place of the error, as `(Renderer, *Failure)`.

**Why not:** a caller of `Execute` with the renderer of `Render` needs the failure alone. `ExecuteRenderer` and `Failure` serve both callers.

### The text of an Error

`Error` could export the text without its command and its suggestion, and each caller would build the failure.

**Why not:** each caller would then repeat the rules of `Main` for an error that wraps an `*Error`. `Failure` keeps one copy of those rules.

### Options of Main

`Main` could take the stream of the failure and the exit statuses as options.

**Why not:** each new need of a program would add an option. `ExecuteRenderer` and `Failure` leave the stream, the statuses and any other step to the code of the program.

## Drawbacks

- The package gains the methods `App.ExecuteRenderer` and `App.Failure`.
- The renderer that `ExecuteRenderer` returns has already written the help and the table of `--flag-debug`. A caller that writes another output with it writes after those.
- A caller passes the environment of the run to the method `Error` of the renderer itself.

## Open questions

None.

## Unresolved and future work

None.

## References

| What | Where |
|---|---|
| Issue 8: a caller of Execute cannot receive the renderer of a failed run | <https://github.com/stealth-scale/go-cli/issues/8> |
| `App.Execute`, `App.Main` and `run` at v0.3.0 | <https://github.com/stealth-scale/go-cli/blob/v0.3.0/app.go#L131-L169> |
| cobra v1.10.2: `ExecuteC` | <https://github.com/spf13/cobra/blob/v1.10.2/command.go#L1083-L1084> |
| `sysexits.h` of FreeBSD: `EX_USAGE` | <https://man.freebsd.org/cgi/man.cgi?query=sysexits> |

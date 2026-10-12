---
rfc: 0003
title: Subcommands and hooks
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-10-12
updated: 2026-10-12
discussion: none
supersedes: none
superseded-by: none
produces-adr: tbd
---

<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0003: Subcommands and hooks

## Summary

The commands of an app form a tree with the app at its root. `App.Command` returns a `*cli.Command[G, C]`. Its method `Command` adds a subcommand. The options of a command apply to it and to its descendants. Their names contain the path of the command, such as `SRV_DB_DSN` and `db.dsn`. A command reads the options of an ancestor through an embedded pointer to their type. A hook runs before the handler of a command and calls the handler through a function `next`. The code of the hook after `next` runs after the handler returns. The app can have a handler of its own, so a program such as `wc` does not need a command.

## Motivation

`kubectl config view`, `git remote add` and `gh pr create` group the commands of a program by area. The package `cli` has one level of commands. The flat help of a program with 20 commands lists every command. Names such as `db-migrate` merge the area and the action into one word.

The commands of one area also share settings and resources, such as the address of a database and the connection to it. Commands share options through an embedded struct now. Each field of the struct needs the tags `env` and `key` to get one name in every command. Every handler opens and closes the shared resource itself. A program with one action has to give it a command name. The program `wc` of the example `14-testing` runs as `wc count`.

The libraries of the same kind nest commands and call hooks:

- cobra v1.10.2 adds commands to a command with `AddCommand`. The persistent flags of a command apply to its subcommands. cobra calls the hooks `PersistentPreRun` and `PreRun` before `Run`. It calls `PostRun` and `PersistentPostRun` after `Run`. Each hook has a variant that returns an error.
- urfave/cli v3.14.0 has the field `Commands []*Command` on a command. A flag applies to the subcommands unless its field `Local` is true. urfave/cli calls the hook `Before` ahead of `Action` and the hook `After` after `Action`.
- kong v1.16.1 declares a command with the tag `cmd` on a struct field, so nested structs nest commands. kong calls methods such as `AfterApply` and `AfterRun` on the structs of the path as hooks.

## Detailed design

### API

```go
// Command is a command of an app with the global options G and the options C. A command has a
// handler, subcommands or both. Its options and its hooks apply to it and to its descendants.
//
// # Concurrency
//
// Command and Use are not safe for concurrent use, as the registrations of an App are not.
type Command[G, C any] struct {
	// contains unexported fields
}

// Command adds the command name with the help text help to the app and returns it. The exported
// fields of C are the options of the command. A run of the command calls handle with the options
// after every source is applied, and returns the error of handle unchanged. A command with a nil
// handle runs only its subcommands.
//
// App.Execute returns an error that wraps ErrDeclaration when name is not a name of lower-case
// words, when the app has a command of that name already, when handle is nil for a command
// without subcommands, and when the command has subcommands and declares a positional argument.
func (a *App[G]) Command[C any](name, help string, handle func(ctx context.Context, run *Run[G, C]) error) *Command[G, C]

// Command adds the subcommand name with the help text help to c and returns it. A subcommand
// follows the rules of App.Command.
func (c *Command[G, C]) Command[D any](name, help string, handle func(ctx context.Context, run *Run[G, D]) error) *Command[G, D]

// Handle sets the handler of the app. A command line without a command calls handle with the
// global options as the options of the run. A later call replaces an earlier one.
//
// App.Execute returns an error that wraps ErrDeclaration when handle is nil.
func (a *App[G]) Handle(handle func(ctx context.Context, run *Run[G, G]) error)

// Hook is a function that a run calls before its handler. run contains the options of the command
// that uses the hook. The hook calls next to run the remaining hooks and the handler. It returns
// the error of next, an error that wraps it, or an error of its own. A hook that does not call
// next ends the run with its own result.
type Hook[G, C any] func(ctx context.Context, run *Run[G, C], next func(ctx context.Context) error) error

// Use adds hook to the app, so every run calls hook before the hooks of its commands. The hooks of
// the app run in the order of the calls of Use.
//
// App.Execute returns an error that wraps ErrDeclaration when hook is nil.
func (a *App[G]) Use(hook Hook[G, G])

// Use adds hook to c, so a run of c or of a descendant of c calls hook after the hooks of the
// ancestors of c. The hooks of c run in the order of the calls of Use.
func (c *Command[G, C]) Use(hook Hook[G, C])

// Command returns the path of the command that the run calls, such as "db migrate". It returns the
// empty string in a run of the handler of the app.
func (r *Run[G, C]) Command() string
```

A program registers a tree of commands and its hooks:

```go
a := cli.New[Global]("srv", "Serve the store.")
a.Use(interrupt)
a.Command("serve", "Run the server.", serve)
db := a.Command("db", "Manage the database.", status)
db.Use(connect)
db.Command("migrate", "Migrate the database to the current schema.", migrate)
db.Command("seed", "Load the fixtures into the database.", seed)
```

Go infers the options type of a command from its handler, so a command without a handler states the type. For a command without options, the type is `struct{}`:

```go
cache := a.Command[struct{}]("cache", "Manage the cache.", nil)
cache.Command("purge", "Remove every entry of the cache.", purge)
```

A command reads the options of an ancestor through an embedded pointer to their type. The package sets the pointer to the options of that ancestor. A hook of the ancestor and every handler of its descendants share one value:

```go
// DB contains the options of the command db and the connection of the hook connect.
type DB struct {
	DSN  string `help:"Data source name of the database." required:""`
	conn *sql.DB
}

// Migrate contains the options of the command db migrate.
type Migrate struct {
	*DB
	Steps int `help:"Number of migrations to apply." default:"1"`
}
```

The embedded `*DB` does not add options to `migrate`. `App.Execute` returns an error that wraps `ErrDeclaration` when `DB` is not the options type of an ancestor and when it is the options type of two ancestors.

The program declares its hooks and its handlers as functions:

```go
// interrupt ends the context of a run at the first interrupt.
func interrupt(ctx context.Context, _ *cli.Run[Global, Global], next func(context.Context) error) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	return next(ctx)
}

// connect opens the connection of the commands of db and closes it after the handler returns.
func connect(ctx context.Context, run *cli.Run[Global, DB], next func(context.Context) error) (err error) {
	db := run.Config()
	if db.conn, err = sql.Open("pgx", db.DSN); err != nil {
		return fmt.Errorf("main: open the database: %w", err)
	}
	defer func() { err = errors.Join(err, db.conn.Close()) }()
	return next(ctx)
}

// migrate applies the first Steps migrations over the connection of the command db.
func migrate(ctx context.Context, run *cli.Run[Global, Migrate]) error {
	m := run.Config()
	return applyMigrations(ctx, m.conn, m.Steps)
}
```

A package of the program can register a subtree because a `*Command` is a value. The options types are then in a package that both packages import:

```go
func Register(db *cli.Command[options.Global, options.DB]) {
	db.Command("dump", "Write the database to the standard output.", dump)
}
```

### The command line

- The first argument that is not a flag selects a command of the app. Each next such argument selects a subcommand of the command before it as long as that command has subcommands.
- A command with subcommands does not declare positional arguments. Every argument after it that is not a flag must then be the name of a subcommand. An unknown name fails with a suggestion from the subcommands of that command.
- After a command without subcommands, every argument that is not a flag is a positional argument.
- The global options can declare positional arguments only in an app without commands because the app is the command at the root.
- A flag can come anywhere after the name of the command that declares it. The flags of the app can still come anywhere.
- A flag of the help writes the help of the last command of the path. `srv db --help` writes the help of `db`.
- A path that ends at a command without a handler writes the help of the command and a blank line to the standard error. The run then fails with a missing command.

The rule for positional arguments makes every misspelled subcommand an error. cobra stops its search at the deepest command that matches. Without a validator of the arguments of that command, cobra reports an unknown command only at the root. `app db migrat` then writes the help of `db` and exits with the status 0 when `db` does not have a handler. When `db` has a handler, cobra runs it with the argument `migrat`.

### Names

The path of a command takes the place of the name of the command in the names of its options. The names of an option of the app do not change.

| Name | Rule | `DSN` of `srv db` | `Steps` of `srv db migrate` |
|---|---|---|---|
| Flag | Unchanged | `--dsn` | `--steps` |
| Environment variable | The app, the path and the flag in upper case with an underscore for each dash | `SRV_DB_DSN` | `SRV_DB_MIGRATE_STEPS` |
| Config key | The path and the flag joined with dots | `db.dsn` | `db.migrate.steps` |

A config file nests the keys of a command under the objects of its path:

```json
{"db": {"dsn": "postgres://localhost/store", "migrate": {"steps": 3}}}
```

`App.Execute` returns an error that wraps `ErrDeclaration` when two options on the path of a command declare one flag, one short flag, one environment variable or one config key. This rule extends the current check of the global options against the options of each command.

### Help

- The help of a command lists its subcommands in the order of their registration, its positional arguments and its options. It lists the options of its ancestors as the global options.
- The environment variable and the config key of each option show the command that declares it.
- The help of the app lists the global options as its options because the app is the command at the root. Its usage line changes from `srv [global options] <command> [options]` to `srv <command> [options]`.
- The usage line of a command with subcommands is `srv db <command> [options]`. It is `srv db [<command>] [options]` when the command has a handler.
- A command without subcommands has the usage line `srv db migrate [options]` with its positional arguments at the end.
- `Help.Command` contains the path of the command, such as `db migrate`. `Help.Commands` contains its subcommands.
- `Help.Options` contains the options of the command. `Help.Globals` contains the options of its ancestors.

### Hooks

- A run calls its hooks after the options resolve and after `--flag-debug` writes its table. The help, an invalid input and a path that ends at a command without a handler end the run before its hooks.
- The hooks of the app run first. The hooks of the other commands of the path follow in the order of the path. The hooks of one command run in the order of the calls of `Use`. The handler runs last.
- The code of the hooks after `next` runs in the reverse order, so the hooks of the app release their resources last.
- The run of a hook contains the global options, the options of the command that uses the hook, the streams of the run and the path of the command that runs.
- A hook passes a value to the handler through the options of its command or through the context. Descendants read those options through their embedded pointers. `migrate` reads the connection that `connect` opens through its embedded `*DB`.
- Each hook passes a context to `next`. The handler receives the context of the last hook.
- `next` runs the remaining hooks and the handler once per call.

Each hook of cobra has a counterpart:

| cobra v1.10.2 | This proposal |
|---|---|
| `PersistentPreRunE` of a command | The code before `next` in a hook of the command |
| `PersistentPostRunE` of a command | The code after `next` in a hook of the command, or a deferred call |
| `PreRunE` and `PostRunE` of a command without subcommands | A hook of the command, or the handler |
| `cmd.SetContext` in a hook | The context that the hook passes to `next` |
| The package variable `EnableTraverseRunHooks` | Not needed, because every hook of the path runs |

### Errors

| Input | Status | Message of Plain |
|---|---|---|
| `srv cache` | `ExitUsage` | The help of `cache` on the standard error, a blank line, then `srv: cache: missing command` |
| `srv db migrat` | `ExitUsage` | `srv: db: unknown command "migrat" (did you mean migrate?)` |
| `srv db --steps 3 migrate` | `ExitUsage` | `srv: db: unknown flag --steps` |
| `srv db migrate --stpes 3` | `ExitUsage` | `srv: db migrate: unknown flag --stpes (did you mean --steps?)` |
| `srv db migrate` without a DSN | `ExitUsage` | `srv: db migrate: missing --dsn` |
| `srv db migrate` with a failing hook | `ExitFailure` | `srv:` and the error text of the hook |

`Error.Command`, `Failure.Command` and the hint of the module charm contain the path of the command, such as `db migrate`.

### Capabilities left out

- **Hooks before the options resolve.** kong calls `BeforeReset`, `BeforeResolve` and `BeforeApply` while it parses. Such a hook would see the defaults in place of the values of the run. A value that it set would lack a source for `--flag-debug`. A program that has to compute an option from a source other than the five sources of a run would reverse this verdict.
- **A default subcommand.** urfave/cli runs the subcommand `DefaultCommand` for a command line without a subcommand. The handler of the command covers such a run. A default subcommand would also accept its own flags before its name.
- **Aliases and hidden commands.** An alias needs rules of its own in the help and in the suggestions for a misspelled name. The help needs one more rule for a hidden command. The examples of the repository do not use either. A program that renames a command and has to accept the old name would reverse this verdict.

### Changes outside the package

- The example `03-commands` gains a command with options, a handler, a hook and two subcommands.
- The program `wc` of the example `14-testing` runs without the command `count`.
- The section Commands of the README shows a subtree and a hook.
- The renderers do not change because they write the fields of `Help` and `Failure`. The golden files with the help of an app change their usage line and their sections.
- The changeset of the package `cli` covers the API because the package is unreleased.

### Migration

- A program compiles without a change because a call statement drops the result of `App.Command`.
- The help of every app changes. Its options move from the global options to the options. Its usage line drops `[global options]`.
- The names of the options of a program without subcommands do not change.
- An app without a handler and without commands fails with an error that wraps `ErrDeclaration`. Such an app wrote its help and failed with a missing command before.

### Tests

- Cases scan a path through two levels of subcommands. The flags of each level come before and after the names of its descendants. A flag of a subcommand before the name of the subcommand fails.
- Cases run the handler of a command with subcommands. Other cases run the handler of the app with and without positional arguments.
- The names of the options of a command contain its path. The config keys nest under the objects of the path.
- A handler reads the options of an ancestor through an embedded pointer. It also reads the value that a hook of the ancestor sets.
- A property over generated trees checks that the hooks of a run start in the order of the path and end in the reverse order.
- Cases cover a hook that does not call `next`, a hook that replaces the error of the handler, and a hook that passes a context to the handler.
- The code of a hook after `next` runs after the handler fails. The help, an invalid input and a missing command end a run before its hooks.
- `Run.Command` returns the path of the command of the run.
- Golden files contain the help of the app, of a command with subcommands, of a command with a handler and subcommands, and of a command two levels down.
- A missing command and an unknown command two levels down fail with the messages of the table of errors.
- Declaration errors cover a command without a handler and without subcommands, a command with subcommands and a positional argument, two siblings with one name, a nil hook, and an embedded pointer without a matching ancestor.
- Coverage and the mutation score are 100%.

## Alternatives considered

### Groups without handlers or options

A type `Group` would contain commands and groups. A group would not have a handler or options of its own. A command would not have subcommands.

**Why not:**

- A hook of a group would see only the global options. The address of a database would then become a global option of every command.
- An option that the commands of a group share needs the tags `env` and `key` on each of its fields to get one name. It cannot come before the name of a subcommand either.
- The tree would have two kinds of node. The app would be a third kind.

### Options of an ancestor through a lookup

kong binds the struct of every command of the path by its type when it calls a method `Run`. A `Run` method takes the struct of a parent as a parameter that way. A function `cli.Options[P](run)` would do the same in this package.

**Why not:** a lookup fails at run time when `P` is not the options type of an ancestor. An embedded pointer of such a type fails in the check of the declarations that `App.Execute` runs before any handler.

### Hooks before and after the handler

cobra calls `PersistentPreRun`, `PreRun`, `Run`, `PostRun` and `PersistentPostRun` in that order. urfave/cli calls `Before` ahead of `Action` and `After` after it.

**Why not:**

- cobra returns before `PostRun` and `PersistentPostRun` when `RunE` fails. A post hook does not release a resource after a failure.
- cobra runs only the `PersistentPreRun` of the nearest command that has one unless a program sets the package variable `EnableTraverseRunHooks`.
- urfave/cli defers the `After` of each command before its chain of `Before` hooks runs. `After` runs also when its own `Before` failed or did not run.
- A `Hook` pairs the two halves. The code after `next` runs only when the code before it ran. A deferred call runs it after a failure.

### Paths in the name of a command

`app.Command("db migrate", help, migrate)` would add the command `migrate` to an implicit command `db`.

**Why not:** the command `db` then needs a second registration for its help text, its options and its hooks. A misspelled path creates a new command without an error.

### Commands from struct tags

kong declares the commands of a program as tagged fields of a struct.

**Why not:** the package `cli` declares a command with its handler. The options of the command are the fields of a struct. A tree of struct fields would add a second way to declare a command.

## Drawbacks

- The types `Command` and `Hook` and the methods `Command.Command`, `Command.Use`, `App.Use`, `App.Handle` and `Run.Command` are new. `App.Command` returns a value.
- A command without a handler states its options type, such as `a.Command[struct{}]("cache", help, nil)`.
- The help of every app changes. The old usage line of an app is in nine files of the repository. Three of them are golden files of examples.
- `Help.Command`, `Error.Command` and `Failure.Command` can contain spaces, such as `db migrate`. A renderer that treats them as one word now gets a path.
- The names of a deep command grow long, such as `SRV_DB_BACKUP_CREATE_TARGET` and `db.backup.create.target`.
- A program that moves a command into a subtree changes the names of its options. Its users then change their environment variables and their config files.
- An embedded pointer selects an ancestor by the type of its options, so two commands on one path cannot share an options type.

## Open questions

None.

## Unresolved and future work

None.

## References

| What | Where |
|---|---|
| cobra v1.10.2: the run hooks and their order | <https://github.com/spf13/cobra/blob/v1.10.2/command.go#L117-L146> |
| cobra v1.10.2: `execute` returns before `PostRun` when `RunE` fails | <https://github.com/spf13/cobra/blob/v1.10.2/command.go#L955-L1045> |
| cobra v1.10.2: `EnableTraverseRunHooks` | <https://github.com/spf13/cobra/blob/v1.10.2/cobra.go#L64-L66> |
| cobra v1.10.2: `Find` stops at the deepest command that matches | <https://github.com/spf13/cobra/blob/v1.10.2/command.go#L757-L779> |
| cobra v1.10.2: `legacyArgs` reports an unknown command at the root | <https://github.com/spf13/cobra/blob/v1.10.2/args.go#L28-L39> |
| cobra v1.10.2: `ExecuteC` writes the help of a command without a handler and returns nil | <https://github.com/spf13/cobra/blob/v1.10.2/command.go#L1148-L1155> |
| urfave/cli v3.14.0: `BeforeFunc` and `AfterFunc` | <https://github.com/urfave/cli/blob/v3.14.0/funcs.go#L8-L15> |
| urfave/cli v3.14.0: the deferred `After` and the chain of `Before` | <https://github.com/urfave/cli/blob/v3.14.0/command_run.go#L247-L352> |
| urfave/cli v3.14.0: the field `Local` of a flag | <https://github.com/urfave/cli/blob/v3.14.0/flag_impl.go#L67> |
| urfave/cli v3.14.0: `Commands` and `DefaultCommand` | <https://github.com/urfave/cli/blob/v3.14.0/command.go> |
| kong v1.16.1: the hooks | <https://github.com/alecthomas/kong/blob/v1.16.1/hooks.go#L3-L31> |
| kong v1.16.1: `Parse` calls the hooks before and after it applies the values | <https://github.com/alecthomas/kong/blob/v1.16.1/kong.go#L323-L352> |
| kong v1.16.1: `Run` binds the parents and calls `AfterRun` after a failed run | <https://github.com/alecthomas/kong/blob/v1.16.1/context.go#L825-L898> |
| kong v1.16.1: the tag `cmd` | <https://github.com/alecthomas/kong/blob/v1.16.1/tag.go> |

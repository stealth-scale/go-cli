// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Package cli runs command-line programs whose options are the fields of Go structs.
//
// A program declares its global options in one struct and the options of each command in a struct
// of its own. The package derives the flag, the environment variable and the config key of each
// field, converts the text of every source to the type of the field, and applies the sources in
// one order. A handler receives the options as typed values, with the source of each.
//
// [New] returns an [App] for the global options, and [App.Command] adds a [Command] with its
// handler. The handler receives a [Run], whose [Run.Global] and [Run.Config] return the options
// after every source is applied:
//
//	type Serve struct {
//		Addr string `help:"Address to listen on." default:":8080" short:"a"`
//	}
//
//	app := cli.New[Global]("srv", "Serve the store.")
//	app.Command("serve", "Run the server.", func(ctx context.Context, run *cli.Run[Global, Serve]) error {
//		return serve(ctx, run.Config().Addr)
//	})
//	os.Exit(app.Main(context.Background(), cli.OS()))
//
// # Commands
//
// The commands of an app form a tree with the app at its root, so the global options are the
// options of the app. [Command.Command] adds a subcommand to a command. A command has a handler,
// subcommands or both, and [App.Handle] sets the handler of the app. The options of a command apply
// to it and to its descendants. A command reads the options of an ancestor through an embedded
// pointer to their type. The package sets the pointer to the options of that ancestor.
//
// A [Hook] runs before the handler and calls it through the function next, so the code of the hook
// after next runs after the handler returns. [App.Use] adds a hook that every run calls, and
// [Command.Use] adds a hook that the runs of a command and of its descendants call. The hooks of the
// app run first, then the hooks of each command of the path, and the handler runs last.
//
// # Names
//
// Each exported field of an options struct is an option with three names:
//   - The flag is the name of the field in kebab case. The field DryRun is --dry-run, and HTTPPort
//     is --http-port.
//   - The environment variable joins the app, the path of the command and the flag in upper case,
//     with an underscore for each dash. A run of the command serve of the app srv reads the field
//     Addr from SRV_SERVE_ADDR, and the subcommand migrate of the command db reads the field Steps
//     from SRV_DB_MIGRATE_STEPS.
//   - The config key joins the path of the command and the flag with dots, such as serve.addr and
//     db.migrate.steps.
//
// A global option has no command in its variable and its key.
//
// A struct field without a conversion is a group. The name of a named field starts the names of
// its options: the field TLS with the field Cert gives the flag --tls-cert, the variable
// SRV_SERVE_TLS_CERT and the key serve.tls.cert. Groups nest, so Cache.Redis.Addr gives
// --cache-redis-addr. An embedded struct adds its options without a group, also when its type is
// unexported.
//
// A name is words of lower-case ASCII letters and digits that single dashes join, the first word
// starting with a letter. The name of the app, of each command, of each group and of each option
// follows that rule.
//
// # Tags
//
//   - help is the text of the option in the help.
//   - default is the value of the option when no other source sets it.
//   - enum is the comma-separated list of the values that the option takes.
//   - required makes a run fail when no source sets the option.
//   - arg makes the field of a command a positional argument, which only the command line sets.
//     The arguments take the command line in the order of their fields. An argument with a default
//     is optional, and a list takes the rest of the arguments.
//   - name replaces the last word of the flag, the variable and the key.
//   - short declares a short flag of one ASCII letter, such as short:"p" for -p.
//   - env and key replace the whole environment variable and the whole config key, so two commands
//     can share one key. The value - removes the variable or the key.
//   - prefix replaces the name of a group. An empty prefix adds the options without a group, and a
//     prefix on an embedded struct makes it a group.
//   - secret masks the value in the help and in the output of --flag-debug.
//
// The tags required, arg and secret count by their presence, such as required:"".
//
// # Types
//
// A field converts from text by the first rule that matches its type: a conversion that
// [App.Convert] registered, a built-in conversion, a pointer that implements
// [encoding.TextUnmarshaler], a pointer that implements [flag.Value], [time.Duration], and the
// kinds string, bool, int, uint and float. The built-in conversions read url.URL through
// url.Parse, time.Location through time.LoadLocation, net.IPNet through net.ParseCIDR and
// fs.FileMode as an octal number such as 0644 or 0o644. A slice takes values that commas separate,
// a map takes entries of the form key=value, and a pointer is nil until a source sets it.
//
// # Sources
//
// A run applies the default, the config file, the environment variable and the flag of each
// option in that order, so a later source replaces the value of an earlier one. A value that fails
// leaves its option at the value of the earlier sources, and the run returns the first error. A
// repeated flag of a slice appends to the values of its earlier occurrences, and of a map adds to
// their entries.
// Every source goes through the same conversion and the same checks. [Run.Source] returns the
// [Source] of a value. The global flag --flag-debug writes the value and the source of every option
// to the standard error before the handler runs.
//
// # Config files
//
// An option of the type [ConfigFile] is the path of a config file. A run reads the file after the
// defaults and before the environment variables. The extension of the path selects a [Format]:
//   - [JSON] for .json and [INI] for .ini, which [New] registers.
//   - A format that [App.Format] registers, such as the YAML format of the module
//     go.stealthscale.io/cli/yaml and the TOML format of go.stealthscale.io/cli/toml.
//
// The YAML and TOML parsers are in modules of their own, so this module does not require a module
// outside the standard library. The file nests the keys of each command and of each group in
// objects. A run fails for a key of the file without an option in the app. [App.IgnoreKeys]
// registers keys that the app decodes itself, and a run skips them. [App.LocateConfig] registers a
// function that returns the path of the config file when neither the flag nor the environment
// variable of the option sets it, such as a search from a directory upward.
//
// # Input files
//
// An option of the type [InputFile] is the path of an input file, and the path - is the standard
// input. [Run.Open] opens the file through [Env.Open], or returns the standard input for -. The
// package does not open an input file while it reads the command line, so the handler opens and
// closes the files of a list one at a time.
//
// # Command line
//
// The command line follows these rules of GNU:
//   - A long flag takes its value after an equals sign or in the next argument.
//   - Short flags combine in one argument, such as -vq. A short flag with a value takes the rest of
//     its argument or the next one, such as -p8080 or -p 8080.
//   - A flag can come anywhere after the name of the command that declares it, and a global flag
//     anywhere.
//   - After a command with subcommands, an argument that is not a flag is the name of a
//     subcommand.
//   - The argument -- ends the flags.
//
// The flags -h and --help write the help of the last command of the path. The help lists the
// options by name and the positional arguments in the order of their fields. It lists the options
// of the ancestors of a command as its global options. It notes the config key of each option only
// when an option of the path is a config file.
//
// # Errors
//
// [App.Execute] returns an [*Error] for an invalid command line, environment variable or config
// file. The error wraps one of [ErrUnknownCommand], [ErrUnknownFlag], [ErrUnknownKey],
// [ErrMissing], [ErrInvalidValue], [ErrUnexpectedArgument] and [ErrConfigFile]. It returns an error
// that wraps [ErrDeclaration] for a registration or options that the app cannot run, and
// [ErrEnv] for an incomplete [Env]. [App.Main] writes the error and returns [ExitUsage] or
// [ExitFailure].
//
// # Output
//
// The [Renderer] of a run writes the help, the [Failure] of the run and the rows of --flag-debug.
// [App.Render] registers one renderer for every run. [App.RenderFor] registers a function that
// selects the renderer of each run from its global options, such as an option of the output format.
// An app without a call uses [Plain], which writes plain text in columns. The module
// go.stealthscale.io/cli/charm renders the same output with Lip Gloss. A renderer receives the
// environment of the run from [Env.Environ], and [Run.Environ] returns that environment to a
// handler.
//
// # Testing
//
// An [Env] contains the arguments, the environment variables, the files and the standard streams
// of a run, so a test runs a program without the state of its process. [Vars] returns the
// environment variables of a map, and [Environ] lists the variables of the same map. [Env.Open]
// has the signature of fs.FS.Open, so a test assigns the Open method of a file system. [NewRun]
// returns the [Run] of a handler for a test that calls the handler without [App.Execute]. The
// package go.stealthscale.io/cli/clitest builds the Env of a test, runs a program with it, and
// writes the transcript of a list of runs for a golden file.
//
// # Concurrency
//
// [App.Execute] and [App.Main] are safe for concurrent use once the registrations have returned.
// Each run has options of its own.
//
// # Dependency position
//
// The package imports only the standard library: bytes, context, encoding, encoding/json, errors,
// flag, fmt, io, io/fs, maps, net, net/url, os, path/filepath, reflect, regexp, slices, strconv,
// strings, time, unicode and unicode/utf8.
package cli

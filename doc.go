// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Package cli runs command-line programs whose options are the fields of Go structs.
//
// A program declares its global options in one struct and the options of each command in a struct
// of its own. The package derives the flag, the environment variable and the config key of each
// field, converts the text of every source to the type of the field, and applies the sources in
// one order. A handler receives the options as typed values, with the source of each.
//
// [New] returns an [App] for the global options, and [App.Command] adds a command with its handler.
// The handler receives a [Run], whose [Run.Global] and [Run.Config] return the options after every
// source is applied:
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
// # Names
//
// Each exported field of an options struct is an option with three names:
//   - The flag is the name of the field in kebab case. The field DryRun is --dry-run, and HTTPPort
//     is --http-port.
//   - The environment variable joins the app, the command and the flag in upper case, with an
//     underscore for each dash. A run of the command serve of the app srv reads the field Addr from
//     SRV_SERVE_ADDR.
//   - The config key joins the command and the flag with dots, such as serve.addr.
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
// fs.FileMode as an octal number such as 0644. A slice takes values that commas separate, a map
// takes entries of the form key=value, and a pointer is nil until a source sets it.
//
// # Sources
//
// A run applies the default, the config file, the environment variable and the flag of each
// option in that order, so a later source replaces the value of an earlier one. A repeated flag of
// a slice appends to the values of its earlier occurrences, and of a map adds to their entries.
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
// objects. A run fails for a key of the file without an option in the app.
//
// # Command line
//
// The command line follows these rules of GNU:
//   - A long flag takes its value after an equals sign or in the next argument.
//   - Short flags combine in one argument, such as -vq. A short flag with a value takes the rest of
//     its argument or the next one, such as -p8080 or -p 8080.
//   - A global flag can come before or after the command.
//   - The argument -- ends the flags.
//
// The flags -h and --help write the help. The help lists the options by name and the positional
// arguments in the order of their fields.
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
// # Testing
//
// An [Env] contains the arguments, the environment variables, the files and the standard streams
// of a run, so a test runs a program without the state of its process. [Vars] returns the
// environment variables of a map. [NewRun] returns the [Run] of a handler for a test that calls the
// handler without [App.Execute].
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

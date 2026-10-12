---
research: 0001
title: Does a Go library that imports only the standard library derive flags, environment variables and config keys from one type, and what must go-cli add?
author: Claude, for Roy Klopper
status: Answered
created: 2026-10-12
updated: 2026-10-12
freshest-source: 2026-10-12
supersedes: none
superseded-by: none
---

<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# Research-0001: Does a Go library that imports only the standard library derive flags, environment variables and config keys from one type, and what must go-cli add?

## The question

Does a maintained Go library meet all six requirements in the following table? If none does, which mechanisms of the existing libraries should go-cli combine and which must it add?

| # | Requirement |
|---|---|
| R1 | A build of the library imports only the standard library. A requirement that only tests use does not count |
| R2 | One Go type declares the flags, the environment variables and the config-file keys of the options |
| R3 | Subcommands with options that a command shares with its subcommands |
| R4 | Generated help |
| R5 | Shell completion for at least bash and zsh |
| R6 | Config files in at least one format. The order of precedence is flag, environment variable, file and default |

A library meets a requirement when its own source at a tagged release contains the mechanism. We admitted source code at tagged releases and our own measurements. We also admitted the documentation and the issue comments of the maintainers. One maintained library that meets all six requirements would make the answer yes.

## The answer

No. We did not find a maintained Go library that meets all six requirements or one that meets five. kong v1.16.1 comes closest. A build of kong imports only the standard library. One struct declares its commands, flags, environment variables, JSON config keys and help. Our reference program needs 35 lines of code with kong and 92 with cobra and viper. kong has no shell completion. It applies a config file over an environment variable. It takes a config key from the flag name without the command path. It reads the environment from the process.

go-cli has to be built. It can combine the declaration by type of kong with the staged precedence and the format functions of ff. It can also take the ordered sources of urfave/cli and the completion protocol of cobra. None of the Go CLI libraries we read derives names from the command path, takes an injected environment or records the source of every value. go-cli has to add these three mechanisms.

## Findings

### Dependencies of a build

We listed the packages that a Linux build of each library imports with `go list -deps` under Go 1.27.2 [S3]. The table counts the packages outside the standard library and outside the module of the library. The release dates come from the Go module proxy [S2].

| Library | Release | Packages | Modules outside the standard library |
|---|---|---|---|
| spf13/cobra | v1.10.2 of 2025-12-03 | 1 | spf13/pflag |
| spf13/pflag | v1.0.10 of 2025-09-02 | 0 | none |
| spf13/viper | v1.21.0 of 2025-09-08 | 30 | 12, among them fsnotify, mapstructure, afero, cast, go-toml, yaml and golang.org/x/text |
| alecthomas/kong | v1.16.1 of 2026-08-09 | 0 | none |
| urfave/cli/v3 | v3.14.0 of 2026-10-02 | 0 | none |
| urfave/cli-altsrc/v3 with the package `json` | v3.1.0 of 2025-10-02 | 1 | gopkg.in/yaml.v3 |
| peterbourgon/ff/v4 with `ffhelp` and `ffjson` | v4.0.0-beta.1 of 2025-08-02 | 0 | none |
| alexflint/go-arg | v1.6.1 of 2025-11-25 | 1 | alexflint/go-scalar |
| jessevdk/go-flags | v1.6.1 of 2024-06-15 | 1 | golang.org/x/sys |
| knadh/koanf/v2 | v2.3.8 of 2026-10-07 | 5 | mapstructure, koanf/maps, copystructure and reflectwalk |
| ardanlabs/conf/v3 | v3.13.0 of 2026-07-26 | 0 | none |
| caarlos0/env/v11 | v11.4.1 of 2026-05-01 | 0 | none |
| sethvargo/go-envconfig | v1.4.3 of 2026-07-19 | 0 | none |
| cristalhq/acmd | v0.12.0 of 2024-04-26 | 0 | none |
| cristalhq/aconfig | v0.19.0 of 2025-06-19 | 1 | mitchellh/mapstructure |
| ilyakaznacheev/cleanenv | v1.5.0 of 2023-07-20 | 5 | BurntSushi/toml, godotenv, yaml.v3 and edn |

- The go.mod of cobra also requires mousetrap for Windows builds and go-md2man and yaml for its `doc` package.
- The proxy lists only alpha and beta releases of ff/v4 [S2].

### The reference program

We wrote the program `srv` once with the standard library and once with each of four libraries [S4]. Every version ran the same 11 scenarios. The program has two commands and eight inputs.

| Command | Input | Type | Default | Environment variable | Config key |
|---|---|---|---|---|---|
| all | `--config` | path | none | none | none |
| all | `--verbose` | bool | false | `SRV_VERBOSE` | `verbose` |
| `serve` | `--addr` | string | `:8080` | `SRV_SERVE_ADDR` | `serve.addr` |
| `serve` | `--timeout` | duration | 30s | `SRV_SERVE_TIMEOUT` | `serve.timeout` |
| `serve` | `--tags` | list of strings | empty | `SRV_SERVE_TAGS` | `serve.tags` |
| `serve` | `--mode` | `dev` or `prod` | `dev` | `SRV_SERVE_MODE` | `serve.mode` |
| `migrate` | `--dry-run` | bool | false | none | none |
| `migrate` | `TARGET` | required positional string | none | none | none |

The config file is JSON. The order of precedence is flag, environment variable, config file and default. Each command prints its resolved values on one line. The scenarios cover these cases:

- The defaults and a config file
- An environment variable over the file and a flag over an environment variable
- A bool flag over the file and a duration flag
- An invalid mode from a flag and from an environment variable
- `migrate` with a target, with `SRV_VERBOSE` and without a target

We wrote each version in the style of the examples of its library. gofmt and `go vet` do not report a finding for any version. We counted the lines that are neither blank nor comments. We built each version with `go build -trimpath -ldflags='-s -w'` under Go 1.27.2 on linux/amd64.

| Version | Code lines | Binary | Scenarios passed | Deviation from the table |
|---|---|---|---|---|
| Standard library | 136 | 2,613,408 B | 11 of 11 | none |
| cobra and viper | 92 | 6,144,263 B | 11 of 11 | none |
| urfave/cli v3 and cli-altsrc | 78 | 9,289,991 B | 11 of 11 | none |
| ff v4 beta | 62 | 3,076,256 B | 11 of 11 | `SRV_ADDR` and the key `addr` instead of the names of the table |
| kong | 35 | 4,276,487 B | 10 of 11 | The key `addr` instead of `serve.addr`. The file takes precedence over the environment |

Each version refers to the option `--timeout` in these places:

| Version | Count | Places |
|---|---|---|
| Standard library | 5 | The struct field, the default, the environment block, the flag and the read |
| cobra and viper | 4 | The struct field with its `mapstructure` tag, the flag, the key of `BindPFlag` and the read |
| urfave/cli v3 | 2 | The flag literal with its environment variable and config key, and the read by name |
| ff v4 | 2 | The flag and the read |
| kong | 2 | The struct field with its tags and the read |

The runs also showed four behaviours.

- kong applied the config file over the environment. With `SRV_SERVE_ADDR=:7000` and `"addr": ":9000"` in the file it printed `:9000`. kong reads the environment when it resets a flag. A resolver overwrites that value later (model.go:396-413 and context.go:619-637 of kong v1.16.1).
- kong did not apply the `envprefix` tag of a command field to the flags of that command. The help of `serve` printed `($ADDR)` for a flag with `env:"ADDR"` under a field with `envprefix:"SRV_SERVE_"`.
- ff parses the config file at every command level. Without `WithConfigIgnoreUndefinedFlags` the root command rejected the key of a `serve` flag with `parse config file: "addr": unknown flag`.
- The cobra, kong, urfave/cli and ff versions all accepted a config file with the misspelled key `adr`. Each printed the default address.

### cobra, pflag and viper

Paths below are relative to the module directories of cobra v1.10.2, pflag v1.0.10 and viper v1.21.0 [S1].

- A command is a `cobra.Command` literal. Its options are pflag calls on `Flags()` or `PersistentFlags()` (cobra command.go:54-146 and pflag string.go:36).
- cobra has no declaration for a positional argument. `Run` receives `args []string`. An `Args` validator checks their number (cobra command.go:93, command.go:136 and args.go:42-129).
- The cobra user guide fills one option from a flag, an environment variable and a config file with four calls and a struct tag. The calls are the flag, `BindPFlag`, `SetDefault` and `BindEnv` or `AutomaticEnv`. The tag is `mapstructure` (cobra site/content/user_guide.md:100-136 and 305-306, viper README.md:296-297 and 738-743). Each call repeats the key as a string.
- viper documents the order of precedence as overrides, flag, env, config, key/value store and default (viper viper.go:11-18). It lowercases every key (viper.go:717). It has no method that returns the source of a value.
- `Unmarshal` decodes only the keys that viper knows from a flag, a binding, a config file or a default. It drops an environment variable without such a key (viper.go:940 and 1891-1897).
- When two commands bind one key, viper keeps only the last flag (viper.go:1101 and 1177).
- The typed getters of viper convert with spf13/cast and discard its error. A malformed value becomes the zero value (viper.go:807 and cast v1.10.0 zz_generated.go:38-41).
- viper keeps a global instance (viper.go:57-61). cobra keeps global switches and initializers (cobra.go:42-66) and a global map of flag completions (completions.go:38-41).
- cobra generates completion scripts for bash, zsh, fish and PowerShell. Each script runs the hidden `__complete` command of the program to get its candidates (completions.go:31-34 and 231-241).
- cobra generates man pages, Markdown, YAML and reStructuredText with its `doc` package (doc/man_docs.go:28, doc/md_docs.go:125, doc/yaml_docs.go:27 and doc/rest_docs.go:138).
- No exported declaration in the three modules has a type parameter.

### kong

Line references in this section point into kong v1.16.1 [S1].

- One struct declares the whole program. A field tagged `cmd:""` is a command. A field tagged `arg:""` is a positional argument (build.go:187-195 and tag.go:270-275).
- kong derives a flag name from the field name in kebab case (camelcase.go:47-90). An environment variable comes from an `env` tag or from the `DefaultEnvars` option. `DefaultEnvars` uses the flag name without the command path (tag.go:280-282 and options.go:525-559).
- The JSON resolver looks up the flag name, then a camel-case variant, then a dotted path (resolver.go:33-63). It ignores the command. Two commands with flags of the same name read the same key.
- The parse runs reset, resolve, apply and validate in that order. A hook can run between two stages (kong.go:331-354).
- The help shows the environment variable of a flag (help.go:97-110). kong suggests a flag or a command within a Levenshtein distance of 2 (context.go:1160-1177).
- The module contains no completion code. A search for `complet`, `bash`, `zsh` and `fish` finds one tag in a test.
- A program can inject the arguments, stdout, stderr and the exit function (options.go:51-56 and 215-221). kong reads the environment with `os.LookupEnv` (model.go:399).
- kong finds a hook by its method name. A hook with a wrong signature fails only when it runs (callbacks.go:210-212 and 234).
- Nine of the 25 non-test files of kong import `reflect`. They use it 311 times. `go doc -short` prints 118 lines for the package.

### urfave/cli v3

For urfave/cli the citations refer to cli v3.14.0 and cli-altsrc v3.1.0 [S1].

- A command is a `cli.Command` literal with `Flags`, `Commands` and `Arguments` (cli command.go:44-140).
- urfave/cli derives nothing from a type. Each flag spells its name, its environment variable and its config key (cli value_source.go:109-143).
- The sources of a flag form an ordered chain. The command line comes first, then the first source with a value, then the default (cli flag_impl.go:137-171). The message of a parse error includes the source of the bad value (flag_impl.go:160-163).
- An action reads a value by name. A mistyped name returns the zero value (cli flag_string.go:56-63 and command.go:546-554).
- Completion covers bash, zsh, fish and PowerShell. Each script runs the program again with `--generate-shell-completion` (cli completion.go:20-46).
- cli-altsrc decodes JSON with yaml.v3 (cli-altsrc json/json_value_source.go:11). It ignores the errors of reading and parsing the file. It reads the file again for every lookup (uri_source_cache.go:54-72 and altsrc.go:116-118).
- urfave/cli has no enum type. It reads the environment with `os.LookupEnv` (cli value_source.go:110).

### ff v4

We cite ff at v4.0.0-beta.1 [S1].

- A flag is a typed call such as `DurationLong`. The environment variable is the flag name with a prefix. The config key is the flag name (parse.go:165-182 and 287-316). Neither contains the command.
- `AddStruct` declares flags from a struct. A field needs its name in a tag. The method skips nested structs (flag_set.go:642-805).
- The parse applies flags, then environment variables, then the config file, then defaults. It tracks the flags that an earlier stage set (parse.go:71-216).
- A config format is one function, `func(r io.Reader, set func(name, value string) error) error` (options.go:32). The JSON parser uses only the standard library. The TOML and YAML parsers need go-toml and yaml.v2.
- A subcommand reads the flags of its parent only after an explicit `SetParent` call (flag_set.go:143-146).
- ff has no completion and no required options. A mistyped command becomes a positional argument (command.go:116-128).
- ff never exits the process. `-h` returns `ErrHelp` to the program (flag_set.go:49-56).

### Other Go libraries

We read each library at the release in the dependency table [S1].

| Library | Declaration | Flags | Environment | Config files | Subcommands | Completion | R1 |
|---|---|---|---|---|---|---|---|
| go-arg | struct tags | yes | per field | no | yes | no | no |
| go-flags | struct tags | yes | per field | INI | yes | bash | no |
| ardanlabs/conf | one `conf` tag | derived | derived with a prefix | YAML in a subpackage | no | no | yes |
| acmd | command literals | standard `flag` | no | no | yes | stub | yes |
| aconfig | struct tags | derived | derived | JSON and others through decoders | no | no | no |
| caarlos0/env | struct tags | no | yes | no | no | no | yes |
| go-envconfig | `env` tag | no | yes | no | no | no | yes |
| koanf | load calls and getters | provider module | provider module | parser modules | no | no | no |
| cleanenv | struct tags | no | yes | YAML, JSON, TOML, EDN and .env | no | no | no |

- ardanlabs/conf derives `--port` and `PREFIX_PORT` from one field. It prefixes a nested field with the name of its parent (fields.go:257-301 and sources.go:42-75). It masks secrets when it prints the configuration (conf.go:128-181).
- aconfig takes the environment, the arguments and an `fs.FS` from the program (aconfig.go:69-90).
- go-envconfig reads the environment through the one-method interface `Lookuper` (envconfig.go:106-112).
- caarlos0/env collects every invalid field in one `AggregateError` (env.go:348-371).
- go-flags gives an INI value precedence over an environment variable (ini.go:598-599).

### Candidates found by the search

We searched catalogues, pkg.go.dev, GitHub and the web for further candidates [S5]. The search scored 13 CLI libraries and 8 configuration libraries with more than 200 stars and a push since 2025-01-01. None meets five or six requirements.

| Library | Release | Stars | Meets | Misses |
|---|---|---|---|---|
| integrii/flaggy | v1.8.0 of 2025-10-04 | 954 | R1, R3, R4 and R5 | R2 and R6 |
| alecthomas/kingpin/v2 | v2.4.0 of 2023-09-30 | 3,566 | R3, R4 and R5 | R1, R2 and R6 |
| gookit/gcli/v3 | v3.8.3 of 2026-07-16 | 381 | R3, R4 and R5 | R1, R2 and R6 |
| heetch/confita | v0.11.0 of 2025-12-08 | 506 | R2 and R6 | R1 for files, R3 and R5 |

fulldump/goconfig and xo/ox have fewer than 200 stars and come close.

- fulldump/goconfig v1.9.0 meets R1, R2, R4 and R6. It has no subcommands and no completion.
- The root package of xo/ox at bb2641a91b18 imports only the standard library and two packages of its own module [S3]. It declares flags with struct tags. It has a command tree, help and completion for bash, zsh, fish and PowerShell. A flag takes its environment and config keys from `ox.Key` (opts.go:606). The TOML and YAML loaders import go-toml and goccy/go-yaml. xo/ox has 35 stars and no tagged release.

### Libraries outside Go

| Library | Declaration | Help text | Environment | Config files | Completion | Test entry |
|---|---|---|---|---|---|---|
| clap 4.6.7 | derive macros on a struct | doc comments | `env` feature | none built in | bash, zsh, fish, PowerShell and Elvish | `try_parse_from` |
| Swift ArgumentParser 1.8.2 | property wrappers | `help:` strings | none | none | bash, zsh and fish | `parse(_:)` |
| Typer 0.27.3 | function signature | docstrings and `help=` | `envvar=` | none built in | bash, zsh, fish and PowerShell | `CliRunner.invoke` |
| picocli 4.7.7 | annotations | `description` | `${env:KEY}` in defaults | `PropertiesDefaultProvider` | bash and zsh | `execute` with writers |
| System.CommandLine 2.0 | option objects | `Description` | none | none. Response files | bash, zsh and PowerShell | `Invoke` with a writer |

- clap derives the long name from the field name in kebab case and the help from the doc comment [S6]. Its `ValueSource` type records the origin of a value as `DefaultValue`, `EnvVariable` or `CommandLine` [S6].
- Click returns the source of a parameter from `Context.get_parameter_source(name)` [S8]. Typer documents the order command line, environment, `default_map` and default [S8].
- picocli calls an `IDefaultValueProvider` between the command line and the declared default. `PropertiesDefaultProvider` reads such values from a properties file [S9].
- Swift ArgumentParser calls `validate()` on a command after parsing. A command checks rules across fields in that method [S7].
- System.CommandLine expands `@file` response files into arguments before it parses them [S10].
- The maintainers of clap, Typer and System.CommandLine leave config files to a separate layer in clap discussion #2763, Typer issue #86 and command-line-api issue #1058 [S6, S8, S10].

### What users report

We read 314 issues of cobra, viper, kong, urfave/cli and ff [S11]. A closed issue is often not a fixed one.

- viper closed 407 issues as not planned from 2025-10-01 to 2026-03-01. Each of the 45 that we opened contains the bot comment "This issue has been closed for inactivity."
- urfave/cli closed 25 issues as completed on 2026-09-27 and 2026-09-28. Issues #1905, #1273 and #2057 contain the comment "I'm closing everything whose last update was in 2025 or earlier".

These themes recur in more than one tracker:

| Theme | Issues |
|---|---|
| Environment and config values skip the checks that flag values get | viper #761 and #397, kong #224 and #596, urfave/cli #2041 and #849, ff #90 |
| Required checks run at the wrong stage or ignore other sources | cobra #655 and #2212, urfave/cli #1725 and #1247, viper #397, kong #516, ff #128 |
| Users cannot tell whether a value was set or by which source | cobra #23, viper #323 and #580, kong #365 and #420, urfave/cli #294 and #1263 |
| Flags after a positional argument fail where the parser builds on the `flag` package | urfave/cli #427 and #533, ff #100 and #124 |
| Global flags are not available in subcommands or their help | cobra #412 and #921, urfave/cli #734 and #585, ff #76 and #97 |
| Shell completion is missing or incomplete | cobra #107 and #350, kong #43, urfave/cli #1905 and #855, ff #132 |
| Positional arguments have no declaration | cobra #378 and #571, urfave/cli #1074 and #2392 |
| Global state blocks parallel tests | cobra #770 and #1290, viper #1855 and #268, urfave/cli #1242 and #2176 |
| Dependencies are too heavy | cobra #1597, viper #887 and #707, ff #154, urfave/cli #2045 |
| Backward compatibility blocks a fix | cobra #340, #706 and #1708, viper #380, urfave/cli #1239 and #1856, ff #55 |

The maintainers stated these limits:

- kong: completion is open because "I would love it, but just haven't had the time" (#43). Environment variables "don't go through the normal resolver system" (#390).
- ff: required flags are not "something that `ff` will or should natively support" (#128). The completion request #132 is still open.
- viper: `Unmarshal` skips automatic environment variables because they "are not in the list of existing keys" (#761). On concurrency bep wrote "I do not intend to do anything about this" (#268).
- urfave/cli: the library "does not advertise concurrency, immutability, or thread-safety" (#1242).
- cobra: the usage output on errors remains because "any change might break people" (#340).

### Where sources disagree

- The kong README states that a resolver provides the `env` tag (README.md:728). The code reads the environment when it resets a flag. A config resolver then overwrites the value (model.go:396-413 and context.go:619-637). Our run confirms the code.
- The doc comment of `WithEnvVarSplit` in ff states that the option turns on environment parsing (options.go:171-179). The parse does not check it (parse.go:94).
- The package comment of aconfig calls it "dependency-free" (doc.go:9). The package imports mitchellh/mapstructure (parser.go:11).
- The cli-altsrc README calls `yaml.YAML(key, configFiles...)` (README.md:26). The function takes one source (yaml/yaml_value_source.go:10).
- `SupportedExts` of viper lists hcl, ini and properties (viper.go:304). The default codec registry decodes only YAML, JSON, TOML and dotenv (encoding.go:166-180).

## What we conclude for go-cli

The rest of this document is our conclusion, drawn from the findings.

### Build rather than adopt

- kong lacks completion, an injectable environment and names scoped by the command. It also applies config files over the environment. Changing that precedence would break every program that sets one flag through a file and an environment variable.
- ff v4 has no stable release. It lacks completion, required options and names scoped by the command. Its owner declines required flags (#128).
- Each flag of urfave/cli repeats its name in its environment variable and its config key. go-cli exists to remove that repetition.
- cobra with viper needs 92 lines and 30 packages outside the standard library for the reference program. It repeats each key in up to five places.

### Mechanisms to take

| Mechanism | Prior art | Evidence |
|---|---|---|
| One struct per command declares the flags, environment variables, config keys and help | kong, clap derive and ardanlabs/conf | 35 lines with kong against 92 with cobra and viper |
| Names derived from the field and the command path, with a tag to override | ardanlabs/conf derives nested names. No library derives the command path | kong and urfave/cli repeat the environment variable per field |
| A staged parse with a set of the values that an earlier stage set | kong reset, resolve, apply and validate, and ff parse.go:71-216 | One precedence for every source |
| A record of the source of each value | clap `ValueSource` and Click `get_parameter_source` | Requested in cobra, viper, kong and urfave/cli |
| The same conversion and validation for every source | none of the Go libraries | Environment and config values skip checks in viper, kong, urfave/cli and ff |
| Required checks that accept a value from any source | none of the Go libraries | cobra checks only the command line (command.go:1192) |
| A config format as one function type | ff options.go:32 | go-cli does not import a YAML or TOML decoder |
| Unknown config keys reported as errors | none of the four library versions | Each accepted the key `adr` |
| An injectable environment, file system, streams and exit | go-envconfig `Lookuper`, aconfig, ff `WithFilesystem`, and kong `Writers` and `Exit` | kong, ff, urfave/cli and cobra read the process environment |
| Dynamic completion through a hidden command | cobra `__complete` and urfave/cli `--generate-shell-completion` | kong and ff have no completion |
| A `Validate` method on the options type | kong and Swift ArgumentParser | Rules across fields need no tag language |
| Custom types through `encoding.TextUnmarshaler` | kong, go-arg, ardanlabs/conf and aconfig | A type parses without registration |
| Errors returned to the program, with no `os.Exit` in the library | ff `ErrHelp`, kong `ExitCoder` and acmd `ErrCode` | The program decides the exit code and the output |
| Help that shows the default, the environment variable and the config key | go-arg shows the first two. kong shows the environment variable | urfave/cli and ff omit config keys |
| One report of every invalid value | caarlos0/env `AggregateError` | A user fixes every bad value in one run |

### Mechanisms to avoid

- viper, urfave/cli and koanf read values by key strings that a program repeats. A typo returns the zero value.
- viper keeps a global instance. cobra keeps global switches. urfave/cli exits through the package variable `OsExiter`.
- kong finds hooks by method name. Only a run checks their signatures.
- kong and go-flags apply a config file over the environment.
- ff parses the config file again at each command level.
- viper discards the errors of cast when it converts a value.

### Limits of the standard library

- The standard library of Go 1.27.2 decodes JSON, XML and CSV. It does not decode YAML, TOML or INI [S12]. A program that reads YAML or TOML has to supply the decoder.
- The `flag` package treats one dash and two dashes alike. It stops at the first argument that is not a flag [S12]. A library needs its own scanner to accept clusters of short flags or flags after a positional argument. kong, pflag and ff each have one.

### Decisions for the RFC

1. Build go-cli. Adopting kong or ff becomes cheaper if a release of either adds completion, the environment over config files and names scoped by the command.
2. Declare each command with one struct type. Derive the flag, the environment variable and the config key from the field and the command path. A tag overrides each derived name.
3. Apply the precedence flag, environment variable, config file and default to every value. Record the source of each value. Run the same conversion, validation and required check for every source.
4. Build in JSON and accept other formats through a parser function that the program supplies.
5. Generate completion for bash, zsh, fish and PowerShell through a hidden command of the program.
6. Take the arguments, an environment lookup, an `fs.FS` and the standard streams in one entry point. The library does not keep package state or exit the process.

## What we could not establish

- We measured one program with two commands. A program with deeper nesting or many shared options can change the ranking of code lines.
- We did not measure parse time or allocations.
- We did not run the versions on macOS or Windows.
- We read only the core of koanf. Its providers are separate modules.
- For the libraries outside Go we read the documentation and selected source files. We did not read their full source.
- The issue sample is ranked by reactions and search terms. It shows what users report. It does not show how often a problem occurs.
- We did not attribute the binary sizes to packages. The sizes depend on Go 1.27.2 and the build flags.
- We did not find a statement by the kong maintainers on whether a config file should take precedence over the environment.

## What would change this answer

- A kong release with completion, an injectable environment, the environment over config files and names scoped by the command would make adoption cheaper than building.
- A stable ff v4 release with completion, required options and names scoped by the command would do the same.
- A tagged release of xo/ox that loads config files with the standard library alone would make xo/ox a candidate.
- A YAML or TOML decoder in the standard library would change the decision on formats.

## Sources

| # | Source | What it is | Retrieved | What it supports |
|---|---|---|---|---|
| S1 | The module zips of the libraries at the releases in the dependency table, from proxy.golang.org, and xo/ox at v0.0.0-20260927161223-bb2641a91b18 | Source code, read in the module cache | 2026-10-12 | Every path and line reference to a Go library |
| S2 | `@v/list` and `@v/<version>.info` of each module on proxy.golang.org | Module metadata | 2026-10-12 | Release dates. ff/v4 has only alpha and beta releases |
| S3 | Our dependency listing with `go list -deps` of each package in a scratch module under Go 1.27.2 on linux/amd64 | Our measurement | 2026-10-12 | The dependency table and the imports of xo/ox and flaggy |
| S4 | Our reference program `srv` with the standard library, cobra and viper, urfave/cli and cli-altsrc, ff and kong. A runner executes the 11 scenarios in a clean environment | Our measurement | 2026-10-12 | Code lines, binary sizes, scenario results, the misspelled key and the four behaviours |
| S5 | The search for candidates in the awesome-go README, pkg.go.dev search, `gh search repos`, the GitHub GraphQL API, web search and Exa | Search results, then the go.mod, README and source of each repository | 2026-10-12 | The candidate table and the two libraries below 200 stars |
| S6 | [clap 4.6.7 derive reference](https://docs.rs/clap/4.6.7/clap/_derive/index.html), [ValueSource](https://docs.rs/clap/4.6.7/clap/parser/enum.ValueSource.html), [clap_complete 4.6.11](https://docs.rs/clap_complete/4.6.11/clap_complete/) and [discussion #2763](https://github.com/clap-rs/clap/discussions/2763) | Maintainers' documentation | 2026-10-12 | clap declaration, help, sources, completion and config files |
| S7 | [Swift ArgumentParser 1.8.2 documentation](https://swiftpackageindex.com/apple/swift-argument-parser/1.8.2/documentation/argumentparser) | Maintainers' documentation | 2026-10-12 | Property wrappers, `validate()` and completion |
| S8 | [Typer 0.27.3 documentation](https://typer.tiangolo.com/tutorial/arguments/envvar/), [Typer issue #86](https://github.com/fastapi/typer/issues/86) and [Click API](https://click.palletsprojects.com/en/stable/api/) | Maintainers' documentation and comments | 2026-10-12 | Typer declaration and order. Click `get_parameter_source` |
| S9 | [picocli 4.7.7 manual](https://picocli.info/) | Maintainers' documentation | 2026-10-12 | Default providers, inherited options and completion |
| S10 | [System.CommandLine syntax](https://learn.microsoft.com/en-us/dotnet/standard/commandline/syntax) and [command-line-api issue #1058](https://github.com/dotnet/command-line-api/issues/1058) | Maintainers' documentation and comments | 2026-10-12 | Response files, global options and config files |
| S11 | Issues of [spf13/cobra](https://github.com/spf13/cobra/issues), [spf13/viper](https://github.com/spf13/viper/issues), [alecthomas/kong](https://github.com/alecthomas/kong/issues), [urfave/cli](https://github.com/urfave/cli/issues) and [peterbourgon/ff](https://github.com/peterbourgon/ff/issues), read with `gh issue view` | Issue reports and maintainer comments | 2026-10-12 | The themes and the stated limits |
| S12 | `go doc flag` and `go list std` of Go 1.27.2 | Standard library documentation | 2026-10-12 | The `flag` package rules and the decoders of the standard library |

S1 to S5 and S12 are our own reading and measurement. S6 to S11 are statements by the maintainers of each project.

## What we searched

| Search | Tool | Date | Useful |
|---|---|---|---|
| The source of 16 Go libraries for declaration, derivation, sources, commands, validation, help, completion, injection, generics and reflection | Module cache and grep | 2026-10-12 | Yes |
| "Command Line", "Standard CLI" and "Configuration" sections of awesome-go | curl of the README | 2026-10-12 | Yes |
| "cli struct tags", "flags env config struct", "command line struct tags subcommands completion" and "cli zero dependencies" | pkg.go.dev search | 2026-10-12 | Yes, except the third query with 0 results |
| "cli flags struct tags", "zero dependencies cli", "flags env config struct", "derive flags from struct", "struct tags command line" and "command line parser struct" | `gh search repos --language go` | 2026-10-12 | Yes, after a rerun with separate keywords |
| 18 phrases from "cli library" to "struct cli" with more than 200 stars | `gh search repos` | 2026-10-12 | Yes |
| Topics such as command-line-parser, argument-parser, cli-framework, struct-tags, subcommands and shell-completion | `gh search repos --topic` | 2026-10-12 | Yes |
| "go cli library without cobra boilerplate", "generate flags from struct go" and six more phrases | Web search | 2026-10-12 | Yes |
| A library that derives flags, environment variables and config keys from one struct with subcommands, help and completion | Exa | 2026-10-12 | No new candidate |
| Kong alternatives | libhunt | 2026-10-12 | No new candidate |
| Reactions, about 25 problem terms per tracker, 18 frustrated phrases and the refusal labels and phrases of each tracker | `gh issue list` on five repositories | 2026-10-12 | Yes |
| clap, Swift ArgumentParser, Typer, Click, picocli and System.CommandLine | Official documentation, tagged sources and issue trackers | 2026-10-12 | Yes |

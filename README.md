<!--
  ~ Copyright Stealth Scale B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# go-cli

`go.stealthscale.io/cli` runs command-line programs whose options are the fields of Go structs. The
package derives the flag, the environment variable and the config key of each field. It converts
every source to the type of the field and applies the sources in one order. A handler receives its
options as typed values.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.stealthscale.io/cli"
)

type Global struct {
	Config cli.ConfigFile `help:"Config file in JSON or INI." default:"srv.json" short:"c"`
}

type Serve struct {
	Addr    string        `help:"Address to listen on." default:":8080" short:"a"`
	Timeout time.Duration `help:"Timeout of a request." default:"30s"`
	Origins []string      `help:"Origins that may connect."`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

func app() *cli.App[Global] {
	a := cli.New[Global]("srv", "Serve the store.")
	a.Command("serve", "Run the server.", serve)
	return a
}

func serve(_ context.Context, run *cli.Run[Global, Serve]) error {
	s := run.Config()
	_, err := fmt.Fprintf(run.Stdout(), "listen on %s with a timeout of %s for %q\n", s.Addr, s.Timeout, s.Origins)
	return err
}
```

The program does not contain code for its flags, its environment variables or its config file. The
help lists them for each option:

```console
$ srv serve --help
Usage: srv serve [options]

Run the server.

Options:
  -a, --addr string         Address to listen on.      default ":8080", env SRV_SERVE_ADDR, key serve.addr
      --origins string,...  Origins that may connect.  env SRV_SERVE_ORIGINS, key serve.origins
      --timeout duration    Timeout of a request.      default "30s", env SRV_SERVE_TIMEOUT, key serve.timeout

Global options:
  -c, --config file  Config file in JSON or INI.                                             default "srv.json", env SRV_CONFIG
      --flag-debug   Print the value and the source of each option before the command runs.  env SRV_FLAG_DEBUG
  -h, --help         Show the help.

$ SRV_SERVE_TIMEOUT=5s srv serve --addr :9000 --origins https://app.example.com
listen on :9000 with a timeout of 5s for ["https://app.example.com"]

$ srv serve --adr :9000
srv: serve: unknown flag --adr (did you mean --addr?)
```

## Install

```sh
go get go.stealthscale.io/cli
```

Every module of go-cli requires Go 1.27.2 or later. The package `cli` imports only the standard
library. The YAML and TOML formats of config files and the Lip Gloss renderer are modules of their
own:

```sh
go get go.stealthscale.io/cli/yaml
go get go.stealthscale.io/cli/toml
go get go.stealthscale.io/cli/charm
```

## Options

Each exported field of an options struct is an option. The package derives three names from the
name of the field:

| Name | Rule | `Timeout` in `srv serve` |
|---|---|---|
| Flag | The name of the field in kebab case | `--timeout` |
| Environment variable | The app, the path of the command and the flag in upper case with an underscore for each dash | `SRV_SERVE_TIMEOUT` |
| Config key | The path of the command and the flag joined with dots | `serve.timeout` |

Kebab case splits the name of a field into words, so `DryRun` becomes `--dry-run` and `HTTPPort`
becomes `--http-port`. The variable and the key of a global option leave out the command. The global
option `Config` has the variable `SRV_CONFIG`. The path of a subcommand contains the names of its
command and of itself, so the option `Steps` of `srv db migrate` has the variable
`SRV_DB_MIGRATE_STEPS` and the key `db.migrate.steps`.

The name of an app, a command, a group or an option has words of lower-case ASCII letters and
digits. Single dashes join the words. The first word starts with a letter.

## Tags

| Tag | Effect | Example |
|---|---|---|
| `help` | Sets the text of the option in the help | `help:"Address to listen on."` |
| `default` | Sets the value when no other source sets the option | `default:":8080"` |
| `enum` | Limits the option to a comma-separated list of values | `enum:"text,json"` |
| `required` | Fails a run when no source sets the option | `required:""` |
| `arg` | Makes the field of a command a positional argument | `arg:""` |
| `name` | Replaces the last word of the flag, the variable and the key | `name:"retain"` |
| `short` | Adds a short flag of one ASCII letter other than `h` | `short:"a"` |
| `env` | Replaces the whole environment variable or removes it with `-` | `env:"API_TOKEN"` |
| `key` | Replaces the whole config key or removes it with `-`. Commands that declare one key share it | `key:"database.url"` |
| `prefix` | Replaces the name of a group or removes the group when it is empty | `prefix:"backend"` |
| `secret` | Masks the value in the help and in the output of `--flag-debug` | `secret:""` |

The tags `required`, `arg` and `secret` take effect by their presence.

## Commands

`App.Command` adds a command with the struct of its options and its handler. Commands that embed one
struct have the same options. An embedded struct adds its options without a group:

```go
type Target struct {
	Env    string `help:"Environment to act on." required:"" enum:"dev,prod" short:"e"`
	Region string `help:"Region of the environment." default:"eu-west-1"`
}

type Plan struct {
	Target
}

type Apply struct {
	Target
	Yes bool `help:"Apply the changes to prod." short:"y"`
}

a := cli.New[Global]("deploy", "Deploy the service.")
a.Command("plan", "Print the changes of a deployment.", plan)
a.Command("apply", "Apply the changes of a deployment.", apply)
```

`App.Command` returns the command, and its method `Command` adds a subcommand. A command has a
handler, subcommands or both. The options of a command apply to its subcommands as well. A
subcommand reads them through an embedded pointer to their type. The package sets the pointer:

```go
type Registry struct {
	URL     string `help:"URL of the registry." default:"registry.example.com"`
	User    string `help:"User of the registry." default:"deploy"`
	session string
}

type Push struct {
	*Registry
	Image string `help:"Image to push." arg:""`
}

registry := a.Command("registry", "Count the images of the registry.", images)
registry.Use(signIn)
registry.Command("push", "Push an image to the registry.", push)
```

A hook runs before the handler of its command and of every subcommand of that command. It calls
`next` to run the handler, so its code after `next` runs after the handler returns or fails. A hook
passes a value to the handler through the options of its command:

```go
func signIn(ctx context.Context, run *cli.Run[Global, Registry], next func(context.Context) error) error {
	r := run.Config()
	r.session = r.User + "@" + r.URL
	return next(ctx)
}
```

`App.Use` adds a hook that every run calls. `App.Handle` sets the handler of the app, so a program
such as `wc` runs without a command.

A handler reads the standard input of its run through `Run.Stdin`, so a pipe passes data to a
command. The program `wc` of the example [`14-testing`](examples/14-testing) counts the words of
its standard input:

```console
$ printf 'one two\nthree\n' | wc --words
3
```

The command line follows the rules of GNU:

- A long flag takes its value after `=` or in the next argument.
- Short flags combine in one argument, such as `-vq`. A short flag with a value takes the rest of
  its argument or the next argument, such as `-p8080` or `-p 8080`.
- A global flag can come anywhere. A flag of a command can come anywhere after the name of the
  command, so both `deploy registry --url r.internal push` and `deploy registry push --url r.internal`
  set the URL of the registry.
- After a command with subcommands, an argument that is not a flag is the name of a subcommand.
- The argument `--` ends the flags.
- The flags `-h` and `--help` write the help of the app or of the last command of the path.

A run fails for an unknown command, flag or config key. The package suggests a similar command, long
flag or key when the app declares one:

```console
$ deploy aply -e dev
deploy: unknown command "aply" (did you mean apply?)

$ deploy registry pusj app:v2
deploy: registry: unknown command "pusj" (did you mean push?)
```

## Positional arguments

A field with the tag `arg` is a positional argument. Only the command line sets a positional
argument. The package assigns the arguments of the command line to the fields in the order of the
fields:

- An argument without a default is required.
- An argument with a default is optional. It must come after the required arguments.
- A slice takes the rest of the arguments and must come last. A slice without a default takes at
  least one argument.
- A lone `-` is an argument and not a flag.

```go
type Add struct {
	Archive string   `help:"Archive that receives the files." arg:""`
	Files   []string `help:"Files to add." arg:""`
	Level   int      `help:"Level of the compression." default:"6" short:"l"`
}
```

```console
$ pack add --help
Usage: pack add [options] ARCHIVE FILES...

Add files to an archive.

Arguments:
      ARCHIVE  Archive that receives the files.
      FILES    Files to add.

Options:
  -l, --level int  Level of the compression.  default "6", env PACK_ADD_LEVEL

Global options:
      --flag-debug  Print the value and the source of each option before the command runs.  env PACK_FLAG_DEBUG
  -h, --help        Show the help.
```

Flags can come between the arguments. `pack add logs app.log -l 9 db.log` sets `Files` to `app.log`
and `db.log`.

## Input files

An option of the type `cli.InputFile` is the path of a file that a command reads. The path `-` is
the standard input. `Run.Open` opens the file of such an option, so one handler reads a file, a
`-` and a pipe:

```go
type Parse struct {
	Doc cli.InputFile `help:"JSON document, or - for the standard input." arg:"" default:"-"`
}

func parse(_ context.Context, run *cli.Run[Global, Parse]) (err error) {
	f, err := run.Open(run.Config().Doc)
	if err != nil {
		return fmt.Errorf("main: parse: %w", err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	var v map[string]any
	if err = json.NewDecoder(f).Decode(&v); err != nil {
		return fmt.Errorf("main: parse %s: %w", run.Config().Doc, err)
	}
	_, err = fmt.Fprintln(run.Stdout(), v["name"])
	return err
}
```

```console
$ cat test.json | tool parse
store
$ tool parse test.json
store
$ tool parse missing.json
tool: main: parse: open missing.json: no such file or directory
```

The package does not open an input file while it reads the command line. A handler opens each
file of a slice such as `[]cli.InputFile` and closes it before the next, so a long list keeps one
file open at a time. The help shows the value of an input file as `file`. The command `add` of the
example [`04-arguments`](examples/04-arguments) reads its files this way.

`Run.Open` opens a file through `Env.Open`. `cli.OS` opens the files of the process, and a test
gives a run its files through `Env.Open`, as the section Testing shows.

## Groups

A named struct field is a group. The name of the group starts the flag, the variable and the key of
each option inside it. Groups nest, so the field `Cache.Redis.Addr` has the flag
`--cache-redis-addr`. The tag `prefix` changes a group:

- A prefix replaces the name of the group.
- An empty prefix removes the group.
- A prefix on an embedded struct makes the struct a group.

```go
type TLS struct {
	Cert string `help:"Certificate file."`
	Key  string `help:"Key file."`
}

type Proxy struct {
	TLS   TLS                      // --tls-cert and --tls-key
	Admin TLS `prefix:"admin-tls"` // --admin-tls-cert and --admin-tls-key
}
```

In the command `serve` of the app `proxy`, the field `TLS.Cert` has the flag `--tls-cert`, the
variable `PROXY_SERVE_TLS_CERT` and the key `serve.tls.cert`.

## Types

The package converts the text of a source by the first rule that matches the type of the field:

1. A conversion that `App.Convert` registers.
2. A built-in conversion. The package parses `url.URL` with `url.Parse`, `time.Location` with
   `time.LoadLocation` and `net.IPNet` with `net.ParseCIDR`. It reads `fs.FileMode` as an octal
   number such as `0644`.
3. The method `UnmarshalText` of a pointer that implements `encoding.TextUnmarshaler`, such as
   `*time.Time`, `*netip.Addr` and `*slog.Level`.
4. The method `Set` of a pointer that implements `flag.Value`.
5. `time.ParseDuration` for `time.Duration` and `strconv` for the kinds string, bool, int, uint and
   float.

The rules apply to the elements of slices, maps and pointers:

- A slice takes values that commas separate, such as `--origins a,b`.
- A map takes entries of the form `key=value`.
- A pointer is nil until a source sets it.
- A repeated flag of a slice or a map adds to its earlier values.

`App.Convert` registers the conversion of a type of your own:

```go
type Celsius float64

a.Convert(func(s string) (Celsius, error) {
	number, ok := strings.CutSuffix(s, "C")
	if !ok {
		return 0, errors.New("want a number with the unit C")
	}
	f, err := strconv.ParseFloat(number, 64)
	return Celsius(f), err
})
```

A run reports the error of the conversion after the input and its source:

```console
$ units check --target 70F
units: check: invalid --target "70F" from flag --target: want a number with the unit C
```

## Config files

An option of the type `cli.ConfigFile` is the path of a config file. The options of the app and of
the commands of a path contain one config file at most. A run reads the file after the defaults and
before the environment variables. Each object of the file contains the keys of a command or of a
group:

```json
{
  "serve": {
    "addr": ":9000",
    "timeout": "10s",
    "origins": ["https://app.example.com"]
  }
}
```

The object of a subcommand is inside the object of its command, so the option `Steps` of
`srv db migrate` has the key `steps` in `{"db": {"migrate": {"steps": 3}}}`.

The help notes the config key of an option only for a command whose path declares a config file.

The extension of the path selects the format. `cli.New` registers JSON for `.json` and INI for
`.ini`. `App.Format` registers a format for other extensions. The modules `yaml` and `toml` register
their formats with one call:

```go
a := cli.New[Global]("srv", "Serve the store.")
yaml.Register(a)           // YAML for .yaml and .yml
a.Format(cli.INI, ".conf") // INI for .conf
```

A run skips a missing file at the default path, so the program also runs without a config file. A
missing file at a path from the environment or the command line fails the run. `srv -c '' serve`
does not read a file because its path is empty. A key that does not match an option also fails the
run:

```console
$ cat srv.json
{"serve": {"adr": ":9000"}}
$ srv serve
srv: serve: unknown key serve.adr in srv.json (did you mean serve.addr?)
```

A config file can also contain keys that the program decodes itself, such as the plans of a
workspace. `App.IgnoreKeys` registers them. A run skips each such key and every key below it, and
the handler reads them from the file at the path of the option:

```go
a.IgnoreKeys("version", "plans")
```

`App.LocateConfig` registers a function that returns the path of the config file when neither the
flag nor the environment variable sets it, such as a search from a directory upward. The function
receives the global options and the `Env` of the run. A search that opens its candidates through
`Env.Open` reads only the files of a test:

```go
a.LocateConfig(func(g *Global, env *cli.Env) (string, error) {
	return search(env, g.Chdir, ".acme.json")
})
```

`--flag-debug` shows a located path with the source `located`. The example
[`16-workspace`](examples/16-workspace) searches for its workspace file and decodes its plans
itself.

## Sources

The value of an option comes from the last source in this list that sets it:

1. The tag `default`, or the zero value of the field.
2. The config file.
3. The environment variable.
4. The command line.

Every source goes through the same conversion and the same checks.

With the global flag `--flag-debug` or the variable `SRV_FLAG_DEBUG`, a run writes the value and the
source of every option to the standard error before the hooks and the handler run. The run prints
`***` for the value of a secret option:

```console
$ cat srv.json
{"serve": {"addr": ":9000", "timeout": "10s", "origins": ["https://app.example.com"]}}
$ SRV_SERVE_TIMEOUT=5s srv serve --origins https://admin.example.com --flag-debug
  OPTION     VALUE                          SOURCE
  --config   "srv.json"                     default
  --addr     ":9000"                        key serve.addr of srv.json
  --origins  ["https://admin.example.com"]  flag --origins
  --timeout  5s                             env SRV_SERVE_TIMEOUT
listen on :9000 with a timeout of 5s for ["https://admin.example.com"]
```

`Run.Source` returns the source of an option to its handler:

```go
src, _ := run.Source(&run.Config().Timeout) // env SRV_SERVE_TIMEOUT
```

## Errors and exit statuses

`App.Main` writes the message of an error to the standard error after the name of the app and
returns the exit status for `os.Exit`:

| Status | Constant | Cause |
|---|---|---|
| 0 | `cli.ExitOK` | The handler succeeds or the run writes the help |
| 1 | `cli.ExitFailure` | The handler fails or the declarations of the app are invalid |
| 2 | `cli.ExitUsage` | The command line, an environment variable or the config file is invalid |

`App.Execute` returns the error instead. It returns a `*cli.Error` for an invalid command line,
environment variable or config file. The error wraps one of the sentinels `ErrUnknownCommand`,
`ErrUnknownFlag`, `ErrUnknownKey`, `ErrMissing`, `ErrInvalidValue`, `ErrUnexpectedArgument` and
`ErrConfigFile`. Its fields `Command`, `Input`, `Value` and `Source` contain the location of the
fault. `Error.Message` returns the text that `App.Main` writes after the name of the app. The
example [`examples/12-errors`](examples/12-errors) handles these errors in place of `App.Main`.

`App.Execute` checks the options of every command before it reads the command line. A wrong tag of
one command fails a run of any command with an error that wraps `ErrDeclaration`, so the first test
of the app finds it.

## Output

The renderer of an app writes the help, the message of a failed run and the table of
`--flag-debug`. `cli.New` sets the renderer `cli.Plain`, which writes the plain text of the other
sections of this README. `App.Render` registers another `cli.Renderer`. The module
`go.stealthscale.io/cli/charm` renders the output with Lip Gloss:

```go
a := cli.New[Global]("srv", "Serve the store.")
charm.Register(a)
```

Its help puts the notes of each option below the text of the option:

```console
$ srv serve --help | cat
Run the server.

USAGE
  srv serve [options]

OPTIONS
  -a, --addr string         Address to listen on.
                            default ":8080" · env SRV_SERVE_ADDR
                            key serve.addr
      --origins string,...  Origins that may connect.
                            env SRV_SERVE_ORIGINS · key serve.origins
      --timeout duration    Timeout of a request.
                            default "30s" · env SRV_SERVE_TIMEOUT
                            key serve.timeout

GLOBAL OPTIONS
  -c, --config file         Config file in JSON or INI.
                            default "srv.json" · env SRV_CONFIG
      --flag-debug          Print the value and the source of each option before
                            the command runs.
                            env SRV_FLAG_DEBUG
  -h, --help                Show the help.
```

The help wraps at the width in the environment variable `COLUMNS`, at most 100 columns. Without
`COLUMNS` it wraps at 80 columns. A terminal receives the help with colors and styles, and a failure
with a red badge, the suggestion and a hint to the help:

```console
$ srv serve --adr :9000
 ERROR  unknown flag --adr
        Did you mean --addr?
        Run srv serve --help for the usage.
```

A pipe or a file receives the help without styles, and the failure and the table of `cli.Plain`.
`NO_COLOR` removes the colors in a terminal, and `CLICOLOR_FORCE` writes the styles to any writer.
A program changes the styles of `charm.DefaultTheme` and passes the theme to `charm.New`:

```go
t := charm.DefaultTheme()
t.Flag = t.Flag.Foreground(lipgloss.Green)
a.Render(charm.New(&t))
```

A renderer of your own implements the three methods of `cli.Renderer`. It can embed `cli.Plain` to
replace only some of them. `App.RenderFor` registers a function that selects the renderer of each
run from its global options, so an option such as `--format json` selects the output of the help,
the failures and `--flag-debug`:

```go
a.RenderFor(func(g *Global) cli.Renderer {
	if g.Format == "json" {
		return jsonRenderer{}
	}
	return cli.Plain{}
})
```

A run applies the valid values of its sources before the function selects the renderer, so the
usage error of a command line with `--format json` is JSON too. The flags count up to the first
fault of the command line. The example [`15-renderer`](examples/15-renderer) writes JSON this way.

`App.Main` writes the failure of a run to the standard error and returns 1 or 2. A program with
exit statuses or streams of its own calls `App.ExecuteRenderer`, which returns the renderer of the
run beside its error. `App.Failure` returns the failure that `App.Main` writes for that error:

```go
r, err := a.ExecuteRenderer(ctx, env)
if f := a.Failure(err); f != nil {
	_ = r.Error(env.Stdout, env.Environ(), f)
}
```

The example `15-renderer` exits with 64 for a usage error this way, and writes a failure in JSON
to the standard output.

`Run.Environ` returns the environment of a run to its handler, so the handler can pass the
variables of its `Env` to a terminal library such as Bubble Tea.

## Testing

The package `go.stealthscale.io/cli/clitest` runs a program in a test. `clitest.Main` runs one
command line and returns the exit status and the output of the run:

```go
func TestServe(t *testing.T) {
	res := clitest.Main(t.Context(), app().Main, &clitest.Input{
		Args:  []string{"serve"},
		Vars:  map[string]string{"SRV_SERVE_TIMEOUT": "5s"},
		Files: clitest.Files{"srv.json": `{"serve": {"addr": ":9000"}}`},
	})
	if res.Status != cli.ExitOK {
		t.Fatalf("Main returns %d: %s", res.Status, res.Stderr)
	}
	if want := "listen on :9000 with a timeout of 5s for []\n"; res.Stdout != want {
		t.Errorf("serve writes %q, want %q", res.Stdout, want)
	}
}
```

A run reads only the arguments, the environment variables, the files and the standard input of its
`Input`. A config file or an input file outside `Input.Files` does not exist for the run, so the
result does not depend on the files of the machine. `os.DirFS("testdata")` gives a run the files of
a directory. Tests that call `clitest` can run in parallel.

`clitest.Transcript` runs a list of command lines and returns the runs as a terminal shows them,
with the files that each run opened. A test compares the transcript with a golden file, such as
with the package `golden` of `go.dokimi.dev/assert`:

```go
runs := []clitest.Input{
	{Comment: "The help of the command serve.", Args: []string{"serve", "--help"}},
	{Comment: "An unknown flag suggests a known one.", Args: []string{"serve", "--adr", ":9000"}},
}
golden.Match(t, "transcript.txt", clitest.Transcript(t.Context(), "srv", app().Main, runs...), golden.ShouldUpdate())
```

`clitest.NewEnv` returns the `Env` of an `Input` for a test that calls `App.Execute` itself, or a
handler through `cli.NewRun`:

```go
env, out := clitest.NewEnv(&clitest.Input{Stdin: "one two\n"})
run, err := cli.NewRun(&Global{}, &Serve{Addr: ":9000"}, env)
```

`FailStdin` and `FailStdout` make every read of the standard input or every write to the standard
output fail, so a test can check the error paths of a handler. The examples test themselves with
`clitest`.

An `Env` of your own builds `Lookup` and `Environ` from one map with `cli.Vars` and `cli.Environ`.
Its `Open` has the signature of `fs.FS.Open`, so the `Open` method of a file system such as
`fstest.MapFS` gives a run its files.
`App.Execute` and `App.Main` are safe for concurrent use once the registrations have returned, so
parallel tests can share one app.

## Modules

| Module | Contents | Imports outside the standard library |
|---|---|---|
| `go.stealthscale.io/cli` | The package `cli` with the formats JSON and INI | None |
| `go.stealthscale.io/cli/yaml` | The YAML format | `go.yaml.in/yaml/v3` |
| `go.stealthscale.io/cli/toml` | The TOML format | `github.com/BurntSushi/toml` |
| `go.stealthscale.io/cli/charm` | The renderer of the output with Lip Gloss | `charm.land/lipgloss/v2`, `github.com/charmbracelet/colorprofile` |

## Examples

[`examples/`](examples) contains 16 runnable programs. The test of each example compares its output
with a golden transcript. [examples/README.md](examples/README.md) lists what each example shows.

## Development

`make check` runs the formatters, the linters and the tests of every module. CI runs the same
targets on every pull request. [CONTRIBUTING.md](CONTRIBUTING.md) describes how to propose a change.

## Security

Report a vulnerability as [SECURITY.md](SECURITY.md) describes. Do not open a public issue for it.

## License

go-cli is licensed under MIT. See [LICENSE](LICENSE).

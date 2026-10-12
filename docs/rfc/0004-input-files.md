---
rfc: 0004
title: Input files and the standard input
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
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

# RFC-0004: Input files and the standard input

## Summary

An option of the type `cli.InputFile` is the path of a file that a command reads. The path `-` is the standard input. `Run.Open` opens the file of such an option, so one handler reads the input of `tool parse test.json`, `tool parse -` and `cat test.json | tool parse`. The run opens every file through `Env.Open`, which replaces `Env.ReadFile`. A test gives a run its input files and its config files through one file system, and the transcript of `clitest` shows each file that a run opens.

## Motivation

A handler reads a pipe through `Run.Stdin` today:

```console
$ cat test.json | tool parse
store
```

Command-line programs also take the path of the input as an argument, with `-` for the standard input, as `cat` and `wc` do. The package `cli` does not open such a path. Each handler opens the file itself through `os.Open`, and treats `-` itself. A test of that handler then reads the files of the machine, because `os.Open` does not go through the `Env` of the run.

The libraries of the same kind open an input file before the handler runs:

- kong v1.16.1 opens a field of the type `*os.File` with `os.Open` while it parses the command line, and maps `-` to `os.Stdin`. The type `filecontent` reads the whole file, or the whole standard input for `-`, into a `[]byte`. The type `existingfile` checks with `os.Stat` that the file exists.
- Python's `argparse.FileType` maps `-` to `sys.stdin`. Its documentation warns that the parser does not close the file when a later argument fails, and Python 3.14 deprecates the type.
- click's `File` maps `-` to the standard input. It opens a file for reading while it converts the value, and closes the file when the context of the command ends.

## Detailed design

### API

```go
// InputFile is the type of an option whose value is the path of an input file. The path - is the
// standard input. The handler opens the file with [Run.Open]. The help shows the value of the
// option as file.
type InputFile string

// Open opens the input file f of the run for reading. For the path -, Open returns the standard
// input of the run, whose Close does nothing. For any other path, Open returns the file that
// [Env.Open] opens.
//
// Error modes:
//   - the error of Env.Open, unchanged.
//   - an *fs.PathError that wraps fs.ErrNotExist for an Env without Open.
func (r *Run[G, C]) Open(f InputFile) (io.ReadCloser, error)
```

`Env` replaces the field `ReadFile` with `Open`:

```go
type Env struct {
	Lookup  func(name string) (string, bool) // reports the value of the environment variable name
	Environ func() []string                  // returns the environment as KEY=value pairs
	// Open opens the file name for reading. A run opens its config file and its input files with
	// it. Open has the signature of fs.FS.Open, so a test assigns the Open method of a file system.
	Open   func(name string) (fs.File, error)
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Args   []string // the arguments after the name of the program
}
```

`cli.OS` sets `Open` to a function that calls `os.Open`. A nil `Open` does not open a file, as if no file existed.

A command that reads one document declares the document as a positional argument with the default `-`:

```go
type Parse struct {
	Doc cli.InputFile `help:"JSON document, or - for the standard input." arg:"" default:"-"`
}

func parse(_ context.Context, run *cli.Run[Global, Parse]) (err error) {
	doc := run.Config().Doc
	f, err := run.Open(doc)
	if err != nil {
		return fmt.Errorf("main: parse: %w", err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	var v map[string]any
	if err = json.NewDecoder(f).Decode(&v); err != nil {
		return fmt.Errorf("main: parse %s: %w", doc, err)
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

The last run exits with the status 1, as `cat` does for a missing file.

A command that reads any number of files declares a slice, such as `Files []cli.InputFile` with the tags `arg:"" default:"-"`. The handler opens the files one at a time, so a command line with thousands of paths keeps one file open at a time.

### Rules

- A run opens a file only when its handler or one of its hooks calls `Run.Open`. The package does not open an input file while it reads the command line.
- `Run.Open` returns the standard input for each `-`. The standard input reads once, so a second `-` in one run reads what the first read left.
- The config file of a run goes through `Env.Open` too. A missing file at the default path of the config file is skipped, as it is now.
- `--flag-debug` shows the path of an input file, such as `"-"` or `"test.json"`.

### Testing

`clitest.NewEnv` sets `Open` to open the files of `Input.Files`. The transcript of `clitest.Transcript` shows each file that the runs open, config files and input files alike:

```text
$ cat test.json
{"name": "store"}
$ tool parse test.json
store
[exit 0]
```

A test that builds an `Env` itself assigns the `Open` method of an `fs.FS`, such as `fstest.MapFS{"test.json": {Data: data}}.Open`.

### Capabilities left out

- **Opening a file while the package reads the command line.** kong, argparse and click open an input file before the handler runs. If a later argument fails, the file is not closed. A command line with many paths also opens every file at once. Each handler that opens and closes its files keeps one file open at a time. A program whose hooks need every input file open before the handler would reverse this verdict.
- **The content of a file in a field.** kong's `filecontent` reads the whole input into a `[]byte`. The handler of a large input then needs memory for the whole input, and its decoder cannot stream. `io.ReadAll` of the reader of `Run.Open` returns the same bytes.
- **A check that the file exists before the handler runs.** kong's `existingfile` checks with `os.Stat`. The file can still vanish between the check and the open, so the open reports the missing file in every case.
- **A config file from the standard input.** A config file from `-` would read the standard input before the handler runs. A program that receives its config only through a pipe would reverse this verdict.
- **Output files with `-` for the standard output.** A redirection of the standard output covers a single output. A file that a command writes needs rules for its creation, its mode and its truncation. A program that writes a file and the standard output in one run would reverse this verdict.
- **A check for a terminal on the standard input.** `cat` and `wc` wait for input on a terminal, and so does a handler that opens `-`. A program that has to refuse input from a terminal checks its standard input itself.

### Changes outside the package

- `clitest.NewEnv` sets `Env.Open`, and the transcript shows each file that a run opens.
- The tests of the modules yaml and toml set `Env.Open` in place of `Env.ReadFile`.
- The program `wc` of the example `14-testing` counts the files of its arguments, and the standard input without an argument.
- The README gains a section on input files.
- The changeset of the package `cli` covers the API because the package is unreleased.

### Migration

- A program that builds its `Env` with `cli.OS` does not change.
- A test that sets `Env.ReadFile` sets `Env.Open` in its place. The Open method of an `fs.FS` has the signature of `Env.Open`.

### Tests

- `Run.Open` returns the standard input for `-` and the file of `Env.Open` for any other path.
- The Close of the standard input does not close the reader of the `Env`.
- `Run.Open` returns the error of `Env.Open` unchanged, and an error that wraps `fs.ErrNotExist` for an `Env` without `Open`.
- A run reads its config file through `Env.Open`, and skips a missing file at the default path.
- The help shows the value of an `InputFile` option as `file`.
- The transcript of `clitest` shows an input file that a run opens.
- Coverage and the mutation score are 100%.

## Alternatives considered

### Env.Open beside Env.ReadFile

`Env` would keep `ReadFile` for config files and gain `Open` for input files.

**Why not:** an `Env` would then need two functions over one file system. A test that builds its own `Env` would set both, and an `Env` that sets one of them would read its config files and its input files from different places.

### Run.Open over Env.ReadFile

`Run.Open` would read a file through `Env.ReadFile` and return a reader of the bytes.

**Why not:** every input file would then be read whole into memory before the handler reads the first byte. A command that streams a large file through a decoder could not run in constant memory.

## Drawbacks

- The package gains the type `InputFile` and the method `Run.Open`. `Env.ReadFile` becomes `Env.Open`.
- The change touches the ten files of the repository that set or call `Env.ReadFile`. Five of them are outside the package `cli`, in the specs of the modules yaml and toml and in `clitest`.
- A handler opens and closes each input file itself, which is five lines for the open, the error and the deferred close.
- A second `-` in one run reads what the first read left, without an error.

## Open questions

None.

## Unresolved and future work

None.

## References

| What | Where |
|---|---|
| kong v1.16.1: the registration of the mappers of files | <https://github.com/alecthomas/kong/blob/v1.16.1/mapper.go#L287-L292> |
| kong v1.16.1: the mappers of `*os.File`, `existingfile` and `filecontent` | <https://github.com/alecthomas/kong/blob/v1.16.1/mapper.go#L617-L750> |
| kong v1.16.1: `FileContentFlag` and `NamedFileContentFlag` | <https://github.com/alecthomas/kong/blob/v1.16.1/mapper.go#L906-L954> |
| Python: `argparse.FileType`, its caveat and its deprecation in 3.14 | <https://docs.python.org/3/library/argparse.html#argparse.FileType> |
| click: the handling of files | <https://click.palletsprojects.com/en/stable/handling-files/> |
| click: `File.convert` registers the close of the file on the context | <https://github.com/pallets/click/blob/main/src/click/types.py> |

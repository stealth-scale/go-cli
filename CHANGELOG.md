# go.stealthscale.io/cli

## 0.4.0

### Minor Changes

- 84352e6: Add `App.ExecuteRenderer` and `App.Failure` to write the failure of a run without `App.Main`.

## 0.3.0

### Minor Changes

- 4e2e996: Add `App.IgnoreKeys` to skip the config keys that an app decodes itself.
- 4e2e996: Add `App.LocateConfig` to search for the config file of a run, and the source kind `Located`.
- 4e2e996: Add `App.RenderFor` to select the renderer of each run from its global options.

### Patch Changes

- 4e2e996: **Breaking:** v0.2.0 replaced `Env.ReadFile` with `Env.Open`.
- 4e2e996: Note the config key of an option in the help only when the path of its command declares a `ConfigFile`.
- 21f29a3: Write the path of each file that a run opens with slashes in the transcript of `clitest`, so the transcript is the same on every system.

## 0.2.0

### Minor Changes

- 79f45ec: Add `InputFile` and `Run.Open` to read an input file or the standard input.

## 0.1.0

### Minor Changes

- d575da3: Add subcommands with `Command.Command`, hooks with `App.Use` and `Command.Use`, and the handler of the app with `App.Handle`.
- b2fdb29: Add the `cli` package to run commands whose options are the fields of Go structs.
- 9bb3e98: Add the package `clitest` to run a program in a test and to write the transcript of its runs.

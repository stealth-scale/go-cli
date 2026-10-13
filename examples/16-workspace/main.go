// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command workspace is a program whose config file is the document of a workspace.
//
//	go run ./16-workspace --chdir svc/store plans
//
// The program searches for its workspace file .acme.json from the directory of --chdir upward,
// and stops at a directory with .git. App.LocateConfig registers the search. The search opens each
// candidate through the Env of the run, so a test controls the files that it finds. --flag-debug
// shows the path that the search returns with the source located.
//
// The workspace file contains the option strict and the plans of the workspace. App.IgnoreKeys
// registers the keys version and plans, so the run skips them and the handler of plans decodes them
// itself. Any other key of the file fails with ErrUnknownKey.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"go.stealthscale.io/cli"
)

// marker is the name of the directory that ends the search for the workspace file.
const marker = ".git"

// The errors of the handler of plans.
var (
	// errNoWorkspace is the error of a run without a workspace file.
	errNoWorkspace = errors.New("main: no workspace file")
	// errDisabled is the error of a disabled plan with --strict.
	errDisabled = errors.New("main: a disabled plan")
)

// Global contains the options of every command.
type Global struct {
	Config cli.ConfigFile `help:"Workspace file."                     default:".acme.json" short:"c"`
	Chdir  string         `help:"Directory to start the search from." default:"."`
	Strict bool           `help:"Fail for a disabled plan."`
}

// document is the part of the workspace file that the handler of plans decodes itself.
type document struct {
	Plans map[string]plan `json:"plans"`
}

// plan is a plan of the workspace.
type plan struct {
	Enabled bool `json:"enabled"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program acme with the command plans. The run skips the keys of the workspace
// file that the handler decodes itself, and locate searches for the file.
func app() *cli.App[Global] {
	a := cli.New[Global]("acme", "Run the plans of a workspace.")
	a.IgnoreKeys("version", "plans")
	a.LocateConfig(locate)
	a.Command("plans", "List the plans of the workspace.", plans)
	return a
}

// locate returns the path of the workspace file. It tests the file name of the default of Config in
// the directory of --chdir and in each parent of that directory. It stops at a directory with .git
// and at the top of the path of --chdir, and then returns the empty path. The path has slashes on
// every system, so --flag-debug writes the same path everywhere.
//
// Error modes:
//   - an error that wraps an error of env.Open other than fs.ErrNotExist, or the error of a close.
func locate(g *Global, env *cli.Env) (string, error) {
	name := filepath.Base(string(g.Config))
	for dir := filepath.Clean(g.Chdir); ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, name)
		found, err := exists(env, path)
		if err != nil {
			return "", err
		}
		if found {
			return filepath.ToSlash(path), nil
		}
		stop, err := exists(env, filepath.Join(dir, marker))
		if err != nil {
			return "", err
		}
		if stop || filepath.Dir(dir) == dir {
			return "", nil
		}
	}
}

// exists reports whether env opens the file name, and closes the file.
//
// Error modes:
//   - an error that wraps an error of env.Open other than fs.ErrNotExist, or the error of the close.
func exists(env *cli.Env, name string) (bool, error) {
	f, err := env.Open(name)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("main: locate the workspace file: %w", err)
	}
	if err := f.Close(); err != nil {
		return false, fmt.Errorf("main: locate the workspace file: %w", err)
	}
	return true, nil
}

// plans writes each plan of the workspace file with its state, in the order of their names. With
// --strict it fails for a disabled plan.
//
// Error modes:
//   - errNoWorkspace for a run without a workspace file.
//   - an error that wraps the error of the open, the decoding or the close of the file.
//   - an error that wraps errDisabled for a disabled plan with --strict.
//   - an error that wraps the error of the standard output.
func plans(_ context.Context, run *cli.Run[Global, struct{}]) (err error) {
	g := run.Global()
	if g.Config == "" {
		return errNoWorkspace
	}
	f, err := run.Open(cli.InputFile(g.Config))
	if err != nil {
		return fmt.Errorf("main: open the workspace file: %w", err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	var doc document
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		return fmt.Errorf("main: decode %s: %w", g.Config, err)
	}
	for _, name := range slices.Sorted(maps.Keys(doc.Plans)) {
		enabled := doc.Plans[name].Enabled
		if g.Strict && !enabled {
			return fmt.Errorf("%w: %s", errDisabled, name)
		}
		state := "disabled"
		if enabled {
			state = "enabled"
		}
		if _, err := fmt.Fprintf(run.Stdout(), "%s: %s\n", name, state); err != nil {
			return fmt.Errorf("main: write the plans: %w", err)
		}
	}
	return nil
}

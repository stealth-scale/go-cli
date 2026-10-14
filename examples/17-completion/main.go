// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command rollout is a program whose words a shell completes.
//
//	source <(go run ./17-completion completion bash)
//
// The package adds the command completion to every app with commands. completion bash writes the
// completion script of bash. For each TAB the script runs completion bash -- WORDS..., and the
// program writes the candidates for the last word. The program derives the candidates from the
// declarations of the app, such as the values of the enum of --region and the files of a config
// file.
//
// App.Complete registers a function for the type Plan. The function reads the names of the plans
// from the config file of the words, so a TAB after apply offers the plans of that file. The run
// skips the key plans of the config file, because the function and the handler of apply decode it.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"time"

	"go.stealthscale.io/cli"
)

// errUnknownPlan is the error of apply for a plan that the config file does not declare.
var errUnknownPlan = errors.New("main: unknown plan")

// Global contains the options of every command.
type Global struct {
	Config  cli.ConfigFile `help:"Config file in JSON." default:"rollout.json" short:"c"`
	Verbose bool           `help:"Log more."                                   short:"v"`
}

// Plan is the name of a plan of the config file.
type Plan string

// Apply contains the options of the command apply.
type Apply struct {
	Plan   Plan   `help:"Plan to apply."      arg:""`
	Region string `help:"Region of the plan."        default:"eu-west" enum:"eu-west,us-east"`
}

// Logs contains the options of the command logs.
type Logs struct {
	Files []cli.InputFile `help:"Log files to read."      default:""   arg:""`
	Since time.Duration   `help:"Age of the oldest line." default:"1h"`
}

// document is the part of the config file that the program decodes itself.
type document struct {
	Plans map[Plan]plan `json:"plans"`
}

// plan is a plan of the config file.
type plan struct {
	Description string `json:"description"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program rollout with the commands apply and logs, and the completion of the
// plans of the config file.
func app() *cli.App[Global] {
	a := cli.New[Global]("rollout", "Roll out the plans of a config file.")
	a.IgnoreKeys("plans")
	a.Complete[Plan](completePlans)
	a.Command("apply", "Apply a plan.", apply)
	a.Command("logs", "Write the log files of the plans.", logs)
	return a
}

// completePlans returns the plans of the config file of the run as candidates, with their
// descriptions, in the order of their names.
//
// Error modes:
//   - the errors of readPlans.
func completePlans(_ context.Context, run *cli.Run[Global, Global], _ string) ([]cli.Candidate, error) {
	plans, err := readPlans(run.Open, run.Global().Config)
	if err != nil {
		return nil, err
	}
	candidates := make([]cli.Candidate, 0, len(plans))
	for _, name := range slices.Sorted(maps.Keys(plans)) {
		candidates = append(candidates, cli.Candidate{Value: string(name), Text: plans[name].Description})
	}
	return candidates, nil
}

// readPlans returns the plans of the config file path, which open opens.
//
// Error modes:
//   - an error that wraps the error of the open, the decoding or the close of the file.
func readPlans(open func(cli.InputFile) (io.ReadCloser, error), path cli.ConfigFile) (_ map[Plan]plan, err error) {
	f, err := open(cli.InputFile(path))
	if err != nil {
		return nil, fmt.Errorf("main: open the config file: %w", err)
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	var doc document
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		return nil, fmt.Errorf("main: decode %s: %w", path, err)
	}
	return doc.Plans, nil
}

// apply writes the plan and the region of the run.
//
// Error modes:
//   - the errors of readPlans.
//   - an error that wraps errUnknownPlan for a plan that the config file does not declare.
//   - an error that wraps the error of the standard output.
func apply(_ context.Context, run *cli.Run[Global, Apply]) error {
	plans, err := readPlans(run.Open, run.Global().Config)
	if err != nil {
		return err
	}
	c := run.Config()
	if _, ok := plans[c.Plan]; !ok {
		return fmt.Errorf("%w: %s", errUnknownPlan, c.Plan)
	}
	if _, err := fmt.Fprintf(run.Stdout(), "apply %s in %s\n", c.Plan, c.Region); err != nil {
		return fmt.Errorf("main: write the plan: %w", err)
	}
	return nil
}

// logs writes the names of the log files of the run and the age of their oldest line.
//
// Error modes:
//   - an error that wraps the error of the standard output.
func logs(_ context.Context, run *cli.Run[Global, Logs]) error {
	c := run.Config()
	if _, err := fmt.Fprintf(run.Stdout(), "logs of %v since %s\n", c.Files, c.Since); err != nil {
		return fmt.Errorf("main: write the logs: %w", err)
	}
	return nil
}

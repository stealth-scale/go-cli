// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command renderer is a program with a renderer of its own.
//
//	OUTPUT=json go run ./15-renderer count --help
//
// The renderer embeds cli.Plain and replaces its three methods. Each method writes JSON when the
// environment of the run contains OUTPUT=json, and calls the method of cli.Plain otherwise:
//   - Help writes the cli.Help of the app or of a command, with its cli.HelpCommand and
//     cli.HelpOption values.
//   - Error writes the cli.Failure of a run without its error.
//   - Debug writes the cli.DebugRow values of --flag-debug with each source as text.
//
// App.Render registers the renderer. A renderer receives the environment of the run, so a test
// selects the output through the variables of its input.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"

	"go.stealthscale.io/cli"
)

// jsonOutput is the pair of the environment that selects the output in JSON.
const jsonOutput = "OUTPUT=json"

// Count contains the options of the command count.
type Count struct {
	Item      string `help:"Item to count."         arg:""`
	Warehouse string `help:"Warehouse of the item."        default:"main" short:"w"`
}

// jsonRenderer is a cli.Renderer that writes the output of the app as JSON when the environment of a
// run contains OUTPUT=json. Otherwise it writes the text of the embedded cli.Plain.
type jsonRenderer struct {
	cli.Plain
}

// failure is the JSON form of a cli.Failure. It leaves out the error, whose text Message contains.
type failure struct {
	App        string `json:"app"`
	Command    string `json:"command,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	Usage      bool   `json:"usage"`
}

// row is the JSON form of a cli.DebugRow, with its source as text.
type row struct {
	Option string `json:"option"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program stock with the command count. jsonRenderer writes its output.
func app() *cli.App[struct{}] {
	a := cli.New[struct{}]("stock", "Count the items of the warehouses.")
	a.Render(jsonRenderer{})
	a.Command("count", "Count an item in a warehouse.", count)
	return a
}

// count writes the number of the item in the warehouse.
func count(_ context.Context, run *cli.Run[struct{}, Count]) error {
	c := run.Config()
	if _, err := fmt.Fprintf(run.Stdout(), "%s: 12 in %s\n", c.Item, c.Warehouse); err != nil {
		return fmt.Errorf("main: write the count: %w", err)
	}
	return nil
}

// encode writes v to w as indented JSON.
func encode(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("main: write JSON: %w", err)
	}
	return nil
}

// Help writes h as JSON, or as the text of cli.Plain.
func (r jsonRenderer) Help(w io.Writer, environ []string, h *cli.Help) error {
	if !slices.Contains(environ, jsonOutput) {
		return r.Plain.Help(w, environ, h)
	}
	return encode(w, h)
}

// Error writes the failure f as JSON, or as the line of cli.Plain.
func (r jsonRenderer) Error(w io.Writer, environ []string, f *cli.Failure) error {
	if !slices.Contains(environ, jsonOutput) {
		return r.Plain.Error(w, environ, f)
	}
	return encode(
		w,
		failure{App: f.App, Command: f.Command, Message: f.Message, Suggestion: f.Suggestion, Usage: f.Usage},
	)
}

// Debug writes the rows of --flag-debug as JSON, or as the table of cli.Plain.
func (r jsonRenderer) Debug(w io.Writer, environ []string, rows []cli.DebugRow) error {
	if !slices.Contains(environ, jsonOutput) {
		return r.Plain.Debug(w, environ, rows)
	}
	out := make([]row, len(rows))
	for i, d := range rows {
		out[i] = row{Option: d.Option, Value: d.Value, Source: d.Source.String()}
	}
	return encode(w, out)
}

// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command renderer is a program whose option --format selects its renderer.
//
//	go run ./15-renderer --format json count --help
//
// App.RenderFor registers a function that selects the renderer of each run from the global options
// of the run. The function returns jsonRenderer for the format json and cli.Plain for the format
// text. jsonRenderer writes JSON:
//   - Help writes the cli.Help of the app or of a command, with its cli.HelpCommand and
//     cli.HelpOption values.
//   - Error writes the cli.Failure of a run without its error.
//   - Debug writes the cli.DebugRow values of --flag-debug with each source as text.
//
// A run applies the valid values of its sources before the function selects the renderer, so the
// usage error of a command line with --format json is JSON. The flags count up to the first fault
// of the command line, so a --format json after an unknown flag does not select JSON. The variable
// STOCK_FORMAT selects the format of every run. The handler of count writes its result in the
// format of the run too.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"go.stealthscale.io/cli"
)

// jsonFormat is the value of the option --format that selects JSON.
const jsonFormat = "json"

// items is the number of every item in every warehouse.
const items = 12

// Global contains the options of every command.
type Global struct {
	Format string `help:"Output format." default:"text" enum:"text,json"`
}

// Count contains the options of the command count.
type Count struct {
	Item      string `help:"Item to count."         arg:""`
	Warehouse string `help:"Warehouse of the item."        default:"main" short:"w"`
}

// jsonRenderer is a cli.Renderer that writes the output of the app as JSON.
type jsonRenderer struct{}

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

// result is the JSON form of the result of the command count.
type result struct {
	Item      string `json:"item"`
	Warehouse string `json:"warehouse"`
	Count     int    `json:"count"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program stock with the command count. The option --format selects its renderer.
func app() *cli.App[Global] {
	a := cli.New[Global]("stock", "Count the items of the warehouses.")
	a.RenderFor(func(g *Global) cli.Renderer {
		if g.Format == jsonFormat {
			return jsonRenderer{}
		}
		return cli.Plain{}
	})
	a.Command("count", "Count an item in a warehouse.", count)
	return a
}

// count writes the number of the item in the warehouse, as JSON for the format json.
func count(_ context.Context, run *cli.Run[Global, Count]) error {
	c := run.Config()
	if run.Global().Format == jsonFormat {
		return encode(run.Stdout(), &result{Item: c.Item, Warehouse: c.Warehouse, Count: items})
	}
	if _, err := fmt.Fprintf(run.Stdout(), "%s: %d in %s\n", c.Item, items, c.Warehouse); err != nil {
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

// Help writes h as JSON.
func (jsonRenderer) Help(w io.Writer, _ []string, h *cli.Help) error {
	return encode(w, h)
}

// Error writes the failure f as JSON.
func (jsonRenderer) Error(w io.Writer, _ []string, f *cli.Failure) error {
	return encode(
		w,
		&failure{App: f.App, Command: f.Command, Message: f.Message, Suggestion: f.Suggestion, Usage: f.Usage},
	)
}

// Debug writes the rows of --flag-debug as JSON.
func (jsonRenderer) Debug(w io.Writer, _ []string, rows []cli.DebugRow) error {
	out := make([]row, len(rows))
	for i, d := range rows {
		out[i] = row{Option: d.Option, Value: d.Value, Source: d.Source.String()}
	}
	return encode(w, out)
}

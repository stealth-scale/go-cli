// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package charm

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"go.stealthscale.io/cli"
)

// The layout of the output.
const (
	indent         = "  "   // the indent of the rows of a section
	gap            = "  "   // the space between a key and its text
	noShort        = "    " // the space of a short flag and its comma before a long flag without one
	separator      = " · "  // the separator of the notes of an option
	defaultColumns = 80     // the width of the output without COLUMNS
	maxColumns     = 100    // the maximum width of the output
	maxKey         = 32     // the maximum width of a key with its text on the same line
	minText        = 20     // the minimum width of a text column
)

// Renderer is a cli.Renderer that writes the output of an app in the styles of a [Theme].
//
// colorprofile decides from the writer and from the environment of the run whether the output
// contains styles. A terminal receives the styles. A pipe or a file receives the same help without
// styles, and the failures and the table of --flag-debug of cli.Plain. NO_COLOR removes the colors
// and keeps bold and faint text. CLICOLOR_FORCE writes the styles to any writer.
//
// The output wraps at the width in the environment variable COLUMNS, at most 100 columns. Without
// COLUMNS it wraps at 80 columns. A Renderer does not change after [New], so concurrent runs can
// share it.
type Renderer struct {
	theme Theme
}

var _ cli.Renderer = (*Renderer)(nil)

// entry is a row of a section of the help. Its key and its notes contain the styles of the theme.
type entry struct {
	key   string   // the name of a command, or the flags and the placeholder of an option
	text  string   // the help text
	notes []string // the notes of an option
}

// New returns a Renderer with a copy of the theme t.
func New(t *Theme) *Renderer {
	return &Renderer{theme: *t}
}

// Register registers a Renderer with [DefaultTheme] as the renderer of app.
func Register[G any](app *cli.App[G]) {
	app.Render(New(new(DefaultTheme())))
}

// Help writes the help h to w. It writes the text and the usage line, then a section for each of
// the commands, the arguments, the options and the global options. The notes of an option follow
// its text on the lines below it.
//
// Error modes:
//   - an error that wraps the error of w.
func (r *Renderer) Help(w io.Writer, environ []string, h *cli.Help) error {
	t := &r.theme
	width := columns(environ)
	var b strings.Builder
	if h.Text != "" {
		hang(&b, "", styled(&t.Text, wrap(h.Text, width)), "")
		b.WriteString("\n")
	}
	usage := []string{t.Program.Render(h.App)}
	if h.Command != "" {
		usage = append(usage, t.Command.Render(h.Command))
	}
	for _, part := range h.Usage {
		usage = append(usage, t.Argument.Render(part))
	}
	b.WriteString(t.Heading.Render("USAGE") + "\n" + indent + strings.Join(usage, " ") + "\n")
	commands := make([]entry, len(h.Commands))
	for i, c := range h.Commands {
		commands[i] = entry{key: t.Command.Render(c.Name), text: c.Text}
	}
	headings := []string{"COMMANDS", "ARGUMENTS", "OPTIONS", "GLOBAL OPTIONS"}
	sections := [][]entry{commands, r.entries(h.Arguments), r.entries(h.Options), r.entries(h.Globals)}
	column := keyColumn(sections)
	for i, entries := range sections {
		r.section(&b, headings[i], entries, column, width)
	}
	if _, err := colorprofile.NewWriter(w, environ).WriteString(b.String()); err != nil {
		return fmt.Errorf("charm: write the help: %w", err)
	}
	return nil
}

// Error writes the failure f to w. A terminal receives a badge with the message and the suggestion.
// After an invalid input it also receives a hint to the help. A pipe or a file receives the line
// of cli.Plain.
//
// Error modes:
//   - an error that wraps the error of w.
func (r *Renderer) Error(w io.Writer, environ []string, f *cli.Failure) error {
	out := colorprofile.NewWriter(w, environ)
	if out.Profile == colorprofile.NoTTY {
		return cli.Plain{}.Error(w, environ, f)
	}
	t := &r.theme
	badge := t.ErrorBadge.Render("ERROR")
	margin := strings.Repeat(" ", lipgloss.Width(badge)+1)
	lines := styled(&t.ErrorText, wrap(f.Message, max(columns(environ)-len(margin), minText)))
	if f.Suggestion != "" {
		lines = append(lines, t.Hint.Render("Did you mean ")+t.Suggestion.Render(f.Suggestion)+t.Hint.Render("?"))
	}
	if f.Usage {
		help := f.App
		if f.Command != "" {
			help += " " + f.Command
		}
		lines = append(lines, t.Hint.Render("Run ")+t.Program.Render(help+" --help")+t.Hint.Render(" for the usage."))
	}
	var b strings.Builder
	hang(&b, badge+" ", lines, margin)
	if _, err := out.WriteString(b.String()); err != nil {
		return fmt.Errorf("charm: write the failure: %w", err)
	}
	return nil
}

// Debug writes the rows of the table of --flag-debug to w. A terminal receives the columns of
// cli.Plain in the styles Heading, Flag, Text and Note. A pipe or a file receives the table of
// cli.Plain.
//
// Error modes:
//   - an error that wraps the error of w.
func (r *Renderer) Debug(w io.Writer, environ []string, rows []cli.DebugRow) error {
	out := colorprofile.NewWriter(w, environ)
	if out.Profile == colorprofile.NoTTY {
		return cli.Plain{}.Debug(w, environ, rows)
	}
	t := &r.theme
	table := [][3]string{{t.Heading.Render("OPTION"), t.Heading.Render("VALUE"), t.Heading.Render("SOURCE")}}
	for _, row := range rows {
		table = append(table, [3]string{
			t.Flag.Render(row.Option), t.Text.Render(row.Value), t.Note.Render(row.Source.String()),
		})
	}
	var first, second int
	for _, cells := range table {
		first = max(first, lipgloss.Width(cells[0]))
		second = max(second, lipgloss.Width(cells[1]))
	}
	var b strings.Builder
	for _, cells := range table {
		b.WriteString(indent + pad(cells[0], first) + gap + pad(cells[1], second) + gap + cells[2] + "\n")
	}
	if _, err := out.WriteString(b.String()); err != nil {
		return fmt.Errorf("charm: write the output of --flag-debug: %w", err)
	}
	return nil
}

// entries returns opts as entries of the help. When an option of opts has a short flag, the long
// flag of an option without one starts after the space of a short flag, so the long flags align.
func (r *Renderer) entries(opts []cli.HelpOption) []entry {
	t := &r.theme
	aligned := slices.ContainsFunc(opts, func(o cli.HelpOption) bool { return o.Short != "" })
	entries := make([]entry, len(opts))
	for i := range opts {
		o := &opts[i]
		key := t.Flag.Render(o.Name)
		switch {
		case o.Short != "":
			key = t.Flag.Render(o.Short) + ", " + key
		case aligned:
			key = noShort + key
		}
		if o.Placeholder != "" {
			key += " " + t.Argument.Render(o.Placeholder)
		}
		entries[i] = entry{key: key, text: o.Text, notes: r.notes(o)}
	}
	return entries
}

// notes returns the notes of o in the styles of the theme. The order is required, the default,
// the environment variable and the config key.
func (r *Renderer) notes(o *cli.HelpOption) []string {
	t := &r.theme
	var notes []string
	if o.Required {
		notes = append(notes, t.Required.Render("required"))
	}
	if o.Default != "" {
		notes = append(notes, t.Note.Render("default "+o.Default))
	}
	if o.Env != "" {
		notes = append(notes, t.Note.Render("env "+o.Env))
	}
	if o.Key != "" {
		notes = append(notes, t.Note.Render("key "+o.Key))
	}
	return notes
}

// section writes a section of the help to b. It writes nothing for a section without entries. The
// text of each entry starts at column, and its notes follow on the lines below the text. The text
// of a key wider than maxKey starts on the line below the key.
func (r *Renderer) section(b *strings.Builder, heading string, entries []entry, column, width int) {
	if len(entries) == 0 {
		return
	}
	t := &r.theme
	text := max(width-column, minText)
	margin := strings.Repeat(" ", column)
	b.WriteString("\n" + t.Heading.Render(heading) + "\n")
	for _, e := range entries {
		var lines []string
		if e.text != "" {
			lines = styled(&t.Text, wrap(e.text, text))
		}
		lines = append(lines, pack(e.notes, t.Note.Render(separator), text)...)
		head := margin
		if key := indent + e.key; len(lines) == 0 || lipgloss.Width(key)+len(gap) > column {
			b.WriteString(key + "\n")
		} else {
			head = pad(key, column)
		}
		hang(b, head, lines, margin)
	}
}

// keyColumn returns the column where the texts of the sections start. The column is the sum of the
// widths of the indent, the widest key of at most maxKey columns and the gap.
func keyColumn(sections [][]entry) int {
	widest := 0
	for _, entries := range sections {
		for _, e := range entries {
			if w := lipgloss.Width(e.key); w <= maxKey {
				widest = max(widest, w)
			}
		}
	}
	return len(indent) + widest + len(gap)
}

// pack joins notes with sep into lines of at most width columns. A note does not break across two
// lines, so a note wider than width has a line of its own.
func pack(notes []string, sep string, width int) []string {
	var lines []string
	for _, n := range notes {
		last := len(lines) - 1
		if last >= 0 && lipgloss.Width(lines[last]+sep+n) <= width {
			lines[last] += sep + n
			continue
		}
		lines = append(lines, n)
	}
	return lines
}

// hang writes lines to b. The first line follows head, and each other line follows margin. hang
// removes the spaces at the end of each line.
func hang(b *strings.Builder, head string, lines []string, margin string) {
	for i, line := range lines {
		if i > 0 {
			head = margin
		}
		b.WriteString(strings.TrimRight(head+line, " ") + "\n")
	}
}

// columns returns the width of the output. The width is the value of the last valid variable
// COLUMNS of environ, at most maxColumns. Without a valid COLUMNS the width is defaultColumns.
func columns(environ []string) int {
	n := defaultColumns
	for _, kv := range environ {
		if v, ok := strings.CutPrefix(kv, "COLUMNS="); ok {
			if c, err := strconv.Atoi(v); err == nil && c > 0 {
				n = c
			}
		}
	}
	return min(n, maxColumns)
}

// wrap returns the lines of text wrapped at width columns.
func wrap(text string, width int) []string {
	return strings.Split(lipgloss.Wrap(text, width, ""), "\n")
}

// styled returns lines in the style s.
func styled(s *lipgloss.Style, lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = s.Render(line)
	}
	return out
}

// pad returns s with spaces up to width columns.
func pad(s string, width int) string {
	return s + strings.Repeat(" ", width-lipgloss.Width(s))
}

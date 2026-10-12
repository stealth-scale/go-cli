// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package charm

import "charm.land/lipgloss/v2"

// Theme contains the styles of the parts of the output of a [Renderer].
type Theme struct {
	Heading    lipgloss.Style // a heading of the help, such as USAGE and OPTIONS
	Program    lipgloss.Style // the name of the app
	Command    lipgloss.Style // the name of a command
	Argument   lipgloss.Style // a part of the usage line after the names, and the placeholder of a value
	Flag       lipgloss.Style // a flag, or the label of a positional argument
	Text       lipgloss.Style // a help text
	Note       lipgloss.Style // the default, the environment variable and the config key of an option
	Required   lipgloss.Style // the word required in the notes of a required option
	ErrorBadge lipgloss.Style // the badge ERROR before the message of a failure
	ErrorText  lipgloss.Style // the message of a failure
	Suggestion lipgloss.Style // the name that a failure suggests
	Hint       lipgloss.Style // the words around the suggested name and around the help command
}

// DefaultTheme returns the theme of [Register]. Its colors come from the 16-color palette of the
// terminal, so the terminal decides each shade on a light and on a dark background.
func DefaultTheme() Theme {
	return Theme{
		Heading:  lipgloss.NewStyle().Bold(true),
		Program:  lipgloss.NewStyle().Bold(true),
		Command:  lipgloss.NewStyle().Foreground(lipgloss.Cyan),
		Argument: lipgloss.NewStyle().Faint(true),
		Flag:     lipgloss.NewStyle().Foreground(lipgloss.Magenta),
		Text:     lipgloss.NewStyle(),
		Note:     lipgloss.NewStyle().Faint(true),
		Required: lipgloss.NewStyle().Foreground(lipgloss.Yellow),
		ErrorBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.BrightWhite).
			Background(lipgloss.Red).
			Padding(0, 1),
		ErrorText:  lipgloss.NewStyle(),
		Suggestion: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Green),
		Hint:       lipgloss.NewStyle().Faint(true),
	}
}

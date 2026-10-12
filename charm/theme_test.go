// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package charm_test

import (
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli/charm"
)

// TestTheme checks the styles of the default theme.
func TestTheme(t *testing.T) {
	t.Parallel()

	t.Run("DefaultTheme", func(t *testing.T) {
		t.Parallel()

		theme := charm.DefaultTheme()
		none := color.Color(lipgloss.NoColor{})
		tests := []struct {
			name      string
			giveStyle lipgloss.Style
			wantColor color.Color
			wantBold  bool
			wantFaint bool
		}{
			{name: "writes a heading in bold", giveStyle: theme.Heading, wantColor: none, wantBold: true},
			{name: "writes the name of the app in bold", giveStyle: theme.Program, wantColor: none, wantBold: true},
			{name: "writes a command in cyan", giveStyle: theme.Command, wantColor: lipgloss.Cyan},
			{name: "writes a placeholder faint", giveStyle: theme.Argument, wantColor: none, wantFaint: true},
			{name: "writes a flag in magenta", giveStyle: theme.Flag, wantColor: lipgloss.Magenta},
			{name: "writes a help text without a style", giveStyle: theme.Text, wantColor: none},
			{name: "writes a note faint", giveStyle: theme.Note, wantColor: none, wantFaint: true},
			{name: "writes the word required in yellow", giveStyle: theme.Required, wantColor: lipgloss.Yellow},
			{name: "writes the message of a failure without a style", giveStyle: theme.ErrorText, wantColor: none},
			{
				name: "writes a suggestion in bold green", giveStyle: theme.Suggestion, wantColor: lipgloss.Green,
				wantBold: true,
			},
			{name: "writes a hint faint", giveStyle: theme.Hint, wantColor: none, wantFaint: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.giveStyle.GetForeground(), tt.wantColor, "the style has the wanted color")
				assert.Equal(t, tt.giveStyle.GetBold(), tt.wantBold, "the style has the wanted bold attribute")
				assert.Equal(t, tt.giveStyle.GetFaint(), tt.wantFaint, "the style has the wanted faint attribute")
			})
		}

		t.Run("writes the badge of a failure in bold bright white on red", func(t *testing.T) {
			t.Parallel()
			badge := theme.ErrorBadge
			assert.True(t, badge.GetBold(), "the badge is bold")
			assert.Equal(t, badge.GetForeground(), color.Color(lipgloss.BrightWhite), "the badge has bright white text")
			assert.Equal(t, badge.GetBackground(), color.Color(lipgloss.Red), "the background of the badge is red")
			assert.Equal(t, badge.GetHorizontalPadding(), 2, "the badge has a space on each side")
		})
	})
}

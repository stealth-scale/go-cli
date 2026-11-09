// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

module go.stealthscale.io/cli/examples

go 1.27.2

require (
	charm.land/lipgloss/v2 v2.0.6
	github.com/charmbracelet/colorprofile v0.4.3
	go.dokimi.dev/assert v0.18.0
	go.stealthscale.io/cli v0.8.1
	go.stealthscale.io/cli/charm v0.2.5
	go.stealthscale.io/cli/toml v0.1.9
	go.stealthscale.io/cli/yaml v0.1.9
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20261023060817-a45b7c86333f // indirect
	github.com/charmbracelet/x/ansi v0.11.9 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-runewidth v0.0.31 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v1.3.0 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace (
	go.stealthscale.io/cli => ../
	go.stealthscale.io/cli/charm => ../charm
	go.stealthscale.io/cli/toml => ../toml
	go.stealthscale.io/cli/yaml => ../yaml
)

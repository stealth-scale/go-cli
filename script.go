// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	_ "embed" // the completion scripts
	"fmt"
	"io"
	"strings"
)

// shell is a shell that the command completion writes a script for.
type shell string

// The shells of the command completion.
const (
	bash       shell = "bash"
	zsh        shell = "zsh"
	fish       shell = "fish"
	powershell shell = "powershell"
)

// The placeholders of a completion script.
const (
	// appPlaceholder takes the name of the app, the name of the program that the script completes.
	appPlaceholder = "{{app}}"
	// functionPlaceholder takes the name of the app with an underscore for each dash, the part of
	// the names of the functions of the script.
	functionPlaceholder = "{{fn}}"
	// functionSeparator replaces each dash of the name of the app in the names of the functions.
	functionSeparator = "_"
)

// The completion scripts of the shells.
var (
	//go:embed scripts/completion.bash
	bashScript string
	//go:embed scripts/completion.zsh
	zshScript string
	//go:embed scripts/completion.fish
	fishScript string
	//go:embed scripts/completion.ps1
	powershellScript string
)

// scripts are the completion scripts by their shell.
var scripts = map[shell]string{bash: bashScript, zsh: zshScript, fish: fishScript, powershell: powershellScript}

// writeScript writes the completion script of sh for the app named app to w.
//
// Error modes:
//   - an error that wraps the error of w.
func writeScript(w io.Writer, sh shell, app string) error {
	r := strings.NewReplacer(appPlaceholder, app,
		functionPlaceholder, strings.ReplaceAll(app, wordSeparator, functionSeparator))
	if _, err := io.WriteString(w, r.Replace(scripts[sh])); err != nil {
		return fmt.Errorf("cli: write the completion script: %w", err)
	}
	return nil
}

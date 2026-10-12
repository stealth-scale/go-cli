// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import "reflect"

// InputFile is the type of an option whose value is the path of an input file. The path - is the
// standard input. A handler opens the file with [Run.Open]. The package does not open the file
// itself, so a handler opens and closes each file of a list one at a time. The help shows the value
// of the option as file.
type InputFile string

// stdinPath is the path of the standard input among the values of an InputFile.
const stdinPath InputFile = "-"

// inputFileType is the type of an option that is an input file.
var inputFileType = reflect.TypeFor[InputFile]()

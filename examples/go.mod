// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

module go.stealthscale.io/cli/examples

go 1.27.2

require (
	go.dokimi.dev/assert v0.2.0
	go.stealthscale.io/cli v0.0.0-00010101000000-000000000000
	go.stealthscale.io/cli/toml v0.0.0-00010101000000-000000000000
	go.stealthscale.io/cli/yaml v0.0.0-00010101000000-000000000000
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)

replace (
	go.stealthscale.io/cli => ../
	go.stealthscale.io/cli/toml => ../toml
	go.stealthscale.io/cli/yaml => ../yaml
)

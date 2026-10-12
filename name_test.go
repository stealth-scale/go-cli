// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"context"
	"testing"
	"testing/fstest"

	"go.dokimi.dev/assert"

	"go.stealthscale.io/cli"
)

// The options of the app in the tests of names.
type (
	nameGlobal struct {
		Config  cli.ConfigFile
		Verbose bool
	}
	redisOptions struct {
		Addr string
	}
	cacheOptions struct {
		Redis redisOptions
	}
	nameOptions struct {
		redisOptions
		Cache     cacheOptions
		Labels    []string `name:"label"`
		URL       string
		X         string
		HTTPPort  int
		HTTP2Port int
		DryRun    bool
	}
)

// TestName checks the flag, the environment variable and the config key that Execute derives for
// each option.
func TestName(t *testing.T) {
	t.Parallel()

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			giveArgs  []string
			giveVars  map[string]string
			giveData  string
			giveField func(run *cli.Run[nameGlobal, nameOptions]) any
			want      cli.Source
		}{
			{
				name:      "derives the flag dry-run from the field DryRun",
				giveArgs:  []string{command, "--dry-run"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().DryRun },
				want:      cli.Source{Kind: cli.Flag, Name: "--dry-run"},
			},
			{
				name:      "derives the flag http-port from the field HTTPPort",
				giveArgs:  []string{command, "--http-port", "80"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().HTTPPort },
				want:      cli.Source{Kind: cli.Flag, Name: "--http-port"},
			},
			{
				name:      "derives the flag http2-port from the field HTTP2Port",
				giveArgs:  []string{command, "--http2-port", "80"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().HTTP2Port },
				want:      cli.Source{Kind: cli.Flag, Name: "--http2-port"},
			},
			{
				name:      "derives the flag url from the field URL",
				giveArgs:  []string{command, "--url", "x"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().URL },
				want:      cli.Source{Kind: cli.Flag, Name: "--url"},
			},
			{
				name:      "derives the flag x from the field X",
				giveArgs:  []string{command, "--x", "x"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().X },
				want:      cli.Source{Kind: cli.Flag, Name: "--x"},
			},
			{
				name:      "joins the groups and the name in the flag",
				giveArgs:  []string{command, "--cache-redis-addr", "x"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().Cache.Redis.Addr },
				want:      cli.Source{Kind: cli.Flag, Name: "--cache-redis-addr"},
			},
			{
				name:      "replaces the last word of the flag with the tag name",
				giveArgs:  []string{command, "--label", "x"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().Labels },
				want:      cli.Source{Kind: cli.Flag, Name: "--label"},
			},
			{
				name:      "joins the app, the command, the groups and the name in the variable",
				giveArgs:  []string{command},
				giveVars:  map[string]string{"SRV_SERVE_CACHE_REDIS_ADDR": "x"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().Cache.Redis.Addr },
				want:      cli.Source{Kind: cli.Environment, Name: "SRV_SERVE_CACHE_REDIS_ADDR"},
			},
			{
				name:      "leaves the embedded struct out of the variable of its option",
				giveArgs:  []string{command},
				giveVars:  map[string]string{"SRV_SERVE_ADDR": "x"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().Addr },
				want:      cli.Source{Kind: cli.Environment, Name: "SRV_SERVE_ADDR"},
			},
			{
				name:      "leaves the embedded struct out of the config key of its option",
				giveArgs:  []string{"--config", configPath, command},
				giveData:  `{"serve": {"addr": "x"}}`,
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().Addr },
				want:      cli.Source{Kind: cli.File, Name: "serve.addr", File: configPath},
			},
			{
				name:      "replaces each dash of the name with an underscore in the variable",
				giveArgs:  []string{command},
				giveVars:  map[string]string{"SRV_SERVE_DRY_RUN": "true"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().DryRun },
				want:      cli.Source{Kind: cli.Environment, Name: "SRV_SERVE_DRY_RUN"},
			},
			{
				name:      "replaces the last word of the variable with the tag name",
				giveArgs:  []string{command},
				giveVars:  map[string]string{"SRV_SERVE_LABEL": "x"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().Labels },
				want:      cli.Source{Kind: cli.Environment, Name: "SRV_SERVE_LABEL"},
			},
			{
				name:      "leaves the command out of the variable of a global option",
				giveArgs:  []string{command},
				giveVars:  map[string]string{"SRV_VERBOSE": "true"},
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Global().Verbose },
				want:      cli.Source{Kind: cli.Environment, Name: "SRV_VERBOSE"},
			},
			{
				name:      "joins the command, the groups and the name in the config key",
				giveArgs:  []string{"--config", configPath, command},
				giveData:  `{"serve": {"cache": {"redis": {"addr": "x"}}}}`,
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().Cache.Redis.Addr },
				want:      cli.Source{Kind: cli.File, Name: "serve.cache.redis.addr", File: configPath},
			},
			{
				name:      "keeps the dashes of the name in the config key",
				giveArgs:  []string{"--config", configPath, command},
				giveData:  `{"serve": {"dry-run": true}}`,
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().DryRun },
				want:      cli.Source{Kind: cli.File, Name: "serve.dry-run", File: configPath},
			},
			{
				name:      "leaves the command out of the config key of a global option",
				giveArgs:  []string{"--config", configPath, command},
				giveData:  `{"verbose": true}`,
				giveField: func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Global().Verbose },
				want:      cli.Source{Kind: cli.File, Name: "verbose", File: configPath},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[nameGlobal, nameOptions](t)
				res := execute(t, app, tt.giveArgs, tt.giveVars, fstest.MapFS{configPath: {Data: []byte(tt.giveData)}})
				assert.NoError(t, res.err, "Execute runs the command")
				src, ok := got.Source(tt.giveField(got))
				assert.True(t, ok, "the run has the source of the option")
				assert.Equal(t, src, tt.want, "the source states the derived name")
			})
		}

		t.Run("replaces each dash of the app name with an underscore in the variable", func(t *testing.T) {
			t.Parallel()
			var got int
			app := cli.New[struct{}]("my-app", appHelp)
			app.Command(
				command,
				commandHelp,
				func(_ context.Context, run *cli.Run[struct{}, struct{ Port int }]) error {
					got = run.Config().Port
					return nil
				},
			)
			res := execute(t, app, []string{command}, map[string]string{"MY_APP_SERVE_PORT": "8"}, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got, 8, "the run reads MY_APP_SERVE_PORT")
		})
	})
}

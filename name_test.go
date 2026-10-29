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
	tokenOptions struct {
		Token string `env:"API_TOKEN"`
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

		nested := []struct {
			name      string
			giveArgs  []string
			giveVars  map[string]string
			giveData  string
			giveField func(run *cli.Run[nameGlobal, stepsOptions]) any
			want      cli.Source
		}{
			{
				name:      "joins the app, the path and the name in the variable of an option of a subcommand",
				giveArgs:  []string{parent, subcommand},
				giveVars:  map[string]string{"SRV_DB_MIGRATE_STEPS": "3"},
				giveField: func(run *cli.Run[nameGlobal, stepsOptions]) any { return &run.Config().Steps },
				want:      cli.Source{Kind: cli.Environment, Name: "SRV_DB_MIGRATE_STEPS"},
			},
			{
				name:      "joins the app, the path of its command and the name in the variable of an inherited option",
				giveArgs:  []string{parent, subcommand},
				giveVars:  map[string]string{"SRV_DB_DSN": "pg"},
				giveField: func(run *cli.Run[nameGlobal, stepsOptions]) any { return &run.Config().DSN },
				want:      cli.Source{Kind: cli.Environment, Name: "SRV_DB_DSN"},
			},
			{
				name:      "joins the path and the name in the config key of an option of a subcommand",
				giveArgs:  []string{"--config", configPath, parent, subcommand},
				giveData:  `{"db": {"migrate": {"steps": 3}}}`,
				giveField: func(run *cli.Run[nameGlobal, stepsOptions]) any { return &run.Config().Steps },
				want:      cli.Source{Kind: cli.File, Name: "db.migrate.steps", File: configPath},
			},
			{
				name:      "nests the keys of a command and of its subcommand in one object of the config file",
				giveArgs:  []string{"--config", configPath, parent, subcommand},
				giveData:  `{"db": {"dsn": "pg", "migrate": {"steps": 3}}}`,
				giveField: func(run *cli.Run[nameGlobal, stepsOptions]) any { return &run.Config().DSN },
				want:      cli.Source{Kind: cli.File, Name: "db.dsn", File: configPath},
			},
		}
		for _, tt := range nested {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _, migrate := nesting[nameGlobal, Store, stepsOptions](t)
				res := execute(t, app, tt.giveArgs, tt.giveVars, fstest.MapFS{configPath: {Data: []byte(tt.giveData)}})
				assert.NoError(t, res.err, "Execute runs the subcommand")
				src, ok := migrate.Source(tt.giveField(migrate))
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

	t.Run("EnvPrefix", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			givePrefix string
			giveVars   map[string]string
			giveField  func(run *cli.Run[nameGlobal, nameOptions]) any
			want       cli.Source
		}{
			{
				name:       "replaces the app name in the variable of an option of a command",
				givePrefix: "STORE",
				giveVars:   map[string]string{"STORE_SERVE_DRY_RUN": "true"},
				giveField:  func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().DryRun },
				want:       cli.Source{Kind: cli.Environment, Name: "STORE_SERVE_DRY_RUN"},
			},
			{
				name:       "replaces the app name in the variable of a global option",
				givePrefix: "STORE",
				giveVars:   map[string]string{"STORE_VERBOSE": "true"},
				giveField:  func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Global().Verbose },
				want:       cli.Source{Kind: cli.Environment, Name: "STORE_VERBOSE"},
			},
			{
				name:       "takes a prefix of words with digits that underscores join",
				givePrefix: "MY_STORE2",
				giveVars:   map[string]string{"MY_STORE2_SERVE_DRY_RUN": "true"},
				giveField:  func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().DryRun },
				want:       cli.Source{Kind: cli.Environment, Name: "MY_STORE2_SERVE_DRY_RUN"},
			},
			{
				name:       "starts the variable of an option of a command with the command for an empty prefix",
				givePrefix: "",
				giveVars:   map[string]string{"SERVE_DRY_RUN": "true"},
				giveField:  func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().DryRun },
				want:       cli.Source{Kind: cli.Environment, Name: "SERVE_DRY_RUN"},
			},
			{
				name:       "starts the variable of a global option with its name for an empty prefix",
				givePrefix: "",
				giveVars:   map[string]string{"VERBOSE": "true"},
				giveField:  func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Global().Verbose },
				want:       cli.Source{Kind: cli.Environment, Name: "VERBOSE"},
			},
			{
				name:       "does not read the variable that starts with the app name",
				givePrefix: "STORE",
				giveVars:   map[string]string{"SRV_SERVE_DRY_RUN": "true"},
				giveField:  func(run *cli.Run[nameGlobal, nameOptions]) any { return &run.Config().DryRun },
				want:       cli.Source{},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, got := serving[nameGlobal, nameOptions](t)
				app.EnvPrefix(tt.givePrefix)
				res := execute(t, app, []string{command}, tt.giveVars, nil)
				assert.NoError(t, res.err, "Execute runs the command")
				src, ok := got.Source(tt.giveField(got))
				assert.True(t, ok, "the run has the source of the option")
				assert.Equal(t, src, tt.want, "the source states the variable of the prefix")
			})
		}

		t.Run("keeps the variable of the tag env", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, tokenOptions](t)
			app.EnvPrefix("STORE")
			res := execute(t, app, []string{command}, map[string]string{"API_TOKEN": "x"}, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			src, ok := got.Source(&got.Config().Token)
			assert.True(t, ok, "the run has the source of the option")
			assert.Equal(t, src, cli.Source{Kind: cli.Environment, Name: "API_TOKEN"},
				"the source states the variable of the tag")
		})

		t.Run("replaces the app name in the variable of --flag-debug", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[nameGlobal, nameOptions](t)
			app.EnvPrefix("STORE")
			res := execute(t, app, []string{command}, map[string]string{"STORE_FLAG_DEBUG": "true"}, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.NotEmpty(t, res.stderr, "Stderr receives the table of --flag-debug")
		})

		t.Run("shows the variable of the prefix in the help", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[nameGlobal, nameOptions](t)
			app.EnvPrefix("STORE")
			res := execute(t, app, []string{command, "--help"}, nil, nil)
			assert.NoError(t, res.err, "Execute writes the help")
			assert.Contains(t, res.stdout, "env STORE_SERVE_DRY_RUN", "the help states the variable of the prefix")
		})

		t.Run("uses the prefix of the last call", func(t *testing.T) {
			t.Parallel()
			app, got := serving[nameGlobal, nameOptions](t)
			app.EnvPrefix("OLD")
			app.EnvPrefix("STORE")
			vars := map[string]string{"OLD_SERVE_DRY_RUN": "true", "STORE_SERVE_DRY_RUN": "true"}
			res := execute(t, app, []string{command}, vars, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			src, ok := got.Source(&got.Config().DryRun)
			assert.True(t, ok, "the run has the source of the option")
			assert.Equal(t, src, cli.Source{Kind: cli.Environment, Name: "STORE_SERVE_DRY_RUN"},
				"the source states the variable of the last prefix")
		})

		invalid := []struct {
			name       string
			givePrefix string
		}{
			{name: "returns ErrDeclaration for a prefix in lower case", givePrefix: "store"},
			{name: "returns ErrDeclaration for a prefix that starts with a digit", givePrefix: "2STORE"},
			{name: "returns ErrDeclaration for a prefix that ends with an underscore", givePrefix: "STORE_"},
			{name: "returns ErrDeclaration for a prefix with two underscores in a row", givePrefix: "MY__STORE"},
			{name: "returns ErrDeclaration for a prefix with a dash", givePrefix: "MY-STORE"},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[nameGlobal, nameOptions](t)
				app.EnvPrefix(tt.givePrefix)
				res := execute(t, app, []string{command}, nil, nil)
				assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			})
		}

		t.Run("states the prefix in the error of an invalid prefix", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[nameGlobal, nameOptions](t)
			app.EnvPrefix("store")
			res := execute(t, app, []string{command}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, res.err.Error(),
				`cli: invalid declaration: the environment prefix "store" is not a name of upper-case words`,
				"the error states the prefix")
		})
	})
}

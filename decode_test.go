// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// decodeContract is the contract of the property that FuzzDecode fuzzes and TestDecode checks.
const decodeContract = "Decode leaves the value unchanged for every tree that it refuses with an *Error"

// errFlaky is the error of decodeFlaky on every second call.
var errFlaky = errors.New("main: the second call fails")

// flakyCalls counts the calls of decodeFlaky.UnmarshalText for each text. Each fixture with a
// field of decodeFlaky has a default of its own, so parallel cases do not share a count.
var flakyCalls = map[string]*atomic.Int64{"in-list": {}, "in-group": {}, "in-embedded": {}}

// The structs of the documents that the specs decode.
type (
	// decodeBinary is a command whose binary a release attaches.
	decodeBinary struct {
		Platforms []string
		Build     decodeBuild
		Sign      *decodeSign
		Name      string `required:""`
		Main      string
	}
	// decodeBuild is the group of the build settings of a binary.
	decodeBuild struct {
		Tags     []string
		Trimpath bool `default:"true"`
	}
	// decodeFuzz is the group of the options of the step fuzz.
	decodeFuzz struct {
		Args []string
		Time string
	}
	// decodeSign is the signature of a release.
	decodeSign struct {
		Key  string
		Mode string `default:"cosign" enum:"cosign,gpg"`
	}
	// decodeStep is a step of a job. Its group comes before its field with a default.
	decodeStep struct {
		Env   map[string]string
		Run   []string
		Cache decodeCache
		Name  string `default:"build"`
	}
	// decodeCache is the group of the cache of a step.
	decodeCache struct {
		Key string
	}
	// decodePlan is a plan of a workspace.
	decodePlan struct {
		DecodeToggle
		Weight int
	}
	// DecodeToggle is the struct that decodePlan embeds.
	DecodeToggle struct {
		Enabled bool `default:"true"`
	}
	// decodeSection is the section go of a document, with a field of each type that Decode takes.
	decodeSection struct {
		Labels   map[string]string
		Plans    map[string]decodePlan
		Hooks    map[string]*decodeStep
		Sign     *decodeSign
		Retries  *int
		Binaries []decodeBinary
		Steps    []*decodeStep
		Ports    []int
		Batches  [][]int
		Fuzz     decodeFuzz
		Timeout  time.Duration
		Jobs     int
		Verbose  bool
	}
	// decodeNames has fields whose members Decode names in kebab case and by the tag name, and three
	// unexported fields that Decode skips.
	decodeNames struct {
		decodeLevel
		inner    decodeTLS
		hidden   string
		Renamed  string `name:"alias"`
		HTTPPort int
		DryRun   bool
	}
	// decodeLevel is a type other than a struct that decodeNames embeds.
	decodeLevel int
	// decodeTLS is the group of the files of TLS.
	decodeTLS struct {
		Cert string
		Key  string
	}
	// decodeGroups has a group under the name of its tag prefix and a group with an empty prefix.
	decodeGroups struct {
		Server decodeTLS `prefix:"server-tls"`
		Client decodeTLS `prefix:""`
	}
	// decodeBase is the struct that decodeEmbedded embeds.
	decodeBase struct {
		Region string
	}
	// decodeHidden is the struct that decodeEmbedded embeds through an unexported pointer.
	decodeHidden struct {
		Secret string
	}
	// decodeEmbedded embeds a struct, an exported pointer to a struct and an unexported pointer to a
	// struct.
	decodeEmbedded struct {
		*DecodeExtra
		*decodeHidden
		decodeBase
		Name string
	}
	// decodeNode is a struct that refers to itself through a list.
	decodeNode struct {
		Children []decodeNode
		Name     string
	}
	// decodeFlaky is a text whose UnmarshalText fails on every second call for each text of
	// flakyCalls. Its default converts when Decode declares it and fails when Decode creates the
	// struct of the default.
	decodeFlaky string
	// decodeFlakyItem has a default of the type decodeFlaky.
	decodeFlakyItem struct {
		Text decodeFlaky `default:"in-list"`
	}
	// decodeFlakyList is a list of decodeFlakyItem.
	decodeFlakyList struct {
		Items []decodeFlakyItem
	}
	// decodeFlakyJob has a group with a default of the type decodeFlaky.
	decodeFlakyJob struct {
		Retry decodeFlakyRetry
	}
	// decodeFlakyRetry is the group of decodeFlakyJob.
	decodeFlakyRetry struct {
		Text decodeFlaky `default:"in-group"`
	}
	// decodeFlakyHost embeds a pointer to a struct with a default of the type decodeFlaky.
	decodeFlakyHost struct {
		*DecodeFlakyBase
	}
	// decodeKeys has maps whose keys convert to integers.
	decodeKeys struct {
		Plans  map[int]decodePlan
		Counts map[int]int
	}
	// decodeRequired has a required list of structs.
	decodeRequired struct {
		Binaries []decodeBinary `required:""`
	}
	// decodeRequiredLists has a required list of lists.
	decodeRequiredLists struct {
		Batches [][]int `required:""`
	}
)

// The structs that decodeEmbedded embeds through exported pointers. Their names are exported, so
// the fields of the pointers are exported too.
type (
	// DecodeExtra is the struct that decodeEmbedded embeds.
	DecodeExtra struct {
		*DecodeDeep
		Zone  string
		Level int `default:"3"`
	}
	// DecodeDeep is the struct that DecodeExtra embeds. It has a group and a field with a default.
	DecodeDeep struct {
		TLS   decodeTLS
		Depth int
		Floor int `default:"1"`
	}
	// DecodeFlakyBase is the struct that decodeFlakyHost embeds.
	DecodeFlakyBase struct {
		Text decodeFlaky `default:"in-embedded"`
		Name string
	}
)

// The structs that embed a pointer to themselves.
type (
	// DeclLoop embeds a pointer to itself.
	DeclLoop struct {
		*DeclLoop
	}
	// DeclRing embeds a pointer to DeclLink, which embeds a pointer to DeclRing.
	DeclRing struct {
		*DeclLink
	}
	// DeclLink has a group of the type declBack before its embedded pointer to DeclRing.
	DeclLink struct {
		Back declBack
		*DeclRing
	}
)

// declBack is the group of DeclLink. It embeds a pointer to DeclRing.
type declBack struct {
	*DeclRing
}

// The structs that Decode refuses to decode into.
type (
	declChan     struct{ Events chan int }
	declPointers struct{ Hooks []*string }
	declDouble   struct{ Sign **decodeSign }
	declNested   struct{ Paths map[string][]string }
	declLists    struct{ Events [][]chan int }
	declArray    struct{ Pair [2]int }
	declAny      struct{ Value any }
	declMapKey   struct{ Plans map[[2]int]decodePlan }
	declShort    struct {
		Name string `short:"n"`
	}
	declEnv struct {
		Name string `env:"NAME"`
	}
	declKey struct {
		Name string `key:"name"`
	}
	declArg struct {
		Name string `arg:""`
	}
	declSecret struct {
		Token string `secret:""`
	}
	declListDefault struct {
		Binaries []decodeBinary `default:"x"`
	}
	declListEnum struct {
		Binaries []decodeBinary `enum:"x"`
	}
	declListPrefix struct {
		Binaries []decodeBinary `prefix:"x"`
	}
	declListsDefault struct {
		Batches [][]int `default:"1"`
	}
	declGroupTag struct {
		Fuzz decodeFuzz `required:""`
	}
	declName struct {
		Name string `name:"Bad_Name"`
	}
	declListName struct {
		Binaries []decodeBinary `name:"Bad_Name"`
	}
	declRequiredDefault struct {
		Name string `required:"" default:"x"`
	}
	declDefault struct {
		Jobs int `default:"many"`
	}
	declTwice struct {
		decodeBase
		Region string
	}
	declGroupTwice struct {
		Name string `name:"fuzz"`
		Fuzz decodeFuzz
	}
	declElement struct{ Items []declChan }
)

// UnmarshalText sets f to b. It returns errFlaky on every second call with a text of flakyCalls.
func (f *decodeFlaky) UnmarshalText(b []byte) error {
	if calls, ok := flakyCalls[string(b)]; ok && calls.Add(1)%2 == 0 {
		return errFlaky
	}
	*f = decodeFlaky(b)
	return nil
}

// decodeMembers generates the names of the members of the trees that decodeTrees generates: the
// member go, the members of decodeSection and of its structs, and one other.
var decodeMembers = prop.SampledFrom(
	"go", "labels", "plans", "hooks", "sign", "retries", "binaries", "steps", "ports", "batches", "fuzz",
	"timeout", "jobs", "verbose", "name", "main", "platforms", "build", "tags", "trimpath", "args", "time",
	"key", "mode", "env", "run", "cache", "weight", "enabled", "other",
)

// decodeTrees generates the trees of a JSON document: objects of the names of decodeMembers,
// arrays, strings, numbers, booleans and nulls.
var decodeTrees = prop.Dict(decodeMembers, prop.Recursive(
	prop.OneOf(
		prop.Just[any](nil),
		prop.Boolean().Map(func(b bool) any { return b }),
		prop.StringMatching(`[a-z0-9.,=/ -]{0,6}`).Map(func(s string) any { return s }),
		prop.Integer[int64](-1000, 1000).Map(func(n int64) any { return json.Number(strconv.FormatInt(n, 10)) }),
	),
	func(self prop.Generator[any]) prop.Generator[any] {
		return prop.OneOf(
			prop.List(self, prop.MaxSize(3)).Map(func(items []any) any { return items }),
			prop.Dict(decodeMembers, self, prop.MaxSize(3)).Map(func(members map[string]any) any { return members }),
		)
	},
), prop.MaxSize(4))

// decodeBody decodes a generated tree into a decodeSection. It checks that a refusal is an *Error
// and leaves the value unchanged.
func decodeBody(c *prop.Case) {
	tree := c.Draw(decodeTrees, "tree")
	key := c.Draw(prop.SampledFrom("", "go"), "key")
	var v decodeSection
	err := cli.Decode(tree, key, &v)
	if err == nil {
		return
	}
	_ = assert.ErrorAs[*cli.Error](c, err, "Decode refuses the tree with an *Error")
	assert.Equal(c, v, decodeSection{}, "Decode leaves the value unchanged")
}

// FuzzDecode fuzzes Decode with the trees of JSON documents.
func FuzzDecode(f *testing.F) {
	prop.Fuzz(f, decodeContract, decodeBody)
}

// TestDecode checks how Decode decodes a member of a tree into the fields of a value, and the trees
// and the types that it refuses.
func TestDecode(t *testing.T) {
	t.Parallel()

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		t.Run("decodes a member into the field of its name in kebab case", func(t *testing.T) {
			t.Parallel()
			var got decodeNames
			tree := map[string]any{"dry-run": true, "http-port": json.Number("8080")}
			assert.NoError(t, cli.Decode(tree, "", &got), "Decode decodes the tree")
			assert.Equal(t, got, decodeNames{HTTPPort: 8080, DryRun: true},
				"Decode names DryRun dry-run and HTTPPort http-port")
		})

		t.Run("decodes a member into the field of its tag name", func(t *testing.T) {
			t.Parallel()
			var got decodeNames
			assert.NoError(t, cli.Decode(map[string]any{"alias": "x"}, "", &got), "Decode decodes the tree")
			assert.Equal(t, got, decodeNames{Renamed: "x"}, "Decode names Renamed by its tag")
		})

		t.Run("keeps the unexported fields of the value", func(t *testing.T) {
			t.Parallel()
			got := decodeNames{decodeLevel: 2, inner: decodeTLS{Cert: "c"}, hidden: "h"}
			assert.NoError(t, cli.Decode(map[string]any{"alias": "x"}, "", &got), "Decode decodes the tree")
			assert.Equal(t, got, decodeNames{decodeLevel: 2, inner: decodeTLS{Cert: "c"}, hidden: "h", Renamed: "x"},
				"Decode sets the field of the member alone")
		})

		unexported := []struct {
			name string
			give map[string]any
			want string
		}{
			{
				name: "returns ErrUnknownKey for a member of an unexported field",
				give: map[string]any{"hidden": "x"},
				want: "cli: unknown key hidden",
			},
			{
				name: "returns ErrUnknownKey for a member of an unexported group",
				give: map[string]any{"inner": map[string]any{}},
				want: "cli: unknown key inner",
			},
			{
				name: "returns ErrUnknownKey for a member of an unexported embedded field of a type other than a struct",
				give: map[string]any{"decode-level": "1"},
				want: "cli: unknown key decode-level",
			},
		}
		for _, tt := range unexported {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got decodeNames
				err := cli.Decode(tt.give, "", &got)
				assert.ErrorIs(t, err, cli.ErrUnknownKey, "Decode skips the unexported field")
				assert.Equal(t, err.Error(), tt.want, "the error states the member")
			})
		}

		t.Run("decodes the object of a group under the name of its prefix", func(t *testing.T) {
			t.Parallel()
			var got decodeGroups
			tree := map[string]any{"server-tls": map[string]any{"cert": "a"}}
			assert.NoError(t, cli.Decode(tree, "", &got), "Decode decodes the tree")
			assert.Equal(t, got, decodeGroups{Server: decodeTLS{Cert: "a"}}, "Decode decodes the group server-tls")
		})

		t.Run("decodes the members of a group with an empty prefix from the object of its parent", func(t *testing.T) {
			t.Parallel()
			var got decodeGroups
			assert.NoError(t, cli.Decode(map[string]any{"cert": "b", "key": "c"}, "", &got), "Decode decodes the tree")
			assert.Equal(t, got, decodeGroups{Client: decodeTLS{Cert: "b", Key: "c"}},
				"Decode decodes the members cert and key into the group Client")
		})

		t.Run("decodes the members of an embedded struct from the object of its parent", func(t *testing.T) {
			t.Parallel()
			var got decodeEmbedded
			assert.NoError(t, cli.Decode(map[string]any{"region": "eu", "name": "n"}, "", &got),
				"Decode decodes the tree")
			assert.Equal(t, got, decodeEmbedded{Region: "eu", Name: "n"},
				"Decode decodes the member region into the embedded struct")
		})

		t.Run("creates the target of an embedded pointer for a member of its struct", func(t *testing.T) {
			t.Parallel()
			var got decodeEmbedded
			assert.NoError(t, cli.Decode(map[string]any{"zone": "west"}, "", &got), "Decode decodes the tree")
			assert.Equal(t, got.DecodeExtra, &DecodeExtra{Zone: "west", Level: 3},
				"Decode creates the target with the defaults of its tags")
		})

		t.Run("keeps an embedded pointer whose struct has no member in the object", func(t *testing.T) {
			t.Parallel()
			var got decodeEmbedded
			assert.NoError(t, cli.Decode(map[string]any{"name": "n"}, "", &got), "Decode decodes the tree")
			assert.Nil(t, got.DecodeExtra, "the embedded pointer stays nil")
		})

		t.Run("decodes into a copy of the target of an embedded pointer", func(t *testing.T) {
			t.Parallel()
			old := &DecodeExtra{Zone: "us", Level: 1}
			got := decodeEmbedded{DecodeExtra: old}
			tree := map[string]any{"level": json.Number("2")}
			assert.NoError(t, cli.Decode(tree, "", &got), "Decode decodes the tree")
			assert.Equal(t, *got.DecodeExtra, DecodeExtra{Zone: "us", Level: 2},
				"the copy keeps the zone of the target")
			assert.Equal(t, *old, DecodeExtra{Zone: "us", Level: 1}, "Decode does not write into the old target")
		})

		t.Run("returns ErrUnknownKey for a member of a struct behind an unexported embedded pointer",
			func(t *testing.T) {
				t.Parallel()
				var got decodeEmbedded
				err := cli.Decode(map[string]any{"secret": "s"}, "", &got)
				assert.ErrorIs(t, err, cli.ErrUnknownKey, "Decode skips the unexported field")
				assert.Equal(t, err.Error(), "cli: unknown key secret", "the error names the member")
				assert.Nil(t, got.decodeHidden, "the unexported pointer stays nil")
			})

		t.Run("creates the targets of two embedded pointers for a member of the inner struct", func(t *testing.T) {
			t.Parallel()
			var got decodeEmbedded
			tree := map[string]any{"depth": json.Number("2")}
			assert.NoError(t, cli.Decode(tree, "", &got), "Decode decodes the tree")
			assert.Equal(t, got.DecodeExtra, &DecodeExtra{DecodeDeep: &DecodeDeep{Depth: 2, Floor: 1}, Level: 3},
				"Decode creates both targets with the defaults of their tags")
		})

		t.Run("decodes a struct that refers to itself through a list", func(t *testing.T) {
			t.Parallel()
			var got decodeNode
			tree := map[string]any{"name": "root", "children": []any{
				map[string]any{"name": "a", "children": []any{map[string]any{"name": "b"}}},
			}}
			assert.NoError(t, cli.Decode(tree, "", &got), "Decode decodes the tree")
			assert.Equal(t, got, decodeNode{Name: "root", Children: []decodeNode{
				{Name: "a", Children: []decodeNode{{Name: "b"}}},
			}}, "Decode decodes each level of the nodes")
		})

		values := []struct {
			name string
			give map[string]any
			want decodeSection
		}{
			{
				name: "converts the text of a value to the type of its field",
				give: map[string]any{"jobs": json.Number("4"), "timeout": "90s", "verbose": "true"},
				want: decodeSection{Jobs: 4, Timeout: 90 * time.Second, Verbose: true},
			},
			{
				name: "sets a pointer to a new value",
				give: map[string]any{"retries": "3"},
				want: decodeSection{Retries: new(3)},
			},
			{
				name: "replaces a list with the values of an array",
				give: map[string]any{"ports": []any{json.Number("80"), "443"}},
				want: decodeSection{Ports: []int{80, 443}},
			},
			{
				name: "replaces a list with the values of a text that commas separate",
				give: map[string]any{"ports": "80,443"},
				want: decodeSection{Ports: []int{80, 443}},
			},
			{
				name: "replaces a list with an empty list for an empty array",
				give: map[string]any{"ports": []any{}},
				want: decodeSection{Ports: []int{}},
			},
			{
				name: "replaces a list of lists with the arrays of an array",
				give: map[string]any{"batches": []any{[]any{json.Number("1"), "2"}, []any{}}},
				want: decodeSection{Batches: [][]int{{1, 2}, {}}},
			},
			{
				name: "replaces a map with the entries of an object",
				give: map[string]any{"labels": map[string]any{"team": "core"}},
				want: decodeSection{Labels: map[string]string{"team": "core"}},
			},
			{
				name: "replaces a map with the entries of a text",
				give: map[string]any{"labels": "team=core,tier=1"},
				want: decodeSection{Labels: map[string]string{"team": "core", "tier": "1"}},
			},
			{
				name: "decodes the object of a group into its fields",
				give: map[string]any{"fuzz": map[string]any{"time": "30s", "args": []any{"-x"}}},
				want: decodeSection{Fuzz: decodeFuzz{Args: []string{"-x"}, Time: "30s"}},
			},
			{
				name: "creates the structs of a list of structs with the defaults of their tags",
				give: map[string]any{"binaries": []any{map[string]any{"name": "ergon"}}},
				want: decodeSection{Binaries: []decodeBinary{{Build: decodeBuild{Trimpath: true}, Name: "ergon"}}},
			},
			{
				name: "creates the targets of a list of pointers to structs",
				give: map[string]any{"steps": []any{map[string]any{"run": []any{"make"}}}},
				want: decodeSection{Steps: []*decodeStep{{Run: []string{"make"}, Name: "build"}}},
			},
			{
				name: "creates the structs of a map of structs",
				give: map[string]any{"plans": map[string]any{"api": map[string]any{"weight": json.Number("2")}}},
				want: decodeSection{Plans: map[string]decodePlan{"api": {Weight: 2, Enabled: true}}},
			},
			{
				name: "creates the targets of a map of pointers to structs",
				give: map[string]any{"hooks": map[string]any{"pre": map[string]any{"name": "lint"}}},
				want: decodeSection{Hooks: map[string]*decodeStep{"pre": {Name: "lint"}}},
			},
			{
				name: "creates the target of a nil pointer to a struct with the defaults of its tags",
				give: map[string]any{"sign": map[string]any{"key": "k"}},
				want: decodeSection{Sign: &decodeSign{Key: "k", Mode: "cosign"}},
			},
			{
				name: "keeps a field whose member is nil",
				give: map[string]any{"jobs": nil, "sign": nil},
				want: decodeSection{},
			},
		}
		for _, tt := range values {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got decodeSection
				assert.NoError(t, cli.Decode(map[string]any{"go": tt.give}, "go", &got), "Decode decodes the section")
				assert.Equal(t, got, tt.want, "Decode sets the fields of the members")
			})
		}

		t.Run("replaces the list of the value", func(t *testing.T) {
			t.Parallel()
			got := decodeSection{Ports: []int{1, 2, 3}}
			assert.NoError(t, cli.Decode(map[string]any{"ports": []any{"4"}}, "", &got), "Decode decodes the tree")
			assert.Equal(t, got.Ports, []int{4}, "the array replaces the list")
		})

		t.Run("replaces the map of the value", func(t *testing.T) {
			t.Parallel()
			got := decodeSection{Labels: map[string]string{"a": "1"}}
			assert.NoError(t, cli.Decode(map[string]any{"labels": map[string]any{"b": "2"}}, "", &got),
				"Decode decodes the tree")
			assert.Equal(t, got.Labels, map[string]string{"b": "2"}, "the object replaces the map")
		})

		t.Run("keeps every field of a group without a member", func(t *testing.T) {
			t.Parallel()
			got := decodeSection{Fuzz: decodeFuzz{Args: []string{"-x"}, Time: "30s"}}
			assert.NoError(t, cli.Decode(map[string]any{"fuzz": map[string]any{"time": "1m"}}, "", &got),
				"Decode decodes the tree")
			assert.Equal(t, got.Fuzz, decodeFuzz{Args: []string{"-x"}, Time: "1m"}, "Decode keeps the arguments")
		})

		t.Run("decodes into a copy of the target of a pointer to a struct", func(t *testing.T) {
			t.Parallel()
			old := &decodeSign{Key: "a", Mode: "gpg"}
			got := decodeSection{Sign: old}
			assert.NoError(t, cli.Decode(map[string]any{"sign": map[string]any{"key": "b"}}, "", &got),
				"Decode decodes the tree")
			expect.Equal(t, *got.Sign, decodeSign{Key: "b", Mode: "gpg"}, "the copy keeps the mode of the target")
			expect.Equal(t, *old, decodeSign{Key: "a", Mode: "gpg"}, "Decode does not write into the old target")
		})

		t.Run("keeps a pointer to a struct without a member", func(t *testing.T) {
			t.Parallel()
			old := &decodeSign{Key: "a"}
			got := decodeSection{Sign: old}
			assert.NoError(t, cli.Decode(map[string]any{"jobs": "1"}, "", &got), "Decode decodes the tree")
			assert.Equal(t, got.Sign, old, "the pointer keeps its target", assert.ByIdentity())
		})

		t.Run("does not apply the defaults to the value", func(t *testing.T) {
			t.Parallel()
			var got decodeSign
			assert.NoError(t, cli.Decode(map[string]any{"key": "k"}, "", &got), "Decode decodes the tree")
			assert.Equal(t, got, decodeSign{Key: "k"}, "the mode keeps its zero value")
		})

		t.Run("leaves the tree unchanged when a caller changes the decoded value", func(t *testing.T) {
			t.Parallel()
			tree := map[string]any{"go": map[string]any{
				"ports":  []any{"80"},
				"labels": map[string]any{"a": "1"},
				"steps":  []any{map[string]any{"run": []any{"make"}}},
			}}
			var got decodeSection
			var err error
			assert.Pure(t, func() string {
				data, encodeErr := json.Marshal(tree)
				assert.NoError(t, encodeErr, "the tree encodes")
				return string(data)
			}, func() {
				err = cli.Decode(tree, "go", &got)
				got.Ports[0], got.Labels["a"], got.Steps[0].Run[0] = 1, "2", "test"
			}, "Decode does not modify the tree, and a change of the value does not change it")
			assert.NoError(t, err, "Decode decodes the tree")
		})

		keys := []struct {
			name    string
			give    map[string]any
			giveKey string
			want    decodeSection
		}{
			{
				name: "decodes the whole tree for the empty key",
				give: map[string]any{"jobs": "3"},
				want: decodeSection{Jobs: 3},
			},
			{
				name:    "decodes the member of a key of two names",
				give:    map[string]any{"ci": map[string]any{"go": map[string]any{"jobs": "3"}}},
				giveKey: "ci.go",
				want:    decodeSection{Jobs: 3},
			},
			{
				name:    "leaves the value unchanged for a key that the tree does not contain",
				give:    map[string]any{"ci": map[string]any{}},
				giveKey: "ci.go",
				want:    decodeSection{Jobs: 7},
			},
			{
				name:    "leaves the value unchanged for a key whose member is nil",
				give:    map[string]any{"ci": nil},
				giveKey: "ci.go",
				want:    decodeSection{Jobs: 7},
			},
		}
		for _, tt := range keys {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := decodeSection{Jobs: 7}
				assert.NoError(t, cli.Decode(tt.give, tt.giveKey, &got), "Decode decodes the member of the key")
				assert.Equal(t, got, tt.want, "Decode sets the fields of the member")
			})
		}

		t.Run("decodes the member of a key into a value of any type that a field can have", func(t *testing.T) {
			t.Parallel()
			tree := map[string]any{
				"jobs":    "4",
				"sign":    map[string]any{"key": "k"},
				"plans":   map[string]any{"api": map[string]any{}},
				"zone":    "UTC",
				"at":      "2026-10-16T12:00:00Z",
				"batches": []any{[]any{"1", "2"}},
			}
			var jobs int
			var sign *decodeSign
			var plans map[string]decodePlan
			var zone *time.Location
			var at *time.Time
			var batches [][]int
			assert.NoError(t, cli.Decode(tree, "jobs", &jobs), "Decode decodes a value")
			assert.NoError(t, cli.Decode(tree, "sign", &sign), "Decode decodes a pointer to a struct")
			assert.NoError(t, cli.Decode(tree, "plans", &plans), "Decode decodes a map of structs")
			assert.NoError(t, cli.Decode(tree, "zone", &zone), "Decode decodes a pointer with a conversion")
			assert.NoError(t, cli.Decode(tree, "at", &at), "Decode decodes a pointer to a struct with a conversion")
			assert.NoError(t, cli.Decode(tree, "batches", &batches), "Decode decodes a list of lists")
			expect.Equal(t, jobs, 4, "Decode converts the value")
			expect.Equal(t, sign, &decodeSign{Key: "k", Mode: "cosign"}, "Decode creates the target")
			expect.Equal(t, plans, map[string]decodePlan{"api": {Enabled: true}}, "Decode creates the plan")
			expect.Equal(t, zone, time.UTC, "Decode loads the location")
			expect.Equal(t, *at, time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC), "Decode parses the time")
			expect.Equal(t, batches, [][]int{{1, 2}}, "Decode converts the values of each list")
		})

		t.Run("decodes a list of a slice type with a conversion as a list of values", func(t *testing.T) {
			t.Parallel()
			var got []net.IP
			tree := map[string]any{"hosts": []any{"10.0.0.1", "::1"}}
			assert.NoError(t, cli.Decode(tree, "hosts", &got), "Decode decodes the member")
			assert.Equal(t, got, []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("::1")},
				"Decode converts each value with the UnmarshalText of net.IP")
		})

		invalid := []struct {
			name string
			give map[string]any
			want string
		}{
			{
				name: "returns ErrInvalidValue for a value that does not convert",
				give: map[string]any{"jobs": "many"},
				want: `cli: invalid go.jobs "many": invalid syntax`,
			},
			{
				name: "returns ErrInvalidValue for a value that the enum of its field excludes",
				give: map[string]any{"sign": map[string]any{"mode": "pgp"}},
				want: `cli: invalid go.sign.mode "pgp": want cosign or gpg`,
			},
			{
				name: "returns ErrInvalidValue for an element of an array that does not convert",
				give: map[string]any{"ports": []any{"80", "x"}},
				want: `cli: invalid go.ports[1] "x": invalid syntax`,
			},
			{
				name: "returns ErrInvalidValue for a value of a text that does not convert",
				give: map[string]any{"ports": "80,x"},
				want: `cli: invalid go.ports "x": invalid syntax`,
			},
			{
				name: "returns ErrInvalidValue for an entry of a text without an equals sign",
				give: map[string]any{"labels": "team"},
				want: `cli: invalid go.labels "team": want key=value`,
			},
			{
				name: "returns ErrInvalidValue for an object for a value",
				give: map[string]any{"jobs": map[string]any{}},
				want: `cli: invalid go.jobs: want a value`,
			},
			{
				name: "returns ErrInvalidValue for an array for a value",
				give: map[string]any{"jobs": []any{"1"}},
				want: `cli: invalid go.jobs: want a value`,
			},
			{
				name: "returns ErrInvalidValue for an object for a pointer to a value",
				give: map[string]any{"retries": map[string]any{}},
				want: `cli: invalid go.retries: want a value`,
			},
			{
				name: "returns ErrInvalidValue for an object for a list of values",
				give: map[string]any{"ports": map[string]any{}},
				want: `cli: invalid go.ports: want an array`,
			},
			{
				name: "returns ErrInvalidValue for an object in an array of values",
				give: map[string]any{"ports": []any{map[string]any{}}},
				want: `cli: invalid go.ports[0]: want a value`,
			},
			{
				name: "returns ErrInvalidValue for a nil element of an array of values",
				give: map[string]any{"ports": []any{nil}},
				want: `cli: invalid go.ports[0]: want a value`,
			},
			{
				name: "returns ErrInvalidValue for a value for a list of lists",
				give: map[string]any{"batches": "1"},
				want: `cli: invalid go.batches "1": want an array`,
			},
			{
				name: "returns ErrInvalidValue for a text in an array of lists",
				give: map[string]any{"batches": []any{"1,2"}},
				want: `cli: invalid go.batches[0] "1,2": want an array`,
			},
			{
				name: "returns ErrInvalidValue for a nil element of an array of lists",
				give: map[string]any{"batches": []any{nil}},
				want: `cli: invalid go.batches[0]: want an array`,
			},
			{
				name: "returns ErrInvalidValue for a value of an inner array that does not convert",
				give: map[string]any{"batches": []any{[]any{"1"}, []any{"2", "x"}}},
				want: `cli: invalid go.batches[1][1] "x": invalid syntax`,
			},
			{
				name: "returns ErrInvalidValue for an array in an inner array",
				give: map[string]any{"batches": []any{[]any{[]any{}}}},
				want: `cli: invalid go.batches[0][0]: want a value`,
			},
			{
				name: "returns ErrInvalidValue for an array for a map of values",
				give: map[string]any{"labels": []any{"a"}},
				want: `cli: invalid go.labels: want an object`,
			},
			{
				name: "returns ErrInvalidValue for an array as the value of an entry of a map of values",
				give: map[string]any{"labels": map[string]any{"a": []any{}}},
				want: `cli: invalid go.labels.a: want a value`,
			},
			{
				name: "returns ErrInvalidValue for a value for a group",
				give: map[string]any{"fuzz": "x"},
				want: `cli: invalid go.fuzz "x": want an object`,
			},
			{
				name: "returns ErrInvalidValue for a value for a pointer to a struct",
				give: map[string]any{"sign": "x"},
				want: `cli: invalid go.sign "x": want an object`,
			},
			{
				name: "returns ErrInvalidValue for a value for a list of structs",
				give: map[string]any{"binaries": "x"},
				want: `cli: invalid go.binaries "x": want an array`,
			},
			{
				name: "returns ErrInvalidValue for a value in an array of structs",
				give: map[string]any{"binaries": []any{"x"}},
				want: `cli: invalid go.binaries[0] "x": want an object`,
			},
			{
				name: "returns ErrInvalidValue for a nil element of an array of structs",
				give: map[string]any{"binaries": []any{nil}},
				want: `cli: invalid go.binaries[0]: want an object`,
			},
			{
				name: "returns ErrInvalidValue for an array for a map of structs",
				give: map[string]any{"plans": []any{}},
				want: `cli: invalid go.plans: want an object`,
			},
			{
				name: "returns ErrInvalidValue for a value as the entry of a map of structs",
				give: map[string]any{"plans": map[string]any{"api": "x"}},
				want: `cli: invalid go.plans.api "x": want an object`,
			},
			{
				name: "returns ErrInvalidValue for the member of the first name in the order of their names",
				give: map[string]any{"verbose": "maybe", "jobs": "x", "binaries": "y"},
				want: `cli: invalid go.binaries "y": want an array`,
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got decodeSection
				err := cli.Decode(map[string]any{"go": tt.give}, "go", &got)
				assert.ErrorIs(t, err, cli.ErrInvalidValue, "Decode returns ErrInvalidValue")
				assert.Equal(t, err.Error(), tt.want, "the error states the key and the reason")
			})
		}

		keyed := []struct {
			name string
			give map[string]any
			want string
		}{
			{
				name: "returns ErrInvalidValue for a key of a map of structs that does not convert",
				give: map[string]any{"plans": map[string]any{"x": map[string]any{}}},
				want: `cli: invalid plans "x": invalid syntax`,
			},
			{
				name: "returns ErrInvalidValue for a key of a map of values that does not convert",
				give: map[string]any{"counts": map[string]any{"x": "1"}},
				want: `cli: invalid counts "x": invalid syntax`,
			},
			{
				name: "returns ErrInvalidValue for the value of an entry of a map of values that does not convert",
				give: map[string]any{"counts": map[string]any{"1": "x"}},
				want: `cli: invalid counts.1 "x": invalid syntax`,
			},
		}
		for _, tt := range keyed {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got decodeKeys
				err := cli.Decode(tt.give, "", &got)
				assert.ErrorIs(t, err, cli.ErrInvalidValue, "Decode returns ErrInvalidValue")
				assert.Equal(t, err.Error(), tt.want, "the error states the map and the key")
			})
		}

		t.Run("returns ErrInvalidValue for a name of the key that is not an object", func(t *testing.T) {
			t.Parallel()
			var got decodeSection
			err := cli.Decode(map[string]any{"ci": "x"}, "ci.go", &got)
			assert.ErrorIs(t, err, cli.ErrInvalidValue, "Decode returns ErrInvalidValue")
			assert.Equal(t, err.Error(), `cli: invalid ci "x": want an object`, "the error states the name")
		})

		t.Run("returns ErrInvalidValue for the whole tree into a value", func(t *testing.T) {
			t.Parallel()
			var jobs int
			err := cli.Decode(map[string]any{}, "", &jobs)
			assert.ErrorIs(t, err, cli.ErrInvalidValue, "Decode returns ErrInvalidValue")
			assert.Equal(t, err.Error(), "cli: invalid tree: want a value", "the error states the tree")
		})

		t.Run("states the key and the value of an invalid member in the *Error", func(t *testing.T) {
			t.Parallel()
			var got decodeSection
			err := cli.Decode(map[string]any{"go": map[string]any{"jobs": "many"}}, "go", &got)
			e := assert.ErrorAs[*cli.Error](t, err, "Decode returns an *Error")
			expect.Equal(t, e.Input, "go.jobs", "the input is the key of the member")
			expect.Equal(t, e.Value, "many", "the value is the text of the member")
			expect.Equal(t, e.Command, "", "the error has no command")
		})

		t.Run("returns ErrUnknownKey with the suggestion of the closest member", func(t *testing.T) {
			t.Parallel()
			var got decodeSection
			tree := map[string]any{"go": map[string]any{"binaries": []any{map[string]any{"name": "ergon", "nme": "x"}}}}
			err := cli.Decode(tree, "go", &got)
			e := assert.ErrorAs[*cli.Error](t, err, "Decode returns an *Error")
			expect.ErrorIs(t, err, cli.ErrUnknownKey, "Decode returns ErrUnknownKey")
			expect.Equal(t, err.Error(), "cli: unknown key go.binaries[0].nme (did you mean go.binaries[0].name?)",
				"the error states the key and the suggestion")
			expect.Equal(t, e.Input, "go.binaries[0].nme", "the input is the key of the member")
			expect.Equal(t, e.Suggestion, "go.binaries[0].name", "the suggestion is the key of the field")
		})

		t.Run("returns ErrUnknownKey without a suggestion for a member far from every field", func(t *testing.T) {
			t.Parallel()
			var got decodeSection
			err := cli.Decode(map[string]any{"go": map[string]any{"zzzzzz": "1"}}, "go", &got)
			assert.ErrorIs(t, err, cli.ErrUnknownKey, "Decode returns ErrUnknownKey")
			assert.Equal(t, err.Error(), "cli: unknown key go.zzzzzz", "the error states no suggestion")
		})

		missing := []struct {
			name string
			give map[string]any
			want string
		}{
			{
				name: "returns ErrMissing for an element without the member of a required field",
				give: map[string]any{"binaries": []any{map[string]any{"main": "./cmd"}}},
				want: "cli: missing binaries[0].name",
			},
			{
				name: "returns ErrMissing for an element whose required member is nil",
				give: map[string]any{"binaries": []any{map[string]any{"name": nil}}},
				want: "cli: missing binaries[0].name",
			},
		}
		for _, tt := range missing {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got decodeSection
				err := cli.Decode(tt.give, "", &got)
				assert.ErrorIs(t, err, cli.ErrMissing, "Decode returns ErrMissing")
				assert.Equal(t, err.Error(), tt.want, "the error states the key of the member")
			})
		}

		t.Run("returns ErrMissing for a tree without the member of a required list", func(t *testing.T) {
			t.Parallel()
			var got decodeRequired
			err := cli.Decode(map[string]any{}, "", &got)
			assert.ErrorIs(t, err, cli.ErrMissing, "Decode returns ErrMissing")
			assert.Equal(t, err.Error(), "cli: missing binaries", "the error states the key of the member")
		})

		t.Run("returns ErrMissing for a tree without the member of a required list of lists", func(t *testing.T) {
			t.Parallel()
			var got decodeRequiredLists
			err := cli.Decode(map[string]any{}, "", &got)
			assert.ErrorIs(t, err, cli.ErrMissing, "Decode returns ErrMissing")
			assert.Equal(t, err.Error(), "cli: missing batches", "the error states the key of the member")
		})

		t.Run("returns the same error on every call for a tree of several faults", func(t *testing.T) {
			t.Parallel()
			tree := map[string]any{"verbose": "maybe", "jobs": "x", "binaries": "y", "fuzz": "z", "other": "w"}
			assert.Deterministic(t, func(tree map[string]any) (string, error) {
				var got decodeSection
				return cli.Decode(tree, "", &got).Error(), nil
			}, tree, "Decode reports the fault of the first member in the order of the names")
		})

		declarations := []struct {
			name string
			give func() error
			want string
		}{
			{
				name: "returns ErrDeclaration for a nil v",
				give: func() error { return cli.Decode(map[string]any{}, "", nil) },
				want: "cli: invalid declaration: Decode needs a non-nil pointer, not <nil>",
			},
			{
				name: "returns ErrDeclaration for a v that is not a pointer",
				give: func() error { return cli.Decode(map[string]any{}, "", decodeSection{}) },
				want: "cli: invalid declaration: Decode needs a non-nil pointer, not cli_test.decodeSection",
			},
			{
				name: "returns ErrDeclaration for a nil pointer",
				give: func() error { return cli.Decode(map[string]any{}, "", (*decodeSection)(nil)) },
				want: "cli: invalid declaration: Decode needs a non-nil pointer, not a nil *cli_test.decodeSection",
			},
			{
				name: "returns ErrDeclaration for a key with an empty name",
				give: func() error { return cli.Decode(map[string]any{}, "ci..go", new(decodeSection)) },
				want: `cli: invalid declaration: the key "ci..go" has an empty name`,
			},
			{
				name: "returns ErrDeclaration for a value of a type without a conversion",
				give: func() error { return cli.Decode(map[string]any{}, "", new(chan int)) },
				want: "cli: invalid declaration: the value of Decode has the type chan int, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a channel",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declChan)) },
				want: "cli: invalid declaration: the field cli_test.declChan.Events has the type chan int, " +
					"which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a list of pointers to strings",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declPointers)) },
				want: "cli: invalid declaration: the field cli_test.declPointers.Hooks has the type []*string, " +
					"which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a pointer to a pointer",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declDouble)) },
				want: "cli: invalid declaration: the field cli_test.declDouble.Sign has the type **cli_test.decodeSign, " +
					"which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a map of structs whose keys have no conversion",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declMapKey)) },
				want: "cli: invalid declaration: the field cli_test.declMapKey.Plans has the type " +
					"map[[2]int]cli_test.decodePlan, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a map of lists",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declNested)) },
				want: "cli: invalid declaration: the field cli_test.declNested.Paths has the type map[string][]string, " +
					"which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a list of lists without a conversion",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declLists)) },
				want: "cli: invalid declaration: each list of the field cli_test.declLists.Events has the type " +
					"[]chan int, which has no conversion",
			},
			{
				name: "returns ErrDeclaration for a value of a list of lists without a conversion",
				give: func() error { return cli.Decode(map[string]any{}, "", new([][]chan int)) },
				want: "cli: invalid declaration: each list of the value of Decode has the type []chan int, " +
					"which has no conversion",
			},
			{
				name: "returns ErrDeclaration for an array",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declArray)) },
				want: "cli: invalid declaration: the field cli_test.declArray.Pair has the type [2]int, " +
					"which has no conversion",
			},
			{
				name: "returns ErrDeclaration for an interface",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declAny)) },
				want: "cli: invalid declaration: the field cli_test.declAny.Value has the type interface {}, " +
					"which has no conversion",
			},
			{
				name: "returns ErrDeclaration for the struct of an element without a conversion",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declElement)) },
				want: "cli: invalid declaration: the field cli_test.declChan.Events has the type chan int, " +
					"which has no conversion",
			},
			{
				name: "returns ErrDeclaration for the tag short",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declShort)) },
				want: "cli: invalid declaration: the field cli_test.declShort.Name takes no tag short in a document",
			},
			{
				name: "returns ErrDeclaration for the tag env",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declEnv)) },
				want: "cli: invalid declaration: the field cli_test.declEnv.Name takes no tag env in a document",
			},
			{
				name: "returns ErrDeclaration for the tag key",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declKey)) },
				want: "cli: invalid declaration: the field cli_test.declKey.Name takes no tag key in a document",
			},
			{
				name: "returns ErrDeclaration for the tag arg",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declArg)) },
				want: "cli: invalid declaration: the field cli_test.declArg.Name takes no tag arg in a document",
			},
			{
				name: "returns ErrDeclaration for the tag secret",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declSecret)) },
				want: "cli: invalid declaration: the field cli_test.declSecret.Token takes no tag secret in a document",
			},
			{
				name: "returns ErrDeclaration for the tag default of a list of structs",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declListDefault)) },
				want: "cli: invalid declaration: the field cli_test.declListDefault.Binaries takes no tag default " +
					"in a document",
			},
			{
				name: "returns ErrDeclaration for the tag enum of a list of structs",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declListEnum)) },
				want: "cli: invalid declaration: the field cli_test.declListEnum.Binaries takes no tag enum in a document",
			},
			{
				name: "returns ErrDeclaration for the tag prefix of a list of structs",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declListPrefix)) },
				want: "cli: invalid declaration: the field cli_test.declListPrefix.Binaries takes no tag prefix " +
					"in a document",
			},
			{
				name: "returns ErrDeclaration for the tag default of a list of lists",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declListsDefault)) },
				want: "cli: invalid declaration: the field cli_test.declListsDefault.Batches takes no tag default " +
					"in a document",
			},
			{
				name: "returns ErrDeclaration for a tag of an option on a group",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declGroupTag)) },
				want: "cli: invalid declaration: the group cli_test.declGroupTag.Fuzz takes no tag required",
			},
			{
				name: "returns ErrDeclaration for a name that is not a name of lower-case words",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declName)) },
				want: `cli: invalid declaration: the field cli_test.declName.Name has the name "Bad_Name", ` +
					"which is not a name of lower-case words",
			},
			{
				name: "returns ErrDeclaration for the name of a list that is not a name of lower-case words",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declListName)) },
				want: `cli: invalid declaration: the field cli_test.declListName.Binaries has the name "Bad_Name", ` +
					"which is not a name of lower-case words",
			},
			{
				name: "returns ErrDeclaration for two fields of one member",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declTwice)) },
				want: "cli: invalid declaration: two fields of cli_test.declTwice have the member region",
			},
			{
				name: "returns ErrDeclaration for a group with the member of another field",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declGroupTwice)) },
				want: "cli: invalid declaration: two fields of cli_test.declGroupTwice have the member fuzz",
			},
			{
				name: "returns ErrDeclaration for a struct that embeds a pointer to itself",
				give: func() error { return cli.Decode(map[string]any{}, "", new(DeclLoop)) },
				want: "cli: invalid declaration: the field cli_test.DeclLoop.DeclLoop points to the struct " +
					"cli_test.DeclLoop, which contains the field",
			},
			{
				name: "returns ErrDeclaration for a struct that embeds a pointer to itself through another struct",
				give: func() error { return cli.Decode(map[string]any{}, "", new(DeclRing)) },
				want: "cli: invalid declaration: the field cli_test.declBack.DeclRing.DeclLink.DeclRing points to " +
					"the struct cli_test.DeclRing, which contains the field",
			},
			{
				name: "returns ErrDeclaration for a required field with a default",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declRequiredDefault)) },
				want: "cli: invalid declaration: the field cli_test.declRequiredDefault.Name is required and has " +
					"a default, which the tag required makes void",
			},
			{
				name: "returns ErrDeclaration for a default that does not convert",
				give: func() error { return cli.Decode(map[string]any{}, "", new(declDefault)) },
				want: `cli: invalid declaration: the field cli_test.declDefault.Jobs has the default "many", ` +
					`whose value "many" is invalid: invalid syntax`,
			},
			{
				name: "returns ErrDeclaration for a tree without the key",
				give: func() error { return cli.Decode(map[string]any{}, "absent", new(declChan)) },
				want: "cli: invalid declaration: the field cli_test.declChan.Events has the type chan int, " +
					"which has no conversion",
			},
		}
		for _, tt := range declarations {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.give()
				assert.ErrorIs(t, err, cli.ErrDeclaration, "Decode returns ErrDeclaration")
				assert.Equal(t, err.Error(), tt.want, "the error states the fault of the declaration")
			})
		}

		flaky := []struct {
			name string
			give func() error
			want string
		}{
			{
				name: "returns ErrDeclaration for a default that fails to convert when Decode creates an element",
				give: func() error {
					return cli.Decode(map[string]any{"items": []any{map[string]any{}}}, "", new(decodeFlakyList))
				},
				want: `cli: invalid declaration: the field cli_test.decodeFlakyItem.Text has the default "in-list", ` +
					`whose value "in-list" is invalid: main: the second call fails`,
			},
			{
				name: "returns ErrDeclaration for a default of a group that fails to convert when Decode creates " +
					"the target of a pointer",
				give: func() error {
					return cli.Decode(map[string]any{"job": map[string]any{}}, "job", new(*decodeFlakyJob))
				},
				want: `cli: invalid declaration: the field cli_test.decodeFlakyRetry.Text has the default ` +
					`"in-group", whose value "in-group" is invalid: main: the second call fails`,
			},
			{
				name: "returns ErrDeclaration for a default that fails to convert when Decode creates the target " +
					"of an embedded pointer",
				give: func() error {
					return cli.Decode(map[string]any{"name": "n"}, "", new(decodeFlakyHost))
				},
				want: `cli: invalid declaration: the field cli_test.DecodeFlakyBase.Text has the default ` +
					`"in-embedded", whose value "in-embedded" is invalid: main: the second call fails`,
			},
		}
		for _, tt := range flaky {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.give()
				assert.ErrorIs(t, err, cli.ErrDeclaration, "Decode returns ErrDeclaration")
				assert.Equal(t, err.Error(), tt.want, "the error states the default and the error of its conversion")
			})
		}

		t.Run("returns the base with the fields that a document states", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, "Decode sets each field that a document states, and every other field keeps its value",
				func(c *prop.Case) {
					base := c.Draw(prop.Of[decodeSection](), "base")
					before := deepCopy(reflect.ValueOf(base)).Interface().(decodeSection)
					tree, want := drawSection(c, base)
					got := base
					assert.NoError(c, cli.Decode(map[string]any{"go": tree}, "go", &got), "Decode decodes the section")
					assert.Equal(c, got, want, "Decode sets the stated fields and keeps the others",
						assert.EquateEmpty())
					assert.Equal(c, base, before, "Decode does not write into a list, a map or a target of the value")
				})
		})

		t.Run("leaves the value unchanged when the document has a fault", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, "Decode leaves the value unchanged when the document has a fault", func(c *prop.Case) {
				got := c.Draw(prop.Of[decodeSection](), "base")
				tree, _ := drawSection(c, got)
				fault := c.Draw(prop.SampledFrom("unknown", "invalid", "kind", "missing"), "fault")
				switch fault {
				case "unknown":
					tree["unknown"] = "x"
				case "invalid":
					tree["verbose"] = "maybe"
				case "kind":
					tree["verbose"] = map[string]any{}
				default:
					tree["binaries"] = []any{map[string]any{}}
				}
				var err error
				assert.Pure(c, func() decodeSection {
					return deepCopy(reflect.ValueOf(got)).Interface().(decodeSection)
				}, func() {
					err = cli.Decode(map[string]any{"go": tree}, "go", &got)
				}, "Decode leaves the value unchanged")
				assert.HasError(c, err, "Decode refuses the document")
			})
		})

		t.Run("leaves the value unchanged for every tree that it refuses", func(t *testing.T) {
			t.Parallel()
			prop.ForAll(t, decodeContract, decodeBody)
		})
	})
}

// drawSection draws the section go of a document over base, and returns its tree and the section
// that Decode returns for it. Each member is present or absent, and a present member states a valid
// value of its field.
func drawSection(c *prop.Case, base decodeSection) (map[string]any, decodeSection) {
	c.Helper()
	tree, want := map[string]any{}, base
	if c.Draw(prop.Boolean(), "jobs present") {
		jobs := c.Draw(prop.Integer(-1000, 1000), "jobs")
		tree["jobs"], want.Jobs = json.Number(strconv.Itoa(jobs)), jobs
	}
	if c.Draw(prop.Boolean(), "verbose present") {
		verbose := c.Draw(prop.Boolean(), "verbose")
		tree["verbose"], want.Verbose = verbose, verbose
	}
	if c.Draw(prop.Boolean(), "timeout present") {
		timeout := c.Draw(prop.Duration(0, time.Hour), "timeout")
		tree["timeout"], want.Timeout = timeout.String(), timeout
	}
	if c.Draw(prop.Boolean(), "retries present") {
		retries := c.Draw(prop.Integer(0, 9), "retries")
		tree["retries"], want.Retries = json.Number(strconv.Itoa(retries)), &retries
	}
	if c.Draw(prop.Boolean(), "ports present") {
		ports := c.Draw(prop.List(prop.Integer(1, 65535), prop.MaxSize(3)), "ports")
		items := make([]any, len(ports))
		for i, port := range ports {
			items[i] = json.Number(strconv.Itoa(port))
		}
		tree["ports"], want.Ports = items, ports
	}
	if c.Draw(prop.Boolean(), "batches present") {
		batches := c.Draw(prop.List(prop.List(prop.Integer(-9, 9), prop.MaxSize(3)), prop.MaxSize(3)), "batches")
		items := make([]any, len(batches))
		for i, batch := range batches {
			values := make([]any, len(batch))
			for j, n := range batch {
				values[j] = json.Number(strconv.Itoa(n))
			}
			items[i] = values
		}
		tree["batches"], want.Batches = items, batches
	}
	if c.Draw(prop.Boolean(), "labels present") {
		labels := c.Draw(prop.Dict(prop.StringMatching(`[a-z]{1,4}`), prop.StringMatching(`[a-z0-9]{0,4}`),
			prop.MaxSize(3)), "labels")
		members := make(map[string]any, len(labels))
		for name, label := range labels {
			members[name] = label
		}
		tree["labels"], want.Labels = members, labels
	}
	if c.Draw(prop.Boolean(), "fuzz present") {
		fuzz := map[string]any{}
		if c.Draw(prop.Boolean(), "fuzz time present") {
			fuzzTime := c.Draw(prop.StringMatching(`[1-9][0-9]{0,2}s`), "fuzz time")
			fuzz["time"], want.Fuzz.Time = fuzzTime, fuzzTime
		}
		if c.Draw(prop.Boolean(), "fuzz args present") {
			args := c.Draw(prop.List(prop.StringMatching(`-[a-z]{1,6}`), prop.MaxSize(3)), "fuzz args")
			items := make([]any, len(args))
			for i, arg := range args {
				items[i] = arg
			}
			fuzz["args"], want.Fuzz.Args = items, args
		}
		tree["fuzz"] = fuzz
	}
	if c.Draw(prop.Boolean(), "sign present") {
		sign, target := map[string]any{}, decodeSign{Mode: "cosign"}
		if base.Sign != nil {
			target = *base.Sign
		}
		if c.Draw(prop.Boolean(), "sign key present") {
			key := c.Draw(prop.StringMatching(`[a-z]{1,6}`), "sign key")
			sign["key"], target.Key = key, key
		}
		if c.Draw(prop.Boolean(), "sign mode present") {
			mode := c.Draw(prop.SampledFrom("cosign", "gpg"), "sign mode")
			sign["mode"], target.Mode = mode, mode
		}
		tree["sign"], want.Sign = sign, &target
	}
	if c.Draw(prop.Boolean(), "binaries present") {
		n := c.Draw(prop.Integer(0, 3), "binaries")
		items, binaries := make([]any, n), make([]decodeBinary, n)
		for i := range n {
			name := c.Draw(prop.StringMatching(`[a-z]{1,6}`), "binary name")
			item := map[string]any{"name": name}
			binary := decodeBinary{Build: decodeBuild{Trimpath: true}, Name: name}
			if c.Draw(prop.Boolean(), "binary main present") {
				main := c.Draw(prop.StringMatching(`\./[a-z]{1,6}`), "binary main")
				item["main"], binary.Main = main, main
			}
			if c.Draw(prop.Boolean(), "binary build present") {
				tags := c.Draw(prop.List(prop.StringMatching(`[a-z]{1,6}`), prop.MaxSize(3)), "binary tags")
				members := make([]any, len(tags))
				for j, tag := range tags {
					members[j] = tag
				}
				item["build"], binary.Build.Tags = map[string]any{"tags": members}, tags
			}
			items[i], binaries[i] = item, binary
		}
		tree["binaries"], want.Binaries = items, binaries
	}
	if c.Draw(prop.Boolean(), "steps present") {
		n := c.Draw(prop.Integer(0, 3), "steps")
		items, steps := make([]any, n), make([]*decodeStep, n)
		for i := range n {
			item, step := map[string]any{}, &decodeStep{Name: "build"}
			if c.Draw(prop.Boolean(), "step name present") {
				name := c.Draw(prop.StringMatching(`[a-z]{1,6}`), "step name")
				item["name"], step.Name = name, name
			}
			items[i], steps[i] = item, step
		}
		tree["steps"], want.Steps = items, steps
	}
	if c.Draw(prop.Boolean(), "plans present") {
		names := c.Draw(prop.List(prop.StringMatching(`[a-z]{1,4}`), prop.MaxSize(3), prop.Unique()), "plan names")
		members, plans := make(map[string]any, len(names)), make(map[string]decodePlan, len(names))
		for _, name := range names {
			member, plan := map[string]any{}, decodePlan{Enabled: true}
			if c.Draw(prop.Boolean(), "plan weight present") {
				weight := c.Draw(prop.Integer(0, 9), "plan weight")
				member["weight"], plan.Weight = json.Number(strconv.Itoa(weight)), weight
			}
			members[name], plans[name] = member, plan
		}
		tree["plans"], want.Plans = members, plans
	}
	if c.Draw(prop.Boolean(), "hooks present") {
		names := c.Draw(prop.List(prop.StringMatching(`[a-z]{1,4}`), prop.MaxSize(3), prop.Unique()), "hook names")
		members, hooks := make(map[string]any, len(names)), make(map[string]*decodeStep, len(names))
		for _, name := range names {
			run := c.Draw(prop.StringMatching(`[a-z]{1,6}`), "hook run")
			members[name] = map[string]any{"run": []any{run}}
			hooks[name] = &decodeStep{Run: []string{run}, Name: "build"}
		}
		tree["hooks"], want.Hooks = members, hooks
	}
	return tree, want
}

// deepCopy returns a copy of v that shares no map, slice or pointer with v. A nil map, slice or
// pointer of v is nil in the copy, and an empty map or slice is empty. The structs of v have no
// unexported fields.
func deepCopy(v reflect.Value) reflect.Value {
	c := reflect.New(v.Type()).Elem()
	switch {
	case v.Kind() == reflect.Pointer && !v.IsNil():
		c.Set(reflect.New(v.Type().Elem()))
		c.Elem().Set(deepCopy(v.Elem()))
	case v.Kind() == reflect.Slice && !v.IsNil():
		c.Set(reflect.MakeSlice(v.Type(), v.Len(), v.Len()))
		for i := range v.Len() {
			c.Index(i).Set(deepCopy(v.Index(i)))
		}
	case v.Kind() == reflect.Map && !v.IsNil():
		c.Set(reflect.MakeMapWithSize(v.Type(), v.Len()))
		for k, e := range v.Seq2() {
			c.SetMapIndex(k, deepCopy(e))
		}
	case v.Kind() == reflect.Struct:
		for i := range v.NumField() {
			c.Field(i).Set(deepCopy(v.Field(i)))
		}
	default:
		c.Set(v)
	}
	return c
}

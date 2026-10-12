// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli_test

import (
	"errors"
	"io/fs"
	"math"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"

	"go.stealthscale.io/cli"
)

// errCelsius is the error of a temperature without the unit C.
var errCelsius = errors.New("main: want a number with the unit C")

// celsius is a temperature that Convert registers as a number with the unit C, such as 21C.
type celsius float64

// point is a struct type that Convert registers as two numbers, such as 3,4.
type point struct {
	X, Y int
}

// level is a type whose pointer implements flag.Value: one of low and high.
type level string

// Set sets l to s, one of low and high.
func (l *level) Set(s string) error {
	if s != "low" && s != "high" {
		return errors.New("main: want low or high")
	}
	*l = level(s)
	return nil
}

// String returns l.
func (l *level) String() string {
	return string(*l)
}

// parseCelsius parses a temperature such as 21C.
func parseCelsius(s string) (celsius, error) {
	n, ok := strings.CutSuffix(s, "C")
	if !ok {
		return 0, errCelsius
	}
	f, err := strconv.ParseFloat(n, 64)
	return celsius(f), err
}

// The options of the commands in the tests of conversions.
type (
	celsiusOptions struct {
		Readings map[celsius]celsius
		Limit    *celsius
		Range    []celsius
		Inside   celsius
	}
	pointOptions struct {
		Origin point
	}
	builtinOptions struct {
		Upstream *url.URL
		Zone     *time.Location
		Peer     *net.IPNet
		Mirror   url.URL
		Allow    net.IPNet
		Socket   fs.FileMode
	}
	methodOptions struct {
		Addr  netip.Addr
		Level level
	}
	kindOptions struct {
		Name    string
		Timeout time.Duration
		Ratio   float64
		Small   int8
		Count   uint16
		Verbose bool
	}
	roundTripOptions struct {
		Int      int64
		Uint     uint64
		Float    float64
		Duration time.Duration
		Bool     bool
	}
)

// TestConvert checks the conversions that Convert registers, the built-in ones, and the
// conversions of the methods, of time.Duration and of the kinds.
func TestConvert(t *testing.T) {
	t.Parallel()

	t.Run("Convert", func(t *testing.T) {
		t.Parallel()

		t.Run("converts each shape of the registered type through parse", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, celsiusOptions](t)
			app.Convert(parseCelsius)
			args := []string{command, "--inside", "21C", "--range", "18C,24C", "--limit", "30C", "--readings", "7C=19C"}
			res := execute(t, app, args, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			limit := celsius(30)
			assert.Equal(t, *got.Config(), celsiusOptions{
				Readings: map[celsius]celsius{7: 19},
				Limit:    &limit,
				Range:    []celsius{18, 24},
				Inside:   21,
			}, "the run converts the value, the list, the pointer and the map through parse")
		})

		t.Run("states the error of parse as the reason", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, celsiusOptions](t)
			app.Convert(parseCelsius)
			res := execute(t, app, []string{command, "--inside", "21F"}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrInvalidValue, "Execute returns ErrInvalidValue")
			assert.Equal(t, res.err.Error(),
				`cli: serve: invalid --inside "21F" from flag --inside: main: want a number with the unit C`,
				"the error states the error of parse")
		})

		t.Run("replaces an earlier registration of the type", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, celsiusOptions](t)
			app.Convert(func(string) (celsius, error) { return 1, nil })
			app.Convert(func(string) (celsius, error) { return 2, nil })
			res := execute(t, app, []string{command, "--inside", "21C"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Inside, celsius(2), "the run converts through the later registration")
		})

		t.Run("replaces a built-in conversion", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, builtinOptions](t)
			upstream := &url.URL{Scheme: "https", Host: "proxy"}
			app.Convert(func(string) (*url.URL, error) { return upstream, nil })
			res := execute(t, app, []string{command, "--upstream", "ignored"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Upstream, upstream, "the run converts through the registration",
				assert.ByIdentity())
		})

		t.Run("makes a struct type an option", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, pointOptions](t)
			app.Convert(func(s string) (point, error) {
				x, y, _ := strings.Cut(s, ",")
				px, err := strconv.Atoi(x)
				if err != nil {
					return point{}, err
				}
				py, err := strconv.Atoi(y)
				return point{X: px, Y: py}, err
			})
			res := execute(t, app, []string{command, "--origin=3,4"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Origin, point{X: 3, Y: 4}, "the run converts the struct through parse")
		})

		t.Run("returns ErrDeclaration for a nil parse", func(t *testing.T) {
			t.Parallel()
			app, _ := serving[struct{}, celsiusOptions](t)
			app.Convert[celsius](nil)
			res := execute(t, app, []string{command}, nil, nil)
			assert.ErrorIs(t, res.err, cli.ErrDeclaration, "Execute returns ErrDeclaration")
			assert.Equal(t, res.err.Error(), "cli: invalid declaration: the conversion of cli_test.celsius is nil",
				"the error names the type")
		})
	})

	t.Run("Execute", func(t *testing.T) {
		t.Parallel()

		t.Run("converts the standard types without a text method of their own", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, builtinOptions](t)
			args := []string{
				command, "--upstream", "https://api.example.com/v1", "--zone", "Europe/Amsterdam",
				"--peer", "10.0.0.0/8", "--mirror", "https://mirror.example.com", "--allow", "192.168.0.0/16",
				"--socket", "0640",
			}
			res := execute(t, app, args, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			zone, err := time.LoadLocation("Europe/Amsterdam")
			assert.NoError(t, err, "the zone loads")
			_, peer, _ := net.ParseCIDR("10.0.0.0/8")
			_, allow, _ := net.ParseCIDR("192.168.0.0/16")
			assert.Equal(t, *got.Config(), builtinOptions{
				Upstream: &url.URL{Scheme: "https", Host: "api.example.com", Path: "/v1"},
				Zone:     zone,
				Peer:     peer,
				Mirror:   url.URL{Scheme: "https", Host: "mirror.example.com"},
				Allow:    *allow,
				Socket:   0o640,
			}, "the run converts each standard type")
		})

		t.Run("converts a file mode with the prefix 0o", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, builtinOptions](t)
			res := execute(t, app, []string{command, "--socket", "0o640"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, got.Config().Socket, fs.FileMode(0o640), "the run reads the digits after 0o as octal")
		})

		t.Run("converts the types with a text method", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, methodOptions](t)
			res := execute(t, app, []string{command, "--addr", "10.0.0.1", "--level", "high"}, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, *got.Config(), methodOptions{Addr: netip.MustParseAddr("10.0.0.1"), Level: "high"},
				"the run converts through UnmarshalText and Set")
		})

		t.Run("converts time.Duration and the kinds", func(t *testing.T) {
			t.Parallel()
			app, got := serving[struct{}, kindOptions](t)
			args := []string{
				command, "--name", "a b", "--timeout", "1m30s", "--ratio", "0.25", "--small", "-8",
				"--count", "65535", "--verbose=false",
			}
			res := execute(t, app, args, nil, nil)
			assert.NoError(t, res.err, "Execute runs the command")
			assert.Equal(t, *got.Config(), kindOptions{
				Name: "a b", Timeout: 90 * time.Second, Ratio: 0.25, Small: -8, Count: math.MaxUint16,
			}, "the run converts each kind")
		})

		invalid := []struct {
			name     string
			giveArgs []string
			want     string
		}{
			{
				name:     "states the error of url.Parse for an invalid URL",
				giveArgs: []string{"--upstream", "%zz"},
				want:     `invalid --upstream "%zz" from flag --upstream: parse "%zz": invalid URL escape "%zz"`,
			},
			{
				name:     "states the error of url.Parse for an invalid URL value",
				giveArgs: []string{"--mirror", "%zz"},
				want:     `invalid --mirror "%zz" from flag --mirror: parse "%zz": invalid URL escape "%zz"`,
			},
			{
				name:     "states the error of time.LoadLocation for an unknown zone",
				giveArgs: []string{"--zone", "Mars/Base"},
				want:     `invalid --zone "Mars/Base" from flag --zone: unknown time zone Mars/Base`,
			},
			{
				name:     "states the error of net.ParseCIDR for an invalid network",
				giveArgs: []string{"--peer", "10.0.0.0/33"},
				want:     `invalid --peer "10.0.0.0/33" from flag --peer: invalid CIDR address: 10.0.0.0/33`,
			},
			{
				name:     "states the error of net.ParseCIDR for an invalid network value",
				giveArgs: []string{"--allow", "10.0.0.0/33"},
				want:     `invalid --allow "10.0.0.0/33" from flag --allow: invalid CIDR address: 10.0.0.0/33`,
			},
			{
				name:     "states the reason of strconv for a file mode that is not octal",
				giveArgs: []string{"--socket", "0699"},
				want:     `invalid --socket "0699" from flag --socket: invalid syntax`,
			},
		}
		for _, tt := range invalid {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[struct{}, builtinOptions](t)
				res := execute(t, app, append([]string{command}, tt.giveArgs...), nil, nil)
				assert.ErrorIs(t, res.err, cli.ErrInvalidValue, "Execute returns ErrInvalidValue")
				assert.Equal(
					t,
					res.err.Error(),
					"cli: serve: "+tt.want,
					"the error states the reason of the conversion",
				)
			})
		}

		methods := []struct {
			name     string
			giveArgs []string
			want     string
		}{
			{
				name:     "states the error of UnmarshalText for an invalid text",
				giveArgs: []string{"--addr", "10.0.0"},
				want:     `invalid --addr "10.0.0" from flag --addr: ParseAddr("10.0.0"): IPv4 address too short`,
			},
			{
				name:     "states the error of Set for an invalid text",
				giveArgs: []string{"--level", "mid"},
				want:     `invalid --level "mid" from flag --level: main: want low or high`,
			},
		}
		for _, tt := range methods {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[struct{}, methodOptions](t)
				res := execute(t, app, append([]string{command}, tt.giveArgs...), nil, nil)
				assert.ErrorIs(t, res.err, cli.ErrInvalidValue, "Execute returns ErrInvalidValue")
				assert.Equal(t, res.err.Error(), "cli: serve: "+tt.want, "the error states the error of the method")
			})
		}

		kinds := []struct {
			name     string
			giveArgs []string
			want     string
		}{
			{
				name:     "states the error of time.ParseDuration for a duration without a unit",
				giveArgs: []string{"--timeout", "90"},
				want:     `invalid --timeout "90" from flag --timeout: time: missing unit in duration "90"`,
			},
			{
				name:     "states the reason of strconv for a number past the range of its kind",
				giveArgs: []string{"--small", "128"},
				want:     `invalid --small "128" from flag --small: value out of range`,
			},
			{
				name:     "states the reason of strconv for a negative unsigned number",
				giveArgs: []string{"--count", "-1"},
				want:     `invalid --count "-1" from flag --count: invalid syntax`,
			},
			{
				name:     "states the reason of strconv for a float that is no number",
				giveArgs: []string{"--ratio", "half"},
				want:     `invalid --ratio "half" from flag --ratio: invalid syntax`,
			},
			{
				name:     "states the reason of strconv for a bool that is no bool",
				giveArgs: []string{"--verbose=maybe"},
				want:     `invalid --verbose "maybe" from flag --verbose: invalid syntax`,
			},
		}
		for _, tt := range kinds {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				app, _ := serving[struct{}, kindOptions](t)
				res := execute(t, app, append([]string{command}, tt.giveArgs...), nil, nil)
				assert.ErrorIs(t, res.err, cli.ErrInvalidValue, "Execute returns ErrInvalidValue")
				assert.Equal(
					t,
					res.err.Error(),
					"cli: serve: "+tt.want,
					"the error states the reason of the conversion",
				)
			})
		}

		t.Run("converts the text of every int64 back to the number", func(t *testing.T) {
			t.Parallel()
			run := roundTrip(t)
			prop.RoundTrip(t,
				func(n int64) (string, error) { return strconv.FormatInt(n, 10), nil },
				func(s string) (int64, error) { c, err := run("int", s); return c.Int, err },
				"the run reads the decimal text of an int64 as the number")
		})

		t.Run("converts the text of every uint64 back to the number", func(t *testing.T) {
			t.Parallel()
			run := roundTrip(t)
			prop.RoundTrip(t,
				func(n uint64) (string, error) { return strconv.FormatUint(n, 10), nil },
				func(s string) (uint64, error) { c, err := run("uint", s); return c.Uint, err },
				"the run reads the decimal text of a uint64 as the number")
		})

		t.Run("converts the shortest text of every float64 back to the number", func(t *testing.T) {
			t.Parallel()
			run := roundTrip(t)
			prop.RoundTrip(t,
				func(f float64) (string, error) { return strconv.FormatFloat(f, 'g', -1, 64), nil },
				func(s string) (float64, error) { c, err := run("float", s); return c.Float, err },
				"the run reads the shortest text of a float64 as the number",
				prop.Using(prop.Float(-math.MaxFloat64, math.MaxFloat64)))
		})

		t.Run("converts the text of every duration back to the duration", func(t *testing.T) {
			t.Parallel()
			run := roundTrip(t)
			prop.RoundTrip(t,
				func(d time.Duration) (string, error) { return d.String(), nil },
				func(s string) (time.Duration, error) { c, err := run("duration", s); return c.Duration, err },
				"the run reads the text of time.Duration.String as the duration")
		})

		t.Run("converts the text of either bool back to the bool", func(t *testing.T) {
			t.Parallel()
			run := roundTrip(t)
			prop.RoundTrip(t,
				func(b bool) (string, error) { return strconv.FormatBool(b), nil },
				func(s string) (bool, error) { c, err := run("bool", s); return c.Bool, err },
				"the run reads the text of a bool as the bool")
		})
	})
}

// roundTrip returns a function that runs an app of the options roundTripOptions with the text of
// one flag, and returns the options of the run.
func roundTrip(t *testing.T) func(flag, text string) (*roundTripOptions, error) {
	t.Helper()
	app, got := serving[struct{}, roundTripOptions](t)
	return func(flag, text string) (*roundTripOptions, error) {
		res := execute(t, app, []string{command, "--" + flag + "=" + text}, nil, nil)
		return got.Config(), res.err
	}
}

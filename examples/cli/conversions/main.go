// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

// Command conversions is a program whose options have types of its own.
//
//	go run ./cli/conversions check --max 64KB --target 19.5C --origin 3,4
//
// The package converts the text of an option through a method of its type when the type has one:
//   - The pointer of ByteSize implements flag.Value. The package calls its method Set.
//   - The pointer of Version implements encoding.TextUnmarshaler. The package calls its method
//     UnmarshalText.
//
// app registers a conversion with App.Convert for each type without such a method:
//   - Celsius is a float64 that converts from text such as 21.5C.
//   - Point is a struct that converts from text such as 3,4.
//   - The conversion of *url.URL replaces the built-in conversion. It refuses a URL whose scheme is
//     not https.
//
// Version and Point are structs with a conversion, so each of them is an option and not a group.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"go.stealthscale.io/cli"
)

// The conversions of the program return these errors.
var (
	errByteSize = errors.New("main: want a whole number with the unit B, KB, MB or GB")
	errVersion  = errors.New("main: want a version of the form v1.2.3")
	errCelsius  = errors.New("main: want a number with the unit C")
	errPoint    = errors.New("main: want two whole numbers of the form x,y")
	errScheme   = errors.New("main: want a URL with the scheme https")
)

// ByteSize is a size in bytes. Its pointer implements flag.Value. Set takes a whole number with the
// unit B, KB, MB or GB, such as 64KB.
type ByteSize int64

// Set sets b to the size s.
//
// Error modes:
//   - errByteSize for s without a unit or with a number that is not whole.
func (b *ByteSize) Set(s string) error {
	for _, unit := range []struct {
		suffix string
		factor int64
	}{{"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"B", 1}} {
		number, ok := strings.CutSuffix(s, unit.suffix)
		if !ok {
			continue
		}
		n, err := strconv.ParseInt(number, 10, 64)
		if err != nil {
			return errByteSize
		}
		*b = ByteSize(n * unit.factor)
		return nil
	}
	return errByteSize
}

// String returns the size in bytes.
func (b *ByteSize) String() string {
	return strconv.FormatInt(int64(*b), 10) + "B"
}

// Version is a version of the form v1.2.3. Its pointer implements encoding.TextUnmarshaler.
type Version struct {
	Major, Minor, Patch int
}

// UnmarshalText sets v to the version of text.
//
// Error modes:
//   - errVersion for text without the v, with fewer or more than three numbers, or with a number
//     that is not whole.
func (v *Version) UnmarshalText(text []byte) error {
	rest, ok := strings.CutPrefix(string(text), "v")
	parts := strings.Split(rest, ".")
	if !ok || len(parts) != 3 {
		return errVersion
	}
	numbers := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return errVersion
		}
		numbers[i] = n
	}
	*v = Version{Major: numbers[0], Minor: numbers[1], Patch: numbers[2]}
	return nil
}

// String returns the version in the form v1.2.3.
func (v Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Celsius is a temperature in degrees Celsius, such as 21.5C.
type Celsius float64

// Point is a position on a grid, such as 3,4.
type Point struct {
	X, Y int
}

// Check contains the options of the command check.
type Check struct {
	Webhook *url.URL   `help:"Webhook of the alerts."`
	Sizes   []ByteSize `help:"Sizes to compare with the largest size."`
	Version Version    `help:"Lowest version of a client."             default:"v1.0.0"`
	Origin  Point      `help:"Origin of the grid."                     default:"0,0"`
	Max     ByteSize   `help:"Largest size of an upload."              default:"1MB"`
	Target  Celsius    `help:"Target temperature."                     default:"21C"`
}

func main() {
	os.Exit(app().Main(context.Background(), cli.OS()))
}

// app returns the program units with the command check. It registers the conversions of Celsius,
// Point and *url.URL.
func app() *cli.App[struct{}] {
	a := cli.New[struct{}]("units", "Check values of the units of the program.")
	a.Convert(parseCelsius)
	a.Convert(parsePoint)
	a.Convert(parseHTTPS)
	a.Command("check", "Print the options and compare each size with the largest size.", check)
	return a
}

// parseCelsius returns the temperature of s, such as 21.5C.
//
// Error modes:
//   - errCelsius for s without the unit C or with a number that strconv.ParseFloat refuses.
func parseCelsius(s string) (Celsius, error) {
	number, ok := strings.CutSuffix(s, "C")
	if !ok {
		return 0, errCelsius
	}
	f, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, errCelsius
	}
	return Celsius(f), nil
}

// parsePoint returns the point of s, such as 3,4.
//
// Error modes:
//   - errPoint for s without a comma or with a number that is not whole.
func parsePoint(s string) (Point, error) {
	x, y, ok := strings.Cut(s, ",")
	px, errX := strconv.Atoi(x)
	py, errY := strconv.Atoi(y)
	if !ok || errX != nil || errY != nil {
		return Point{}, errPoint
	}
	return Point{X: px, Y: py}, nil
}

// parseHTTPS returns the URL of s. The URL must have the scheme https.
//
// Error modes:
//   - the error of url.Parse.
//   - errScheme for a URL with another scheme.
func parseHTTPS(s string) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("main: parse the URL: %w", err)
	}
	if u.Scheme != "https" {
		return nil, errScheme
	}
	return u, nil
}

// check writes the options. It then writes for each size whether the size fits the largest size.
func check(_ context.Context, run *cli.Run[struct{}, Check]) error {
	c := run.Config()
	webhook := "<nil>"
	if c.Webhook != nil {
		webhook = c.Webhook.String()
	}
	out := fmt.Appendf(nil, "max %s, version %s, target %gC, origin %d,%d, webhook %s\n",
		c.Max.String(), c.Version, float64(c.Target), c.Origin.X, c.Origin.Y, webhook)
	for _, size := range c.Sizes {
		verdict := "fits"
		if size > c.Max {
			verdict = "is too large"
		}
		out = fmt.Appendf(out, "size %s %s\n", size.String(), verdict)
	}
	if _, err := run.Stdout().Write(out); err != nil {
		return fmt.Errorf("main: write the check: %w", err)
	}
	return nil
}

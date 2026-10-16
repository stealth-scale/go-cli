// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"encoding"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// The types that a conversion of the package recognises.
var (
	durationType        = reflect.TypeFor[time.Duration]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
	flagValueType       = reflect.TypeFor[flag.Value]()
)

// converter converts the text s to a value of one type. It returns the reason when s does not
// convert, which a message states after the input and its source.
type converter func(s string) (reflect.Value, string)

// conversions are the conversions of the types of fields by their type: the built-in ones, and in an
// app the ones that [App.Convert] registers.
type conversions map[reflect.Type]converter

// builtins returns the conversions of the standard types that have no text method of their own:
// url.URL and *url.URL through url.Parse, *time.Location through time.LoadLocation, net.IPNet and
// *net.IPNet through net.ParseCIDR, and fs.FileMode as an octal number such as 0644 or 0o644.
func builtins() conversions {
	return conversions{
		reflect.TypeFor[*url.URL]():       convertWith(url.Parse),
		reflect.TypeFor[*time.Location](): convertWith(time.LoadLocation),
		reflect.TypeFor[url.URL](): func(s string) (reflect.Value, string) {
			u, err := url.Parse(s)
			if err != nil {
				return reflect.Value{}, err.Error()
			}
			return reflect.ValueOf(*u), ""
		},
		reflect.TypeFor[*net.IPNet](): func(s string) (reflect.Value, string) {
			_, n, err := net.ParseCIDR(s)
			if err != nil {
				return reflect.Value{}, err.Error()
			}
			return reflect.ValueOf(n), ""
		},
		reflect.TypeFor[net.IPNet](): func(s string) (reflect.Value, string) {
			_, n, err := net.ParseCIDR(s)
			if err != nil {
				return reflect.Value{}, err.Error()
			}
			return reflect.ValueOf(*n), ""
		},
		reflect.TypeFor[fs.FileMode](): func(s string) (reflect.Value, string) {
			n, err := strconv.ParseUint(strings.TrimPrefix(s, "0o"), 8, 32)
			return kindValue(reflect.TypeFor[fs.FileMode](), err, func(v reflect.Value) { v.SetUint(n) })
		},
	}
}

// converter returns the conversion of a text to a value of the type t, or nil for a type without
// one. It tries, in order, the conversion of t in c, encoding.TextUnmarshaler and flag.Value of a
// pointer to t, time.Duration, and the kinds string, bool, int, uint and float.
func (c conversions) converter(t reflect.Type) converter {
	if conv, ok := c[t]; ok {
		return conv
	}
	switch p := reflect.PointerTo(t); {
	case p.Implements(textUnmarshalerType):
		return func(s string) (reflect.Value, string) {
			v := reflect.New(t)
			u, _ := reflect.TypeAssert[encoding.TextUnmarshaler](v)
			if err := u.UnmarshalText([]byte(s)); err != nil {
				return reflect.Value{}, err.Error()
			}
			return v.Elem(), ""
		}
	case p.Implements(flagValueType):
		return func(s string) (reflect.Value, string) {
			v := reflect.New(t)
			f, _ := reflect.TypeAssert[flag.Value](v)
			if err := f.Set(s); err != nil {
				return reflect.Value{}, err.Error()
			}
			return v.Elem(), ""
		}
	case t == durationType:
		return convertWith(time.ParseDuration)
	default:
		return kindConverter(t)
	}
}

// convertWith returns the conversion of a text to a value of the type T through parse, with the
// text of the error of parse as the reason.
func convertWith[T any](parse func(s string) (T, error)) converter {
	return func(s string) (reflect.Value, string) {
		v, err := parse(s)
		if err != nil {
			return reflect.Value{}, err.Error()
		}
		return reflect.ValueOf(&v).Elem(), ""
	}
}

// kindConverter returns the conversion of a text to a value of the type t by the kind of t, or nil
// for a kind other than string, bool, int, uint and float. An integer takes decimal digits, and
// the reason of a failure is the error of strconv without its function and its input.
func kindConverter(t reflect.Type) converter {
	switch t.Kind() {
	case reflect.String:
		return func(s string) (reflect.Value, string) {
			return kindValue(t, nil, func(v reflect.Value) { v.SetString(s) })
		}
	case reflect.Bool:
		return func(s string) (reflect.Value, string) {
			b, err := strconv.ParseBool(s)
			return kindValue(t, err, func(v reflect.Value) { v.SetBool(b) })
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(s string) (reflect.Value, string) {
			n, err := strconv.ParseInt(s, 10, t.Bits())
			return kindValue(t, err, func(v reflect.Value) { v.SetInt(n) })
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(s string) (reflect.Value, string) {
			n, err := strconv.ParseUint(s, 10, t.Bits())
			return kindValue(t, err, func(v reflect.Value) { v.SetUint(n) })
		}
	case reflect.Float32, reflect.Float64:
		return func(s string) (reflect.Value, string) {
			n, err := strconv.ParseFloat(s, t.Bits())
			return kindValue(t, err, func(v reflect.Value) { v.SetFloat(n) })
		}
	default:
		return nil
	}
}

// kindValue returns a new value of the type t that set fills, or the reason of err. The reason of
// an error of strconv leaves out its function and its input.
func kindValue(t reflect.Type, err error, set func(v reflect.Value)) (reflect.Value, string) {
	if err != nil {
		if ne, ok := errors.AsType[*strconv.NumError](err); ok {
			err = ne.Err
		}
		return reflect.Value{}, err.Error()
	}
	v := reflect.New(t).Elem()
	set(v)
	return v, ""
}

// Convert registers parse as the conversion of a text to a value of the type T. A field of the
// type T, a slice of T, a pointer to T and a map with keys or values of T then convert through
// parse. A registration replaces every other conversion of T, a built-in one included, and a later
// registration of T replaces an earlier one. A struct type with a conversion is an option and not
// a group.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when parse is nil.
func (a *App[G]) Convert[T any](parse func(s string) (T, error)) {
	if parse == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: the conversion of %s is nil", ErrDeclaration, reflect.TypeFor[T]()))
		return
	}
	a.converters[reflect.TypeFor[T]()] = convertWith(parse)
}

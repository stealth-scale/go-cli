// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"encoding"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// The extensions of the config files of the formats that [New] registers.
const (
	jsonExt = ".json"
	iniExt  = ".ini"
)

// ConfigFile is the type of an option whose value is the path of a config file. The default, the
// environment variable and the flag of the option select the file, and a run reads it after the
// defaults and before the environment variables. The option has no config key, and the options of
// one run declare one ConfigFile at most. The extension of the path selects the [Format].
//
// A run skips a file at the default path that does not exist, and fails for a file at a path from
// the environment or the command line that does not exist.
type ConfigFile string

// configFileType is the type of an option that is a config file.
var configFileType = reflect.TypeFor[ConfigFile]()

// Format decodes the data of a config file into a tree. An object of the file is a map[string]any,
// an array is a []any, and any other value converts through its text: the result of MarshalText for
// an encoding.TextMarshaler, and of fmt.Sprint for every other value. A nil value is no value.
type Format func(data []byte) (map[string]any, error)

// Format registers decode as the format of the config files with the extensions exts, such as
// ".yaml" and ".yml". An extension compares without regard to case, and a later registration of an
// extension replaces an earlier one. [New] registers [JSON] for ".json" and [INI] for ".ini".
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when decode is nil, when exts is empty,
// and for an extension that is not a dot followed by a name.
func (a *App[G]) Format(decode Format, exts ...string) {
	if decode == nil || len(exts) == 0 {
		a.errs = append(a.errs, fmt.Errorf("%w: a format needs a decoder and an extension", ErrDeclaration))
		return
	}
	for _, ext := range exts {
		if len(ext) < 2 || filepath.Ext(ext) != ext {
			a.errs = append(
				a.errs,
				fmt.Errorf("%w: the extension %q is not a dot followed by a name", ErrDeclaration, ext),
			)
			return
		}
	}
	for _, ext := range exts {
		a.formats[strings.ToLower(ext)] = decode
	}
}

// applyConfig reads the config file of the run, and sets the option of each of its keys. It skips
// a key of an option of another command.
//
// Error modes:
//   - an *Error that wraps ErrConfigFile for a file without a registered format, a file that the
//     Env cannot read, and a file that its format cannot decode.
//   - an *Error that wraps ErrUnknownKey for a key that no option of the app declares.
//   - the errors of assign.
func (a *App[G]) applyConfig(env *Env, d *declaration[G], in *input[G]) error {
	i := slices.IndexFunc(in.list, func(o *option) bool { return o.config })
	if i < 0 {
		return nil
	}
	path, explicit := configPath(env, in, in.list[i])
	if path == "" {
		return nil
	}
	ext := filepath.Ext(path)
	decode, ok := a.formats[strings.ToLower(ext)]
	if !ok {
		return &Error{
			Err:   ErrConfigFile,
			Input: path,
			text:  "config file " + path + ": no format for " + strconv.Quote(ext),
		}
	}
	var data []byte
	err := error(&fs.PathError{Op: "read", Path: path, Err: fs.ErrNotExist})
	if env.ReadFile != nil {
		data, err = env.ReadFile(path)
	}
	if errors.Is(err, fs.ErrNotExist) && !explicit {
		return nil
	}
	if err != nil {
		return &Error{Err: ErrConfigFile, Input: path, text: "config file " + path + ": " + err.Error()}
	}
	tree, err := decode(data)
	if err != nil {
		return &Error{Err: ErrConfigFile, Input: path, text: "config file " + path + ": " + err.Error()}
	}
	values := map[string][]string{}
	flatten("", tree, d.dicts, values)
	known := slices.Sorted(maps.Keys(d.dicts))
	for _, key := range slices.Sorted(maps.Keys(values)) {
		src := Source{Kind: File, Name: key, File: path}
		j := slices.IndexFunc(in.list, func(o *option) bool { return o.key == key })
		switch {
		case j >= 0:
			o, vals := in.list[j], values[key]
			if len(vals) == 1 {
				vals = o.split(vals[0])
			}
			if err := o.assign(vals, src, false); err != nil {
				return err
			}
		case !slices.Contains(known, key):
			return &Error{
				Err:        ErrUnknownKey,
				Input:      key,
				Source:     src,
				Suggestion: suggest(key, known, ""),
				text:       "unknown key " + key + " in " + path,
			}
		}
	}
	return nil
}

// configPath returns the path of the config file cfg: its last flag, else its environment variable,
// else its default. It reports whether the flag or the environment variable states the path.
func configPath[G any](env *Env, in *input[G], cfg *option) (string, bool) {
	path, explicit := cfg.value.String(), false
	if cfg.env != "" {
		if s, ok := env.lookup(cfg.env); ok {
			path, explicit = s, true
		}
	}
	for _, f := range in.flags {
		if f.opt == cfg {
			path, explicit = f.value, true
		}
	}
	return path, explicit
}

// flatten adds the members of tree to values by their keys under prefix, the names of the nested
// objects joined with dots. dicts marks the key of a map option, whose object adds its members as
// entries of the form key=value. A member with the value nil is left out.
func flatten(prefix string, tree map[string]any, dicts map[string]bool, values map[string][]string) {
	for k, v := range tree {
		if prefix != "" {
			k = prefix + keySeparator + k
		}
		switch v := v.(type) {
		case nil:
		case map[string]any:
			if !dicts[k] {
				flatten(k, v, dicts, values)
				continue
			}
			entries := make([]string, 0, len(v))
			for _, name := range slices.Sorted(maps.Keys(v)) {
				entries = append(entries, name+entrySeparator+text(v[name]))
			}
			values[k] = entries
		case []any:
			texts := make([]string, len(v))
			for i, item := range v {
				texts[i] = text(item)
			}
			values[k] = texts
		default:
			values[k] = []string{text(v)}
		}
	}
}

// text returns the text of a value of a config file: the result of MarshalText for an
// encoding.TextMarshaler, such as the time.Time of a TOML date, and of fmt.Sprint for any other
// value.
func text(v any) string {
	if m, ok := v.(encoding.TextMarshaler); ok {
		if b, err := m.MarshalText(); err == nil {
			return string(b)
		}
	}
	return fmt.Sprint(v)
}

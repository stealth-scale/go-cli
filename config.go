// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"encoding"
	"errors"
	"fmt"
	"io"
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

// IgnoreKeys registers config keys that the app decodes itself. A run skips each such key of the
// config file and every key below it, so the file can contain a document of the app beside the
// keys of its options. Dots join the names of the objects of a key, as in the tag key. A later call
// adds its keys to the keys of the earlier calls.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] for a key with an empty name, and for
// a key that overlaps the config key of an option. Keys overlap when they are equal or when one of
// them lies inside the other.
func (a *App[G]) IgnoreKeys(keys ...string) {
	for _, key := range keys {
		if slices.Contains(strings.Split(key, keySeparator), "") {
			a.errs = append(a.errs, fmt.Errorf("%w: the ignored config key %q has an empty name", ErrDeclaration, key))
			return
		}
	}
	a.ignored = append(a.ignored, keys...)
}

// LocateConfig registers locate, which returns the path of the config file of a run when neither
// the flag nor the environment variable of its ConfigFile option sets the path. global contains the
// global options after their defaults, environment variables and flags, and env is the Env of the
// run. A search that tests its candidates through env.Open reads only the files of the Env, so a
// test controls the files that it finds. The ConfigFile option takes the path with the source
// [Located]. With the empty path the run does not read a config file, and a located path that does
// not exist fails as an explicit path does. A later call replaces an earlier one.
//
// locate runs on the goroutine of [App.Execute], once for each run that needs a path. Runs of the
// app can call it concurrently, so locate must be safe for concurrent use. It must not change the
// options. An app with [App.Render] does not apply its sources for the help, so its help does not
// call locate.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] when locate is nil, and the error of
// locate unchanged.
func (a *App[G]) LocateConfig(locate func(global *G, env *Env) (string, error)) {
	if locate == nil {
		a.errs = append(a.errs, fmt.Errorf("%w: the function that locates the config file is nil", ErrDeclaration))
		return
	}
	a.locate = locate
}

// checkIgnored returns an error that wraps ErrDeclaration for an ignored key that overlaps the
// config key of an option of d. Keys overlap when they are equal or when one of them lies inside
// the other. It compares the keys of the options in their order.
func (a *App[G]) checkIgnored(d *declaration[G]) error {
	keys := slices.Sorted(maps.Keys(d.dicts))
	for _, ignored := range a.ignored {
		for _, key := range keys {
			overlap := key == ignored || strings.HasPrefix(key, ignored+keySeparator) ||
				strings.HasPrefix(ignored, key+keySeparator)
			if overlap {
				return fmt.Errorf("%w: the ignored config key %s overlaps the config key %s of an option",
					ErrDeclaration, ignored, key)
			}
		}
	}
	return nil
}

// applyConfig reads the config file of the run, and sets the option of each of its keys in the order
// of the keys. It skips a key of an option of another command and an ignored key. It applies every
// valid key and returns the first error, and it skips a file that it cannot read or decode. When
// neither the flag nor the environment variable of the ConfigFile option sets the path, the
// function of LocateConfig returns the path.
//
// Error modes:
//   - the error of the function of LocateConfig, unchanged.
//   - an *Error that wraps ErrConfigFile for a file without a registered format, a file that the
//     Env cannot read, and a file that its format cannot decode.
//   - an *Error that wraps ErrUnknownKey for a key that no option of the app declares.
//   - the errors of assign.
func (a *App[G]) applyConfig(env *Env, d *declaration[G], in *input[G]) error {
	i := slices.IndexFunc(in.list, func(o *option) bool { return o.config })
	if i < 0 {
		return nil
	}
	cfg := in.list[i]
	path, explicit := configPath(env, in, cfg)
	if !explicit && a.locate != nil {
		// locate reads the global options of the environment variables and the flags. resolve
		// applies both again after the file, so it reports the errors of that later pass.
		_ = in.applyEnv(env)
		_ = in.applyFlags()
		located, err := a.locate(d.global, env)
		if err != nil {
			return err
		}
		cfg.value.SetString(located)
		cfg.set, cfg.source = true, Source{Kind: Located}
		path, explicit = located, true
	}
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
	file, err := env.open(path)
	if errors.Is(err, fs.ErrNotExist) && !explicit {
		return nil
	}
	if err != nil {
		return &Error{Err: ErrConfigFile, Input: path, text: "config file " + path + ": " + err.Error()}
	}
	data, err := io.ReadAll(file)
	err = errors.Join(err, file.Close())
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
	var first error
	for _, key := range slices.Sorted(maps.Keys(values)) {
		ignored := slices.ContainsFunc(a.ignored, func(ignored string) bool {
			return key == ignored || strings.HasPrefix(key, ignored+keySeparator)
		})
		if ignored {
			continue
		}
		src := Source{Kind: File, Name: key, File: path}
		j := slices.IndexFunc(in.list, func(o *option) bool { return o.key == key })
		switch {
		case j >= 0:
			o, vals := in.list[j], values[key]
			if len(vals) == 1 {
				vals = o.split(vals[0])
			}
			if err := o.assign(vals, src, false); first == nil {
				first = err
			}
		case first == nil && !slices.Contains(known, key):
			first = &Error{
				Err:        ErrUnknownKey,
				Input:      key,
				Source:     src,
				Suggestion: suggest(key, known, ""),
				text:       "unknown key " + key + " in " + path,
			}
		}
	}
	return first
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

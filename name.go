// Copyright Stealth Scale B.V. 2026
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// The separators of the names of an option.
const (
	// wordSeparator joins the words of a name, such as dry-run, and the groups of a flag, such as
	// tls-cert.
	wordSeparator = "-"
	// keySeparator joins the command and the groups of a config key, such as serve.tls.cert.
	keySeparator = "."
	// envSeparator joins the words of an environment variable, such as SRV_SERVE_TLS_CERT.
	envSeparator = "_"
)

// namePattern matches the name of an app, a command, a group or an option: words of lower-case ASCII
// letters and digits that single dashes join, the first word starting with a letter.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// envPattern matches the prefix of [App.EnvPrefix]: words of upper-case ASCII letters and digits that
// single underscores join, the first word starting with a letter.
var envPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$`)

// scope is the position of an options struct in the names of its options and in the tree of the
// commands of the app.
type scope struct {
	envPrefix string          // the start of each environment variable, such as SRV, or empty for none
	path      string          // the Go path of the struct in messages, such as main.Serve.Cache
	command   []string        // the names of the path of the command, or nil for the options of the app
	groups    []string        // the names of the groups that contain the struct, the outermost first
	ancestors []reflect.Value // the pointers to the options of the ancestors of the command
}

// nest returns the scope of the struct field f in s. A named field is a group, so the scope has its
// name in kebab case as its last group. An embedded field does not add a group. The tag prefix
// replaces the name of the group, and an empty prefix does not add one.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a field with a tag of an option, and for a group name
//     that is not a name of lower-case words.
func (s *scope) nest(f *reflect.StructField) (scope, error) {
	field := s.path + "." + f.Name
	for _, t := range optionTags {
		if _, ok := f.Tag.Lookup(string(t)); ok {
			return scope{}, fmt.Errorf("%w: the group %s takes no tag %s", ErrDeclaration, field, t)
		}
	}
	name := kebab(f.Name)
	if f.Anonymous {
		name = ""
	}
	if prefix, ok := f.Tag.Lookup(string(tagPrefix)); ok {
		name = prefix
	}
	nested := *s
	nested.path = field
	if name == "" {
		return nested, nil
	}
	if !namePattern.MatchString(name) {
		return scope{}, fmt.Errorf("%w: the group %s has the name %q, which is not a name of lower-case words",
			ErrDeclaration, field, name)
	}
	nested.groups = append(slices.Clone(s.groups), name)
	return nested, nil
}

// names returns the flag, the environment variable and the config key of the option name in s. The
// flag joins the groups and the name with dashes. The environment variable starts with the prefix of
// s and joins the path of the command, the groups and the name in upper case with underscores. The
// config key joins the path of the command, the groups and the name with dots.
func (s *scope) names(name string) (flag, env, key string) {
	words := append(slices.Clone(s.groups), name)
	flag = strings.Join(words, wordSeparator)
	words = slices.Concat(s.command, words)
	key = strings.Join(words, keySeparator)
	env = strings.ToUpper(strings.ReplaceAll(strings.Join(words, envSeparator), wordSeparator, envSeparator))
	if s.envPrefix != "" {
		env = s.envPrefix + envSeparator + env
	}
	return flag, env, key
}

// fieldName returns the name of the field f: its tag name, or its Go name in kebab case. field is
// the Go path of the field in a message, such as main.Serve.Addr.
//
// Error modes:
//   - an error that wraps ErrDeclaration for a name that is not a name of lower-case words.
func fieldName(f *reflect.StructField, field string) (string, error) {
	name := kebab(f.Name)
	if tagged, ok := f.Tag.Lookup(string(tagName)); ok {
		name = tagged
	}
	if !namePattern.MatchString(name) {
		return "", fmt.Errorf("%w: the field %s has the name %q, which is not a name of lower-case words",
			ErrDeclaration, field, name)
	}
	return name, nil
}

// kebab returns field, the name of a Go field, in kebab case. A word starts at an upper-case letter
// that follows a lower-case letter or a digit, and at the last upper-case letter of an acronym that
// a lower-case letter follows. DryRun is dry-run, HTTPPort is http-port and HTTP2Port is
// http2-port.
func kebab(field string) string {
	var b strings.Builder
	runes := []rune(field)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			afterWord := unicode.IsLower(prev) || unicode.IsDigit(prev)
			endsAcronym := unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if afterWord || endsAcronym {
				b.WriteString(wordSeparator)
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// EnvPrefix registers prefix as the start of every environment variable of the app, in place of the
// name of the app in upper case. With the prefix STORE, the option Addr of the command serve has the
// variable STORE_SERVE_ADDR, and the global flag --flag-debug has STORE_FLAG_DEBUG. The empty prefix
// leaves the start out, so the same option has SERVE_ADDR and a global option Config has CONFIG. The
// tag env of an option replaces its whole variable, so the prefix does not apply to it. A later call
// replaces an earlier one.
//
// [App.Execute] returns an error that wraps [ErrDeclaration] for a prefix that is not a name of
// upper-case words: words of upper-case ASCII letters and digits that single underscores join, the
// first word starting with a letter.
func (a *App[G]) EnvPrefix(prefix string) {
	if prefix != "" && !envPattern.MatchString(prefix) {
		a.errs = append(a.errs, fmt.Errorf("%w: the environment prefix %q is not a name of upper-case words",
			ErrDeclaration, prefix))
	}
	a.envPrefix = prefix
}

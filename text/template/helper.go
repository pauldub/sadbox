// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Helper functions to make constructing templates easier.

package template

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"code.google.com/p/sadbox/text/template/parse"
)

// Functions and methods to parse templates.

// Must is a helper that wraps a call to a function returning (*Set, error)
// and panics if the error is non-nil. It is intended for use in variable
// initializations such as
//	var set = template.Must(template.Parse("text"))
func Must(s *Set, err error) *Set {
	if err != nil {
		panic(err)
	}
	return s
}

func Parse(text string) (*Set, error) {
	s, err := new(Set).Parse(text)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Set) Parse(text string) (*Set, error) {
	return s.ParseNamed(text, "source")
}

// ParseNamed is the same as Parse, but defines a name for debugging purposes.
// Useful when parsing multiple files or sources, to know which one caused an
// error.
func (s *Set) ParseNamed(text, name string) (*Set, error) {
	s.init()
	tmpl := make(map[string]*parse.DefineNode)
	err := parse.Parse(tmpl, name, text, s.leftDelim, s.rightDelim, builtins, s.parseFuncs)
	if err != nil {
		return nil, err
	}
	for k, v := range tmpl {
		if s.tmpl[k] != nil {
			return nil, fmt.Errorf("template: %s: multiple definition of template %q", name, k)
		}
		s.tmpl[k] = v
	}
	return s, nil
}

// ParseFiles creates a new Set and parses the template definitions from
// the named files. There must be at least one file. If an error occurs,
// parsing stops and the returned set is nil.
func ParseFiles(filenames ...string) (*Set, error) {
	return parseFiles(new(Set), filenames...)
}

// ParseFiles parses the named files and adds its template to the set. There
// must be at least one file. If an error occurs, parsing stops and the
// returned set is nil; otherwise it is s.
func (s *Set) ParseFiles(filenames ...string) (*Set, error) {
	return parseFiles(s, filenames...)
}

// parseFiles is the helper for the method and function.
func parseFiles(s *Set, filenames ...string) (*Set, error) {
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return nil, fmt.Errorf("template: no files named in call to ParseFiles")
	}
	for _, filename := range filenames {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		if _, err = s.ParseNamed(string(b), filename); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// ParseGlob creates a new Set and parses the template definitions from the
// files identified by the pattern, which must match at least one file. The
// returned template will have the (base) name and (parsed) contents of the
// first file matched by the pattern. ParseGlob is equivalent to calling
// ParseFiles with the list of files matched by the pattern.
func ParseGlob(pattern string) (*Set, error) {
	return parseGlob(new(Set), pattern)
}

// ParseGlob parses the template definitions in the files identified by the
// pattern and associates the resulting templates with t. The pattern is
// processed by filepath.Glob and must match at least one file. ParseGlob is
// equivalent to calling t.ParseFiles with the list of files matched by the
// pattern.
func (s *Set) ParseGlob(pattern string) (*Set, error) {
	return parseGlob(s, pattern)
}

// parseGlob is the implementation of the function and method ParseGlob.
func parseGlob(s *Set, pattern string) (*Set, error) {
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("template: pattern matches no files: %#q", pattern)
	}
	return parseFiles(s, filenames...)
}

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"

	"code.google.com/p/sadbox/template/parse"
)

// Set stores a collection of parsed templates.
type Set struct {
	tmpl       map[string]*parse.DefineNode
	leftDelim  string
	rightDelim string
	// We use two maps, one for parsing and one for execution.
	// This separation makes the API cleaner since it doesn't
	// expose reflection to the client.
	parseFuncs FuncMap
	execFuncs  map[string]reflect.Value
}

func (s *Set) init() {
	if s.tmpl == nil {
		s.tmpl = make(map[string]*parse.DefineNode)
	}
	if s.execFuncs == nil {
		s.execFuncs = make(map[string]reflect.Value)
	}
	if s.parseFuncs == nil {
		s.parseFuncs = make(FuncMap)
	}
}

//func (s *Set) Reset() {
//	s.tmpl = nil
//	s.leftDelim = ""
//	s.rightDelim = ""
//	s.parseFuncs = nil
//	s.execFuncs = nil
//}

// Delims sets the action delimiters to the specified strings, to be used in
// subsequent calls to Parse. An empty delimiter stands for the corresponding
// default: "{{" or "}}".
// The return value is the set, so calls can be chained.
func (s *Set) Delims(left, right string) *Set {
	s.leftDelim = left
	s.rightDelim = right
	return s
}

// Funcs adds the elements of the argument map to the template's function map.
// It panics if a value in the map is not a function with appropriate return
// type. However, it is legal to overwrite elements of the map. The return
// value is the set, so calls can be chained.
func (s *Set) Funcs(funcMap FuncMap) *Set {
	s.init()
	addValueFuncs(s.execFuncs, funcMap)
	addFuncs(s.parseFuncs, funcMap)
	return s
}

// Clone returns a duplicate of the template, including all associated
// templates. The actual representation is not copied, but the name space of
// associated templates is, so further calls to Parse in the copy will add
// templates to the copy but not to the original. Clone can be used to prepare
// common templates and use them with variant definitions for other templates
// by adding the variants after the clone is made.
func (s *Set) Clone() *Set {
	ns := new(Set).Delims(s.leftDelim, s.rightDelim)
	ns.init()
	for k, v := range s.tmpl {
		ns.tmpl[k] = v
	}
	for k, v := range s.parseFuncs {
		ns.parseFuncs[k] = v
	}
	for k, v := range s.execFuncs {
		ns.execFuncs[k] = v
	}
	return ns
}

// Functions and methods to parse templates -----------------------------------

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

// parseNamed parses the given text and adds the resulting templates to the
// set. The name is only used for debugging purposes: useful to parse multiple
// files or glob, for example, to know which file caused an error.
func (s *Set) parseNamed(text, name string) (*Set, error) {
	s.init()
	// Maybe instead of passing s.tmpl we should create a new map, and only
	// add the parsed templates if no error occurred?
	err := parse.Parse(s.tmpl, name, text, s.leftDelim, s.rightDelim, builtins, s.parseFuncs)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// Parse creates a new Set with the template definitions from the given text.
// If an error occurs, parsing stops and the returned set is nil.
func Parse(text string) (*Set, error) {
	return new(Set).Parse(text)
}

// Parse parses the given text and adds the resulting templates to the set.
// If an error occurs, parsing stops and the returned set is nil; otherwise
// it is s.
func (s *Set) Parse(text string) (*Set, error) {
	return s.parseNamed(text, "source")
}

// ParseFiles creates a new Set with the template definitions from the named
// files. There must be at least one file. If an error occurs, parsing stops
// and the returned set is nil.
func ParseFiles(filenames ...string) (*Set, error) {
	return new(Set).ParseFiles(filenames...)
}

// ParseFiles parses the named files and adds the resulting templates to the
// set. There must be at least one file. If an error occurs, parsing stops and
// the returned set is nil; otherwise it is s.
func (s *Set) ParseFiles(filenames ...string) (*Set, error) {
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return nil, fmt.Errorf("template: no files named in call to ParseFiles")
	}
	for _, filename := range filenames {
		b, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		if _, err = s.parseNamed(string(b), filename); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// ParseGlob creates a new Set with the template definitions from the
// files identified by the pattern. The pattern is processed by filepath.Glob
// and must match at least one file. ParseGlob is equivalent to calling
// ParseFiles with the list of files matched by the pattern. If an error
// occurs, parsing stops and the returned set is nil.
func ParseGlob(pattern string) (*Set, error) {
	return new(Set).ParseGlob(pattern)
}

// ParseGlob parses the template definitions in the files identified by the
// pattern and adds the resulting templates to the set. The pattern is
// processed by filepath.Glob and must match at least one file. ParseGlob is
// equivalent to calling s.ParseFiles with the list of files matched by the
// pattern. If an error occurs, parsing stops and the returned set is nil;
// otherwise it is s.
func (s *Set) ParseGlob(pattern string) (*Set, error) {
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("template: pattern matches no files: %#q", pattern)
	}
	return parseFiles(s, filenames...)
}

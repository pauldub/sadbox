// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"

	"code.google.com/p/sadbox/template/escape"
	"code.google.com/p/sadbox/template/parse"
)

// Set stores a collection of parsed templates.
//
// To create a new set call Parse (or other Parse* functions):
//
//     set, err := template.Parse(`{{define "hello"}}Hello, World.{{end}}`)
//     if err != nil {
//         // do something with the parsing error...
//     }
//
// To add more templates to the set call Set.Parse (or any Set.Parse* methods):
//
//     set, err = set.Parse(`{{define "bye"}}Good bye, World.{{end}}`)
//     if err != nil {
//         // do something with the parsing error...
//     }
//
// To execute a template call Set.Execute passing an io.Writer, the name of
// the template to execute and related data:
//
//     err = set.Execute(os.Stderr, "hello", nil)
//     if err != nil {
//         // do something with the execution error...
//     }
type Set struct {
	Tree       parse.Tree
	leftDelim  string
	rightDelim string
	// We use two maps, one for parsing and one for execution.
	// This separation makes the API cleaner since it doesn't
	// expose reflection to the client.
	parseFuncs FuncMap
	execFuncs  map[string]reflect.Value
}

// init initializes the set fields to default values.
func (s *Set) init() {
	if s.Tree == nil {
		s.Tree = make(parse.Tree)
	}
	if s.execFuncs == nil {
		s.execFuncs = make(map[string]reflect.Value)
	}
	if s.parseFuncs == nil {
		s.parseFuncs = make(FuncMap)
	}
}

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
func (s *Set) Clone() (*Set, error) {
	ns := new(Set).Delims(s.leftDelim, s.rightDelim)
	ns.init()
	for k, v := range s.parseFuncs {
		ns.parseFuncs[k] = v
	}
	for k, v := range s.execFuncs {
		ns.execFuncs[k] = v
	}
	err := ns.Tree.AddTree(s.Tree.CopyTree())
	if err != nil {
		return nil, err
	}
	return ns, nil
}

// Escape rewrites the set executing contextual HTML escaping in all
// templates, like in the standard html/template package.
//
// This must be called only once, after all templates were added to the set.
//
// If escaping fails, all templates are removed from the set, so that unsafe
// templates can't be executed.
func (s *Set) Escape() (*Set, error) {
	var err error
	s.Tree, err = escape.EscapeTree(s.Tree)
	s.Funcs(escape.FuncMap)
	return s, err
}

// Parsing --------------------------------------------------------------------

// parse parses the given text and adds the resulting templates to the set.
// The name is only used for debugging purposes: useful to parse multiple
// files or glob, for example, to know which file caused an error.
// Adding templates after the set executed results in error.
func (s *Set) parse(text, name string) (*Set, error) {
	s.init()
	if tree, err := parse.Parse(text, name, s.leftDelim, s.rightDelim,
		builtins, s.parseFuncs); err != nil {
		return nil, err
	} else if err = s.Tree.AddTree(tree); err != nil {
		return nil, err
	}
	return s, nil
}

// Parse parses the given text and adds the resulting templates to the set.
// If an error occurs, parsing stops and the returned set is nil; otherwise
// it is s.
func (s *Set) Parse(text string) (*Set, error) {
	return s.parse(text, "source")
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
		if b, err := ioutil.ReadFile(filename); err != nil {
			return nil, err
		} else if _, err = s.parse(string(b), filename); err != nil {
			return nil, err
		}
	}
	return s, nil
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
		return nil, fmt.Errorf("template: pattern matches no files: %#q",
			pattern)
	}
	return s.ParseFiles(filenames...)
}

// Convenience parsing wrappers -----------------------------------------------

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

// Parse creates a new Set with the template definitions from the given text.
// If an error occurs, parsing stops and the returned set is nil.
func Parse(text string) (*Set, error) {
	return new(Set).Parse(text)
}

// ParseFiles creates a new Set with the template definitions from the named
// files. There must be at least one file. If an error occurs, parsing stops
// and the returned set is nil.
func ParseFiles(filenames ...string) (*Set, error) {
	return new(Set).ParseFiles(filenames...)
}

// ParseGlob creates a new Set with the template definitions from the
// files identified by the pattern. The pattern is processed by filepath.Glob
// and must match at least one file. ParseGlob is equivalent to calling
// ParseFiles with the list of files matched by the pattern. If an error
// occurs, parsing stops and the returned set is nil.
func ParseGlob(pattern string) (*Set, error) {
	return new(Set).ParseGlob(pattern)
}

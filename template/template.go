// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"reflect"

	"code.google.com/p/sadbox/text/template/parse"
)

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

func (s *Set) Reset() {
	s.tmpl = nil
	s.leftDelim = ""
	s.rightDelim = ""
	s.parseFuncs = nil
	s.execFuncs = nil
}

// Delims sets the action delimiters to the specified strings, to be used in
// subsequent calls to Parse. An empty delimiter stands for the corresponding
// default: "{{" or "}}".
// The return value is the template, so calls can be chained.
func (s *Set) Delims(left, right string) *Set {
	s.leftDelim = left
	s.rightDelim = right
	return s
}

// Funcs adds the elements of the argument map to the template's function map.
// It panics if a value in the map is not a function with appropriate return
// type. However, it is legal to overwrite elements of the map. The return
// value is the template, so calls can be chained.
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

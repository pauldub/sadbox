// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zap

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"code.google.com/p/sadbox/zap/parse"
)

// Zapper extends the template language from text/template and html/template
// and compiles the result to templates compatible with those packages.
type Zapper struct {
	tree       map[string]*parse.Tree
	leftDelim  string
	rightDelim string
	funcs      map[string]interface{}
}

// init initializes default values.
func (z *Zapper) init() {
	if z.tree == nil {
		z.tree = make(map[string]*parse.Tree)
	}
	if z.funcs == nil {
		z.funcs = make(map[string]interface{})
	}
}

// init initializes default values.
func (z *Zapper) addTree(t map[string]*parse.Tree) error {
	z.init()
	for k, v := range t {
		if parse.IsEmptyTree(v.Root) {
			continue
		}
		if z.tree[k] != nil {
			return fmt.Errorf("zapper: duplicated template %q", k)
		}
		z.tree[k] = v
	}
	return nil
}

// Delims sets the template delimiters to the specified strings, to be used in
// subsequent calls to Zap. An empty delimiter stands for the corresponding
// default: {{ or }}.
func (z *Zapper) Delims(left, right string) *Zapper {
	z.leftDelim = left
	z.rightDelim = right
	return z
}

// Funcs adds template functions to be recognized by the zapper.
func (z *Zapper) Funcs(funcs map[string]interface{}) *Zapper {
	z.init()
	for k, v := range funcs {
		z.funcs[k] = v
	}
	return z
}

// Zap compiles all parsed templates and writes the result to the given writer.
// The resulting template is guaranteed to be compatible with the template
// language from text/template and html/template packages.
func (z *Zapper) Zap(w io.Writer) error {
	if err := parse.Compile(z.tree); err != nil {
		return err
	}
	for _, v := range z.tree {
		fmt.Fprint(w, v.Root)
	}
	return nil
}

// Parsing --------------------------------------------------------------------

// parse parses the given text and adds the resulting templates to the zapper.
func (z *Zapper) parse(text, name string) (*Zapper, error) {
	z.init()
	if tree, err := parse.Parse(text, name, z.leftDelim, z.rightDelim,
		builtins, z.funcs); err != nil {
		return nil, err
	} else if err = z.addTree(tree); err != nil {
		return nil, err
	}
	return z, nil
}

// Parse parses the given text and adds the resulting templates to the zapper.
func (z *Zapper) Parse(text string) (*Zapper, error) {
	return z.parse(text, "source")
}

// ParseFiles parses the given files and adds the resulting templates to the
// zapper.
func (z *Zapper) ParseFiles(filenames ...string) (*Zapper, error) {
	if len(filenames) == 0 {
		// Not really a problem, but be consistent.
		return nil, fmt.Errorf("zapper: no files named in call to ParseFiles")
	}
	for _, filename := range filenames {
		if b, err := ioutil.ReadFile(filename); err != nil {
			return nil, err
		} else if _, err = z.parse(string(b), filename); err != nil {
			return nil, err
		}
	}
	return z, nil
}

// ParseGlob parses templates based on a glob pattern and adds them to the
// zapper.
func (z *Zapper) ParseGlob(pattern string) (*Zapper, error) {
	filenames, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(filenames) == 0 {
		return nil, fmt.Errorf("zapper: pattern matches no files: %#q",
			pattern)
	}
	return z.ParseFiles(filenames...)
}

var builtins = map[string]interface{}{
	"and":      true,
	"call":     true,
	"html":     true,
	"index":    true,
	"js":       true,
	"len":      true,
	"not":      true,
	"or":       true,
	"print":    true,
	"printf":   true,
	"println":  true,
	"urlquery": true,
}

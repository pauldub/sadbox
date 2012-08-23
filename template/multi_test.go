// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

// Tests for mulitple-template parsing and execution.

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

const (
	noError  = true
	hasError = false
)

type multiParseTest struct {
	name    string
	input   string
	ok      bool
	names   []string
	results []string
}

var multiParseTests = []multiParseTest{
	{"empty", "", noError,
		nil,
		nil},
	{"one", `{{define "foo"}} FOO {{end}}`, noError,
		[]string{"foo"},
		[]string{`" FOO "`}},
	{"two", `{{define "foo"}} FOO {{end}}{{define "bar"}} BAR {{end}}`, noError,
		[]string{"foo", "bar"},
		[]string{`" FOO "`, `" BAR "`}},
	// errors
	{"missing end", `{{define "foo"}} FOO `, hasError,
		nil,
		nil},
	{"malformed name", `{{define "foo}} FOO `, hasError,
		nil,
		nil},
}

func TestMultiParse(t *testing.T) {
	for _, test := range multiParseTests {
		template, err := Parse(test.input)
		switch {
		case err == nil && !test.ok:
			t.Errorf("%q: expected error; got none", test.name)
			continue
		case err != nil && test.ok:
			t.Errorf("%q: unexpected error: %v", test.name, err)
			continue
		case err != nil && !test.ok:
			// expected error, got one
			if *debug {
				fmt.Printf("%s: %s\n\t%s\n", test.name, test.input, err)
			}
			continue
		}
		if template == nil {
			continue
		}
		if len(template.tmpl.GetAll()) != len(test.names) {
			t.Errorf("%s: wrong number of templates; wanted %d got %d", test.name, len(test.names), len(template.tmpl.GetAll()))
			continue
		}
		for i, name := range test.names {
			tmpl := template.tmpl.Get(name)
			if tmpl == nil {
				t.Errorf("%s: can't find template %q", test.name, name)
				continue
			}
			result := tmpl.String()
			if result != test.results[i] {
			// TODO: string will show a single template, and not a root with defines
			//	t.Errorf("%s=(%v): got\n\t%v\nexpected\n\t%v", test.name, test.input, result, test.results[i])
			}
		}
	}
}

var multiExecTests = []execTest{
	{"empty", "", "", nil, true},
	{"text", `{{define "foo"}}some text{{end}}`, "some text", nil, true},
	{"invoke x", `{{define "foo"}}{{template "x" .SI}}{{end}}`, "TEXT", tVal, true},
	{"invoke x no args", `{{define "foo"}}{{template "x"}}{{end}}`, "TEXT", tVal, true},
	{"invoke dot int", `{{define "foo"}}{{template "dot" .I}}{{end}}`, "17", tVal, true},
	{"invoke dot []int", `{{define "foo"}}{{template "dot" .SI}}{{end}}`, "[3 4 5]", tVal, true},
	{"invoke dotV", `{{define "foo"}}{{template "dotV" .U}}{{end}}`, "v", tVal, true},
	{"invoke nested int", `{{define "foo"}}{{template "nested" .I}}{{end}}`, "17", tVal, true},
	{"variable declared by template", `{{define "foo"}}{{template "nested" $x:=.SI}},{{index $x 1}}{{end}}`, "[3 4 5],4", tVal, true},

	// User-defined function: test argument evaluator.
	{"testFunc literal", `{{define "foo"}}{{oneArg "joe"}}{{end}}`, "oneArg=joe", tVal, true},
	{"testFunc .", `{{define "foo"}}{{oneArg .}}{{end}}`, "oneArg=joe", "joe", true},
}

// These strings are also in testdata/*.
const multiText1 = `
	{{define "x"}}TEXT{{end}}
	{{define "dotV"}}{{.V}}{{end}}
`

const multiText2 = `
	{{define "dot"}}{{.}}{{end}}
	{{define "nested"}}{{template "dot" .}}{{end}}
`

// TODO: must redesign these tests to use required {{define}}
func TestMultiExecute(t *testing.T) {
	/*
	// Declare a couple of templates first.
	template, err := Parse(multiText1)
	if err != nil {
		t.Fatalf("parse error for 1: %s", err)
	}
	template, err = template.Parse(multiText2)
	if err != nil {
		t.Fatalf("parse error for 2: %s", err)
	}
	testExecute(multiExecTests, template, t, false)
	*/
}

func TestParseFiles(t *testing.T) {
	/*
	_, err := ParseFiles("DOES NOT EXIST")
	if err == nil {
		t.Error("expected error for non-existent file; got none")
	}
	template, err := new(Set).ParseFiles("testdata/file1.tmpl", "testdata/file2.tmpl")
	if err != nil {
		t.Fatalf("error parsing files: %v", err)
	}
	testExecute(multiExecTests, template, t, false)
	*/
}

func TestParseGlob(t *testing.T) {
	/*
	_, err := ParseGlob("DOES NOT EXIST")
	if err == nil {
		t.Error("expected error for non-existent file; got none")
	}
	template, err := new(Set).ParseGlob("[x")
	if err == nil {
		t.Error("expected error for bad pattern; got none")
	}
	template, err = new(Set).ParseGlob("testdata/file*.tmpl")
	if err != nil {
		t.Fatalf("error parsing files: %v", err)
	}
	testExecute(multiExecTests, template, t, false)
	*/
}

// In these tests, actual content (not just template definitions) comes from the parsed files.

var templateFileExecTests = []execTest{
	{"test", `{{define "foo"}}{{template "tmpl1"}}{{template "tmpl2"}}{{end}}`, "template1-y-template2-x-", 0, true},
}

func TestParseFilesWithData(t *testing.T) {
	template, err := new(Set).ParseFiles("testdata/tmpl1.tmpl", "testdata/tmpl2.tmpl")
	if err != nil {
		t.Fatalf("error parsing files: %v", err)
	}
	testExecute(templateFileExecTests, template, t, false)
}

func TestParseGlobWithData(t *testing.T) {
	template, err := new(Set).ParseGlob("testdata/tmpl*.tmpl")
	if err != nil {
		t.Fatalf("error parsing files: %v", err)
	}
	testExecute(templateFileExecTests, template, t, false)
}

const (
	cloneText1 = `{{define "a"}}{{template "b"}}{{template "c"}}{{end}}`
	cloneText2 = `{{define "b"}}b{{end}}`
	cloneText3 = `{{define "c"}}root{{end}}`
	cloneText4 = `{{define "c"}}clone{{end}}`
)

func TestClone(t *testing.T) {
	// Create some templates and clone the root.
	template, err := new(Set).Parse(cloneText1)
	if err != nil {
		t.Fatal(err)
	}
	_, err = template.Parse(cloneText2)
	if err != nil {
		t.Fatal(err)
	}
	clone, _ := template.Clone()
	// Add variants to both.
	_, err = template.Parse(cloneText3)
	if err != nil {
		t.Fatal(err)
	}
	_, err = clone.Parse(cloneText4)
	if err != nil {
		t.Fatal(err)
	}
	// Verify that the clone is self-consistent.
	for k, v := range clone.tmpl.GetAll() {
		if v == nil {
			t.Errorf("clone %q contain nil node", k)
		}
	}
	// Execute root.
	var b bytes.Buffer
	err = template.Execute(&b, "a", 0)
	if err != nil {
		t.Fatal(err)
	}
	if b.String() != "broot" {
		t.Errorf("expected %q got %q", "broot", b.String())
	}
	// Execute copy.
	b.Reset()
	err = clone.Execute(&b, "a", 0)
	if err != nil {
		t.Fatal(err)
	}
	if b.String() != "bclone" {
		t.Errorf("expected %q got %q", "bclone", b.String())
	}
}

func TestRedefinition(t *testing.T) {
	var tmpl *Set
	var err error
	if tmpl, err = new(Set).Parse(`{{define "test"}}foo{{end}}`); err != nil {
		t.Fatalf("parse 1: %v", err)
	}
	if _, err = tmpl.Parse(`{{define "test"}}bar{{end}}`); err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "duplicated template") {
		t.Fatalf("expected redefinition error; got %v", err)
	}
}

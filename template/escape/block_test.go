// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecBlock(t *testing.T) {
	source := `
	{{define "Base"}}
		<p>{{block "body"}}
			Base body.
		{{end}}</p>
	{{end}}

	{{define "Home"}}
		{{fill "Base" .}}
			{{block "body"}}
				Home body.
			{{end}}
		{{end}}
	{{end}}

	{{define "Cond1"}}
		{{fill "Base" .}}
			{{block "body" .}}
				{{if .what}}
					{{.what}} body.
				{{else}}
					Undefined body.
				{{end}}
			{{end}}
		{{end}}
	{{end}}

	{{define "Cond2"}}
		{{fill "Base" .}}
			{{if .what}}
				{{block "body" .}}
					{{.what}} body.
				{{end}}
			{{else}}
				{{block "body" .}}
					Undefined body.
				{{end}}
			{{end}}
		{{end}}
	{{end}}

	{{define "Foo"}}{{block "foo"}}Foo{{end}}{{end}}

	{{define "FooBar"}}{{fill "Foo"}}{{block "foo"}}{{fill "Foo"}}{{end}}Bar{{end}}{{end}}{{end}}

	{{define "FooBarBaz"}}{{fill "Foo"}}{{block "foo"}}{{fill "FooBar"}}{{end}}Baz{{end}}{{end}}{{end}}
	`

	type test struct {
		name    string
		prefix  string
		suffix  string
		content string
		data    map[string]string
	}
	data := map[string]string{"what": "Gophers"}
	tests := []test{
		{"Base", "<p>", "</p>", "Base body.", nil},
		{"Home", "<p>", "</p>", "Home body.", nil},
		{"Cond1", "<p>", "</p>", "Undefined body.", nil},
		{"Cond1", "<p>", "</p>", "Gophers body.", data},
		{"Cond2", "<p>", "</p>", "Undefined body.", nil},
		{"Cond2", "<p>", "</p>", "Gophers body.", data},
		{"Foo", "", "", "Foo", nil},
		{"FooBar", "", "", "FooBar", nil},
		{"FooBarBaz", "", "", "FooBarBaz", nil},
	}

	tmpl, err := New("root").Parse(source)
	if err != nil {
		t.Fatal(err)
	}

	b := new(bytes.Buffer)
	for _, v := range tests {
		b.Reset()
		if err = tmpl.ExecuteTemplate(b, v.name, v.data); err != nil {
			t.Fatal(err)
		}
		s := strings.TrimSpace(b.String())
		if !strings.HasPrefix(s, v.prefix) || !strings.HasSuffix(s, v.suffix) || strings.Index(s, v.content) == -1 {
			t.Errorf("Expected %q, got %q", v.content, s)
		}
	}
}

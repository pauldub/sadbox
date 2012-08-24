// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"bytes"
	"testing"
	htmlTemplate "html/template"
	textTemplate "text/template"
	textParse    "text/template/parse"

	"code.google.com/p/sadbox/template/parse"
)

var benchTemplate = `
{{define "page"}}
<!doctype html>
<html>
  <head>
	<title>{{template "title" .}}</title>
  </head>
  <body>
	{{template "header" .}}
	<ul class="navigation">
	{{range .Menu}}
	  <li><a href="{{.Link}}">{{.Text}}</a></li>
	{{else}}
		<li>No menu items</li>
	{{end}}
	</ul>
	<div class="table">
	  <table>
	  {{range .Rows}}
		<tr>
		{{range .}}
		  <td>{{.}}</td>
		{{end}}
		</tr>
	  {{end}}
	  </table>
	</div>
	{{template "footer" .}}
  </body>
</html>
{{end}}

{{define "title"}}{{.PageTitle}}{{end}}

{{define "header"}}
  <div class="header">
	<h1>{{.PageTitle}}</h1>
  </div>
{{end}}

{{define "footer"}}
  <div class="footer">
	<p>Copyright 2012 The Gorilla Authors.</p>
  </div>
{{end}}
`

var benchData = map[string]interface{}{
	"PageTitle": "Benchmark",
	"Menu": []map[string]string{
		{"Link": "/", "Text": "Home"},
		{"Link": "/downloads", "Text": "Downloads"},
		{"Link": "/products", "Text": "Products"},
	},
	"Rows": [][]int{
		{1, 2, 3, 4, 5},
		{6, 7, 8, 9, 10},
		{11, 12, 13, 14, 15},
		{16, 17, 18, 19, 20},
	},
}

func BenchmarkParser(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tree, err := parse.Parse("page", benchTemplate, "{{", "}}")
		if err != nil {
			panic(err)
		}
		if len(tree) != 4 {
			panic("template not parsed")
		}
	}
}

func BenchmarkStdParser(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tree, err := textParse.Parse("page", benchTemplate, "{{", "}}")
		if err != nil {
			panic(err)
		}
		if len(tree) != 4 {
			panic("template not parsed")
		}
	}
}

// ----------------------------------------------------------------------------

func BenchmarkTextExecutor(b *testing.B) {
	set, err := Parse(benchTemplate)
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err = set.Execute(&buf, "page", benchData)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkStdTextExecutor(b *testing.B) {
	set, err := textTemplate.New("bench").Parse(benchTemplate)
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err = set.ExecuteTemplate(&buf, "page", benchData)
		if err != nil {
			panic(err)
		}
	}
}

// ----------------------------------------------------------------------------

func BenchmarkHtmlExecutor(b *testing.B) {
	set, err := Parse(benchTemplate)
	if err != nil {
		panic(err)
	}
	set, err = set.Escape()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err = set.Execute(&buf, "page", benchData)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkStdHtmlExecutor(b *testing.B) {
	set, err := htmlTemplate.New("bench").Parse(benchTemplate)
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		err = set.ExecuteTemplate(&buf, "page", benchData)
		if err != nil {
			panic(err)
		}
	}
}

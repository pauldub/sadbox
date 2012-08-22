// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"testing"
	"text/template/parse"
)

var benchTemplate = `
{{define "page"}}
<!doctype html>
<html>
  <head>
	<title>{{.PageTitle}}</title>
  </head>
  <body>
	<div class="header">
	  <h1>{{.PageTitle}}</h1>
	</div>
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
  </body>
</html>
{{end}}`

// to bench execution one day
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

func BenchmarkSad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tree := make(map[string]*DefineNode)
		err := Parse(tree, "page", benchTemplate, "{{", "}}")
		if err != nil {
			panic(err)
		}
		if len(tree) != 1 {
			panic("template not parsed")
		}
	}
}

func BenchmarkStd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		tree, err := parse.Parse("page", benchTemplate, "{{", "}}")
		if err != nil {
			panic(err)
		}
		if len(tree) != 1 {
			panic("template not parsed")
		}
	}
}

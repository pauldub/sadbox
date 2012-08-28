// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package zap extends the template language from text/template and html/template
and compiles the result to templates compatible with those packages.

For now, only one extension is available: template inheritance. Let's see how
it works.

Note: if you are familiar with Django templates, Jinja2, Mako, dust.js
or many others, you already know the concept.

Templates can include {{block}} actions that define replaceable template parts.
We define a block just to mark a content as a placeholder, like this:

	{{define "base"}}
	  <html>
		<body>
		  <header>
			Welcome to the Zap World.
		  </header>

		  {{block "content"}}
			<p>This is just a placeholder.</p>
		  {{end}}

		  <footer>
			Copyright 2012 The Zap Team.
		  </footer>
		</body>
	  </html>
	{{end}}

The idea is that we can reuse this same page layout in other templates, just
replacing the "content" block. For this we will need a template that "extends"
the previous one. Here is how we declare it:

	{{define "tutorial" "base"}}
	  ...
	{{end}}

The {{define}} action accepts a second string, which is the name of the
template being extended. You can read it as "define tutorial extending base".

A template that extends another one can do only one thing: define blocks
to replace the ones from the base template. Here we override the "content"
block:

	{{define "tutorial" "base"}}
	  {{block "content"}}
		<p>Welcome to the Zap tutorial.</p>
	  {{end}}
	{{end}}

When we compile this template, the result will be a template with the layout
from the "base" template, and the "content" block replaced by the one it
defines.

The zap package can compute this, and output templates that text/template or
html/template can execute. Here's how:

	zapper, err := new(zap.Zapper).ParseFiles("base.html", "tutorial.html")
	if err != nil {
		// ... do something with the parsing error
	}
	err = zapper.Zap(w)
	if err != nil {
		// ... do something with the compilation error
	}

After this, the compiled templates are written to the writer passed to Zap
and can be used with the standard template packages.
*/
package zap

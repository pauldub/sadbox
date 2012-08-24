// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package template is a simplified version of the standard packages text/template
and html/template. It merges both packages into a single API. Here is a summary
of the differences:

	- All templates must be defined using a {{define}} action. This makes
	  all templates self-contained. The API becomes cleaner and more uniform:
	  template names come from a single place, the templates themselves.
	  From an API perspective, this means that template.New("name") doesn't
	  exist anymore and all templates must be executed by name, since there's
	  no more implicit "root" template.
	- Templates are grouped in sets, so there're no more standalone Template
	  instances. All actions -- parsing, executing -- are performed in a set.
	- Escaping as provided by the html/template package must be performed
	  manually calling set.Escape() after all templates were added to a set.
	- Two new actions were added: {{block}} and {{fill}}, which allow
	  skeleton templates to be filled by other templates. This must be
	  familiar to Python developers because it is similar to what Django,
	  Jinja2 or Mako provide through template inheritance.

The rest is basically the same, the grammar is the same, and the syntax is the
same, as it is built on top of the zen foundations from these packages:

	http://golang.org/pkg/text/template
	http://golang.org/pkg/html/template

But let's show a quick usage example while our docs don't provide full
description of the template language and detailed explanations. Here we go.

Templates are stored in a collection of related templates, called a "Set".
Templates that call each other must belong to the same set. To create a new
set we call Parse:

	set, err := template.Parse(`{{define "hello"}}Hello, World.{{end}}`)
	if err != nil {
		// do something with the parsing error...
	}

This adds all templates defined using {{define "name"}}...{{end}} to the set.
In the example above, it adds a single template named "hello".
Duplicated template names result in an error; template names must be unique.
To add more templates we call Parse again, this time on the set we created:

	set, err = set.Parse(`{{define "bye"}}Good bye, World.{{end}}`)
	if err != nil {
		// do something with the parsing error...
	}

Now our set has two templates, and we can execute any of them calling
Set.Execute and passing an io.Writer, the name of the template to execute
and related data:

	err = set.Execute(os.Stderr, "hello", nil)
	if err != nil {
		// do something with the execution error...
	}

For HTML usage it is a good idea to enable contextual escaping. We do this
calling Set.Escape after all templates were added to a set:

	set, err := template.Parse(`{{define "hello"}}Hello, World.{{end}}`)
	// ...
	set, err := set.Escape()

Without calling Escape the template works like in the text/template package.
After calling Escape it behaves like in the html/template package, escaping
HTML/CSS/JS data contextually, as needed.

The set must not be changed after escaping was performed.

And that's all for now.
*/
package template

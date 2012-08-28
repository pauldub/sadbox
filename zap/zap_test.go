// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zap

import (
	"bytes"
	"testing"
	textTemplate "text/template"
	htmlTemplate "html/template"
)

func TestBlock(t *testing.T) {
	tpl := `
	{{define "t1"}}foo-{{block "b1"}}t1b1-{{end}}bar{{end}}
	{{define "t2" "t1"}}{{block "b1"}}t2b1-{{block "b2"}}t2b2-{{end}}{{end}}{{end}}
	{{define "t3" "t2"}}{{block "b2"}}t3b2-{{end}}{{end}}
	{{define "t4" "t2"}}{{block "b2"}}t4b2-{{end}}{{end}}
	`
	expect := map[string]string{
		"t1": "foo-t1b1-bar",
		"t2": "foo-t2b1-t2b2-bar",
		"t3": "foo-t2b1-t3b2-bar",
		"t4": "foo-t2b1-t4b2-bar",
	}

	zapper, err := new(Zapper).Parse(tpl)
	if err != nil {
		t.Fatal(err)
	}
	buf := new(bytes.Buffer)
	if err := zapper.Zap(buf); err != nil {
		t.Fatal(err)
	}

	tpl = buf.String()
	txt := textTemplate.Must(textTemplate.New("_").Parse(tpl))
	htm := htmlTemplate.Must(htmlTemplate.New("_").Parse(tpl))

	for name, value := range expect {
		buf.Reset()
		err = txt.ExecuteTemplate(buf, name, nil)
		if err != nil {
			t.Fatal(err)
		}
		result := buf.String()
		if result != value {
			t.Errorf("expected %q, got %q", value, result)
		}

		buf.Reset()
		err = htm.ExecuteTemplate(buf, name, nil)
		if err != nil {
			t.Fatal(err)
		}
		result = buf.String()
		if result != value {
			t.Errorf("expected %q, got %q", value, result)
		}
	}
}

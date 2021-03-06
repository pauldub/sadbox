// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gettext

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"os"
	"runtime"
	"testing"
)

func decode(value []byte) ([]byte, error) {
	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(value)))
	b, err := base64.StdEncoding.Decode(decoded, value)
	if err != nil {
		return nil, err
	}
	return decoded[:b], nil
}

func newFile(testName string, t *testing.T) (f *os.File) {
	// Use a local file system, not NFS.
	// On Unix, override $TMPDIR in case the user
	// has it set to an NFS-mounted directory.
	dir := ""
	if runtime.GOOS != "windows" {
		dir = "/tmp"
	}
	f, err := ioutil.TempFile(dir, "_Go_"+testName)
	if err != nil {
		t.Fatalf("open %s: %s", testName, err)
	}
	return
}

// From Python's gettext tests
var gnuMoData = `3hIElQAAAAAGAAAAHAAAAEwAAAALAAAAfAAAAAAAAACoAAAAFQAAAKkAAAAjAAAAvwAAAKEAAADj
AAAABwAAAIUBAAALAAAAjQEAAEUBAACZAQAAFgAAAN8CAAAeAAAA9gIAAKEAAAAVAwAABQAAALcD
AAAJAAAAvQMAAAEAAAADAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAABQAAAAYAAAACAAAAAFJh
eW1vbmQgTHV4dXJ5IFlhY2gtdABUaGVyZSBpcyAlcyBmaWxlAFRoZXJlIGFyZSAlcyBmaWxlcwBU
aGlzIG1vZHVsZSBwcm92aWRlcyBpbnRlcm5hdGlvbmFsaXphdGlvbiBhbmQgbG9jYWxpemF0aW9u
CnN1cHBvcnQgZm9yIHlvdXIgUHl0aG9uIHByb2dyYW1zIGJ5IHByb3ZpZGluZyBhbiBpbnRlcmZh
Y2UgdG8gdGhlIEdOVQpnZXR0ZXh0IG1lc3NhZ2UgY2F0YWxvZyBsaWJyYXJ5LgBtdWxsdXNrAG51
ZGdlIG51ZGdlAFByb2plY3QtSWQtVmVyc2lvbjogMi4wClBPLVJldmlzaW9uLURhdGU6IDIwMDAt
MDgtMjkgMTI6MTktMDQ6MDAKTGFzdC1UcmFuc2xhdG9yOiBKLiBEYXZpZCBJYsOhw7FleiA8ai1k
YXZpZEBub29zLmZyPgpMYW5ndWFnZS1UZWFtOiBYWCA8cHl0aG9uLWRldkBweXRob24ub3JnPgpN
SU1FLVZlcnNpb246IDEuMApDb250ZW50LVR5cGU6IHRleHQvcGxhaW47IGNoYXJzZXQ9aXNvLTg4
NTktMQpDb250ZW50LVRyYW5zZmVyLUVuY29kaW5nOiBub25lCkdlbmVyYXRlZC1CeTogcHlnZXR0
ZXh0LnB5IDEuMQpQbHVyYWwtRm9ybXM6IG5wbHVyYWxzPTI7IHBsdXJhbD1uIT0xOwoAVGhyb2F0
d29iYmxlciBNYW5ncm92ZQBIYXkgJXMgZmljaGVybwBIYXkgJXMgZmljaGVyb3MAR3V2ZiB6YnFo
eXIgY2ViaXZxcmYgdmFncmVhbmd2YmFueXZtbmd2YmEgbmFxIHlicG55dm1uZ3ZiYQpmaGNjYmVn
IHNiZSBsYmhlIENsZ3ViYSBjZWJ0ZW56ZiBvbCBjZWJpdnF2YXQgbmEgdmFncmVzbnByIGdiIGd1
ciBUQUgKdHJnZ3JrZyB6cmZmbnRyIHBuZ255YnQgeXZvZW5lbC4AYmFjb24Ad2luayB3aW5rAA==`

func TestReadMo(t *testing.T) {
	equalString := func(s1, s2 string) {
		if s1 != s2 {
			t.Errorf("Expected %q, got %q.", s2, s1)
		}
	}

	mr := new(MoReader)

	b, err := decode([]byte(gnuMoData))
	if err != nil {
		t.Fatal(err)
	}
	c := NewCatalog()
	if err := mr.Read(c, bytes.NewReader(b)); err != nil {
		t.Fatal(err)
	}

	// gettext
	equalString(c.Get("albatross"), "")
	equalString(c.Get("mullusk"), "bacon")
	equalString(c.Get("Raymond Luxury Yach-t"), "Throatwobbler Mangrove")
	equalString(c.Get("nudge nudge"), "wink wink")
	// ngettext
	equalString(c.GetPlural("There is %s file", 1), "Hay %s fichero")
	equalString(c.GetPlural("There is %s file", 2), "Hay %s ficheros")
}

func TestWriteMo(t *testing.T) {
	equalString := func(s1, s2 string) {
		if s1 != s2 {
			t.Errorf("Expected %q, got %q.", s2, s1)
		}
	}

	mr := new(MoReader)
	mw := new(MoWriter)

	c := NewCatalog()
	c.Add(&SimpleMessage{Src: "foo", Dst: "bar"})
	c.Add(&SimpleMessage{Src: "baz", Dst: "ding"})
	c.Add(&PluralMessage{
		Src: []string{"bubble", "bubbles"},
		Dst: []string{"bolha", "bolhas"},
	})

	f1 := newFile("testWriteMo", t)
	if err := mw.Write(c, f1); err != nil {
		t.Fatal(err)
	}
	f1.Close()

	f2, err := os.Open(f1.Name())
	if err != nil {
		t.Fatal(err)
	}
	c2 := NewCatalog()
	if err := mr.Read(c2, f2); err != nil {
		t.Fatal(err)
	}
	f2.Close()

	// gettext
	equalString(c2.Get("foo"), "bar")
	equalString(c2.Get("baz"), "ding")
	// ngettext
	equalString(c2.GetPlural("bubble", 2), "bolhas")
}

func TestWriteMo2(t *testing.T) {
	equalString := func(s1, s2 string) {
		if s1 != s2 {
			t.Errorf("Expected %q, got %q.", s2, s1)
		}
	}

	mr := new(MoReader)
	mw := new(MoWriter)

	b, err := decode([]byte(gnuMoData))
	if err != nil {
		t.Fatal(err)
	}
	c := NewCatalog()
	if err := mr.Read(c, bytes.NewReader(b)); err != nil {
		t.Fatal(err)
	}

	f1 := newFile("testWriteMo", t)
	if err := mw.Write(c, f1); err != nil {
		t.Fatal(err)
	}
	f1.Close()

	f2, err := os.Open(f1.Name())
	if err != nil {
		t.Fatal(err)
	}
	c2 := NewCatalog()
	if err := mr.Read(c2, f2); err != nil {
		t.Fatal(err)
	}
	f2.Close()

	// gettext
	equalString(c.Get("albatross"), "")
	equalString(c.Get("mullusk"), "bacon")
	equalString(c.Get("Raymond Luxury Yach-t"), "Throatwobbler Mangrove")
	equalString(c.Get("nudge nudge"), "wink wink")
	// ngettext
	equalString(c.GetPlural("There is %s file", 1), "Hay %s fichero")
	equalString(c.GetPlural("There is %s file", 2), "Hay %s ficheros")
}

func TestContext(t *testing.T) {
	equalString := func(s1, s2 string) {
		if s1 != s2 {
			t.Errorf("Expected %q, got %q.", s2, s1)
		}
	}

	mr := new(MoReader)
	mw := new(MoWriter)

	testCatalog := func(c *Catalog) {
		equalString(c.Get("food"), "comida")
		equalString(c.Get("music"), "")

		c.SetContext("kids")
		equalString(c.Get("food"), "merenda")
		equalString(c.Get("music"), "melodia")

		c.SetContext("slang")
		equalString(c.Get("food"), "rango")
		equalString(c.Get("music"), "sonzera")
	}

	c := NewCatalog()
	c.Add(&SimpleMessage{Src: "food", Dst: "comida"})
	m := &SimpleMessage{Src: "food", Dst: "merenda", Ctx: "kids", HasCtx: true}
	c.Add(m)
	m = &SimpleMessage{Src: "food", Dst: "rango", Ctx: "slang", HasCtx: true}
	c.Add(m)
	m = &SimpleMessage{Src: "music", Dst: "melodia", Ctx: "kids", HasCtx: true}
	c.Add(m)
	m = &SimpleMessage{Src: "music", Dst: "sonzera", Ctx: "slang", HasCtx: true}
	c.Add(m)

	f1 := newFile("testWriteMo", t)
	if err := mw.Write(c, f1); err != nil {
		t.Fatal(err)
	}
	f1.Close()

	f2, err := os.Open(f1.Name())
	if err != nil {
		t.Fatal(err)
	}
	c2 := NewCatalog()
	if err := mr.Read(c2, f2); err != nil {
		t.Fatal(err)
	}
	f2.Close()
	testCatalog(c2)
}

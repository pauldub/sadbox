package parse

import (
	"testing"
)

func TestBlock(t *testing.T) {
	/*
	src := `
	{{define "t1"}}foo-{{block "b1"}}t1b1-{{end}}bar-{{end}}
	{{define "t2" "t1"}}{{block "b1"}}t2b1-{{block "b2"}}t2b2-{{end}}{{end}}{{end}}
	{{define "t3" "t2"}}{{block "b2"}}t3b2-{{end}}{{end}}
	{{define "t4" "t2"}}{{block "b2"}}t4b2-{{end}}{{end}}
	`

	treeSet, err := Parse(src, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	Compile(treeSet)

	for k, v := range treeSet {
		if IsEmptyTree(v.Root) {
			continue
		}
		t.Errorf("%v: %v", k, v.Root.String())
	}
	*/
}

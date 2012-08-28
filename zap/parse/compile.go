package parse

import (
	"fmt"
)

/*
// renders: foo t1b1 bar
{{define "t1"}}
	foo
	{{block "b1"}}
		t1b1
	{{end}}
	bar
{{end}}

// meaning: define "t2" extending "t1"
// renders: foo t2b1 t2b2 bar
{{define "t2" "t1"}}
	{{block "b1"}}
		t2b1
		{{block "b2"}}
			t2b2
		{{end}}
	{{end}}
{{end}}

// meaning: define "t3" extending "t2"
// renders: foo t2b1 t3b2 bar
{{define "t3" "t2"}}
	{{block "b2"}}
		t3b2
	{{end}}
{{end}}
*/
// TODO: pass compilation options. Right now this is just a dirty hack.
func Compile(treeSet map[string]*Tree) error {
	for name, _ := range treeSet {
		names, err := inlineParentList(treeSet, name)
		if err != nil {
			return err
		}
		for len(names) > 0 {
			// inline in reverse order
			inlineParent(treeSet, names[len(names)-1])
			names = names[:len(names)-1]
		}
	}
	for _, v := range treeSet {
		if err := inlineBlocks(v.Root.List); err != nil {
			return err
		}
	}
	return nil
}

func inlineParent(treeSet map[string]*Tree, name string) {
	// to be discarded
	define := treeSet[name].Root
	// to replace the original
	parent := treeSet[define.Parent].Root.CopyDefine()
	parent.Name = define.Name
	src := make(map[string]*BlockNode)
	extractBlocks(src, define.List)
	dst := make(map[string]*BlockNode)
	extractBlocks(dst, parent.List)
	for k, v := range dst {
		if block := src[k]; block != nil {
			v.List = block.List
		}
	}
	treeSet[name].Root = parent
}

// inlineParentList returns the parent templates that need inlining for a given
// template name.  It returns an error if a dependency is not found or
// recursive dependency is detected.
func inlineParentList(treeSet map[string]*Tree, name string) (deps []string, err error) {
	for {
		define := treeSet[name]
		if define == nil || define.Root == nil {
			return nil, fmt.Errorf("template not found: %q", name)
		}
		parentName := define.Root.Parent
		if parentName == "" {
			break
		}
		for _, v := range deps {
			if v == name {
				deps = append(deps, name)
				return nil, fmt.Errorf("impossible recursion: %#v", deps)
			}
		}
		deps = append(deps, name)
		name = parentName
	}
	return
}

func extractBlocks(dst map[string]*BlockNode, n Node) {
	switch n := n.(type) {
	case *BlockNode:
		dst[n.Name] = n
		extractBlocks(dst, n.List)
	case *DefineNode:
		extractBlocks(dst, n.List)
	case *IfNode:
		extractBlocks(dst, n.List)
		extractBlocks(dst, n.ElseList)
	case *ListNode:
		for _, node := range n.Nodes {
			extractBlocks(dst, node)
		}
	case *RangeNode:
		extractBlocks(dst, n.List)
		extractBlocks(dst, n.ElseList)
	case *WithNode:
		extractBlocks(dst, n.List)
		extractBlocks(dst, n.ElseList)
	}
}

func inlineBlocks(n Node) error {
	switch n := n.(type) {
	case *BlockNode:
		return fmt.Errorf("block node can't be replaced by itself")
	case *DefineNode:
		inlineBlocks(n.List)
	case *IfNode:
		inlineBlocks(n.List)
		inlineBlocks(n.ElseList)
	case *ListNode:
		for k, node := range n.Nodes {
			if block, ok := node.(*BlockNode); ok {
				n.Nodes[k] = block.List
				inlineBlocks(block.List)
			} else {
				inlineBlocks(node)
			}
		}
	case *RangeNode:
		inlineBlocks(n.List)
		inlineBlocks(n.ElseList)
	case *WithNode:
		inlineBlocks(n.List)
		inlineBlocks(n.ElseList)
	}
	return nil
}

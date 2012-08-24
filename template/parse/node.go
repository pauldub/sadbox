// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Parse nodes.

package parse

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// A node is an element in the parse tree. The interface is trivial.
type Node interface {
	Type() NodeType
	String() string
	// Copy does a deep copy of the Node and all its components.
	// To avoid type assertions, some XxxNodes also have specialized
	// CopyXxx methods that return *XxxNode.
	Copy() Node
}

// NodeType identifies the type of a parse tree node.
type NodeType int

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeText       NodeType = iota // Plain text.
	NodeAction                     // A simple action such as field evaluation.
	NodeBlock                      // A block action.
	NodeBool                       // A boolean constant.
	NodeCommand                    // An element of a pipeline.
	NodeDefine                     // A template definition.
	NodeDot                        // The cursor, dot.
	nodeElse                       // An else action. Not added to tree.
	nodeEnd                        // An end action. Not added to tree.
	NodeField                      // A field or method name.
	NodeFill                       // A fill action.
	NodeIdentifier                 // An identifier; always a function name.
	NodeIf                         // An if action.
	NodeList                       // A list of Nodes.
	NodeNil                        // An untyped nil constant.
	NodeNumber                     // A numerical constant.
	NodePipe                       // A pipeline of commands.
	NodeRange                      // A range action.
	NodeTree                       // A tree of define nodes.
	NodeString                     // A string constant.
	NodeTemplate                   // A template inv
	NodeVariable                   // A $ variable.
	NodeWith                       // A with action.
)

// Nodes.

// ListNode holds a sequence of nodes.
type ListNode struct {
	NodeType
	Nodes []Node // The element nodes in lexical order.
}

func newList() *ListNode {
	return &ListNode{NodeType: NodeList}
}

func (l *ListNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

func (l *ListNode) String() string {
	b := new(bytes.Buffer)
	for _, n := range l.Nodes {
		fmt.Fprint(b, n)
	}
	return b.String()
}

func (l *ListNode) CopyList() *ListNode {
	if l == nil {
		return l
	}
	n := newList()
	for _, elem := range l.Nodes {
		n.append(elem.Copy())
	}
	return n
}

func (l *ListNode) Copy() Node {
	return l.CopyList()
}

// TextNode holds plain text.
type TextNode struct {
	NodeType
	Text []byte // The text; may span newlines.
}

func newText(text string) *TextNode {
	return &TextNode{NodeType: NodeText, Text: []byte(text)}
}

func (t *TextNode) String() string {
	return fmt.Sprintf("%q", t.Text)
}

func (t *TextNode) Copy() Node {
	return &TextNode{NodeType: NodeText, Text: append([]byte{}, t.Text...)}
}

// PipeNode holds a pipeline with optional declaration
type PipeNode struct {
	NodeType
	Line int             // The line number in the input.
	Decl []*VariableNode // Variable declarations in lexical order.
	Cmds []*CommandNode  // The commands in lexical order.
}

func newPipeline(line int, decl []*VariableNode) *PipeNode {
	return &PipeNode{NodeType: NodePipe, Line: line, Decl: decl}
}

func (p *PipeNode) append(command *CommandNode) {
	p.Cmds = append(p.Cmds, command)
}

func (p *PipeNode) String() string {
	s := ""
	if len(p.Decl) > 0 {
		for i, v := range p.Decl {
			if i > 0 {
				s += ", "
			}
			s += v.String()
		}
		s += " := "
	}
	for i, c := range p.Cmds {
		if i > 0 {
			s += " | "
		}
		s += c.String()
	}
	return s
}

func (p *PipeNode) CopyPipe() *PipeNode {
	if p == nil {
		return p
	}
	var decl []*VariableNode
	for _, d := range p.Decl {
		decl = append(decl, d.Copy().(*VariableNode))
	}
	n := newPipeline(p.Line, decl)
	for _, c := range p.Cmds {
		n.append(c.Copy().(*CommandNode))
	}
	return n
}

func (p *PipeNode) Copy() Node {
	return p.CopyPipe()
}

// ActionNode holds an action (something bounded by delimiters).
// Control actions have their own nodes; ActionNode represents simple
// ones such as field evaluations.
type ActionNode struct {
	NodeType
	Line int       // The line number in the input.
	Pipe *PipeNode // The pipeline in the action.
}

func newAction(line int, pipe *PipeNode) *ActionNode {
	return &ActionNode{NodeType: NodeAction, Line: line, Pipe: pipe}
}

func (a *ActionNode) String() string {
	return fmt.Sprintf("{{%s}}", a.Pipe)

}

func (a *ActionNode) Copy() Node {
	return newAction(a.Line, a.Pipe.CopyPipe())

}

// CommandNode holds a command (a pipeline inside an evaluating action).
type CommandNode struct {
	NodeType
	Args []Node // Arguments in lexical order: Identifier, field, or constant.
}

func newCommand() *CommandNode {
	return &CommandNode{NodeType: NodeCommand}
}

func (c *CommandNode) append(arg Node) {
	c.Args = append(c.Args, arg)
}

func (c *CommandNode) String() string {
	s := ""
	for i, arg := range c.Args {
		if i > 0 {
			s += " "
		}
		s += arg.String()
	}
	return s
}

func (c *CommandNode) Copy() Node {
	if c == nil {
		return c
	}
	n := newCommand()
	for _, c := range c.Args {
		n.append(c.Copy())
	}
	return n
}

// IdentifierNode holds an identifier.
type IdentifierNode struct {
	NodeType
	Ident string // The identifier's name.
}

// NewIdentifier returns a new IdentifierNode with the given identifier name.
func NewIdentifier(ident string) *IdentifierNode {
	return &IdentifierNode{NodeType: NodeIdentifier, Ident: ident}
}

func (i *IdentifierNode) String() string {
	return i.Ident
}

func (i *IdentifierNode) Copy() Node {
	return NewIdentifier(i.Ident)
}

// VariableNode holds a list of variable names. The dollar sign is
// part of the name.
type VariableNode struct {
	NodeType
	Ident []string // Variable names in lexical order.
}

func newVariable(ident string) *VariableNode {
	return &VariableNode{NodeType: NodeVariable, Ident: strings.Split(ident, ".")}
}

func (v *VariableNode) String() string {
	s := ""
	for i, id := range v.Ident {
		if i > 0 {
			s += "."
		}
		s += id
	}
	return s
}

func (v *VariableNode) Copy() Node {
	return &VariableNode{NodeType: NodeVariable, Ident: append([]string{}, v.Ident...)}
}

// DotNode holds the special identifier '.'. It is represented by a nil pointer.
type DotNode bool

func newDot() *DotNode {
	return nil
}

func (d *DotNode) Type() NodeType {
	return NodeDot
}

func (d *DotNode) String() string {
	return "."
}

func (d *DotNode) Copy() Node {
	return newDot()
}

// NilNode holds the special identifier 'nil' representing an untyped nil constant.
// It is represented by a nil pointer.
type NilNode bool

func newNil() *NilNode {
	return nil
}

func (d *NilNode) Type() NodeType {
	return NodeNil
}

func (d *NilNode) String() string {
	return "nil"
}

func (d *NilNode) Copy() Node {
	return newNil()
}

// FieldNode holds a field (identifier starting with '.').
// The names may be chained ('.x.y').
// The period is dropped from each ident.
type FieldNode struct {
	NodeType
	Ident []string // The identifiers in lexical order.
}

func newField(ident string) *FieldNode {
	return &FieldNode{NodeType: NodeField, Ident: strings.Split(ident[1:], ".")} // [1:] to drop leading period
}

func (f *FieldNode) String() string {
	s := ""
	for _, id := range f.Ident {
		s += "." + id
	}
	return s
}

func (f *FieldNode) Copy() Node {
	return &FieldNode{NodeType: NodeField, Ident: append([]string{}, f.Ident...)}
}

// BoolNode holds a boolean constant.
type BoolNode struct {
	NodeType
	True bool // The value of the boolean constant.
}

func newBool(true bool) *BoolNode {
	return &BoolNode{NodeType: NodeBool, True: true}
}

func (b *BoolNode) String() string {
	if b.True {
		return "true"
	}
	return "false"
}

func (b *BoolNode) Copy() Node {
	return newBool(b.True)
}

// NumberNode holds a number: signed or unsigned integer, float, or complex.
// The value is parsed and stored under all the types that can represent the value.
// This simulates in a small amount of code the behavior of Go's ideal constants.
type NumberNode struct {
	NodeType
	IsInt      bool       // Number has an integral value.
	IsUint     bool       // Number has an unsigned integral value.
	IsFloat    bool       // Number has a floating-point value.
	IsComplex  bool       // Number is complex.
	Int64      int64      // The signed integer value.
	Uint64     uint64     // The unsigned integer value.
	Float64    float64    // The floating-point value.
	Complex128 complex128 // The complex value.
	Text       string     // The original textual representation from the input.
}

func newNumber(text string, typ itemType) (*NumberNode, error) {
	n := &NumberNode{NodeType: NodeNumber, Text: text}
	switch typ {
	case itemCharConstant:
		rune, _, tail, err := strconv.UnquoteChar(text[1:], text[0])
		if err != nil {
			return nil, err
		}
		if tail != "'" {
			return nil, fmt.Errorf("malformed character constant: %s", text)
		}
		n.Int64 = int64(rune)
		n.IsInt = true
		n.Uint64 = uint64(rune)
		n.IsUint = true
		n.Float64 = float64(rune) // odd but those are the rules.
		n.IsFloat = true
		return n, nil
	case itemComplex:
		// fmt.Sscan can parse the pair, so let it do the work.
		if _, err := fmt.Sscan(text, &n.Complex128); err != nil {
			return nil, err
		}
		n.IsComplex = true
		n.simplifyComplex()
		return n, nil
	}
	// Imaginary constants can only be complex unless they are zero.
	if len(text) > 0 && text[len(text)-1] == 'i' {
		f, err := strconv.ParseFloat(text[:len(text)-1], 64)
		if err == nil {
			n.IsComplex = true
			n.Complex128 = complex(0, f)
			n.simplifyComplex()
			return n, nil
		}
	}
	// Do integer test first so we get 0x123 etc.
	u, err := strconv.ParseUint(text, 0, 64) // will fail for -0; fixed below.
	if err == nil {
		n.IsUint = true
		n.Uint64 = u
	}
	i, err := strconv.ParseInt(text, 0, 64)
	if err == nil {
		n.IsInt = true
		n.Int64 = i
		if i == 0 {
			n.IsUint = true // in case of -0.
			n.Uint64 = u
		}
	}
	// If an integer extraction succeeded, promote the float.
	if n.IsInt {
		n.IsFloat = true
		n.Float64 = float64(n.Int64)
	} else if n.IsUint {
		n.IsFloat = true
		n.Float64 = float64(n.Uint64)
	} else {
		f, err := strconv.ParseFloat(text, 64)
		if err == nil {
			n.IsFloat = true
			n.Float64 = f
			// If a floating-point extraction succeeded, extract the int if needed.
			if !n.IsInt && float64(int64(f)) == f {
				n.IsInt = true
				n.Int64 = int64(f)
			}
			if !n.IsUint && float64(uint64(f)) == f {
				n.IsUint = true
				n.Uint64 = uint64(f)
			}
		}
	}
	if !n.IsInt && !n.IsUint && !n.IsFloat {
		return nil, fmt.Errorf("illegal number syntax: %q", text)
	}
	return n, nil
}

// simplifyComplex pulls out any other types that are represented by the complex number.
// These all require that the imaginary part be zero.
func (n *NumberNode) simplifyComplex() {
	n.IsFloat = imag(n.Complex128) == 0
	if n.IsFloat {
		n.Float64 = real(n.Complex128)
		n.IsInt = float64(int64(n.Float64)) == n.Float64
		if n.IsInt {
			n.Int64 = int64(n.Float64)
		}
		n.IsUint = float64(uint64(n.Float64)) == n.Float64
		if n.IsUint {
			n.Uint64 = uint64(n.Float64)
		}
	}
}

func (n *NumberNode) String() string {
	return n.Text
}

func (n *NumberNode) Copy() Node {
	nn := new(NumberNode)
	*nn = *n // Easy, fast, correct.
	return nn
}

// StringNode holds a string constant. The value has been "unquoted".
type StringNode struct {
	NodeType
	Quoted string // The original text of the string, with quotes.
	Text   string // The string, after quote processing.
}

func newString(orig, text string) *StringNode {
	return &StringNode{NodeType: NodeString, Quoted: orig, Text: text}
}

func (s *StringNode) String() string {
	return s.Quoted
}

func (s *StringNode) Copy() Node {
	return newString(s.Quoted, s.Text)
}

// endNode represents an {{end}} action. It is represented by a nil pointer.
// It does not appear in the final parse tree.
type endNode bool

func newEnd() *endNode {
	return nil
}

func (e *endNode) Type() NodeType {
	return nodeEnd
}

func (e *endNode) String() string {
	return "{{end}}"
}

func (e *endNode) Copy() Node {
	return newEnd()
}

// elseNode represents an {{else}} action. Does not appear in the final tree.
type elseNode struct {
	NodeType
	Line int // The line number in the input.
}

func newElse(line int) *elseNode {
	return &elseNode{NodeType: nodeElse, Line: line}
}

func (e *elseNode) Type() NodeType {
	return nodeElse
}

func (e *elseNode) String() string {
	return "{{else}}"
}

func (e *elseNode) Copy() Node {
	return newElse(e.Line)
}

// BranchNode is the common representation of if, range, and with.
type BranchNode struct {
	NodeType
	Line     int       // The line number in the input.
	Pipe     *PipeNode // The pipeline to be evaluated.
	List     *ListNode // What to execute if the value is non-empty.
	ElseList *ListNode // What to execute if the value is empty (nil if absent).
}

func (b *BranchNode) String() string {
	name := ""
	switch b.NodeType {
	case NodeIf:
		name = "if"
	case NodeRange:
		name = "range"
	case NodeWith:
		name = "with"
	default:
		panic("unknown branch type")
	}
	if b.ElseList != nil {
		return fmt.Sprintf("{{%s %s}}%s{{else}}%s{{end}}", name, b.Pipe, b.List, b.ElseList)
	}
	return fmt.Sprintf("{{%s %s}}%s{{end}}", name, b.Pipe, b.List)
}

// IfNode represents an {{if}} action and its commands.
type IfNode struct {
	BranchNode
}

func newIf(line int, pipe *PipeNode, list, elseList *ListNode) *IfNode {
	return &IfNode{BranchNode{NodeType: NodeIf, Line: line, Pipe: pipe, List: list, ElseList: elseList}}
}

func (i *IfNode) Copy() Node {
	return newIf(i.Line, i.Pipe.CopyPipe(), i.List.CopyList(), i.ElseList.CopyList())
}

// RangeNode represents a {{range}} action and its commands.
type RangeNode struct {
	BranchNode
}

func newRange(line int, pipe *PipeNode, list, elseList *ListNode) *RangeNode {
	return &RangeNode{BranchNode{NodeType: NodeRange, Line: line, Pipe: pipe, List: list, ElseList: elseList}}
}

func (r *RangeNode) Copy() Node {
	return newRange(r.Line, r.Pipe.CopyPipe(), r.List.CopyList(), r.ElseList.CopyList())
}

// WithNode represents a {{with}} action and its commands.
type WithNode struct {
	BranchNode
}

func newWith(line int, pipe *PipeNode, list, elseList *ListNode) *WithNode {
	return &WithNode{BranchNode{NodeType: NodeWith, Line: line, Pipe: pipe, List: list, ElseList: elseList}}
}

func (w *WithNode) Copy() Node {
	return newWith(w.Line, w.Pipe.CopyPipe(), w.List.CopyList(), w.ElseList.CopyList())
}

// TemplateNode represents a {{template}} action.
type TemplateNode struct {
	NodeType
	Line int       // The line number in the input.
	Name string    // The name of the template (unquoted).
	Pipe *PipeNode // The command to evaluate as dot for the template.
}

func newTemplate(line int, name string, pipe *PipeNode) *TemplateNode {
	return &TemplateNode{NodeType: NodeTemplate, Line: line, Name: name, Pipe: pipe}
}

func (t *TemplateNode) String() string {
	if t.Pipe == nil {
		return fmt.Sprintf("{{template %q}}", t.Name)
	}
	return fmt.Sprintf("{{template %q %s}}", t.Name, t.Pipe)
}

func (t *TemplateNode) Copy() Node {
	return newTemplate(t.Line, t.Name, t.Pipe.CopyPipe())
}

// BlockNode represents a {{block}} action.
type BlockNode struct {
	NodeType
	Line int       // The line number in the input.
	Name string    // The name of the block (unquoted).
	Pipe *PipeNode // The command to evaluate as dot for the block.
	List *ListNode // Contents of the block.
}

func newBlock(line int, name string, pipe *PipeNode, list *ListNode) *BlockNode {
	return &BlockNode{NodeType: NodeBlock, Line: line, Name: name, Pipe: pipe, List: list}
}

func (b *BlockNode) String() string {
	if b.Pipe == nil {
		return fmt.Sprintf("{{block %q}}%s{{end}}", b.Name, b.List)
	}
	return fmt.Sprintf("{{block %q %s}}%s{{end}}", b.Name, b.Pipe, b.List)
}

func (b *BlockNode) Copy() Node {
	return newBlock(b.Line, b.Name, b.Pipe.CopyPipe(), b.List.CopyList())
}

// FillNode represents a {{fill}} action.
type FillNode struct {
	BlockNode
}

func newFill(line int, name string, pipe *PipeNode, list *ListNode) *FillNode {
	return &FillNode{BlockNode{NodeType: NodeFill, Line: line, Name: name, Pipe: pipe, List: list}}
}

func (f *FillNode) String() string {
	if f.Pipe == nil {
		return fmt.Sprintf("{{fill %q}}%s{{end}}", f.Name, f.List)
	}
	return fmt.Sprintf("{{fill %q %s}}%s{{end}}", f.Name, f.Pipe, f.List)
}

func (f *FillNode) Copy() Node {
	return newFill(f.Line, f.Name, f.Pipe.CopyPipe(), f.List.CopyList())
}

// validate walks a node list and verifies actions inside a {{fill}}.
// A fill node can only contain nodes that don't generate output, except
// block nodes or text nodes containing only whitespace. So we whitelist
// {{block}}, {{if}}, {{with}} and {{range}}. This is checked during parsing.
func (f *FillNode) validate(n *ListNode) {
	for _, v := range n.Nodes {
		if !f.isValid(v) {
			panic(fmt.Errorf("invalid action inside <fill>: %s", v))
		}
	}
}

func (f *FillNode) isValid(n Node) bool {
	switch n := n.(type) {
	case *BlockNode:
	case *ListNode:
		f.validate(n)
	case *IfNode:
		f.validate(n.List)
		f.validate(n.ElseList)
	case *WithNode:
		f.validate(n.List)
		f.validate(n.ElseList)
	case *RangeNode:
		f.validate(n.List)
		f.validate(n.ElseList)
	case *TextNode:
		return len(bytes.TrimSpace(n.Text)) == 0
	default:
		return false
	}
	return true
}

// DefineNode represents a {{define}} action.
type DefineNode struct {
	NodeType
	Line int       // The line number in the input.
	Name string    // The name of the template (unquoted).
	List *ListNode // Contents of the template.
}

func newDefine(line int, name string, list *ListNode) *DefineNode {
	return &DefineNode{NodeType: NodeDefine, Line: line, Name: name, List: list}
}

func (d *DefineNode) String() string {
	return fmt.Sprintf("{{define %q}}%s{{end}}", d.Name, d.List)
}

func (d *DefineNode) CopyDefine() *DefineNode {
	return newDefine(d.Line, d.Name, d.List.CopyList())
}

func (d *DefineNode) Copy() Node {
	return d.CopyDefine()
}

// Tree stores a collection of DefineNode's.
type Tree map[string]*DefineNode

func (t Tree) Type() NodeType {
	return NodeTree
}

// Add adds a node to the tree.
func (t Tree) Add(node *DefineNode) error {
	if _, ok := t[node.Name]; ok {
		return fmt.Errorf("template: duplicated template name %q", node.Name)
	}
	t[node.Name] = node
	return nil
}

// AddTree adds all nodes from the given tree to this tree.
func (t Tree) AddTree(t2 Tree) error {
	for _, n := range t2 {
		if err := t.Add(n); err != nil {
			return err
		}
	}
	return nil
}

// Strings returns a parseable representation of all templates in the tree.
func (t Tree) String() string {
	b := new(bytes.Buffer)
	for _, n := range t {
		fmt.Fprint(b, n)
	}
	return b.String()
}

// Copy returns a deep copy of the tree.
func (t Tree) Copy() Node {
	nt := Tree{}
	for k, v := range t {
		nt[k]= v.CopyDefine()
	}
	return nt
}

// CopyShallow returns a shallow copy of the tree.
func (t Tree) CopyShallow() Tree {
	nt := Tree{}
	for k, v := range t {
		nt[k]= v
	}
	return nt
}

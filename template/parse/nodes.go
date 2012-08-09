package parse

import (
	"bytes"
	"fmt"
	"strconv"
	"sync/atomic"
)

// NodeType identifies the type of a parse tree node.
type NodeType int

// Type returns itself.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeBool NodeType = iota //
	NodeFloat                //
	NodeInt                  //
	NodeNil	                 //
	NodeString               //
	NodeList                 //
	NodePipe                 //
	NodeTemplate             // {template} tag
	NodeText                 // plain text
	NodeEnd                  // {end} tag
)

var nodeTypeIndex = int32(NodeEnd)

// RegisterNodeType registers a NodeType for custom nodes.
//
//     var NodeFoo = parse.RegisterNodeType()
func RegisterNodeType() NodeType {
	atomic.AddInt32(&nodeTypeIndex, 1)
	return NodeType(nodeTypeIndex)
}

// A node is an element in the parse tree.
type Node interface {
	Type() NodeType
	String() string
	Position() (int, int)
	// Copy does a deep copy of the Node and all its components.
	// To avoid type assertions, some XxxNodes also have specialized
	// CopyXxx methods that return *XxxNode.
	Copy() Node
}

// BaseNode is embedded in all node types.
type BaseNode struct {
	NodeType
	Line   int
	Column int
}

// Position returns the line and column where the node started in the input.
func (n BaseNode) Position() (int, int) {
	return n.Line, n.Column
}

// ----------------------------------------------------------------------------

// ListNode holds a sequence of nodes.
type ListNode struct {
	BaseNode
	Nodes []Node // The element nodes in lexical order.
}

func NewList(line, column int) *ListNode {
	return &ListNode{
		BaseNode: BaseNode{NodeList, line, column},
	}
}

func (n *ListNode) append(node Node) {
	n.Nodes = append(n.Nodes, node)
}

func (n *ListNode) String() string {
	b := new(bytes.Buffer)
	for _, node := range n.Nodes {
		fmt.Fprint(b, node)
	}
	return b.String()
}

func (n *ListNode) CopyList() *ListNode {
	if n == nil {
		return n
	}
	nn := NewList(n.Line, n.Column)
	for _, elem := range n.Nodes {
		nn.append(elem.Copy())
	}
	return nn
}

func (n *ListNode) Copy() Node {
	return n.CopyList()
}

// ----------------------------------------------------------------------------

func NewTemplate(line, column int, namespace, name string, list *ListNode) *TemplateNode {
	return &TemplateNode{
		BaseNode:  BaseNode{NodeTemplate, line, column},
		Namespace: namespace,
		Name:      name,
		List:      list,
	}
}

// TemplateNode holds the contents of a {template} tag.
type TemplateNode struct {
	BaseNode
	Namespace string
	Name      string
	List      *ListNode
}

func (n *TemplateNode) String() string {
	return fmt.Sprint("{template %q}%s{end}", n.Name, n.List)
}

func (n *TemplateNode) CopyTemplate() *ListNode {
	return NewTemplate(n.Line, n.Column, n.Namespace, n,Name, n.List.CopyList())
}

func (n *TemplateNode) Copy() Node {
	return n.CopyTemplate()
}

// ----------------------------------------------------------------------------

func NewBool(line, column int, text string) *BoolNode {
	return &BoolNode{
		BaseNode: BaseNode{NodeBool, line, column},
		True:     text == "true",
	}
}

// BoolNode holds a boolean constant.
type BoolNode struct {
	BaseNode
	True bool
}

func (n *BoolNode) String() string {
	if n.True {
		return "true"
	}
	return "false"
}

func (n *BoolNode) Copy() Node {
	nn := *n
	return &nn
}

// ----------------------------------------------------------------------------

func NewFloat(line, column int, text string) (*FloatNode, error) {
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, err
	}
	return &FloatNode{
		BaseNode: BaseNode{NodeFloat, line, column},
		Float:    value,
		Text:     text,
	}, nil
}

// FloatNode holds a float constant.
type FloatNode struct {
	BaseNode
	Float float64 // The floating-point value.
	Text  string  // The original textual representation from the input.
}

func (n *FloatNode) String() string {
	return n.Text
}

func (n *FloatNode) Copy() Node {
	nn := *n
	return &nn
}

// ----------------------------------------------------------------------------

func NewInt(line, column int, text string) (*IntNode, error) {
	value, err := strconv.ParseInt(text, 0, 64)
	if err != nil {
		return nil, err
	}
	return &IntNode{
		BaseNode: BaseNode{NodeInt, line, column},
		Int:      value,
		Text:     text,
	}, nil
}

// IntNode holds an int constant.
type IntNode struct {
	BaseNode
	Int  int64  // The signed integer value.
	Text string // The original textual representation from the input.
}

func (n *IntNode) String() string {
	return n.Text
}

func (n *IntNode) Copy() Node {
	nn := *n
	return &nn
}

// ----------------------------------------------------------------------------

func NewNil(line, column int) *NilNode {
	return &NilNode{
		BaseNode: BaseNode{NodeNil, line, column},
	}
}

// NilNode holds a nil constant.
type NilNode struct {
	BaseNode
}

func (n *NilNode) String() string {
	return "nil"
}

func (n *NilNode) Copy() Node {
	return NewNil(n.Line, n.Column)
}

// ----------------------------------------------------------------------------

func NewString(line, column int, orig, text string) *StringNode {
	return &StringNode{
		BaseNode: BaseNode{NodeString, line, column},
		Quoted: orig,
		Text: text,
	}
}

// StringNode holds a string constant.
type StringNode struct {
	BaseNode
	Quoted string // The original text of the string, with quotes.
	Text   string // The string, after quote processing.
}

func (n *StringNode) String() string {
	return n.Quoted
}

func (n *StringNode) Copy() Node {
	return NewString(n.Line, n.Column, n.Quoted, n.Text)
}

// ----------------------------------------------------------------------------

// TODO
func NewPipe(line, column int) *PipeNode {
	return &PipeNode{
		BaseNode: BaseNode{NodePipe, line, column},
	}
}

type PipeNode struct {
	BaseNode
}

func (n *PipeNode) String() string {
	/// TODO
	return ""
}

func (n *PipeNode) Copy() Node {
	return n.CopyPipe()
}

func (n *PipeNode) CopyPipe() *PipeNode {
	return NewPipe(n.Line, n.Column)
}

// ----------------------------------------------------------------------------

func NewText(line, column int, text []byte) *TextNode {
	return &TextNode{
		BaseNode: BaseNode{NodeText, line, column},
		Text:     text,
	}
}

// TextNode holds plain text.
type TextNode struct {
	BaseNode
	Text []byte // The text; may span newlines.
}

func (n *TextNode) String() string {
	return fmt.Sprintf("%q", n.Text)
}

func (n *TextNode) Copy() Node {
	return NewText(n.Line, n.Column, append([]byte{}, n.Text...))
}

// ----------------------------------------------------------------------------

// EndNode represents an {end} tag. It does not appear in the final parse tree.
type EndNode struct {
	BaseNode
}

func NewEnd(line, column int) *EndNode {
	return &EndNode{
		BaseNode: BaseNode{NodeEnd, line, column},
	}
}

func (n *EndNode) String() string {
	return "{end}"
}

func (n *EndNode) Copy() Node {
	return NewEnd(n.Line, n.Column)
}

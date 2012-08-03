// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pluralforms

import (
	"fmt"
	"strconv"
)

// Operator precedence levels.
var precedence = map[tokenType]int{
	tokenNot:   6,
	tokenMul:   5,
	tokenDiv:   5,
	tokenMod:   5,
	tokenAdd:   4,
	tokenSub:   4,
	tokenEq:    3,
	tokenNotEq: 3,
	tokenGt:    3,
	tokenGte:   3,
	tokenLt:    3,
	tokenLte:   3,
	tokenOr:    2,
	tokenAnd:   1,
}

// Map of operators that are right-associative. We don't have any. :P
var rightAssociativity = map[tokenType]bool{}

// ----------------------------------------------------------------------------

// parse parses an expression and returns a parse tree.
func parse(expr string) (node, error) {
	p := &parser{stream: newTokenStream(expr)}
	return p.parse()
}

// parser parses basic arithmetic expressions and returns a parse tree.
//
// It uses the recursive descent "precedence climbing" algorithm from:
//
//     http://www.engr.mun.ca/~theo/Misc/exp_parsing.htm
type parser struct {
	stream *tokenStream
}

// expect consumes the next token if it matches the given type, or returns
// an error.
func (p *parser) expect(t tokenType) {
	next := p.stream.pop()
	if next.typ != t {
		panic(fmt.Sprintf("Expected token %q, got %q", t, next.typ))
	}
}

// parse consumes the token stream and returns a parse tree.
func (p *parser) parse() (n node, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	n = p.parseExpression(0)
	p.expect(tokenEOF)
	return n, err
}

// parseExpression parses and returns an expression node.
func (p *parser) parseExpression(prec int) node {
	n := p.parsePrimary()
	var t token
	for {
		t = p.stream.pop()
		q := precedence[t.typ]
		if !isBinaryOp(t) || q < prec {
			break
		}
		if !rightAssociativity[t.typ] {
			q += 1
		}
		n = newBinaryOpNode(t, n, p.parseExpression(q))
	}
	p.stream.push(t)
	if prec == 0 && t.typ == tokenIf {
		return p.parseTernary(n)
	}
	return n
}

// parsePrimary parses and returns a primary node.
func (p *parser) parsePrimary() node {
	t := p.stream.pop()
	if isUnaryOp(t) {
		return newUnaryOpNode(t, p.parseExpression(precedence[t.typ]))
	} else if t.typ == tokenLeftParen {
		n := p.parseExpression(0)
		p.expect(tokenRightParen)
		return n
	} else if isValue(t) {
		return newValueNode(t)
	}
	panic(fmt.Sprintf("Unexpected token %q", t))
}

// parseTernary parses and returns a ternary operator node.
func (p *parser) parseTernary(n node) node {
	var t token
	for {
		if t = p.stream.pop(); t.typ != tokenIf {
			break
		}
		n1 := p.parseExpression(0)
		p.expect(tokenElse)
		n2 := p.parseExpression(0)
		n = &ifNode{n, n1, n2}
	}
	p.stream.push(t)
	return n
}

// ----------------------------------------------------------------------------

// isBinaryOp returns true if the given token is a binary operator.
func isBinaryOp(t token) bool {
	switch t.typ {
	case tokenMul, tokenDiv, tokenMod,
		tokenAdd, tokenSub,
		tokenEq, tokenNotEq, tokenGt, tokenGte, tokenLt, tokenLte,
		tokenOr, tokenAnd:
		return true
	}
	return false
}

// isUnaryOp returns true if the given token is a unary operator.
func isUnaryOp(t token) bool {
	switch t.typ {
	case tokenNot:
		return true
	}
	return false
}

// isValue returns true if the given token is a literal or variable.
func isValue(t token) bool {
	switch t.typ {
	case tokenBool, tokenInt, tokenVar:
		return true
	}
	return false
}

// ----------------------------------------------------------------------------

// newBinaryOpNode returns a tree for the given binary operator
// and child nodes.
func newBinaryOpNode(t token, n1, n2 node) node {
	switch t.typ {
	case tokenMul:
		return &mulNode{n1: n1, n2: n2}
	case tokenDiv:
		return &divNode{n1: n1, n2: n2}
	case tokenMod:
		return &modNode{n1: n1, n2: n2}
	case tokenAdd:
		return &addNode{n1: n1, n2: n2}
	case tokenSub:
		return &subNode{n1: n1, n2: n2}
	case tokenEq:
		return &eqNode{n1: n1, n2: n2}
	case tokenNotEq:
		return &notEqNode{n1: n1, n2: n2}
	case tokenGt:
		return &gtNode{n1: n1, n2: n2}
	case tokenGte:
		return &gteNode{n1: n1, n2: n2}
	case tokenLt:
		return &ltNode{n1: n1, n2: n2}
	case tokenLte:
		return &lteNode{n1: n1, n2: n2}
	case tokenOr:
		return &orNode{n1: n1, n2: n2}
	case tokenAnd:
		return &andNode{n1: n1, n2: n2}
	}
	panic("unreachable")

}

// newUnaryOpNode returns a tree for the given unary operator and child node.
func newUnaryOpNode(t token, n1 node) node {
	switch t.typ {
	case tokenNot:
		return &notNode{n1: n1}
	}
	panic("unreachable")
}

// newValueNode returns a node for the given literal or variable.
func newValueNode(t token) node {
	switch t.typ {
	case tokenBool:
		return boolNode(false)
	case tokenInt:
		if value, err := strconv.ParseInt(t.val, 10, 0); err == nil {
			return intNode(value)
		}
		return invalidExpression
	case tokenVar:
		return varNode(0)
	}
	panic("unreachable")
}

// ----------------------------------------------------------------------------

type node interface {
	Eval(ctx int) node
	String() string
}

// ----------------------------------------------------------------------------

var invalidExpression = errorNode("Invalid expression")

type errorNode string

func (n errorNode) Eval(ctx int) node {
	return n
}

func (n errorNode) String() string {
	return string(n)
}

// ----------------------------------------------------------------------------

type boolNode bool

func (n boolNode) Eval(ctx int) node {
	return n
}

func (n boolNode) String() string {
	return fmt.Sprintf("%v", bool(n))
}

// ----------------------------------------------------------------------------

type intNode int

func (n intNode) Eval(ctx int) node {
	return n
}

func (n intNode) String() string {
	return fmt.Sprintf("%v", int(n))
}

// ----------------------------------------------------------------------------

type varNode int

func (n varNode) Eval(ctx int) node {
	return intNode(ctx)
}

func (n varNode) String() string {
	return "n"
}

// ----------------------------------------------------------------------------

type notNode struct {
	n1 node
}

func (n *notNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(boolNode); ok {
		return !x
	}
	return invalidExpression
}

func (n *notNode) String() string {
	return fmt.Sprintf("(!%s)", n.n1)
}

// ----------------------------------------------------------------------------

type mulNode struct {
	n1 node
	n2 node
}

func (n *mulNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(intNode); ok {
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return x * y
		}
	}
	return invalidExpression
}

func (n *mulNode) String() string {
	return fmt.Sprintf("(%s*%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type divNode struct {
	n1 node
	n2 node
}

func (n *divNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(intNode); ok {
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return x / y
		}
	}
	return invalidExpression
}

func (n *divNode) String() string {
	return fmt.Sprintf("(%s/%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type modNode struct {
	n1 node
	n2 node
}

func (n *modNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(intNode); ok {
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return x % y
		}
	}
	return invalidExpression
}

func (n *modNode) String() string {
	return fmt.Sprintf("(%s%%%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type addNode struct {
	n1 node
	n2 node
}

func (n *addNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(intNode); ok {
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return x + y
		}
	}
	return invalidExpression
}

func (n *addNode) String() string {
	return fmt.Sprintf("(%s+%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type subNode struct {
	n1 node
	n2 node
}

func (n *subNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(intNode); ok {
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return x - y
		}
	}
	return invalidExpression
}

func (n *subNode) String() string {
	return fmt.Sprintf("(%s-%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type eqNode struct {
	n1 node
	n2 node
}

func (n *eqNode) Eval(ctx int) node {
	switch x := n.n1.Eval(ctx).(type) {
	case boolNode:
		if y, ok := n.n2.Eval(ctx).(boolNode); ok {
			return boolNode(x == y)
		}
	case intNode:
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return boolNode(x == y)
		}
	}
	return invalidExpression
}

func (n *eqNode) String() string {
	return fmt.Sprintf("(%s==%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type notEqNode struct {
	n1 node
	n2 node
}

func (n *notEqNode) Eval(ctx int) node {
	switch x := n.n1.Eval(ctx).(type) {
	case boolNode:
		if y, ok := n.n2.Eval(ctx).(boolNode); ok {
			return boolNode(x != y)
		}
	case intNode:
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return boolNode(x != y)
		}
	}
	return invalidExpression
}

func (n *notEqNode) String() string {
	return fmt.Sprintf("(%s!=%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type gtNode struct {
	n1 node
	n2 node
}

func (n *gtNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(intNode); ok {
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return boolNode(x > y)
		}
	}
	return invalidExpression
}

func (n *gtNode) String() string {
	return fmt.Sprintf("(%s>%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type gteNode struct {
	n1 node
	n2 node
}

func (n *gteNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(intNode); ok {
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return boolNode(x >= y)
		}
	}
	return invalidExpression
}

func (n *gteNode) String() string {
	return fmt.Sprintf("(%s>=%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type ltNode struct {
	n1 node
	n2 node
}

func (n *ltNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(intNode); ok {
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return boolNode(x < y)
		}
	}
	return invalidExpression
}

func (n *ltNode) String() string {
	return fmt.Sprintf("(%s<%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type lteNode struct {
	n1 node
	n2 node
}

func (n *lteNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(intNode); ok {
		if y, ok := n.n2.Eval(ctx).(intNode); ok {
			return boolNode(x <= y)
		}
	}
	return invalidExpression
}

func (n *lteNode) String() string {
	return fmt.Sprintf("(%s<=%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type orNode struct {
	n1 node
	n2 node
}

func (n *orNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(boolNode); ok {
		if y, ok := n.n2.Eval(ctx).(boolNode); ok {
			return boolNode(x || y)
		}
	}
	return invalidExpression
}

func (n *orNode) String() string {
	return fmt.Sprintf("(%s||%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type andNode struct {
	n1 node
	n2 node
}

func (n *andNode) Eval(ctx int) node {
	if x, ok := n.n1.Eval(ctx).(boolNode); ok {
		if y, ok := n.n2.Eval(ctx).(boolNode); ok {
			return boolNode(x && y)
		}
	}
	return invalidExpression
}

func (n *andNode) String() string {
	return fmt.Sprintf("(%s&&%s)", n.n1, n.n2)
}

// ----------------------------------------------------------------------------

type ifNode struct {
	cond node
	n1   node
	n2   node
}

func (n *ifNode) Eval(ctx int) node {
	if x, ok := n.cond.Eval(ctx).(boolNode); ok {
		if x {
			return n.n1.Eval(ctx)
		} else {
			return n.n2.Eval(ctx)
		}
	}
	return invalidExpression
}

func (n *ifNode) String() string {
	return fmt.Sprintf("(%s?%s:%s)", n.cond, n.n1, n.n2)
}

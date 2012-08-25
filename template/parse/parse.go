// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse builds parse trees for templates as defined by text/template
// and html/template. Clients should use those packages to construct templates
// rather than this one, which provides shared internal data structures not
// intended for general use.
package parse

import (
	"fmt"
	"runtime"
	"strconv"
	"unicode"
)

// Parse parses a string and returns a SetNode with the parsed templates.
func Parse(text, name, leftDelim, rightDelim string, funcs ...map[string]interface{}) (Tree, error) {
	p := &parser{
		name:  name,
		tree:  Tree{},
		funcs: funcs,
		vars:  []string{"$"},
	}
	return p.parse("", text, leftDelim, rightDelim)
}

type parser struct {
	name      string // used for debugging only.
	tree      Tree
	funcs     []map[string]interface{}
	lex       *lexer
	token     [2]item // two-token lookahead for parser.
	peekCount int
	vars      []string // variables defined at the moment.
}

// next returns the next token.
func (p *parser) next() item {
	if p.peekCount > 0 {
		p.peekCount--
	} else {
		p.token[0] = p.lex.nextItem()
	}
	return p.token[p.peekCount]
}

// backup backs the input stream up one token.
func (p *parser) backup() {
	p.peekCount++
}

// backup2 backs the input stream up two tokens
func (p *parser) backup2(t1 item) {
	p.token[1] = t1
	p.peekCount = 2
}

// peek returns but does not consume the next token.
func (p *parser) peek() item {
	if p.peekCount > 0 {
		return p.token[p.peekCount-1]
	}
	p.peekCount = 1
	p.token[0] = p.lex.nextItem()
	return p.token[0]
}

// Parsing.

// errorf formats the error and terminates processing.
func (p *parser) errorf(format string, args ...interface{}) {
	format = fmt.Sprintf("template: %s:%d: %s", p.name, p.lex.lineNumber(), format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (p *parser) error(err error) {
	p.errorf("%s", err)
}

// expect consumes the next token and guarantees it has the required type.
func (p *parser) expect(expected itemType, context string) item {
	token := p.next()
	if token.typ != expected {
		p.errorf("expected %s in %s; got %s", expected, context, token)
	}
	return token
}

// expectOneOf consumes the next token and guarantees it has one of the required types.
func (p *parser) expectOneOf(expected1, expected2 itemType, context string) item {
	token := p.next()
	if token.typ != expected1 && token.typ != expected2 {
		p.errorf("expected %s or %s in %s; got %s", expected1, expected2, context, token)
	}
	return token
}

// unexpected complains about the token and terminates processing.
func (p *parser) unexpected(token item, context string) {
	p.errorf("unexpected %s in %s", token, context)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (p *parser) recover(errp *error) {
	if e := recover(); e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		*errp = e.(error)
	}
}

// atEOF returns true if, possibly after spaces, we're at EOF.
func (p *parser) atEOF() bool {
	for {
		token := p.peek()
		switch token.typ {
		case itemEOF:
			return true
		case itemText:
			for _, r := range token.val {
				if !unicode.IsSpace(r) {
					return false
				}
			}
			p.next() // skip spaces.
			continue
		}
		break
	}
	return false
}

// parse is the top-level parser for a template: it only parses {{define}}
// actions. It runs to EOF.
func (p *parser) parse(name, text, leftDelim, rightDelim string) (tree Tree, err error) {
	defer p.recover(&err)
	p.lex = lex(name, text, leftDelim, rightDelim)
	for {
		switch p.next().typ {
		case itemEOF:
			return p.tree, nil
		case itemLeftDelim:
			p.expect(itemDefine, "template root")
			if err := p.tree.Add(p.parseDefinition()); err != nil {
				p.error(err)
			}
		}
	}
	return p.tree, nil
}

// parseDefinition parses a {{define}} ...  {{end}} template definition and
// installs the definition in the treeSet map.  The "define" keyword has already
// been scanned.
func (p *parser) parseDefinition() *DefineNode {
	const context = "define clause"
	defer p.popVars(1)
	line := p.lex.lineNumber()
	token := p.expectOneOf(itemString, itemRawString, context)
	name, err := strconv.Unquote(token.val)
	if err != nil {
		p.error(err)
	}
	p.expect(itemRightDelim, context)
	list, end := p.itemList()
	if end.Type() != nodeEnd {
		p.errorf("unexpected %s in %s", end, context)
	}
	return newDefine(line, name, list)
}

// itemList:
//	textOrAction*
// Terminates at {{end}} or {{else}}, returned separately.
func (p *parser) itemList() (list *ListNode, next Node) {
	list = newList()
	for p.peek().typ != itemEOF {
		n := p.textOrAction()
		switch n.Type() {
		case nodeEnd, nodeElse:
			return list, n
		}
		list.append(n)
	}
	p.errorf("unexpected EOF")
	return
}

// textOrAction:
//	text | action
func (p *parser) textOrAction() Node {
	switch token := p.next(); token.typ {
	case itemText:
		return newText(token.val)
	case itemLeftDelim:
		return p.action()
	default:
		p.unexpected(token, "input")
	}
	return nil
}

// Action:
//	control
//	command ("|" command)*
// Left delim is past. Now get actions.
// First word could be a keyword such as range.
func (p *parser) action() (n Node) {
	switch token := p.next(); token.typ {
	case itemElse:
		return p.elseControl()
	case itemEnd:
		return p.endControl()
	case itemIf:
		return p.ifControl()
	case itemRange:
		return p.rangeControl()
	case itemTemplate:
		return p.templateControl()
	case itemWith:
		return p.withControl()
	case itemBlock:
		return p.blockControl()
	case itemFill:
		return p.fillControl()
	}
	p.backup()
	// Do not pop variables; they persist until "end".
	return newAction(p.lex.lineNumber(), p.pipeline("command"))
}

// Pipeline:
//	field or command
//	pipeline "|" pipeline
func (p *parser) pipeline(context string) (pipe *PipeNode) {
	var decl []*VariableNode
	// Are there declarations?
	for {
		if v := p.peek(); v.typ == itemVariable {
			p.next()
			if next := p.peek(); next.typ == itemColonEquals || (next.typ == itemChar && next.val == ",") {
				p.next()
				variable := newVariable(v.val)
				if len(variable.Ident) != 1 {
					p.errorf("illegal variable in declaration: %s", v.val)
				}
				decl = append(decl, variable)
				p.vars = append(p.vars, v.val)
				if next.typ == itemChar && next.val == "," {
					if context == "range" && len(decl) < 2 {
						continue
					}
					p.errorf("too many declarations in %s", context)
				}
			} else {
				p.backup2(v)
			}
		}
		break
	}
	pipe = newPipeline(p.lex.lineNumber(), decl)
	for {
		switch token := p.next(); token.typ {
		case itemRightDelim:
			if len(pipe.Cmds) == 0 {
				p.errorf("missing value for %s", context)
			}
			return
		case itemBool, itemCharConstant, itemComplex, itemDot, itemField, itemIdentifier,
			itemNumber, itemNil, itemRawString, itemString, itemVariable:
			p.backup()
			pipe.append(p.command())
		default:
			p.unexpected(token, context)
		}
	}
	return
}

func (p *parser) parseControl(context string) (lineNum int, pipe *PipeNode, list, elseList *ListNode) {
	lineNum = p.lex.lineNumber()
	defer p.popVars(len(p.vars))
	pipe = p.pipeline(context)
	var next Node
	list, next = p.itemList()
	switch next.Type() {
	case nodeEnd: //done
	case nodeElse:
		elseList, next = p.itemList()
		if next.Type() != nodeEnd {
			p.errorf("expected end; found %s", next)
		}
		elseList = elseList
	}
	return lineNum, pipe, list, elseList
}

// If:
//	{{if pipeline}} itemList {{end}}
//	{{if pipeline}} itemList {{else}} itemList {{end}}
// If keyword is past.
func (p *parser) ifControl() Node {
	return newIf(p.parseControl("if"))
}

// Range:
//	{{range pipeline}} itemList {{end}}
//	{{range pipeline}} itemList {{else}} itemList {{end}}
// Range keyword is past.
func (p *parser) rangeControl() Node {
	return newRange(p.parseControl("range"))
}

// With:
//	{{with pipeline}} itemList {{end}}
//	{{with pipeline}} itemList {{else}} itemList {{end}}
// If keyword is past.
func (p *parser) withControl() Node {
	return newWith(p.parseControl("with"))
}

// End:
//	{{end}}
// End keyword is past.
func (p *parser) endControl() Node {
	p.expect(itemRightDelim, "end")
	return newEnd()
}

// Else:
//	{{else}}
// Else keyword is past.
func (p *parser) elseControl() Node {
	p.expect(itemRightDelim, "else")
	return newElse(p.lex.lineNumber())
}

// Template:
//	{{template stringValue pipeline}}
// Template keyword is past.  The name must be something that can evaluate
// to a string.
func (p *parser) templateControl() Node {
	var name string
	switch token := p.next(); token.typ {
	case itemString, itemRawString:
		s, err := strconv.Unquote(token.val)
		if err != nil {
			p.error(err)
		}
		name = s
	default:
		p.unexpected(token, "template invocation")
	}
	var pipe *PipeNode
	if p.next().typ != itemRightDelim {
		p.backup()
		// Do not pop variables; they persist until "end".
		pipe = p.pipeline("template")
	}
	return newTemplate(p.lex.lineNumber(), name, pipe)
}

// Block:
//	{{block stringValue pipeline}} itemList {{end}}
// Block keyword is past.
func (p *parser) blockControl() Node {
	return newBlock(p.blockOrFill("block"))
}

// Fill:
//	{{fill stringValue pipeline}} itemList {{end}}
// Fill keyword is past.
func (p *parser) fillControl() Node {
	n := newFill(p.blockOrFill("fill"))
	n.validate(n.List)
	return n
}

// blockOrFillControl parses a {{block}} or {{fill}}.
func (p *parser) blockOrFill(context string) (lineNum int, name string, pipe *PipeNode, list *ListNode) {
	lineNum = p.lex.lineNumber()
	token := p.expectOneOf(itemString, itemRawString, context)
	name, err := strconv.Unquote(token.val)
	if err != nil {
		p.error(err)
	}
	if p.next().typ != itemRightDelim {
		p.backup()
		pipe = p.pipeline(context)
	}
	list, end := p.itemList()
	if end.Type() != nodeEnd {
		p.errorf("expected <end> in %s", context)
	}
	return lineNum, name, pipe, list
}

// command:
// space-separated arguments up to a pipeline character or right delimiter.
// we consume the pipe character but leave the right delim to terminate the action.
func (p *parser) command() *CommandNode {
	cmd := newCommand()
Loop:
	for {
		switch token := p.next(); token.typ {
		case itemRightDelim:
			p.backup()
			break Loop
		case itemPipe:
			break Loop
		case itemError:
			p.errorf("%s", token.val)
		case itemIdentifier:
			if !p.hasFunction(token.val) {
				p.errorf("function %q not defined", token.val)
			}
			cmd.append(NewIdentifier(token.val))
		case itemDot:
			cmd.append(newDot())
		case itemNil:
			cmd.append(newNil())
		case itemVariable:
			cmd.append(p.useVar(token.val))
		case itemField:
			cmd.append(newField(token.val))
		case itemBool:
			cmd.append(newBool(token.val == "true"))
		case itemCharConstant, itemComplex, itemNumber:
			number, err := newNumber(token.val, token.typ)
			if err != nil {
				p.error(err)
			}
			cmd.append(number)
		case itemString, itemRawString:
			s, err := strconv.Unquote(token.val)
			if err != nil {
				p.error(err)
			}
			cmd.append(newString(token.val, s))
		default:
			p.unexpected(token, "command")
		}
	}
	if len(cmd.Args) == 0 {
		p.errorf("empty command")
	}
	return cmd
}

// hasFunction reports if a function name exists in the Tree's maps.
func (p *parser) hasFunction(name string) bool {
	for _, funcMap := range p.funcs {
		if funcMap == nil {
			continue
		}
		if funcMap[name] != nil {
			return true
		}
	}
	return false
}

// popVars trims the variable list to the specified length
func (p *parser) popVars(n int) {
	p.vars = p.vars[:n]
}

// useVar returns a node for a variable reference. It errors if the
// variable is not defined.
func (p *parser) useVar(name string) Node {
	v := newVariable(name)
	for _, varName := range p.vars {
		if varName == v.Ident[0] {
			return v
		}
	}
	p.errorf("undefined variable %q", v.Ident[0])
	return nil
}

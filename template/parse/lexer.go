// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"strings"
	"unicode/utf8"
)

const (
	eof = -1
)

// stateFn represents the state of the scanner; it returns the next state.
type stateFn func(*lexer) stateFn

// newLexer returns a new lexer for the given input.
func newLexer(input string) *lexer {
	input = strings.Replace(input, "\r\n", "\n", -1)
	return &lexer{
		input: input,
		state: lexFile,
		queue: newQueue(),
	}
}

// lexer scans an input and emits tokens.
type lexer struct {
	input string  // input being scanned
	pin   int     // pinned position in the input
	pos   int     // current position in the input
	width int     // width of last rune read from input
	state stateFn // the next lexing function to enter
	queue *queue  // queue of emitted tokens
}

// emit -----------------------------------------------------------------------

func (l *lexer) emit(typ tokenType) {
	l.queue.push(token{
		typ: typ,
		pos: l.position(l.pin),
		val: l.input[l.pin:l.pos],
	})
	l.pin = l.pos
}

func (l *lexer) emitValue(typ tokenType, pos int, value string) {
	l.queue.push(token{
		typ: typ,
		pos: l.position(pos),
		val: value,
	})
	l.pin = l.pos
}

func (l *lexer) skip() {
	l.pin = l.pos
}

// position returns a struct with line and column for a given input position.
func (l *lexer) position(pos int) position {
	if pos >= 0 && pos <= len(l.input) {
		row := 1 + strings.Count(l.input[:pos], "\n")
		col := 1 + pos
		if row > 1 {
			col -= 1 + strings.LastIndex(l.input[:pos], "\n")
		}
		return position{row, col}
	}
	return position{-1, -1}
}

// next -----------------------------------------------------------------------

// next returns the next emitted token.
func (l *lexer) next() token {
	for {
		if l.queue.count > 0 {
			return l.queue.pop()
		}
		if l.state == nil {
			l.emitValue(tokenError, -1, "state is nil")
			l.state = lexError
			continue
		}
		l.state = l.state(l)
	}
	panic("unreachable")
}

// nextRune returns the next rune from the input.
func (l *lexer) nextRune() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// backup steps back one rune. Can only be called once per call of nextRune.
func (l *lexer) backup() {
	l.pos -= l.width
}

// accept ---------------------------------------------------------------------

// accept consumes the input if it matches a rune.
func (l *lexer) accept(r rune) bool {
	if l.nextRune() == r {
		return true
	}
	l.backup()
	return false
}

// acceptOne consumes a rune if it matches a set of runes and returns
// the consumed rune.
func (l *lexer) acceptOne(valid string) rune {
	if i := strings.IndexRune(valid, l.nextRune()); i >= 0 {
		return rune(valid[i])
	}
	l.backup()
	return -1
}

// acceptMany consumes the input while it matches a set of runes and returns
// the consumed string.
func (l *lexer) acceptMany(valid string) string {
	pos := l.pos
	for strings.IndexRune(valid, l.nextRune()) >= 0 {
	}
	l.backup()
	return l.input[pos:l.pos]
}

// acceptPrefix consumes the input it matches a prefix.
func (l *lexer) acceptPrefix(prefix string) bool {
	if l.pos < len(l.input) && strings.HasPrefix(l.input[l.pos:], prefix) {
		l.width = utf8.RuneCountInString(prefix)
		l.pos += l.width
		return true
	}
	return false
}

// acceptUntilRune consumes the input until the given rune is found
// (inclusive) and returns the consumed string. The position doesn't change
// if there's no match.
func (l *lexer) acceptUntilRune(r rune) string {
	pos := l.pos
	for {
		if l.accept(r) {
			return l.input[pos:l.pos]
		}
		if l.nextRune() == eof {
			break
		}
	}
	l.pos = pos
	return ""
}

// acceptUntilPrefix consumes the input until the given prefix is found
// (inclusive) and returns the consumed string. The position doesn't change
// if there's no match.
func (l *lexer) acceptUntilPrefix(prefix string) string {
	if l.pos < len(l.input) {
		if i := strings.Index(l.input[l.pos:], prefix); i >= 0 {
			s := l.input[l.pos:l.pos+i]
			l.width = utf8.RuneCountInString(s)
			l.pos += l.width
			return s
		}
	}
	return ""
}

// test -----------------------------------------------------------------------

// ...
func (l *lexer) testOne(valid string) bool {
	ok := strings.IndexRune(valid, l.nextRune()) >= 0
	l.backup()
	return ok
}

// ----------------------------------------------------------------------------

// newQueue returns a new queue for tokens.
func newQueue() *queue {
	return &queue{tokens: make([]token, 10)}
}

// queue is a FIFO queue for tokens that resizes as needed.
//
// It is used by internally by the lexer.
type queue struct {
	tokens []token
	head   int
	tail   int
	count  int
}

// push adds a token to the queue.
func (q *queue) push(t token) {
	if q.head == q.tail && q.count > 0 {
		// Resize.
		tokens := make([]token, len(q.tokens)*2)
		copy(tokens, q.tokens[q.head:])
		copy(tokens[len(q.tokens)-q.head:], q.tokens[:q.head])
		q.head = 0
		q.tail = len(q.tokens)
		q.tokens = tokens
	}
	q.tokens[q.tail] = t
	q.tail = (q.tail + 1) % len(q.tokens)
	q.count++
}

// pop removes and returns a token from the queue in first to last order.
func (q *queue) pop() token {
	if q.count > 0 {
		t := q.tokens[q.head]
		q.head = (q.head + 1) % len(q.tokens)
		q.count--
		return t
	}
	panic("queue is empty")
}

// ----------------------------------------------------------------------------

// newStack returns a token stack for the given input.
func newStack(input string) *stack {
	return &stack{lexer: newLexer(input), tokens: make([]token, 10)}
}

// stack is a LIFO stack for tokens that resizes as needed.
//
// It is used by consumers of the lexer.
type stack struct {
	lexer  *lexer
	tokens []token
	count  int
}

// pop returns the next token from the stack.
func (s *stack) pop() token {
	if s.count == 0 {
		return s.lexer.next()
	}
	s.count--
	return s.tokens[s.count]
}

// push puts a token back to the stack.
func (s *stack) push(t token) {
	if s.count >= len(s.tokens) {
		// Resizes as needed.
		tokens := make([]token, len(s.tokens)*2)
		copy(tokens, s.tokens)
		s.tokens = tokens
	}
	s.tokens[s.count] = t
	s.count++
}

// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pluralforms

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	eof     = -1
	numbers = "0123456789"
	symbols = "*/%+-=!<>|&?:"
)

// tokenType is the type of lex tokens.
type tokenType int

func (i tokenType) String() string {
	if tokenName[i] == "" {
		return fmt.Sprintf("token%d", int(i))
	}
	return tokenName[i]
}

const (
	tokenError tokenType = iota
	tokenEOF
	tokenBool
	tokenInt
	tokenVar
	tokenMul        // *
	tokenDiv        // /
	tokenMod        // %
	tokenAdd        // +
	tokenSub        // - (binary)
	tokenEq         // ==
	tokenNotEq      // !=
	tokenGt         // >
	tokenGte        // >=
	tokenLt         // <
	tokenLte        // <=
	tokenOr         // ||
	tokenAnd        // &&
	tokenIf         // ?
	tokenElse       // :
	tokenNot        // !
	tokenLeftParen  // (
	tokenRightParen // )
)

// Make the types prettyprint.
var tokenName = map[tokenType]string{
	tokenError:      "error",
	tokenEOF:        "EOF",
	tokenBool:       "bool",
	tokenInt:        "int",
	tokenVar:        "var",
	tokenMul:        "*",
	tokenDiv:        "/",
	tokenMod:        "%",
	tokenAdd:        "+",
	tokenSub:        "-",
	tokenEq:         "==",
	tokenNotEq:      "!=",
	tokenGt:         ">",
	tokenGte:        ">=",
	tokenLt:         "<",
	tokenLte:        "<=",
	tokenOr:         "||",
	tokenAnd:        "&&",
	tokenIf:         "?",
	tokenElse:       ":",
	tokenNot:        "!",
	tokenLeftParen:  "(",
	tokenRightParen: ")",
}

// Used to get the corrsponding token for a string.
var stringToToken = map[string]tokenType{
	"*":  tokenMul,
	"/":  tokenDiv,
	"%":  tokenMod,
	"+":  tokenAdd,
	"-":  tokenSub,
	"==": tokenEq,
	"!=": tokenNotEq,
	">":  tokenGt,
	">=": tokenGte,
	"<":  tokenLt,
	"<=": tokenLte,
	"||": tokenOr,
	"&&": tokenAnd,
	"?":  tokenIf,
	":":  tokenElse,
	"!":  tokenNot,
	"(":  tokenLeftParen,
	")":  tokenRightParen,
}

// ----------------------------------------------------------------------------

// token is a token returned from the lexer.
type token struct {
	typ tokenType
	val string
}

func (t token) String() string {
	if t.val != "" {
		return fmt.Sprintf("<%s:%s>", t.typ, t.val)
	}
	return fmt.Sprintf("<%s>", t.typ)
}

// ----------------------------------------------------------------------------

// newLexer returns a lexer for the given expression.
func newLexer(expr string) *lexer {
	return &lexer{input: expr}
}

// lexer scans an expression and returns tokens.
//
// Very simplified version of the lexer from text/template.
type lexer struct {
	input string // expression being scanned
	pos   int    // current position in the input
	width int    // width of last rune read from input
}

// next returns the next token from the input.
func (l *lexer) next() token {
	for {
		r := l.nextRune()
		switch r {
		case ' ':
			// ignore spaces.
		case eof:
			return token{typ: tokenEOF}
		case 'n':
			return token{typ: tokenVar}
		case '*', '/', '%', '+', '-', '?', ':', '(', ')':
			return token{typ: stringToToken[string(r)]}
		default:
			l.backup()
			if s := l.nextRun(numbers); s != "" {
				return token{typ: tokenInt, val: s}
			}
			if s := l.nextRun(symbols); s != "" {
				if typ, ok := stringToToken[s]; ok {
					return token{typ: typ}
				}
			}
			return token{tokenError,
				fmt.Sprintf("Invalid character %s", string(r))}
		}
	}
	panic("unreachable")
}

// next returns the next rune from the input.
func (l *lexer) nextRune() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// nextRun returns a run of runes from the valid set.
func (l *lexer) nextRun(valid string) string {
	pos := l.pos
	for strings.IndexRune(valid, l.nextRune()) >= 0 {
	}
	l.backup()
	return l.input[pos:l.pos]
}

// backup steps back one rune. Can only be called once per call of nextRune.
func (l *lexer) backup() {
	l.pos -= l.width
}

// ----------------------------------------------------------------------------

// newTokenStream returns a token stream for the given expression.
func newTokenStream(expr string) *tokenStream {
	return &tokenStream{lexer: newLexer(expr), tokens: make([]token, 5)}
}

// tokenStream is a LIFO stack for tokens.
type tokenStream struct {
	lexer  *lexer
	tokens []token
	count  int
}

// pop returns the next token from the stream.
func (s *tokenStream) pop() token {
	if s.count == 0 {
		return s.lexer.next()
	}
	s.count--
	return s.tokens[s.count]
}

// push puts a token back to the stream.
func (s *tokenStream) push(t token) {
	if s.count >= len(s.tokens) {
		// Resizes as needed.
		tokens := make([]token, len(s.tokens)*2)
		copy(tokens, s.tokens)
		s.tokens = tokens
	}
	s.tokens[s.count] = t
	s.count++
}

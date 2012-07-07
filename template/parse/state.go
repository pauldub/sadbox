// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

var tags = map[string]tokenType{
	"else":      tokenElse,
	"end":       tokenEnd,
	"if":        tokenIf,
	"namespace": tokenNamespace,
	"range":     tokenRange,
	"raw":       tokenRaw,
	"sp":        tokenSp,
	"template":  tokenTemplate,
	"with":      tokenWith,
}

var symbols = map[string]tokenType{
	"(":  tokenLeftParen,
	")":  tokenRightParen,
	"*":  tokenMul,
	"/":  tokenDiv,
	"%":  tokenMod,
	"+":  tokenAdd,
	"-":  tokenSub,
	"?":  tokenQuestion,
	"=":  tokenAssign,
	"==": tokenEq,
	":":  tokenColon,
	":=": tokenColonAssign,
	"!":  tokenNot,
	"!=": tokenNotEq,
	">":  tokenGt,
	">=": tokenGte,
	"<":  tokenLt,
	"<=": tokenLte,
	"&&": tokenAnd,
	"||": tokenOr,
}

// state functions ------------------------------------------------------------

// lexFile is the initial state of the lexer and the one used when we are
// outside of any tags.
func lexFile(l *lexer) stateFn {
	// We are only looking for two things: slashes and left braces,
	// for possible comments or tags.
	for {
		switch l.nextRune() {
		case '/':
			return lexComment
		case '{':
			return lexTagStart
		case eof:
			emitText(l, 1)
			l.emitValue(tokenEOF, l.pos, "")
			return lexEOF
		}
	}
	panic("unreachable")
}

// lexEOF is the state after the input is consumed. It emits a tokenEOF and
// returns itself, so further calls to next() will always return EOF.
func lexEOF(l *lexer) stateFn {
	l.emitValue(tokenEOF, l.pos, "")
	return lexEOF
}

// lexError is the state after an error was found. It emits a tokenError and
// returns itself, so further calls to next() will always return error.
func lexError(l *lexer) stateFn {
	l.emitValue(tokenError, -1, "")
	return lexError
}

// lexComment lexes a possible single line or multi-line comment.
//
// Single line comments start with two slashes and go until the end of the
// line, but must be preceded by a whitespace. Multi-line comments start with
// slash-star and end with star-slash, and do not nest.
//
// A slash was already consumed when this is called.
func lexComment(l *lexer) stateFn {
	switch l.nextRune() {
	case '/':
		ok := l.pos-2 < 0
		if !ok {
			r, _ := utf8.DecodeLastRuneInString(l.input[:l.pos-2])
			ok = isSpace(r)
		}
		if ok {
			emitText(l, 2)
			l.acceptUntilRune('\n')
			l.emit(tokenComment)
		}
	case '*':
		emitText(l, 2)
		l.acceptUntilPrefix("*/")
		l.emit(tokenComment)
	}
	return lexFile
}

// lexTagStart scans the beginning of a tag.
//
// A left curly brace was already consumed when this is called.
func lexTagStart(l *lexer) stateFn {
	emitText(l, 1)
	l.emitValue(tokenLeftBrace, l.pos-2, "")
	l.skip()
	return lexTagContent
}

// lexTagEnd scans the end of a tag.
//
// A right curly brace was already consumed when this is called.
func lexTagEnd(l *lexer) stateFn {
	l.emitValue(tokenRightBrace, l.pos-2, "")
	l.skip()
	return lexFile
}

// lexTagContent scans the contents of a tag.
//
// A curly brace was already consumed and emitted when this is called.
func lexTagContent(l *lexer) stateFn {
	r := l.nextRune()
	switch r {
	case '}':
		return lexTagEnd
	case '[':
		return lexIndex
	case '"':
		return lexQuote
	case '.':
		return lexIdentifier
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		// We only support decimal numbers starting with a digit.
		// No floats starting with dot.
		l.backup()
		return lexNumber
	case ' ', '\t', '\r':
		l.acceptMany(" \t\r")
		l.skip()
	case '(', ')', '*', '/', '%', '+', '-', '?':
		l.emitValue(symbols[l.input[l.pin:l.pos]], l.pin, "")
	case '=', ':', '!', '<', '>':
		// =, ==
		// :, :=
		// !, !=
		// <, <=
		// >, >=
		// TODO: *=, /=, %=, +=, -=
		l.accept('=')
		l.emitValue(symbols[l.input[l.pin:l.pos]], l.pin, "")
	case '&', '|':
		// &&
		// ||
		if !l.accept(r) {
			return errorf(l, "expected %s%s", string(r), string(r))
		}
		l.emitValue(symbols[l.input[l.pin:l.pos]], l.pin, "")
	case eof, '\n':
		return errorf(l, "unclosed action")
	default:
		if isAlphaNumeric(r) {
			l.backup()
			return lexIdentifier
		}
		return errorf(l, "unrecognized character in tag: %#U", r)
	}
	return lexTagContent
}

func lexIdentifier(l *lexer) stateFn {
	return nil
}

// lexQuote scans a quoted string.
func lexQuote(l *lexer) stateFn {
Loop:
	for {
		switch l.nextRune() {
		case '\\':
			if r := l.nextRune(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return errorf(l, "unterminated quoted string")
		case '"':
			break Loop
		}
	}
	l.emit(tokenString)
	return lexTagContent
}

func lexNumber(l *lexer) stateFn {
	return nil
}

func lexIndex(l *lexer) stateFn {
	return errorf(l, "indexes are not supported yet")
}

// helpers --------------------------------------------------------------------

func errorf(l *lexer, message string, a ...interface{}) stateFn {
	l.emitValue(tokenError, l.pin, fmt.Sprintf(message, a...))
	return lexError
}

// emitText emits the text accumulated until current position minus offset,
// which is a known valid position.
func emitText(l *lexer, offset int) {
	pin := l.pos-offset
	if pin > l.pin {
		l.emitValue(tokenText, l.pin, l.input[l.pin:pin])
		l.pin = pin
	}
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

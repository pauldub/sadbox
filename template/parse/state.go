// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TODO
// - track "inside template" state
// - only emit single line comments preceding a template tag

const (
	sDecDigits  = "0123456789"
	sHexDigits  = sDecDigits + "ABCDEF"
	sAlphaLower = "abcdefghijklmnopqrstuvwxyz"
	sAlphaUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sAlphaNum   = sAlphaLower + sAlphaUpper + sDecDigits
	sWhitespace = " \t\n\r"
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
	"*":  tokenStar,
	"*=": tokenStarEq,
	"/":  tokenSlash,
	"/=": tokenSlashEq,
	"%":  tokenPercent,
	"%=": tokenPercentEq,
	"+":  tokenPlus,
	"+=": tokenPlusEq,
	"-":  tokenMinus,
	"-=": tokenMinusEq,
	"?":  tokenQuestion,
	"=":  tokenEq,
	"==": tokenEqEq,
	":":  tokenColon,
	":=": tokenColonEq,
	"!":  tokenNot,
	"!=": tokenNotEq,
	">":  tokenGt,
	">=": tokenGtEq,
	"<":  tokenLt,
	"<=": tokenLtEq,
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
			if !l.define {
				return lexComment
			}
		case '{':
			emitText(l, 1)
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
	}
	return lexFile
}

// lexTagStart scans the beginning of a tag.
//
// A left curly brace was already consumed when this is called.
func lexTagStart(l *lexer) stateFn {
	l.emitValue(tokenLeftBrace, l.pos-1, "")
	return lexTagContent
}

// lexTagEnd scans the end of a tag.
//
// A right curly brace was already consumed when this is called.
func lexTagEnd(l *lexer) stateFn {
	l.emitValue(tokenRightBrace, l.pos-1, "")
	return lexFile
}

// lexTagComment scans a comment tag: {# comment #}.
func lexTagComment(l *lexer) stateFn {
	i := strings.Index(l.input[l.pos:], "#}")
	if i < 0 {
		return errorf(l, "unclosed comment tag")
	}
	l.pos += i + 2
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
	case ' ', '\t', '\r':
		l.acceptMany(" \t\r")
		l.skip()
	case '(', ')', '?':
		l.emitValue(symbols[l.input[l.pin:l.pos]], l.pin, "")
	case '#':
		return lexTagComment
	case '*', '/', '%', '+', '-', '=', ':', '!', '<', '>':
		// *, *=
		// /, /=
		// %, %=
		// +, +=
		// -, -=
		// =, ==
		// :, :=
		// !, !=
		// <, <=
		// >, >=
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
		return errorf(l, "unclosed tag")
	default:
		if isDigit(r) {
			// We only support decimal/hexadecimal numbers starting with a digit.
			// No floats starting with dot. No unary operators included.
			l.backup()
			return lexNumber
		}
		if isAlphaNumeric(r) {
			l.backup()
			return lexIdentifier
		}
		return errorf(l, "unrecognized character in tag: %#U", r)
	}
	return lexTagContent
}

// lexIdentifier scans an alphanumeric or field.
func lexIdentifier(l *lexer) stateFn {
Loop:
	for {
		switch r := l.nextRune(); {
		case isAlphaNumeric(r):
			// absorb.
		case r == '.' && l.input[l.pin] == '.':
			// field chaining; absorb into one token.
		default:
			l.backup()
			word := l.input[l.pin:l.pos]
			switch {
			case word[0] == '.':
				l.emit(tokenVar)
			case word == "true", word == "false":
				l.emit(tokenBool)
			case word == "raw":
				return lexRaw
			default:
				l.emit(tokenIdent)
			}
			break Loop
		}
	}
	return lexTagContent
}

// lexQuote scans a quoted string.
func lexQuote(l *lexer) stateFn {
	if scanQuote(l) {
		l.emit(tokenString)
		return lexTagContent
	}
	return errorf(l, "unterminated quoted string")
}

// lexNumber scans a number: a float, decimal integer or hex integer.
func lexNumber(l *lexer) stateFn {
	typ, ok := scanNumber(l)
	if !ok {
		return errorf(l, "bad number syntax: %q", l.input[l.pin:l.pos])
	}
	// Emits tokenFloat or tokenInt.
	l.emit(typ)
	return lexTagContent
}

func lexIndex(l *lexer) stateFn {
	return errorf(l, "indexes are not supported yet")
}

func lexRaw(l *lexer) stateFn {
	pin := l.pin
	l.emitValue(tokenRaw, l.pin, "")
	stop := "{end}"
	if l.acceptMany(sWhitespace) {
		l.skip()
		if l.accept('"') {
			if scanQuote(l) {
				stop = l.input[l.pin+1:l.pos-1]
			} else {
				return errorf(l, "unterminated quoted string")
			}
		}
	}
	if !l.accept('}') {
		return errorf(l, "bad raw tag: %q", l.input[pin:l.pos])
	}
	l.skip()
	i := strings.Index(l.input[l.pos:], stop)
	if i < 0 {
		return errorf(l, "unterminated raw content: %q", l.input[pin:l.pos])
	}
	l.pos += i
	l.emit(tokenRawText)
	l.pos += len(stop)
	l.skip()
	return lexFile
}

// scanNumber scans a number.
//
// It returns a tokenFloat or tokenInt) and a flag indicating if an error
// was found.
//
// Floats must be in decimal and must either:
//
//     - Have digits both before and after the decimal point (both can be
//       a single 0), e.g. 0.5, 100.0, or
//     - Have a lower-case e that represents scientific notation,
//       e.g. 3e-3, 6.02e23.
//
// Integers can be:
//
//     - decimal (e.g. 827)
//     - hexadecimal (must begin with 0x and must use capital A-F,
//       e.g. 0x1A2B).
//
// Unary operator minus is not scanned here.
func scanNumber(l *lexer) (t tokenType, ok bool) {
	t = tokenInt
	if l.acceptPrefix("0x") {
		// Hexadecimal.
		if !l.acceptMany(sHexDigits) {
			// Requires at least one digit.
			return
		}
		if l.accept('.') {
			// No dots for hexadecimals.
			return
		}
	} else {
		// Decimal.
		if !l.acceptMany(sDecDigits) {
			// Requires at least one digit.
			return
		}
		if l.accept('.') {
			// Float.
			if !l.acceptMany(sDecDigits) {
				// Requires a digit after the dot.
				return
			}
			t = tokenFloat
		} else {
			// Integer.
			if l.input[l.pin] == '0' {
				// Integers can't start with 0.
				return
			}
		}
		if l.accept('e') {
			l.acceptOne("+-")
			if !l.acceptMany(sDecDigits) {
				// A digit is required after the scientific notation.
				return
			}
			t = tokenFloat
		}
	}
	// Next thing must not be alphanumeric.
	if l.acceptMany(sAlphaNum) {
		return
	}
	return t, true
}

// lexQuote scans a quoted string.
func scanQuote(l *lexer) bool {
Loop:
	for {
		switch l.nextRune() {
		case '\\':
			if r := l.nextRune(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return false
		case '"':
			break Loop
		}
	}
	return true
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

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
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

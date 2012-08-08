package parse

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenError    // scanning error
	// Primitives
	TokenBool     // 'true' or 'false'
	TokenFloat    // float
	TokenInt      // decimal or hexadecimal integer
	TokenNil      // 'nil'
	TokenString	  // string or raw string
	// Words and text
	TokenIdent    // alphanumeric identifier
	TokenTag      // registered tag name
	TokenEnd      // tag close: 'end'
	TokenFunc     // registered function name
	TokenText     // plain text outside of tags
	// Special characters used in tags
	TokenLDelim   // registered left tag delimiter; '{' by default
	TokenRDelim   // registered right tag delimiter; '}' by default
	TokenLParen   // (
	TokenRParen   // )
	TokenLBrace   // {
	TokenRBrace   // }
	TokenLBracket // [
	TokenRBracket // ]
	TokenDot      // .
	TokenPipe     // |
	TokenComma    // ,
	TokenColon    // :
	TokenNot      // !
	TokenMul      // *
	TokenDiv      // /
	TokenMod      // %
	TokenAdd      // +
	TokenSub      // -
	TokenEq       // =
	TokenColonEq  // :=
	TokenMulEq    // *=
	TokenDivEq    // /=
	TokenModEq    // %=
	TokenAddEq    // +=
	TokenSubEq    // -=
	TokenEqEq     // ==
	TokenNotEq    // !=
	TokenGt       // >
	TokenLt       // <
	TokenGtEq     // >=
	TokenLtEq     // <=
	TokenAnd      // &&
	TokenOr       // ||
	TokenChar     // any other printable ASCII char
)

var tokenName = map[TokenType]string{
	TokenEOF:      "EOF",
	TokenError:    "error",
	TokenBool:     "bool",
	TokenFloat:    "float",
	TokenInt:      "int",
	TokenNil:      "nil",
	TokenString:   "string",
	TokenIdent:    "ident",
	TokenTag:      "tag",
	TokenEnd:      "end",
	TokenFunc:     "func",
	TokenText:     "text",
	TokenLDelim:   "left delim",
	TokenRDelim:   "right delim",
	TokenLParen:   "(",
	TokenRParen:   ")",
	TokenLBrace:   "{",
	TokenRBrace:   "}",
	TokenLBracket: "[",
	TokenRBracket: "]",
	TokenDot:      ".",
	TokenPipe:     "|",
	TokenComma:    ",",
	TokenColon:    ":",
	TokenNot:      "!",
	TokenMul:      "*",
	TokenDiv:      "/",
	TokenMod:      "%",
	TokenAdd:      "+",
	TokenSub:      "-",
	TokenEq:       "=",
	TokenColonEq:  ":=",
	TokenMulEq:    "*=",
	TokenDivEq:    "/=",
	TokenModEq:    "%=",
	TokenAddEq:    "+=",
	TokenSubEq:    "-=",
	TokenEqEq:     "==",
	TokenNotEq:    "!=",
	TokenGt:       ">",
	TokenLt:       "<",
	TokenGtEq:     ">=",
	TokenLtEq:     "<=",
	TokenAnd:      "&&",
	TokenOr:       "||",
	TokenChar:     "char",
}

var stringToType = map[string]TokenType {
	"(":  TokenLParen,
	")":  TokenRParen,
	"{":  TokenLBrace,
	"}":  TokenRBrace,
	"[":  TokenLBracket,
	"]":  TokenRBracket,
	".":  TokenDot,
	"|":  TokenPipe,
	",":  TokenComma,
	":":  TokenColon,
	"!":  TokenNot,
	"*":  TokenMul,
	"/":  TokenDiv,
	"%":  TokenMod,
	"+":  TokenAdd,
	"-":  TokenSub,
	"=":  TokenEq,
	">":  TokenGt,
	"<":  TokenLt,
	":=": TokenColonEq,
	"*=": TokenMulEq,
	"/=": TokenDivEq,
	"%=": TokenModEq,
	"+=": TokenAddEq,
	"-=": TokenSubEq,
	"==": TokenEqEq,
	"!=": TokenNotEq,
	">=": TokenGtEq,
	"<=": TokenLtEq,
	"&&": TokenAnd,
	"||": TokenOr,
}

func (t TokenType) String() string {
	s := tokenName[t]
	if s == "" {
		return fmt.Sprintf("Token%d", int(t))
	}
	return s
}

// Token represents a scanned token.
type Token struct {
	Typ TokenType
	Val string
	Pos int
}

func (t Token) String() string {
	if t.Val == "" {
		return fmt.Sprintf("%s:%d", t.Typ, t.Pos)
	} else if len(t.Val) > 10 {
		return fmt.Sprintf("%s:%d:%.35q...", t.Typ, t.Pos, t.Val)
	}
	return fmt.Sprintf("%s:%d:%q", t.Typ, t.Pos, t.Val)
}

// ----------------------------------------------------------------------------

// stateFn represents the state of the scanner as a function that returns
// the next state.
type stateFn func(*Scanner) stateFn

const (
	EOF      = -1
	lDelim   = "{"
	rDelim   = "}"
	lComment = "#"
	rComment = "#"
)

// balanced delimiters shortcut.
var braces = map[string]string {
	"(": ")",
	"{": "}",
	"[": "]",
}

// newScanner creates a new scanner for the source string.
func newScanner(name, source, left, right string) *Scanner {
	if left == "" {
		left = lDelim
	}
	if right == "" {
		right = rDelim
	}
	s := &Scanner{
		name:   name,
		src:    source,
		lDelim: left,
		rDelim: right,
		tokens: make(chan Token),
	}
	go func() {
		for s.state = scanRoot; s.state != nil; {
			s.state = s.state(s)
		}
		close(s.tokens)
	}()
	return s
}

// Scanner scans a template source and emits tokens.
//
// Based on the lexer from text/template/parse; it scans some more things and
// stores state to detect the current tag being scanned and unbalanced braces.
type Scanner struct {
	name   string     // name of the source; used only for errors reports
	src    string     // template being scanned
	pin    int        // current pinned position
	pos    int        // current offset position
	width  int        // width of the last rune read
	lDelim string     // left tag delimiter
	rDelim string     // right tag delimiter
	state  stateFn    // the next scanning function to enter
	tokens chan Token // channel of scanned tokens
	stack  []Token    // LIFO stack for token backup
	braces []string   // balance stack for braces, brackets and parentheses
}

// Public API -----------------------------------------------------------------

// Position returns the line and column numbers for the given input position.
func (s *Scanner) Position(pos int) (int, int) {
	line := 1 + strings.Count(s.src[:pos], "\n")
	start := 1 + strings.LastIndex(s.src[:pos], "\n")
	column := utf8.RuneCountInString(s.src[start:pos])
	return line, column
}

// Next consumes and returns the next rune in the input. It returns EOF at the
// end of the input.
func (s *Scanner) Next() (r rune) {
	if s.pos >= len(s.src) {
		s.width = 0
		return EOF
	}
	r, s.width = utf8.DecodeRuneInString(s.src[s.pos:])
	s.pos += s.width
	return r
}

// Peek returns but does not consume the next rune in the input.
func (s *Scanner) Peek() rune {
	r := s.Next()
	s.backup()
	return r
}

// NextToken returns the next token in the input.
func (s *Scanner) NextToken() Token {
	if size := len(s.stack); size > 0 {
		tok := s.stack[size-1]
		s.stack = s.stack[:size-1]
		return tok
	}
	if tok, ok := <-s.tokens; ok {
		return tok
	}
	return Token{TokenError, "token stack is empty", s.pos}
}

// PushToken pushes back a token to the token stack. This is a LIFO stack:
// the last pushed token is the first returned by NextToken.
func (s *Scanner) PushToken(t Token) {
	s.stack = append(s.stack, t)
}

// Non-public API -------------------------------------------------------------

// backup steps back one rune. Can only be called once per call of Next.
func (s *Scanner) backup() {
	s.pos -= s.width
}

// skip skips over the pending input before this point.
func (s *Scanner) skip() {
	s.pin = s.pos
}

// emit sends a token to the tokens channel.
func (s *Scanner) emit(typ TokenType) {
	s.tokens <- Token{typ, s.src[s.pin:s.pos], s.pin}
	s.pin = s.pos
}

// error emits an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating the scan.
func (s *Scanner) errorf(pos int, format string, args ...interface{}) stateFn {
	s.tokens <- Token{TokenError, fmt.Sprintf(format, args...), pos}
	return nil
}

// isTag returns true if the given identifier is a registered tag.
func (s *Scanner) isTag(ident string) bool {
	// TODO
	return false
}

// isFunc returns true if the given identifier is a registered function.
func (s *Scanner) isFunc(ident string) bool {
	// TODO
	return false
}

// pushBrace increments the balance stack for braces, brackets and parentheses.
func (s *Scanner) pushBrace(delim string) {
	s.braces = append(s.braces, delim)
}

// popBrace decrements the balance stack for braces, brackets and parentheses.
func (s *Scanner) popBrace(delim string) error {
	if size := len(s.braces); size > 0 {
		exp := s.braces[size-1]
		if exp != delim {
			return fmt.Errorf("unbalanced delimiters: expected %q, got %q",
				exp, delim)
		}
		s.braces = s.braces[:size-1]
	} else {
		return fmt.Errorf("unbalanced delimiters: unexpected %q", delim)
	}
	return nil
}

// State functions ------------------------------------------------------------

// scanRoot scans a template root.
func scanRoot(s *Scanner) stateFn {
	for {
		if strings.HasPrefix(s.src[s.pos:], s.lDelim) {
			if s.pos > s.pin {
				s.emit(TokenText)
			}
			return scanLeftDelim
		}
		if s.Next() == EOF {
			break
		}
	}
	if s.pos > s.pin {
		s.emit(TokenText)
	}
	s.emit(TokenEOF)
	return nil
}

// scanLeftDelim scans the left delimiter, which is known to be present.
func scanLeftDelim(s *Scanner) stateFn {
	if strings.HasPrefix(s.src[s.pos:], s.lDelim+lComment) {
		// Comment tag: ignore.
		pos := s.pos
		s.pos += len(s.lDelim) + len(lComment)
		i := strings.Index(s.src[s.pos:], rComment + s.rDelim)
		if i < 0 {
			return s.errorf(pos, "unclosed comment tag")
		}
		s.pos += i + len(rComment) + len(s.rDelim)
		s.skip()
		return scanRoot
	}
	s.pos += len(s.lDelim)
	s.emit(TokenLDelim)
	return scanInsideTag
}

// scanRightDelim scans the right delimiter, which is known to be present.
func scanRightDelim(s *Scanner) stateFn {
	s.pos += len(s.rDelim)
	s.emit(TokenRDelim)
	// TODO: this can be scanTemplate
	return scanRoot
}

// scanInsideTag scans the elements inside tag delimiters.
func scanInsideTag(s *Scanner) stateFn {
	if atRightDelim(s) {
		// Only close the tag when there are no remaining open braces.
		return scanRightDelim
	}
	pos := s.pos
	switch r := s.Next(); {
	case r == EOF:
		return s.errorf(pos, "unclosed action")
	case isSpace(r):
		s.skip()
	case r == '"':
		return scanString
	case '0' <= r && r <= '9':
		s.backup()
		return scanNumber
	case isAlphaNumeric(r):
		s.backup()
		return scanIdent
	case isSymbol(r):
		s.backup()
		return scanSymbol
	case r <= unicode.MaxASCII && unicode.IsPrint(r):
		s.emit(TokenChar)
	default:
		return s.errorf(pos, "unrecognized character inside tag: %#U", r)
	}
	return scanInsideTag
}

// scanString scans a quoted string.
func scanString(s *Scanner) stateFn {
	pos := s.pos - 1
Loop:
	for {
		switch s.Next() {
		case '\\':
			if r := s.Next(); r != EOF && r != '\n' {
				break
			}
			fallthrough
		case EOF, '\n':
			return s.errorf(pos, "unterminated quoted string")
		case '"':
			break Loop
		}
	}
	s.emit(TokenString)
	return scanInsideTag
}

// scanNumber scans a number.
func scanNumber(s *Scanner) stateFn {
	pos := s.pos
	typ, ok := scanNumberType(s)
	if !ok {
		return s.errorf(pos, "bad number syntax: %q", s.src[s.pin:s.pos])
	}
	// Emits TokenFloat or TokenInt.
	s.emit(typ)
	return scanInsideTag
}

// scanNumberType scans a number.
//
// It returns a TokenFloat or TokenInt and a flag indicating if an error
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
//     - hexadecimal (must begin with lower-case 0x and must use capital A-F,
//       e.g. 0x1A2B).
//
// Unary operator minus is not scanned here.
func scanNumberType(s *Scanner) (t TokenType, ok bool) {
	r := s.Next()
	if r == '0' {
		// hexadecimal int or float
		r = s.Next()
		if r == 'x' {
			// hexadecimal int
			for isHexadecimal(s.Next()) {
			}
			s.backup()
			if !atNumberTerminator(s) {
				return TokenError, false
			}
			return TokenInt, true
		} else {
			// float
			// Not manny options after 0: only '.' or 'e'.
			s.backup()
			return scanFractionOrExponent(s)
		}
	}
	// decimal int or float
	for isDecimal(r) {
		r = s.Next()
	}
	s.backup()
	if r != '.' && r != 'e' {
		// decimal int
		if !atNumberTerminator(s) {
			return TokenError, false
		}
		return TokenInt, true
	}
	// float
	return scanFractionOrExponent(s)
}

// scanFractionOrExponent scans a fraction or exponent part of a float.
// The next character is expected to be '.' or 'e'.
func scanFractionOrExponent(s *Scanner) (t TokenType, ok bool) {
	r := s.Next()
	switch r {
	case '.':
		r = s.Next()
	case 'e':
		r = s.Next()
		if r == '-' || r == '+' {
			r = s.Next()
		}
	default:
		return TokenError, false
	}
	seenDecimal := false
	for isDecimal(r) {
		r = s.Next()
		seenDecimal = true
	}
	s.backup()
	if !seenDecimal || !atNumberTerminator(s) {
		return TokenError, false
	}
	return TokenFloat, true
}

// scanIdent scans an alphanumeric identifier.
func scanIdent(s *Scanner) stateFn {
	pos := s.pos
Loop:
	for {
		switch r := s.Next(); {
		case isAlphaNumeric(r):
			// absorb.
		default:
			s.backup()
			if !atIdentTerminator(s) {
				return s.errorf(pos, "bad character %#U", r)
			}
			switch word := s.src[s.pin:s.pos]; {
			case word == "true", word == "false":
				s.emit(TokenBool)
			case word == "nil":
				s.emit(TokenNil)
			case word == "end":
				s.emit(TokenEnd)
			case s.isTag(word):
				// TODO: check if tag has end, and build a stack to know inside
				// which tag we are
				s.emit(TokenTag)
			case s.isFunc(word):
				s.emit(TokenFunc)
			default:
				s.emit(TokenIdent)
			}
			break Loop
		}
	}
	return scanInsideTag
}

// scanString scans a symbol.
func scanSymbol(s *Scanner) stateFn {
	pos := s.pos
	r := s.Next()
	switch r {
	case '.', ',':
		s.emit(stringToType[string(r)])
	case '&':
		// &&
		if s.Next() != '&' {
			return s.errorf(pos, "expected &&")
		}
		s.emit(TokenAnd)
	case '|':
		// |, ||
		if s.Next() == '|' {
			s.emit(TokenOr)
		} else {
			s.backup()
			s.emit(TokenPipe)
		}
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
		t := string(r)
		if s.Next() == '=' {
			t += "="
		} else {
			s.backup()
		}
		s.emit(stringToType[t])
	case '(', '{', '[':
		str := string(r)
		s.pushBrace(braces[str])
		s.emit(stringToType[str])
	case ')', '}', ']':
		str := string(r)
		if err := s.popBrace(str); err != nil {
			return s.errorf(pos, err.Error())
		}
		s.emit(stringToType[str])
	default:
		// should never happen.
		return s.errorf(pos, "unrecognized symbol: %#U", r)
	}
	return scanInsideTag
}

// Helpers --------------------------------------------------------------------

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r':
		return true
	}
	return false
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isSymbol reports whether r is a recognized symbol character.
func isSymbol(r rune) bool {
	switch r {
	case '.', ',', '*', '/', '%', '+', '-', '=', ':', '!', '<', '>',
		'&', '|', '(', '{', '[', ')', '}', ']':
		return true
	}
	return false
}

// isDecimal reports whether r is a decimal character.
func isDecimal(r rune) bool {
	return '0' <= r && r <= '9'
}

// isHexadecimal reports whether r is an hexadecimal character.
func isHexadecimal(r rune) bool {
	return ('0' <= r && r <= '9') || ('A' <= r && r <= 'F')
}

// atRightDelim reports whether the input is at a right delimiter.
func atRightDelim(s *Scanner) bool {
	return len(s.braces) == 0 && strings.HasPrefix(s.src[s.pos:], s.rDelim)
}

// atNumberTerminator reports whether the input is a valid character to
// appear after a number.
func atNumberTerminator(s *Scanner) bool {
	switch r := s.Peek(); {
	// All spaces and most symbols (excludes dot and open braces).
	case isSpace(r):
		return true
	case isSymbol(r):
		switch r {
		case '.', '(', '[', '{':
			return false
		}
		return true
	case atRightDelim(s):
		return true
	}
	return false
}

// atIdentTerminator reports whether the input is a valid character to
// appear after an identifier.
func atIdentTerminator(s *Scanner) bool {
	switch r := s.Peek(); {
	// All spaces and symbols.
	case isSpace(r):
		return true
	case isSymbol(r):
		return true
	case atRightDelim(s):
		return true
	}
	return false
}

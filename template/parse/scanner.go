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
	TokenText     // plain text outside of tags
	// Special characters used inside tags
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

var symbolToType = map[string]TokenType {
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
	Typ TokenType // type of the token
	Val string    // value of the token
	Pos int       // position of the token in the input, in bytes
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
		for s.state = scanTemplate; s.state != nil; {
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

// next consumes and returns the next rune in the input. It returns EOF at the
// end of the input.
func (s *Scanner) next() (r rune) {
	if s.pos >= len(s.src) {
		s.width = 0
		return EOF
	}
	r, s.width = utf8.DecodeRuneInString(s.src[s.pos:])
	s.pos += s.width
	return r
}

// peek returns but does not consume the next rune in the input.
func (s *Scanner) peek() rune {
	r := s.next()
	s.backup()
	return r
}

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
func (s *Scanner) errorf(format string, args ...interface{}) stateFn {
	s.tokens <- Token{TokenError, fmt.Sprintf(format, args...), s.pin}
	return nil
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

// scanTemplate scans a template root.
func scanTemplate(s *Scanner) stateFn {
	for {
		if strings.HasPrefix(s.src[s.pos:], s.lDelim) {
			if s.pos > s.pin {
				s.emit(TokenText)
			}
			return scanLeftDelim
		}
		if s.next() == EOF {
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
		s.pos += len(s.lDelim) + len(lComment)
		i := strings.Index(s.src[s.pos:], rComment + s.rDelim)
		if i < 0 {
			return s.errorf("unclosed comment tag")
		}
		s.pos += i + len(rComment) + len(s.rDelim)
		s.skip()
		return scanTemplate
	}
	s.pos += len(s.lDelim)
	s.emit(TokenLDelim)
	return scanInsideTag
}

// scanRightDelim scans the right delimiter, which is known to be present.
func scanRightDelim(s *Scanner) stateFn {
	s.pos += len(s.rDelim)
	s.emit(TokenRDelim)
	return scanTemplate
}

// scanInsideTag scans the elements inside tag delimiters.
func scanInsideTag(s *Scanner) stateFn {
	if atRightDelim(s) {
		// Only close the tag when there are no remaining open braces.
		return scanRightDelim
	}
	switch r := s.next(); {
	case r == EOF:
		return s.errorf("unclosed tag")
	case isSpace(r):
		s.skip()
	case r == '"':
		return scanString
	case isDecimal(r):
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
		return s.errorf("unrecognized character inside tag: %#U", r)
	}
	return scanInsideTag
}

// scanString scans a quoted string.
func scanString(s *Scanner) stateFn {
Loop:
	for {
		switch s.next() {
		case '\\':
			if r := s.next(); r != EOF && r != '\n' {
				break
			}
			fallthrough
		case EOF, '\n':
			return s.errorf("unterminated quoted string")
		case '"':
			break Loop
		}
	}
	s.emit(TokenString)
	return scanInsideTag
}

// scanNumber scans a number.
//
// It returns a TokenFloat or TokenInt, if everything goes well, and a state
// function if an error was found.
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
func scanNumber(s *Scanner) stateFn {
	r := s.next()
	if r == '0' {
		// hexadecimal int or float
		r = s.next()
		if r == 'x' {
			// hexadecimal int
			for isHexadecimal(s.next()) {
			}
			s.backup()
			s.emit(TokenInt)
			return scanInsideTag
		} else {
			// float
			// Not manny options after 0: only '.' or 'e'.
			s.backup()
			return scanFractionOrExponent(s)
		}
	}
	// decimal int or float
	for isDecimal(r) {
		r = s.next()
	}
	s.backup()
	if r != '.' && r != 'e' {
		// decimal int
		s.emit(TokenInt)
		return scanInsideTag
	}
	// float
	return scanFractionOrExponent(s)
}

// scanFractionOrExponent scans a fraction or exponent part of a float.
// The next character is expected to be '.' or 'e'.
func scanFractionOrExponent(s *Scanner) stateFn {
	r := s.next()
	switch r {
	case '.':
		r = s.next()
	case 'e':
		r = s.next()
		if r == '-' || r == '+' {
			r = s.next()
		}
	default:
		return s.errorf("bad float syntax: %q",	s.src[s.pin:s.pos])
	}
	seenDecimal := false
	for isDecimal(r) {
		r = s.next()
		seenDecimal = true
	}
	s.backup()
	if !seenDecimal {
		return s.errorf("expected a decimal after '.' or 'e'")
	}
	s.emit(TokenFloat)
	return scanInsideTag
}

// scanIdent scans an alphanumeric identifier.
func scanIdent(s *Scanner) stateFn {
Loop:
	for {
		switch r := s.next(); {
		case isAlphaNumeric(r):
			// absorb.
		default:
			s.backup()
			switch word := s.src[s.pin:s.pos]; {
			case word == "true", word == "false":
				s.emit(TokenBool)
			case word == "nil":
				s.emit(TokenNil)
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
	r := s.next()
	switch r {
	case '.', ',':
		s.emit(symbolToType[string(r)])
	case '&':
		// &&
		if s.next() != '&' {
			return s.errorf("expected &&")
		}
		s.emit(TokenAnd)
	case '|':
		// |, ||
		if s.next() == '|' {
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
		if s.next() == '=' {
			t += "="
		} else {
			s.backup()
		}
		s.emit(symbolToType[t])
	case '(', '{', '[':
		str := string(r)
		s.pushBrace(braces[str])
		s.emit(symbolToType[str])
	case ')', '}', ']':
		str := string(r)
		if err := s.popBrace(str); err != nil {
			return s.errorf(err.Error())
		}
		s.emit(symbolToType[str])
	default:
		// Should never happen.
		return s.errorf("unrecognized symbol: %#U", r)
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
	return symbolToType[string(r)] > 0
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

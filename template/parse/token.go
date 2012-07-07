// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
)

// tokenType is the type of lex tokens.
// TODO make this extensible using a registry system.
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
	tokenIdent        // [a-zA-Z_][a-zA-Z0-9_]+(\.[a-zA-Z_][a-zA-Z0-9_]+)*
	tokenDot          // identifier starting with dot
	tokenComment
	tokenText
	tokenRawText
	// Primitives
	tokenBool
	tokenFloat
	tokenInt
	tokenNull
	tokenString
	// Tags
	tokenElse
	tokenEnd
	tokenIf
	tokenNamespace
	tokenTemplate
	tokenRange
	tokenRaw
	tokenSp
	tokenWith
	// Operators
	tokenNeg          // - (unary)
	tokenNot          // ! ("not" in Closure Template)
	tokenMul          // *
	tokenDiv          // /
	tokenMod          // %
	tokenAdd          // +
	tokenSub          // - (binary)
	tokenEq           // ==
	tokenNotEq        // !=
	tokenGt           // >
	tokenGte          // >=
	tokenLt           // <
	tokenLte          // <=
	tokenOr           // || ("or" in Closure Template)
	tokenAnd          // && ("and" in Closure Template)
	tokenQuestion     // ?
	tokenColon        // : ("else" or key:value delimiter in maps)
	// Symbols
	tokenDollar       // $
	tokenLeftParen    // (
	tokenRightParen   // )
	tokenLeftBrace    // {
	tokenRightBrace   // }
	tokenLeftBracket  // [
	tokenRightBracket // ]

	tokenAssign
	tokenColonAssign
)

var tokenName = map[tokenType]string{
	tokenError:        "error",
	tokenEOF:          "EOF",
	tokenDot:          "dot",
	tokenIdent:        "ident",
	tokenComment:      "comment",
	tokenText:         "text",
	tokenRawText:      "raw text",
	// Primitives
	tokenBool:         "bool",
	tokenFloat:        "float",
	tokenInt:          "int",
	tokenNull:         "null",
	tokenString:       "string",
	// Tags
	tokenElse:         "{else}",
	tokenEnd:          "{end}",
	tokenIf:           "{if}",
	tokenNamespace:    "{namespace}",
	tokenTemplate:     "{template}",
	tokenRange:        "{range}",
	tokenRaw:          "{raw}",
	tokenSp:           "{sp}",
	tokenWith:         "{with}",
	// Operators
	tokenNeg:          "-",
	tokenNot:          "!",
	tokenMul:          "*",
	tokenDiv:          "/",
	tokenMod:          "%",
	tokenAdd:          "+",
	tokenSub:          "-",
	tokenEq:           "==",
	tokenNotEq:        "!=",
	tokenGt:           ">",
	tokenGte:          ">=",
	tokenLt:           "<",
	tokenLte:          "<=",
	tokenOr:           "||",
	tokenAnd:          "&&",
	tokenQuestion:     "?",
	tokenColon:        ":",
	// Symbols
	tokenDollar:       "$",
	tokenLeftParen:    "(",
	tokenRightParen:   ")",
	tokenLeftBrace:    "{",
	tokenRightBrace:   "}",
	tokenLeftBracket:  "[",
	tokenRightBracket: "]",

	tokenAssign:       "=",
	tokenColonAssign:  ":=",
}

// ----------------------------------------------------------------------------

type position struct {
	line   int
	column int
}

// token is a token returned from the lexer.
type token struct {
	typ tokenType
	pos position
	val string
}

func (t token) String() string {
	if t.val != "" {
		return fmt.Sprintf("<%s:%s>", t.typ, t.val)
	}
	return fmt.Sprintf("<%s>", t.typ)
}

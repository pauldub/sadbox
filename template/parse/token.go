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
	tokenIdent        // alphanumeric
	tokenVar          // identifier starting with dot
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
	tokenNot          // ! ("not")
	tokenNotEq        // !=
	tokenStar         // *
	tokenStarEq       // *=
	tokenSlash        // /
	tokenSlashEq      // /=
	tokenPercent      // %
	tokenPercentEq    // %=
	tokenPlus         // +
	tokenPlusEq       // +=
	tokenMinus        // -
	tokenMinusEq      // -=
	tokenEq           // =
	tokenEqEq         // ==
	tokenGt           // >
	tokenGtEq         // >=
	tokenLt           // <
	tokenLtEq         // <=
	tokenOr           // || ("or" in Closure Template)
	tokenAnd          // && ("and" in Closure Template)
	tokenQuestion     // ?
	tokenColon        // : ("else" or key:value delimiter in maps)
	tokenColonEq      // :=
	// Symbols
	tokenDollar       // $
	tokenLeftParen    // (
	tokenRightParen   // )
	tokenLeftBrace    // {
	tokenRightBrace   // }
	tokenLeftBracket  // [
	tokenRightBracket // ]
)

var tokenName = map[tokenType]string{
	tokenError:        "error",
	tokenEOF:          "EOF",
	tokenIdent:        "ident",
	tokenVar:          "dot",
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
	tokenNotEq:        "!=",
	tokenStar:         "*",
	tokenStarEq:       "*=",
	tokenSlash:        "/",
	tokenSlashEq:      "/=",
	tokenPercent:      "%",
	tokenPercentEq:    "%=",
	tokenPlus:         "+",
	tokenPlusEq:       "+=",
	tokenMinus:        "-",
	tokenMinusEq:      "-=",
	tokenEq:           "=",
	tokenEqEq:         "==",
	tokenGt:           ">",
	tokenGtEq:         ">=",
	tokenLt:           "<",
	tokenLtEq:         "<=",
	tokenOr:           "||",
	tokenAnd:          "&&",
	tokenQuestion:     "?",
	tokenColon:        ":",
	tokenColonEq:      ":=",
	// Symbols
	tokenDollar:       "$",
	tokenLeftParen:    "(",
	tokenRightParen:   ")",
	tokenLeftBrace:    "{",
	tokenRightBrace:   "}",
	tokenLeftBracket:  "[",
	tokenRightBracket: "]",



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

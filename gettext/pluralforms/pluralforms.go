// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pluralforms

import (
	"strings"
)

// PluralFunc is used to select a plural form index for a given amount.
type PluralFunc func(int) int

// DefaultPluralFunc is the default plural selector, used for English
// and others.
var DefaultPluralFunc = pluralFunc2

// Parse parses a Plural-Forms expression and returns a PluralFunc
// capable of evaluating it.
//
// If the expression is malformed it returns an error. Even if it doesn't
// return an error the returned PluralFunc can still fail to evaluate.
// If this occurs it returns -1 (an invalid index).
func Parse(expr string) (PluralFunc, error) {
	expr = strings.Replace(expr, " ", "", -1)
	if f, ok := pluralFuncs[expr]; ok {
		return f, nil
	}
	return createPluralFunc(expr)
}

// createPluralFunc parses a Plural-Forms expression and returns a PluralFunc
// capable of evaluating it.
func createPluralFunc(expr string) (PluralFunc, error) {
	tree, err := parse(expr)
	if err != nil {
		return nil, err
	}
	return func(n int) int {
		switch v := tree.Eval(n).(type) {
		case intNode:
			return int(v)
		case boolNode:
			if v {
				return 1
			}
			return 0
		}
		return -1
	}, nil
}

// Precomputed plural funcs taken from the gettext manual. We avoid parsing
// expressions that match one of these forms. See:
//
//     http://www.gnu.org/software/gettext/manual/gettext.html#Plural-forms
//
// Send us variants or new ones to be added to this list.
var pluralFuncs = map[string]PluralFunc{
	"0":    pluralFunc1,
	"n!=1": pluralFunc2,
	"n>1":  pluralFunc3,
	"n%10==1&&n%100!=11?0:n!=0?1:2":                                    pluralFunc4,
	"n==1?0:n==2?1:2":                                                  pluralFunc5,
	"n==1?0:(n==0||(n%100>0&&n%100<20))?1:2":                           pluralFunc6,
	"n%10==1&&n%100!=11?0:n%10>=2&&(n%100<10||n%100>=20)?1:2":          pluralFunc7,
	"n%10==1&&n%100!=11?0:n%10>=2&&n%10<=4&&(n%100<10||n%100>=20)?1:2": pluralFunc8,
	"(n==1)?0:(n>=2&&n<=4)?1:2":                                        pluralFunc9,
	"n==1?0:n%10>=2&&n%10<=4&&(n%100<10||n%100>=20)?1:2":               pluralFunc10,
	"n%100==1?0:n%100==2?1:n%100==3||n%100==4?2:3":                     pluralFunc11,
}

// Only one form:
//
//     Plural-Forms: nplurals=1; plural=0;
//
func pluralFunc1(n int) int {
	return 0
}

// Two forms, singular used for one only:
//
//     Plural-Forms: nplurals=2; plural=n != 1;
//
// Default one, used for English (and others).
func pluralFunc2(n int) int {
	if n == 1 {
		return 0
	}
	return 1
}

// Two forms, singular used for zero and one:
//
//     Plural-Forms: nplurals=2; plural=n>1;
//
func pluralFunc3(n int) int {
	if n > 1 {
		return 1
	}
	return 0
}

// Three forms, special case for zero:
//
//     Plural-Forms: nplurals=3; plural=n%10==1 && n%100!=11 ? 0 : n != 0 ? 1 : 2;
//
func pluralFunc4(n int) int {
	if n%10 == 1 && n%100 != 11 {
		return 0
	}
	if n != 0 {
		return 1
	}
	return 2
}

// Three forms, special cases for one and two:
//
//     Plural-Forms: nplurals=3; plural=n==1 ? 0 : n==2 ? 1 : 2;
//
func pluralFunc5(n int) int {
	if n == 1 {
		return 0
	}
	if n == 2 {
		return 1
	}
	return 2
}

// Three forms, special case for numbers ending in 00 or [2-9][0-9]:
//
//     Plural-Forms: nplurals=3; plural=n==1 ? 0 : (n==0 || (n%100 > 0 && n%100 < 20)) ? 1 : 2;
//
func pluralFunc6(n int) int {
	if n == 1 {
		return 0
	}
	if n == 0 || (n%100 > 0 && n%100 < 20) {
		return 1
	}
	return 2
}

// Three forms, special case for numbers ending in 1[2-9]:
//
//     Plural-Forms: nplurals=3; plural=n%10==1 && n%100!=11 ? 0 : n%10>=2 && (n%100<10 || n%100>=20) ? 1 : 2;
//
func pluralFunc7(n int) int {
	if n%10 == 1 && n%100 != 11 {
		return 0
	}
	if n%10 >= 2 && (n%100 < 10 || n%100 >= 20) {
		return 1
	}
	return 2
}

// Three forms, special cases for numbers ending in 1 and 2, 3, 4, except
// those ending in 1[1-4]:
//
//     Plural-Forms: nplurals=3; plural=n%10==1 && n%100!=11 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2;
//
func pluralFunc8(n int) int {
	if n%10 == 1 && n%100 != 11 {
		return 0
	}
	if n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20) {
		return 1
	}
	return 2
}

// Three forms, special cases for 1 and 2, 3, 4:
//
//     Plural-Forms: nplurals=3; plural=(n==1) ? 0 : (n>=2 && n<=4) ? 1 : 2;
//
func pluralFunc9(n int) int {
	if n == 1 {
		return 0
	}
	if n >= 2 && n <= 4 {
		return 1
	}
	return 2
}

// Three forms, special case for one and some numbers ending in 2, 3, or 4:
//
//     Plural-Forms: nplurals=3; plural=n==1 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2;
//
func pluralFunc10(n int) int {
	if n == 1 {
		return 0
	}
	if n%10 >= 2 && n%10 <= 4 && (n%100 < 10 || n%100 >= 20) {
		return 1
	}
	return 2
}

// Four forms, special case for one and all numbers ending in 02, 03, or 04:
//
//     Plural-Forms: nplurals=4; plural=n%100==1 ? 0 : n%100==2 ? 1 : n%100==3 || n%100==4 ? 2 : 3;
//
func pluralFunc11(n int) int {
	if n%100 == 1 {
		return 0
	}
	if n%100 == 2 {
		return 1
	}
	if n%100 == 3 || n%100 == 4 {
		return 2
	}
	return 3
}

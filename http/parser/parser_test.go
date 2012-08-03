// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser

import (
	"testing"
)

func stringSliceEqual(v1, v2 []string) bool {
	m := make(map[string]bool)
	for _, v := range v1 {
		m[v] = true
	}
	for _, v := range v2 {
		if !m[v] {
			return false
		}
	}
	return true
}

func stringMapEqual(v1, v2 map[string]string) bool {
	for k, v := range v1 {
		if value, ok := v2[k]; !ok || value != v {
			return false
		}
	}
	return true
}

func TestParseList(t *testing.T) {
	tests := []struct{
		Value string
		List  []string
	}{
		{`a,b,c`, []string{`a`, `b`, `c`}},
		{`path"o,l"og"i"cal, example`, []string{`path"o,l"og"i"cal`, `example`}},
		{`a, b, "c", "d", "e,f", g, h`, []string{`a`, `b`, `"c"`, `"d"`, `"e,f"`, `g`, `h`}},
		{`a="b\"c", d="e\,f", g="h\\i"`, []string{`a="b"c"`, `d="e,f"`, `g="h\i"`}},
	}

	for _, test := range tests {
		v := ParseList(test.Value)
		if !stringSliceEqual(test.List, v) {
			t.Errorf("Expected %v, got %v", test.List, v)
		}
	}
}

func TestParsePairs(t *testing.T) {
	tests := []struct{
		Value string
		Pairs map[string]string
	}{
		{`a,b,c`, map[string]string{`a`: ``, `b`: ``, `c`: ``}},
		{`a="b\"c", d="e\,f", g="h\\i"`, map[string]string{`a`: `b"c`, `d`: `e,f`, `g`: `h\i`}},
	}

	for _, test := range tests {
		v := ParsePairs(test.Value)
		if !stringMapEqual(test.Pairs, v) {
			t.Errorf("Expected %v, got %v", test.Pairs, v)
		}
	}
}

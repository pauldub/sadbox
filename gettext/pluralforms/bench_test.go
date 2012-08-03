// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pluralforms

import (
	"testing"
)

// - June 25, 2012, Core i5-2400:
//   BenchmarkParser  100000	     17934 ns/op
//   BenchmarkEval	 1000000	      1546 ns/op

func BenchmarkParser(b *testing.B) {
	expr := "n%10==1&&n%100!=11?0:n%10>=2&&n%10<=4&&(n%100<10||n%100>=20)?1:2"
	for i := 0; i < b.N; i++ {
		_, err := parse(expr)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEval(b *testing.B) {
	expr := "n%10==1&&n%100!=11?0:n%10>=2&&n%10<=4&&(n%100<10||n%100>=20)?1:2"
	fn, err := createPluralFunc(expr)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		n := fn(i)
		if n == -1 {
			b.Fatalf("Expression failed to evaluate")
		}
	}
}

func BenchmarkPluralFunc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		n := pluralFunc8(i)
		if n == -1 {
			b.Fatalf("Expression failed to evaluate")
		}
	}
}

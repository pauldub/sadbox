// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gettext

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode"
)

// parseFmt converts a string that relies on reordering ability to a standard
// format, e.g., the string "%2$d bytes on %1$s." becomes "%d bytes on %s.".
// The returned indices are used to format the resulting string using
// sprintf().
func parseFmt(trn, msg string) (string, []int) {
	var idx []int
	end := len(trn)
	buf := new(bytes.Buffer)
	for i := 0; i < end; {
		lasti := i
		for i < end && trn[i] != '%' {
			i++
		}
		if i > lasti {
			buf.WriteString(trn[lasti:i])
		}
		if i >= end {
			break
		}
		i++
		if i < end && trn[i] == '%' {
			// escaped percent
			buf.WriteString("%%")
			i++
		} else {
			buf.WriteByte('%')
			lasti = i
			for i < end && unicode.IsDigit(rune(trn[i])) {
				i++
			}
			if i > lasti {
				if i < end && trn[i] == '$' {
					// extract number, skip dollar sign
					pos, _ := strconv.ParseInt(trn[lasti:i], 10, 0)
					idx = append(idx, int(pos))
					i++
				} else {
					buf.WriteString(trn[lasti:i])
				}
			}
		}
	}
	return buf.String(), idx
}

// sprintf applies fmt.Sprintf() on a string that relies on reordering
// ability, e.g., for the string "%2$d bytes free on %1$s.", the order of
// arguments must be inverted.
func sprintf(format string, order []int, a ...interface{}) string {
	if order == nil {
		return fmt.Sprintf(format, a...)
	}
	b := make([]interface{}, len(order))
	l := len(a)
	for k, v := range order {
		if v < l {
			b[k] = a[v]
		}
	}
	return fmt.Sprintf(format, b...)
}

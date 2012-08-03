// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gorilla/http/parser offers utilities to parse HTTP headers.
*/
package parser

import (
	"bytes"
	"strings"
)

// ParseList parses a comma-separated list of values as described by RFC 2068.
//
// Ported from urllib2.parse_http_list, from the Python standard library.
func ParseList(value string) []string {
	var list []string
	var escape, quote bool
	b := new(bytes.Buffer)
	for _, r := range value {
		if escape {
			b.WriteRune(r)
			escape = false
			continue
		}
		if quote {
			if r == '\\' {
				escape = true
				continue
			} else if r == '"' {
				quote = false
			}
			b.WriteRune(r)
			continue
		}
		if r == ',' {
			list = append(list, strings.TrimSpace(b.String()))
			b.Reset()
			continue
		}
		if r == '"' {
			quote = true
		}
		b.WriteRune(r)
	}
	// Append last part.
	if s := b.String(); s != "" {
		list = append(list, strings.TrimSpace(s))
	}
	return list
}

// ParsePairs extracts key/value pairs from a comma-separated list of values as
// described by RFC 2068.
//
// The resulting values are unquoted. If a value doesn't contain a "=", the
// key is the value itself and the value is an empty string.
func ParsePairs(value string) map[string]string {
	m := make(map[string]string)
	for _, pair := range ParseList(value) {
		if i := strings.Index(pair, "="); i < 0 {
			m[pair] = ""
		} else {
			v := pair[i+1:]
			if v[0] == '"' && v[len(v)-1] == '"' {
				// Unquote it.
				v = v[1:len(v)-1]
			}
			m[pair[:i]] = v
		}
	}
	return m
}

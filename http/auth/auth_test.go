// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package auth

import (
	"encoding/base64"
	"net/http"
	"testing"
)

func TestBasic(t *testing.T) {
	v1 := base64.StdEncoding.EncodeToString([]byte("foo:bar"))
	v2 := base64.StdEncoding.EncodeToString([]byte("foo:bar:baz"))
	valid := []string{
		"Basic " + v1,
	}
	invalidScheme := []string{
		"basic " + v1,
		"Digest " + v1,
	}
	invalidCredential := []string{
		"Basic " + v2,
	}
	for _, v := range valid {
		r, _ := http.NewRequest("GET", "http://localhost", nil)
		r.Header.Set("Authorization", v)
		b, err := NewBasicFromRequest(r)
		if err != nil {
			t.Errorf("NewBasicFromRequest should not fail for %q (error: %q)", v, err)
		} else if b.Username != "foo" || b.Password != "bar" {
			t.Errorf(`Expected "foo:bar", got "%s:%s"`, b.Username, b.Password)
		}
		b, err = NewBasic(v[6:])
		if err != nil {
			t.Errorf("NewBasic should not fail for %q (error: %q)", v, err)
		} else if b.Username != "foo" || b.Password != "bar" {
			t.Errorf(`Expected "foo:bar", got "%s:%s"`, b.Username, b.Password)
		}
	}
	for _, v := range invalidScheme {
		r, _ := http.NewRequest("GET", "http://localhost", nil)
		r.Header.Set("Authorization", v)
		_, err := NewBasicFromRequest(r)
		if err == nil {
			t.Errorf("NewBasic should fail for %q", v)
		}
	}
	for _, v := range invalidCredential {
		_, err := NewBasic(v[6:])
		if err == nil {
			t.Errorf("NewBasic should fail for %q", v)
		}
	}
}

// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package gorilla/http/auth parses "Authorization" request headers.

The framework is defined by RFC2617, "HTTP Authentication: Basic and Digest
Access Authentication":

	http://tools.ietf.org/html/rfc2617
*/
package auth

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"code.google.com/p/gorilla/http/parser"
)

// ParseRequest extracts an "Authorization" header from a request and returns
// its scheme and credentials.
func ParseRequest(r *http.Request) (scheme, credentials string, err error) {
	h, ok := r.Header["Authorization"]
	if !ok || len(h) == 0 {
		return "", "", errors.New("The authorization header is not set.")
	}
	return Parse(h[0])
}

// Parse parses an "Authorization" header and returns its scheme and
// credentials.
func Parse(value string) (scheme, credentials string, err error) {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return "", "", errors.New("The authorization header is malformed.")
}

// ----------------------------------------------------------------------------

// NewBasicFromRequest extracts an "Authorization" header from a request and
// returns the parsed credentials from a "basic" http authentication scheme.
func NewBasicFromRequest(r *http.Request) (*Basic, error) {
	scheme, credentials, err := ParseRequest(r)
	if err == nil {
		if scheme == "Basic" {
			return NewBasic(credentials)
		} else {
			err = errors.New("The basic authentication header is invalid.")
		}
	}
	return nil, err
}

// NewBasic parses credentials from a "basic" http authentication scheme.
func NewBasic(credentials string) (*Basic, error) {
	if b, err := base64.StdEncoding.DecodeString(credentials); err == nil {
		parts := strings.Split(string(b), ":")
		if len(parts) == 2 {
			return &Basic{
				Username: parts[0],
				Password: parts[1],
			}, nil
		}
	}
	return nil, errors.New("The basic authentication header is malformed.")
}

// Basic stores username and password for the "basic" http authentication
// scheme. Reference:
//
//    http://tools.ietf.org/html/rfc2617#section-2
type Basic struct {
	Username string
	Password string
}

// ----------------------------------------------------------------------------

// NewDigestFromRequest extracts an "Authorization" header from a request and
// returns the parsed credentials from a "digest" http authentication scheme.
func NewDigestFromRequest(r *http.Request) (*Digest, error) {
	scheme, credentials, err := ParseRequest(r)
	if err == nil {
		if scheme == "Digest" {
			return NewDigest(credentials)
		} else {
			err = errors.New("The digest authentication header is invalid.")
		}
	}
	return nil, err
}

// NewDigest parses credentials from a "digest" http authentication scheme.
func NewDigest(credentials string) (*Digest, error) {
	// TODO: validate required keys.
	return &Digest{Values: parser.ParsePairs(credentials)}, nil
}

// Basic stores credentials for the "digest" http authentication scheme.
// Reference:
//
//    http://tools.ietf.org/html/rfc2617#section-2
//
// This is just a placeholder.
type Digest struct {
	Values map[string]string
}

// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package i18n

// Catalog types return translations for messages and plurals.
type Catalog interface {
	// Get returns a translation for the given key.
	// Extra arguments or optional, used to format the translation.
	Get(key string, a ...interface{}) string
	// GetPlural returns a plural translation for the given key and number.
	// Extra arguments or optional, used to format the translation.
	//
	// Note: while ngettext accepts two string arguments, other systems
	// normally just accept a key. To follow ngettext strictly,
	// gettext-based catalogs must wrap a call to GetPlural.
	GetPlural(key string, num int, a ...interface{}) string
}

// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gettext

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"code.google.com/p/gorilla/gettext/pluralforms"
)

const (
	magicBigEndian    uint32 = 0xde120495
	magicLittleEndian uint32 = 0x950412de
)

// MoReader loads catalogs from GNU MO files.
//
// Currently only UTF-8 encoding is supported. An encoding translator
// may be added in the future.
type MoReader struct {
}

// Read loads a catalog from the given reader.
func (mr *MoReader) Read(c *Catalog, r io.ReadSeeker) error {
	// First word identifies the byte order.
	var order binary.ByteOrder
	var magic uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return err
	}
	if magic == magicLittleEndian {
		order = binary.LittleEndian
	} else if magic == magicBigEndian {
		order = binary.BigEndian
	} else {
		return errors.New("Unable to identify the file byte order")
	}
	// Next two words:
	// byte 4: major revision number
	// byte 6: minor revision number
	rev := make([]uint16, 2)
	for k, _ := range rev {
		if err := binary.Read(r, order, &rev[k]); err != nil {
			return err
		}
	}
	if rev[0] > 1 || rev[1] > 1 {
		return fmt.Errorf("Major and minor MO revision numbers must be "+
			"0 or 1, got %d and %d", rev[0], rev[1])
	}
	// Next five words:
	// byte 8:  number of messages
	// byte 12: index of messages table
	// byte 16: index of translations table
	// byte 20: size of hashing table
	// byte 24: offset of hashing table
	idx := make([]uint32, 5)
	for k, _ := range idx {
		if err := binary.Read(r, order, &idx[k]); err != nil {
			return err
		}
	}
	count, mTableIdx, tTableIdx := int(idx[0]), int64(idx[1]), int64(idx[2])
	// Build a translations table of strings and translations.
	// Plurals are stored separately with the first message as key.
	var mLen, mIdx, tLen, tIdx uint32
	for i := 0; i < count; i++ {
		// Get message length and position.
		r.Seek(mTableIdx, 0)
		if err := binary.Read(r, order, &mLen); err != nil {
			return err
		}
		if err := binary.Read(r, order, &mIdx); err != nil {
			return err
		}
		// Get message.
		mb := make([]byte, mLen)
		r.Seek(int64(mIdx), 0)
		if err := binary.Read(r, order, mb); err != nil {
			return err
		}
		// Get translation length and position.
		r.Seek(tTableIdx, 0)
		if err := binary.Read(r, order, &tLen); err != nil {
			return err
		}
		if err := binary.Read(r, order, &tIdx); err != nil {
			return err
		}
		// Get translation.
		tb := make([]byte, tLen)
		r.Seek(int64(tIdx), 0)
		if err := binary.Read(r, order, tb); err != nil {
			return err
		}
		// Move cursor to next message.
		mTableIdx += 8
		tTableIdx += 8
		// Is this is the file header?
		if len(mb) == 0 {
			readMoHeader(c, string(tb))
			continue
		}
		// Check for context.
		mStr, tStr := string(mb), string(tb)
		var ctx string
		var hasCtx bool
		if ctxIdx := strings.Index(mStr, "\x04"); ctxIdx != -1 {
			ctx = mStr[:ctxIdx]
			mStr = mStr[ctxIdx+1:]
			hasCtx = true
		}
		// Add the message.
		if keyIdx := strings.Index(mStr, "\x00"); keyIdx == -1 {
			// Singular.
			c.Add(&SimpleMessage{
				Src:    mStr,
				Dst:    tStr,
				Ctx:    ctx,
				HasCtx: hasCtx,
			})
		} else {
			// Plural.
			c.Add(&PluralMessage{
				Src:    strings.Split(mStr, "\x00"),
				Dst:    strings.Split(tStr, "\x00"),
				Ctx:    ctx,
				HasCtx: hasCtx,
			})
		}
	}
	if header, ok := c.Header["plural-forms"]; ok {
		fn, err := mr.getPluralFunc(header)
		if err != nil {
			return err
		}
		c.PluralFunc = fn
	}
	return nil
}

func (mr *MoReader) getPluralFunc(header string) (pluralforms.PluralFunc, error) {
	for _, part := range strings.Split(header, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == "plural" {
			if fn, err := pluralforms.Parse(kv[1]); err == nil {
				return fn, nil
			} else {
				return nil, err
			}
		}
	}
	return nil, fmt.Errorf("Malformed Plural-Forms header: %q", header)
}

// readMoHeader parses the translations metadata following GNU .mo conventions.
func readMoHeader(c *Catalog, header string) {
	var lastk string
	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i := strings.Index(line, ":"); i != -1 {
			k := strings.ToLower(strings.TrimSpace(line[:i]))
			v := strings.TrimSpace(line[i+1:])
			c.Header[k] = v
			lastk = k
		} else if lastk != "" {
			c.Header[lastk] += "\n" + line
		}
	}
}

// ----------------------------------------------------------------------------

// MoWriter compiles catalogs to GNU MO files.
//
// Currently only UTF-8 encoding is supported. An encoding translator
// may be added in the future.
type MoWriter struct {
}

// Write compiles a catalog to the given writer.
func (mw *MoWriter) Write(c *Catalog, w io.WriteSeeker) error {
	order := binary.LittleEndian
	count := len(c.Messages) + 1 // +1 for the header
	idxs, msgs := newMoMessageWriter(c)
	mTableIdx := 28
	tTableIdx := mTableIdx + count*8
	table := []uint32{
		magicLittleEndian, // byte 0:  magic number
		uint32(0),         // byte 4:  major+minor revision number
		uint32(count),     // byte 8:  number of messages
		uint32(mTableIdx), // byte 12: index of messages table
		uint32(tTableIdx), // byte 16: index of translations table
		uint32(0),         // byte 20: size of hashing table
		uint32(0),         // byte 24: offset of hashing table
	}
	if err := binary.Write(w, order, table); err != nil {
		return err
	}
	// At byte 28
	if err := binary.Write(w, order, idxs); err != nil {
		return err
	}
	// At byte 28 + (count*8) + (count*8)
	if err := binary.Write(w, order, msgs); err != nil {
		return err
	}
	return nil
}

// moMessageWriter pre-computes values for MoWriter.Write.
type moMessageWriter struct {
	src     *bytes.Buffer
	dst     *bytes.Buffer
	srcIdx  uint32
	dstIdx  uint32
	srcList []uint32
	dstList []uint32
}

func newMoMessageWriter(c *Catalog) (idxs []uint32, msgs []byte) {
	count := len(c.Messages) + 1 // +1 for the header
	m := &moMessageWriter{
		src:    new(bytes.Buffer),
		dst:    new(bytes.Buffer),
		srcIdx: uint32(28 + count*16),
	}
	m.append(m.getHeader(c))
	for _, msg := range sortedMessages(c) {
		m.append(msg)
	}
	// Merge everything.
	for i := 0; i < len(m.dstList); i += 2 {
		// Increment offset for translations.
		m.dstList[i+1] += m.srcIdx
	}
	m.src.Write(m.dst.Bytes())
	idxs = append(m.srcList, m.dstList...)
	return idxs, m.src.Bytes()
}

func (m *moMessageWriter) getHeader(c *Catalog) Message {
	b := new(bytes.Buffer)
	for k, v := range c.Header {
		b.WriteString(k + ": " + v)
	}
	return &SimpleMessage{
		Src: "",
		Dst: b.String(),
	}
}

func (m *moMessageWriter) append(msg Message) {
	src := ""
	dst := ""
	if ctx, err := msg.Context(); err == nil {
		src = ctx + "\x04"
	}
	switch t := msg.(type) {
	case *SimpleMessage:
		src += t.Src
		dst = t.Dst
	case *PluralMessage:
		src += strings.Join(t.Src, "\x00")
		dst = strings.Join(t.Dst, "\x00")
	}
	m.src.WriteString(src + "\x00")
	m.dst.WriteString(dst + "\x00")
	sLen, dLen := uint32(len(src)), uint32(len(dst))
	m.srcList = append(m.srcList, sLen, m.srcIdx)
	m.dstList = append(m.dstList, dLen, m.dstIdx)
	m.srcIdx += sLen + 1
	m.dstIdx += dLen + 1
}

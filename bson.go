// Copyright 2010, Evan Shaw. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bson implements BSON as specified at http://bsonspec.org/#/specification.
package bson

import (
	"bytes"
	"encoding/binary"
	"os"
)

// A Doc represents a BSON document.
type Doc map[string]interface{}

type ObjectId [12]byte

func (o *ObjectId) MarshalBSON() (byte, []byte, os.Error) { return 0x07, o[:], nil }

// Regexp represents a regular expression string. This structure is for encoding
// purposes only and will not be parsed or executed by this package.
type Regexp struct {
	// Expr is the regular expression string itself.
	Expr string
	// Options is the options string. Valid options are:
	//	i	Case insensitive matching
	//	l	Make \w, \W, etc. locale-dependent
	//	m	Multiline matching
	//	s	Dotall mode
	//	u	Make \w, \W, etc. match Unicode
	//	x	Verbose mode
	// Options must be specified in alphabetical order.
	Options string
}

func (r *Regexp) MarshalBSON() (byte, []byte, os.Error) {
	return 0x0B, []byte(r.Expr + "\x00" + r.Options + "\x00"), nil
}

type JavaScript string

func (j JavaScript) MarshalBSON() (byte, []byte, os.Error) {
	b := make([]byte, 4+len(j)+1)
	order.PutUint32(b, uint32(len(j)+1))
	copy(b[4:], []byte(string(j)))
	return 0x0D, b, nil
}

type JavaScriptWithScope struct {
	Code JavaScript
	Scope Doc
}

func (j *JavaScriptWithScope) MarshalBSON() (code byte, b []byte, err os.Error) {
	scope, err := Marshal(j.Scope)
	if err != nil {
		return
	}
	size := 4 + 4 + len(j.Code) + 1 + len(scope)
	b = make([]byte, 0, size)
	buf := bytes.NewBuffer(b)
	binary.Write(buf, order, uint32(size))
	binary.Write(buf, order, uint32(len(j.Code)+1))
	buf.WriteString(string(j.Code))
	buf.WriteByte(0)
	buf.Write(scope)
	b = b[:size]
	return 0x0F, b, nil
}

type Symbol string

func (s Symbol) MarshalBSON() (byte, []byte, os.Error) {
	b := make([]byte, 4+len(s)+1)
	order.PutUint32(b, uint32(len(s)+1))
	copy(b[4:], []byte(string(s)))
	return 0x0E, b, nil
}

type MaxKey struct{}

func (m MaxKey) MarshalBSON() (byte, []byte, os.Error) { return 0x7F, nil, nil }

type MinKey struct{}

func (m MinKey) MarshalBSON() (byte, []byte, os.Error) { return 0xFF, nil, nil }

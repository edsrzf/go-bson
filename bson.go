// Copyright 2010, Evan Shaw. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bson implements BSON as specified at http://bsonspec.org/#/specification.
package bson

import (
	"os"
)

// A Doc represents a BSON document.
type Doc map[string]interface{}

type ObjectId [12]byte

func (o *ObjectId) MarshalBSON() (byte, []byte, os.Error) { return 0x07, o[:], nil }

type Regexp string

func (r Regexp) MarshalBSON() (byte, []byte, os.Error) { return 0x0B, []byte(string(r)), nil }

type JavaScript string

func (j JavaScript) MarshalBSON() (byte, []byte, os.Error) { return 0x0D, []byte(string(j)), nil }

type Symbol string

func (s Symbol) MarshalBSON() (byte, []byte, os.Error) { return 0x0E, []byte(string(s)), nil }

type MaxKey struct{}

func (m MaxKey) MarshalBSON() (byte, []byte, os.Error) { return 0x7F, nil, nil }

type MinKey struct{}

func (m MinKey) MarshalBSON() (byte, []byte, os.Error) { return 0xFF, nil, nil }

// Copyright 2010, Evan Shaw. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bson

import (
	"bytes"
	"container/vector"
	"encoding/binary"
	"os"
	"time"
)

type DecodeError string

func (e DecodeError) String() string {
	return string(e)
}

type decodeState struct {
	*bytes.Buffer
}

func (d *decodeState) decodeDoc() (Doc, os.Error) {
	v := make(Doc)
	// discard total length; it doesn't help us
	d.Next(4)
	kind, err := d.ReadByte()
	for kind > 0 && err == nil {
		var key string
		key, err = d.readString()
		if err != nil {
			break
		}

		var val interface{}
		val, err = d.decodeElem(kind)
		if err != nil {
			break
		}
		v[key] = val
		kind, err = d.ReadByte()
	}
	return v, err
}

func (d *decodeState) readString() (string, os.Error) {
	b := d.Bytes()
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return "", DecodeError("unterminated string")
	}
	s := string(b[:i])
	// discard the bytes we used
	d.Next(i + 1)
	return s, nil
}

func (d *decodeState) decodeElem(kind byte) (interface{}, os.Error) {
	switch kind {
	case 0x01:
		// float
		var f float64
		err := binary.Read(d, order, &f)
		return f, err
	case 0x02:
		// string
		var l int32
		err := binary.Read(d, order, &l)
		b := make([]byte, l-1)
		d.Read(b)
		// discard the null terminator
		d.ReadByte()
		return string(b), err
	case 0x03:
		// document
		return d.decodeDoc()
	case 0x04:
		// array
		// byte length doesn't help
		d.Next(4)
		var s vector.Vector
		kind, err := d.ReadByte()
		for kind > 0 && err == nil {
			// discard key (always 1-byte string + null)
			d.Next(2)
			var el interface{}
			el, err = d.decodeElem(kind)
			s.Push(el)
			kind, err = d.ReadByte()
		}
		return []interface{}(s), err
	case 0x05:
		// binary data
		var l int32
		err := binary.Read(d, order, &l)
		// assuming binary/generic data; discarding actual kind
		d.ReadByte()
		b := make([]byte, l)
		d.Read(b)
		return b, err
	case 0x07:
		// object ID
		var o ObjectId
		_, err := d.Read(o[:])
		return &o, err
	case 0x08:
		// boolean
		b, err := d.ReadByte()
		return b != 0, err
	case 0x09:
		// time
		var t int64
		err := binary.Read(d, order, &t)
		return time.SecondsToLocalTime(t), err
	case 0x0A:
		// null
		return nil, nil
	case 0x0B:
		// regex
		r, err := d.readString()
		// discard options
		d.readString()
		return &Regexp{Expr: r}, err
	case 0x0D:
		// javascript
		var l int32
		err := binary.Read(d, order, &l)
		j := make([]byte, l-1)
		d.Read(j)
		d.ReadByte()
		return JavaScript(j), err
	case 0x0E:
		// symbol
		var l int32
		err := binary.Read(d, order, &l)
		s := make([]byte, l-1)
		d.Read(s)
		d.ReadByte()
		return Symbol(s), err
	case 0x10:
		// int32
		var i int32
		err := binary.Read(d, order, &i)
		return i, err
	case 0x12:
		// int64
		var i int64
		err := binary.Read(d, order, &i)
		return i, err
	case 0x7F:
		// max key
		return MaxKey{}, nil
	case 0xFF:
		// min key
		return MinKey{}, nil
	default:
		println(kind)
		panic("Unsupported type")
	}
	return nil, nil
}

func Unmarshal(data []byte) (Doc, os.Error) {
	d := &decodeState{bytes.NewBuffer(data)}
	return d.decodeDoc()
}

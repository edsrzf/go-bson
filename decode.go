// Copyright 2010, Evan Shaw. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bson

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) String() string {
	if e.Type == nil {
		return "bson: Unmarshal(nil)"
	}

	if _, ok := e.Type.(*reflect.PtrType); !ok {
		return "bson: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "bson: Unmarshal(nil " + e.Type.String() + ")"
}

type DecodeError string

func (e DecodeError) String() string {
	return string(e)
}

type decodeState struct {
	*bytes.Buffer
}

func (d *decodeState) error(err os.Error) {
	if err != nil {
		panic(err)
	}
}

func (d *decodeState) decodeDoc(v interface{}) {
	val := reflect.NewValue(v)
	unboxing: for {
		switch v := val.(type) {
		case *reflect.InterfaceValue:
			if v.IsNil() {
				d.error(&InvalidUnmarshalError{v.Type()})
			}
			val = v.Elem()
		case *reflect.MapValue:
			d.decodeMapDoc(v)
			break unboxing
		case *reflect.PtrValue:
			if v.IsNil() {
				d.error(&InvalidUnmarshalError{v.Type()})
			}
			val = v.Elem()
		case *reflect.StructValue:
			d.decodeStructDoc(v)
			break unboxing
		default:
			d.error(&InvalidUnmarshalError{val.Type()})
		}
	}
}

func (d *decodeState) decodeMapDoc(v *reflect.MapValue) {
	mapType := v.Type().(*reflect.MapType)
	_, stringKey := mapType.Key().(*reflect.StringType)
	if !stringKey {
		d.error(&InvalidUnmarshalError{v.Type()})
	}
	elType := mapType.Elem()

	kind, key, b := d.readChunk()
	for kind > 0 {
		val := d.decodeElem(kind, b)

		refVal := reflect.NewValue(val)
		var vType reflect.Type
		if refVal != nil {
			vType = refVal.Type()
		}
		if elType != vType {
			iVal := reflect.MakeZero(elType)
			iVal.SetValue(refVal)
			refVal = iVal
		}
		v.SetElem(reflect.NewValue(key), refVal)
		kind, key, b = d.readChunk()
	}
}

func (d *decodeState) decodeStructDoc(v *reflect.StructValue) {
	st := v.Type().(*reflect.StructType)

	kind, key, b := d.readChunk()
	for kind > 0 {
		val := d.decodeElem(kind, b)

		var fieldVal reflect.Value
		var f reflect.StructField
		found := false
		for i := 0; i < st.NumField(); i++ {
			f = st.Field(i)
			if f.Tag == key {
				found = true
				break
			}
		}
		if !found {
			f, found = st.FieldByName(key)
		}
		if !found {
			lowKey := strings.ToLower(key)
			f, found = st.FieldByNameFunc(func(s string) bool { return lowKey == strings.ToLower(s) })
		}
		if found {
			fieldVal = v.FieldByIndex(f.Index)
		} else {
			continue
		}

		refVal := reflect.NewValue(val)
		fieldVal.SetValue(refVal)

		kind, key, b = d.readChunk()
	}
}

func (d *decodeState) readCString() string {
	s, err := d.ReadString(0)
	d.error(err)
	return s[:len(s)-1]
}

func (d *decodeState) readString() string {
	var l int32
	err := binary.Read(d, order, &l)
	d.error(err)
	b := make([]byte, l)
	d.Read(b)
	// discard null terminator
	return string(b[:l - 1])
}

const (
	doubleNull = -1
	lengthEncodedMinus = -2
	lengthEncoded = -6
	lengthEncodedPlus = -7
)

var lengths = []int32 {
	elFloat: 8,
	elString: lengthEncoded,
	elDoc: lengthEncodedMinus,
	elArray: lengthEncodedMinus,
	elBinary: lengthEncodedPlus,
	elObjectID: 12,
	elBool: 1,
	elDatetime: 8,
	elNull: 0,
	elRegexp: doubleNull,
	elJavaScript: lengthEncoded,
	elSymbol: lengthEncoded,
	elJavaScope: lengthEncodedMinus,
	elInt32: 4,
	elInt64: 8,
	elMax: 0,
	elMin: 0,
}

func (d *decodeState) readChunk() (kind byte, key string, b []byte) {
	kind, err := d.ReadByte()
	d.error(err)
	if kind == 0 {
		return
	}

	key = d.readCString()
	switch n := lengths[kind]; {
	case n == doubleNull:
		s1, err := d.ReadString(0)
		d.error(err)
		s2, err := d.ReadString(0)
		d.error(err)
		b = []byte(s1 + s2)
	case n < 0:
		// length-encoded with a possible offset
		var l int32
		binary.Read(d, order, &l)
		b = make([]byte, l + (lengthEncoded - n))
		d.Read(b)
	case n == 0:
	default:
		b = make([]byte, n)
		d.Read(b)
	}
	return
}

func (d *decodeState) decodeElem(kind byte, b []byte) interface{} {
	switch kind {
	case elFloat:
		f := math.Float64frombits(order.Uint64(b))
		return f
	case elString:
		return string(b[:len(b)-1])
	case elDoc:
		m := make(map[string]interface{})
		d2 := &decodeState{bytes.NewBuffer(b)}
		d2.decodeDoc(m)
		return m
	case elArray:
		// byte length doesn't help
		d2 := &decodeState{bytes.NewBuffer(b)}
		var s []interface{}
		kind, _, b := d2.readChunk()
		for kind > 0 {
			var el interface{}
			el = d2.decodeElem(kind, b)
			s = append(s, el)
			kind, _, b = d2.readChunk()
		}
		return s
	case elBinary:
		// assuming binary/generic data; discarding actual kind
		// TODO: consider making a copy of this data so that we won't
		// be holding references to potentially large blocks of
		// memory
		return b[1:]
	case elObjectID:
		var o ObjectId
		copy(o[:], b)
		return &o
	case elBool:
		return b[0] != 0
	case elDatetime:
		t := int64(order.Uint64(b))
		return time.SecondsToUTC(t)
	case elNull:
		return nil
	case elRegexp:
		pos := bytes.IndexByte(b, 0)
		// TODO: consider copying
		r := string(b[:pos])
		// discard options
		return &Regexp{Expr: r}
	case elJavaScript:
		return &JavaScript{Code: string(b[:len(b)-1])}
	case elSymbol:
		return Symbol(b[:len(b)-1])
	case elJavaScope:
		d2 := &decodeState{bytes.NewBuffer(b)}
		code := d2.readString()
		// discard length
		d2.Next(4)
		scope := make(map[string]interface{})
		d2.decodeDoc(scope)
		return &JavaScript{code, scope}
	case elInt32:
		return int32(order.Uint32(b))
	case elInt64:
		return int64(order.Uint64(b))
	case elMax:
		return MaxKey{}
	case elMin:
		return MinKey{}
	default:
		panic("unsupported type")
	}
	return nil
}

func Unmarshal(data []byte, v interface{}) (err os.Error) {
	defer func() {
		if r := recover(); r != nil {
			switch rval := r.(type) {
			case runtime.Error:
				panic(r)
			case os.Error:
				err = rval
			case string:
				err = os.NewError(rval)
			default:
				panic(r)
			}
		}
	}()

	d := &decodeState{bytes.NewBuffer(data)}
	// discard doc length -- it doesn't help
	d.Next(4)
	d.decodeDoc(v)
	return
}

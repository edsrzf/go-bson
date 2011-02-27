// Copyright 2010, Evan Shaw. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bson

import (
	"bytes"
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
	b []byte
	r int
}

func (d *decodeState) error(err os.Error) {
	if err != nil {
		panic(err)
	}
}

func indirect(val reflect.Value) reflect.Value {
	for {
		switch v := val.(type) {
		case *reflect.InterfaceValue:
			if v.IsNil() {
				panic(&InvalidUnmarshalError{v.Type()})
			}
			val = v.Elem()
		case *reflect.PtrValue:
			if v.IsNil() {
				panic(&InvalidUnmarshalError{v.Type()})
			}
			val = v.Elem()
		default:
			return val
		}
	}
	panic("unreachable")
}

func (d *decodeState) decodeDoc(val reflect.Value) {
	val = indirect(val)
	switch v := val.(type) {
	case *reflect.MapValue:
		if v.IsNil() {
			mt := v.Type().(*reflect.MapType)
			v.Set(reflect.MakeMap(mt))
		}
		d.decodeMapDoc(v)
	case *reflect.StructValue:
		d.decodeStructDoc(v)
	default:
		d.error(&InvalidUnmarshalError{val.Type()})
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
		val := reflect.MakeZero(elType)
		d.decodeElem(kind, b, val)

		v.SetElem(reflect.NewValue(key), val)
		kind, key, b = d.readChunk()
	}
}

func (d *decodeState) decodeStructDoc(v *reflect.StructValue) {
	st := v.Type().(*reflect.StructType)

	kind, key, b := d.readChunk()
	for kind > 0 {
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
			d.decodeElem(kind, b, fieldVal)
		}

		kind, key, b = d.readChunk()
	}
}

func (d *decodeState) readCString() string {
	pos := bytes.IndexByte(d.b[d.r:], 0)
	if pos < 0 {
		panic("unterminated C string")
	}
	end := d.r + pos
	s := string(d.b[d.r:end])
	d.r = end + 1
	return s
}

func (d *decodeState) readString() string {
	l := int(order.Uint32(d.b[d.r:]))
	d.r += 4
	s := string(d.b[d.r : d.r+l-1])
	d.r += l
	return s
}

const (
	doubleNull         = -1
	lengthEncodedMinus = -2
	lengthEncoded      = -6
	lengthEncodedPlus  = -7
)

var lengths = []int{
	elFloat:      8,
	elString:     lengthEncoded,
	elDoc:        lengthEncodedMinus,
	elArray:      lengthEncodedMinus,
	elBinary:     lengthEncodedPlus,
	elObjectID:   12,
	elBool:       1,
	elDatetime:   8,
	elNull:       0,
	elRegexp:     doubleNull,
	elJavaScript: lengthEncoded,
	elSymbol:     lengthEncoded,
	elJavaScope:  lengthEncodedMinus,
	elInt32:      4,
	elInt64:      8,
	elMax:        0,
	elMin:        0,
}

func (d *decodeState) readChunk() (kind byte, key string, b []byte) {
	kind = d.b[d.r]
	d.r++
	if kind == 0 {
		return
	}

	key = d.readCString()
	switch n := lengths[kind]; {
	case n == doubleNull:
		start := d.r
		pos := bytes.IndexByte(d.b[d.r:], 0)
		d.r += pos + 1
		pos = bytes.IndexByte(d.b[d.r:], 0)
		end := d.r + pos + 1
		b = d.b[start:end]
		d.r = end
	case n < 0:
		// length-encoded with a possible offset
		l := int(order.Uint32(d.b[d.r:]))
		d.r += 4
		end := d.r + l + lengthEncoded - n
		b = d.b[d.r:end]
		d.r = end
	case n == 0:
	default:
		end := d.r + n
		b = d.b[d.r:end]
		d.r = end
	}
	return
}

func (d *decodeState) decodeElem(kind byte, b []byte, val reflect.Value) {
	iv, ok := val.(*reflect.InterfaceValue)
	if ok {
		iv.Set(reflect.NewValue(d.decodeElemInterface(kind, b)))
		return
	}

	switch kind {
	case elFloat:
		f := math.Float64frombits(order.Uint64(b))
		fv, ok := val.(*reflect.FloatValue)
		if !ok {
			goto error
		}
		fv.Set(f)
	case elString:
		s := string(b[:len(b)-1])
		sv, ok := val.(*reflect.StringValue)
		if !ok {
			goto error
		}
		sv.Set(s)
	case elDoc:
		d2 := &decodeState{b: b}
		d2.decodeDoc(val)
	case elArray:
		// byte length doesn't help
		sv, ok := val.(*reflect.SliceValue)
		if !ok {
			goto error
		}
		elType := sv.Type().(*reflect.SliceType).Elem()
		d2 := &decodeState{b: b}
		kind, _, b := d2.readChunk()
		for kind > 0 {
			el := reflect.MakeZero(elType)
			d2.decodeElem(kind, b, el)
			sv = reflect.Append(sv, el)
			kind, _, b = d2.readChunk()
		}
	case elBinary:
		sv, ok := val.(*reflect.SliceValue)
		if !ok {
			goto error
		}
		bv := reflect.NewValue(b[1:])
		sliceType := sv.Type().(*reflect.SliceType)
		if sliceType != bv.Type() {
			goto error
		}
		sv.SetValue(bv)
	case elObjectID:
		var o ObjectID
		copy(o[:], b)
		ov := reflect.NewValue(&o)
		if val.Type() != ov.Type() {
			goto error
		}
		val.SetValue(ov)
	case elBool:
		bv := reflect.NewValue(b[0] != 0)
		if val.Type() != bv.Type() {
			goto error
		}
		val.SetValue(bv)
	case elDatetime:
		t := int64(order.Uint64(b))
		tv := reflect.NewValue(time.SecondsToUTC(t))
		if val.Type() != tv.Type() {
			goto error
		}
		val.SetValue(tv)
	case elNull:
		type nillable interface {
			IsNil() bool
		}
		_, canNil := val.(nillable)
		if !canNil {
			goto error
		}
		val.SetValue(reflect.MakeZero(val.Type()))
	case elRegexp:
		pos := bytes.IndexByte(b, 0)
		r := string(b[:pos])
		// discard options
		rv := reflect.NewValue(&Regexp{Expr: r})
		if val.Type() != rv.Type() {
			goto error
		}
		val.SetValue(rv)
	case elJavaScript:
		j := &JavaScript{Code: string(b[:len(b)-1])}
		jv := reflect.NewValue(j)
		if val.Type() != jv.Type() {
			goto error
		}
		val.SetValue(jv)
	case elSymbol:
		s := string(b[:len(b)-1])
		sv, ok := val.(*reflect.StringValue)
		if !ok {
			goto error
		}
		sv.Set(s)
	case elJavaScope:
		d2 := &decodeState{b: b}
		code := d2.readString()
		// discard length
		d2.r += 4
		scope := make(map[string]interface{})
		d2.decodeDoc(reflect.NewValue(scope))
		j := &JavaScript{code, scope}
		jv := reflect.NewValue(j)
		if val.Type() != jv.Type() {
			goto error
		}
		val.SetValue(jv)
	case elInt32:
		n := order.Uint32(b)
		switch v := val.(type) {
		case *reflect.FloatValue:
			n := float64(n)
			v.Set(n)
		case *reflect.IntValue:
			n := int64(n)
			if v.Overflow(n) {
				goto error
			}
			v.Set(n)
		case *reflect.UintValue:
			n := uint64(n)
			if v.Overflow(n) {
				goto error
			}
			v.Set(n)
		}
	case elInt64:
		n := order.Uint64(b)
		switch v := val.(type) {
		case *reflect.FloatValue:
			n := float64(n)
			v.Set(n)
		case *reflect.IntValue:
			n := int64(n)
			if v.Overflow(n) {
				goto error
			}
			v.Set(n)
		case *reflect.UintValue:
			n := uint64(n)
			if v.Overflow(n) {
				goto error
			}
			v.Set(n)
		}
	case elMax:
		m := MaxKey{}
		mv := reflect.NewValue(m)
		if val.Type() != mv.Type() {
			goto error
		}
		val.SetValue(mv)
	case elMin:
		m := MinKey{}
		mv := reflect.NewValue(m)
		if val.Type() != mv.Type() {
			goto error
		}
		val.SetValue(mv)
	}
	return

error:
	panic("invalid type for decoding")
}

func (d *decodeState) decodeElemInterface(kind byte, b []byte) interface{} {
	switch kind {
	case elFloat:
		f := math.Float64frombits(order.Uint64(b))
		return f
	case elString:
		return string(b[:len(b)-1])
	case elDoc:
		m := make(map[string]interface{})
		d2 := &decodeState{b: b}
		d2.decodeDoc(reflect.NewValue(m))
		return m
	case elArray:
		// byte length doesn't help
		d2 := &decodeState{b: b}
		var s []interface{}
		kind, _, b := d2.readChunk()
		for kind > 0 {
			el := d2.decodeElemInterface(kind, b)
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
		var o ObjectID
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
		r := string(b[:pos])
		// discard options
		return &Regexp{Expr: r}
	case elJavaScript:
		return &JavaScript{Code: string(b[:len(b)-1])}
	case elSymbol:
		return Symbol(b[:len(b)-1])
	case elJavaScope:
		d2 := &decodeState{b: b}
		code := d2.readString()
		// discard length
		d2.r += 4
		scope := make(map[string]interface{})
		d2.decodeDoc(reflect.NewValue(scope))
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

	// discard doc length -- it doesn't help
	d := &decodeState{b: data[4:]}
	d.decodeDoc(reflect.NewValue(v))
	return
}

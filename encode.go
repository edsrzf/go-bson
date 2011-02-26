// Copyright 2010, Evan Shaw. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bson

import (
	"bytes"
	"encoding/binary"
	"os"
	"reflect"
	"strconv"
	"time"
)

var order = binary.LittleEndian

type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) String() string {
	return "bson: unsupported type: " + e.Type.String()
}

type Marshaler interface {
	MarshalBSON() (byte, []byte, os.Error)
}

type encodeState struct {
	*bytes.Buffer
}

func (e *encodeState) marshal(v interface{}) (err os.Error) {
	// write a value just to reserve some space
	e.Write([]byte{0, 0, 0, 0})
	rval := reflect.NewValue(v)
unboxing:
	for {
		switch v := rval.(type) {
		case *reflect.InterfaceValue:
			rval = v.Elem()
		case *reflect.MapValue:
			keys := v.Keys()
			for _, rkey := range keys {
				key := rkey.Interface().(string)
				val := v.Elem(rkey)
				e.writeKeyVal(key, val.Interface())
			}
			break unboxing
		case *reflect.PtrValue:
			rval = v.Elem()
		case *reflect.StructValue:
			t := rval.Type().(*reflect.StructType)
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				/*if field.Anonymous {
					continue
				}*/
				key := field.Tag
				if key == "" {
					key = field.Name
				}
				e.writeKeyVal(key, v.Field(i).Interface())
			}
			break unboxing
		default:
			panic("invalid type")
		}
	}
	// terminate the element list
	e.WriteByte(0x00)
	b := e.Bytes()
	e.Buffer = bytes.NewBuffer(b)
	l := int32(len(b))
	toWrite := []byte{byte(l), byte(l >> 8), byte(l >> 16), byte(l >> 24)}
	copy(e.Bytes(), toWrite)
	return nil
}

func (e *encodeState) writeBegin(kind byte, key string) os.Error {
	e.WriteByte(kind)
	e.WriteString(key)
	return e.WriteByte(0x00)
}

func (e *encodeState) writeKeyVal(key string, val interface{}) os.Error {
	switch v := val.(type) {
	case Marshaler:
		kind, b, err := v.MarshalBSON()
		if err != nil {
			return err
		}
		e.writeBegin(kind, key)
		_, err = e.Write(b)
		return err
	case float32:
		e.writeBegin(elFloat, key)
		return binary.Write(e, order, float64(v))
	case float64:
		e.writeBegin(elFloat, key)
		return binary.Write(e, order, v)
	case string:
		e.writeBegin(elString, key)
		l := int32(len(v)) + 1
		binary.Write(e, order, l)
		e.WriteString(v)
		return e.WriteByte(0x00)
	case []byte:
		e.writeBegin(elBinary, key)
		l := int32(len(v))
		binary.Write(e, order, l)
		// binary/generic subtype
		e.WriteByte(0x00)
		_, err := e.Write(v)
		return err
	case bool:
		e.writeBegin(elBool, key)
		if v {
			return e.WriteByte(0x01)
		}
		return e.WriteByte(0x00)
	case *time.Time:
		e.writeBegin(elDatetime, key)
		return binary.Write(e, order, v.Seconds())
	case nil:
		return e.writeBegin(elNull, key)
	case int8:
		return e.writeInt32(key, int32(v))
	case uint8:
		return e.writeInt32(key, int32(v))
	case int16:
		return e.writeInt32(key, int32(v))
	case uint16:
		return e.writeInt32(key, int32(v))
	case int32:
		return e.writeInt32(key, v)
	case uint32:
		return e.writeInt32(key, int32(v))
	case int:
		return e.writeInt64(key, int64(v))
	case uint:
		return e.writeInt64(key, int64(v))
	case int64:
		return e.writeInt64(key, v)
	case uint64:
		return e.writeInt64(key, int64(v))
	}

	return e.writeReflect(key, reflect.NewValue(val))
}

func (e *encodeState) writeInt32(key string, val int32) os.Error {
	e.writeBegin(elInt32, key)
	return binary.Write(e, order, val)
}

func (e *encodeState) writeInt64(key string, val int64) os.Error {
	e.writeBegin(elInt64, key)
	return binary.Write(e, order, val)
}

func (e *encodeState) writeReflect(key string, val reflect.Value) os.Error {
	switch v := val.(type) {
	case *reflect.MapValue:
		e.writeBegin(elDoc, key)
		keys := v.Keys()
		e2 := &encodeState{bytes.NewBuffer(nil)}
		for i, k := range keys {
			e2.writeKeyVal(keys[i].Interface().(string), v.Elem(k).Interface())
		}
		b := e2.Bytes()
		binary.Write(e, order, int32(len(b)+5))
		e.Write(b)
		return e.WriteByte(0x00)
	case reflect.ArrayOrSliceValue:
		e.writeBegin(elArray, key)
		l := v.Len()
		e2 := &encodeState{bytes.NewBuffer(nil)}
		for i := 0; i < l; i++ {
			e2.writeKeyVal(strconv.Itoa(i), v.Elem(i).Interface())
		}
		b := e2.Bytes()
		binary.Write(e, order, int32(len(b)+5))
		e.Write(b)
		return e.WriteByte(0x00)
	case *reflect.PtrValue:
		return e.writeKeyVal(key, v.Elem().Interface())
	case *reflect.StructValue:
		e.writeBegin(elDoc, key)
		t := v.Type().(*reflect.StructType)
		l := t.NumField()
		for i := 0; i < l; i++ {
			field := t.Field(i)
			name := field.Tag
			if name == "" {
				name = field.Name
			}
			e.writeKeyVal(name, v.Field(i).Interface())
		}
		return e.WriteByte(0x00)
	}
	return &UnsupportedTypeError{val.Type()}
}

// Marshal returns the BSON encoding of v.
//
// Marshal traverses v recursively. If an encountered value implements the
// Marshaler interface, Marshal calls its MarshalBSON method to produce BSON.
//
// Otherwise, Marshal uses the following type-dependent default encodings:
//
// Boolean values encode as BSON booleans.
//
// Floating point values encode as BSON doubles.
//
// 8-bit, 16-bit, and 32-bit integer values encode as BSON int32 values.
//
// 64-bit and implementation-sized integer values encode as BSON int64 values.
//
// String values encode as BSON strings.
//
// Byte slices encode as BSON generic binary data. Other types of BSON binary
// data are not supported.
//
// Array and slice values other than byte slices encode as BSON arrays.
//
// Struct values encode as BSON embedded documents. Each field becomes a member
// of the object. By default the object's key name is the struct field name
// converted to lower case. If the struct has a tag, that will be used instead.
//
// Map values encode as BSON embedded documents. The map's key type must be a
// string; the object keys are used directly as map keys.
//
// Pointer values encode the value pointed to. A nil pointer encodes as the
// BSON null object.
//
// Interface values encode as the value contained in the interface. A nil
// interface value encodes as the null BSON object.
//
// Channel, complex, and function values cannot be encoded in BSON. Attempting
// to encode such a value causes Marshal to return an InvalidTypeError.
//
// BSON cannot represent cyclic data structures and Marshal cannot handle them.
// Passing cyclic structures to Marshal will result in an infinite recursion.
func Marshal(v interface{}) ([]byte, os.Error) {
	e := &encodeState{bytes.NewBuffer(nil)}
	err := e.marshal(v)
	if err != nil {
		return nil, err
	}
	return e.Bytes(), nil
}

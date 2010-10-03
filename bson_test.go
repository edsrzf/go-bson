// Copyright 2010, Evan Shaw. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bson

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

type bsonTest struct {
	doc  Doc
	bson []byte
}

var bsonTests = []bsonTest{
	bsonTest{nil, []byte("\x05\x00\x00\x00\x00")},
	bsonTest{Doc{}, []byte("\x05\x00\x00\x00\x00")},
	bsonTest{Doc{"test": float64(3.14159)}, []byte("\x13\x00\x00\x00\x01test\x00\x6E\x86\x1B\xF0\xF9\x21\x09\x40\x00")},
	bsonTest{Doc{"hello": "world"}, []byte("\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00")},
	bsonTest{Doc{"test": Doc{}}, []byte("\x10\x00\x00\x00\x03test\x00\x05\x00\x00\x00\x00\x00")},
	bsonTest{Doc{"test": []byte{0xDE, 0xAD, 0xBE, 0xEF}}, []byte("\x14\x00\x00\x00\x05test\x00\x04\x00\x00\x00\x00\xDE\xAD\xBE\xEF\x00")},
	bsonTest{Doc{"test": &ObjectId{0x4C, 0x9B, 0x8F, 0xB4, 0xA3, 0x82, 0xAA, 0xFE, 0x17, 0xC8, 0x6E, 0x63}}, []byte("\x17\x00\x00\x00\x07test\x00\x4C\x9B\x8F\xB4\xA3\x82\xAA\xFE\x17\xC8\x6E\x63\x00")},
	bsonTest{Doc{"test": true}, []byte("\x0C\x00\x00\x00\x08test\x00\x01\x00")},
	bsonTest{Doc{"test": false}, []byte("\x0C\x00\x00\x00\x08test\x00\x00\x00")},
	bsonTest{Doc{"test": &time.Time{2008, 9, 17, 20, 4, 26, time.Wednesday, 0, "UTC"}}, []byte("\x13\x00\x00\x00\x09test\x00\xCA\x62\xD1\x48\x00\x00\x00\x00\x00")},
	bsonTest{Doc{"test": nil}, []byte("\x0B\x00\x00\x00\x0Atest\x00\x00")},
	bsonTest{Doc{"test": &Regexp{".*", ""}}, []byte("\x0F\x00\x00\x00\x0Btest\x00.*\x00\x00\x00")},
	bsonTest{Doc{"test": JavaScript("function foo(){};")}, []byte("\x21\x00\x00\x00\x0Dtest\x00\x12\x00\x00\x00function foo(){};\x00\x00")},
	bsonTest{Doc{"test": Symbol("aSymbol")}, []byte("\x17\x00\x00\x00\x0Etest\x00\x08\x00\x00\x00aSymbol\x00\x00")},
	bsonTest{Doc{"test": &JavaScriptWithScope{"function foo(){};", Doc{"hello": "world"}}}, []byte("\x3B\x00\x00\x00\x0Ftest\x00\x30\x00\x00\x00\x12\x00\x00\x00function foo(){};\x00\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00\x00")},
	bsonTest{Doc{"test": int32(10)}, []byte("\x0F\x00\x00\x00\x10test\x00\x0A\x00\x00\x00\x00")},
	bsonTest{Doc{"test": int64(256)}, []byte("\x13\x00\x00\x00\x12test\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00")},
	bsonTest{Doc{"test": MaxKey{}}, []byte("\x0B\x00\x00\x00\x7Ftest\x00\x00")},
	bsonTest{Doc{"test": MinKey{}}, []byte("\x0B\x00\x00\x00\xFFtest\x00\x00")},
	bsonTest{Doc{"BSON": []interface{}{"awesome", float64(5.05), int32(1986)}}, []byte("\x31\x00\x00\x00\x04BSON\x00\x26\x00\x00\x00\x02\x30\x00\x08\x00\x00\x00awesome\x00\x01\x31\x00\x33\x33\x33\x33\x33\x33\x14\x40\x10\x32\x00\xC2\x07\x00\x00\x00\x00")},
}

func TestMarshal(t *testing.T) {
	for i, test := range bsonTests {
		bson, err := Marshal(test.doc)
		if err != nil {
			t.Errorf("#%d error: %s", i, err.String())
		}
		if !bytes.Equal(bson, test.bson) {
			t.Errorf("#%d expected\n%q\ngot\n%q", i, test.bson, bson)
		}
	}
}

func TestUnmarshal(t *testing.T) {
	for i, test := range bsonTests {
		doc, err := Unmarshal(test.bson)
		if err != nil {
			t.Errorf("#%d error: %s", i, err.String())
		}
		if !reflect.DeepEqual(test.doc, doc) {
			t.Errorf("#%d expected\n%v\ngot\n%v", i, test.doc, doc)
		}
	}
}

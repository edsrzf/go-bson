// Copyright 2010, Evan Shaw. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bson

import (
	"bytes"
	"reflect"
	"testing"
)

type marshalTest struct {
	doc  Doc
	bson []byte
}

var marshalTests = []marshalTest{
	marshalTest{Doc{}, []byte("\x05\x00\x00\x00\x00")},
	marshalTest{Doc{"hello": "world"}, []byte("\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00")},
	marshalTest{Doc{"BSON": []interface{}{"awesome", float64(5.05), int32(1986)}}, []byte("\x31\x00\x00\x00\x04BSON\x00\x26\x00\x00\x00\x02\x30\x00\x08\x00\x00\x00awesome\x00\x01\x31\x00\x33\x33\x33\x33\x33\x33\x14\x40\x10\x32\x00\xC2\x07\x00\x00\x00\x00")},
}

func TestMarshal(t *testing.T) {
	for i, test := range marshalTests {
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
	for i, test := range marshalTests {
		doc, err := Unmarshal(test.bson)
		if err != nil {
			t.Errorf("#%d error: %s", i, err.String())
		}
		if !reflect.DeepEqual(test.doc, doc) {
			t.Errorf("#%d expected\n%v\ngot\n%v", i, test.doc, doc)
		}
	}
}

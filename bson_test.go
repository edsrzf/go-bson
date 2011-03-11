// Copyright 2010, Evan Shaw. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bson

import (
	"bytes"
	//"fmt"
	"reflect"
	"testing"
	"time"
)

var bsonTests = []struct {
	doc  interface{}
	bson []byte
}{
	{map[string]interface{}{}, []byte("\x05\x00\x00\x00\x00")},
	{&struct{}{}, []byte("\x05\x00\x00\x00\x00")},
	{map[string]interface{}{"test": 3.14159}, []byte("\x13\x00\x00\x00\x01test\x00\x6E\x86\x1B\xF0\xF9\x21\x09\x40\x00")},
	{map[string]float64{"e": 2.71828}, []byte("\x10\x00\x00\x00\x01e\x00\x90\xf7\xaa\x95\t\xbf\x05@\x00")},
	{&struct {
		E float64 "e"
	}{2.71828},
		[]byte("\x10\x00\x00\x00\x01e\x00\x90\xf7\xaa\x95\t\xbf\x05@\x00")},
	{map[string]interface{}{"hello": "world"}, []byte("\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00")},
	{map[string]string{"it's a": "string"}, []byte("\x18\x00\x00\x00\x02it's a\x00\x07\x00\x00\x00string\x00\x00")},
	{&struct {
		S string "it's a"
	}{"string"},
		[]byte("\x18\x00\x00\x00\x02it's a\x00\x07\x00\x00\x00string\x00\x00")},
	{map[string]interface{}{"test": map[string]interface{}{}}, []byte("\x10\x00\x00\x00\x03test\x00\x05\x00\x00\x00\x00\x00")},
	{&struct{ Test map[string]int }{}, []byte("\x10\x00\x00\x00\x03Test\x00\x05\x00\x00\x00\x00\x00")},
	{&struct{ Test map[string]string }{map[string]string{"inner": "doc"}}, []byte("\x1F\x00\x00\x00\x03Test\x00\x14\x00\x00\x00\x02inner\x00\x04\x00\x00\x00doc\x00\x00\x00")},
	{map[string]interface{}{"test": []byte{0xDE, 0xAD, 0xBE, 0xEF}}, []byte("\x14\x00\x00\x00\x05test\x00\x04\x00\x00\x00\x00\xDE\xAD\xBE\xEF\x00")},
	{&struct{ Test []byte }{[]byte{0xDE, 0xAD, 0xBE, 0xEF}}, []byte("\x14\x00\x00\x00\x05Test\x00\x04\x00\x00\x00\x00\xDE\xAD\xBE\xEF\x00")},
	{map[string]interface{}{"test": &ObjectID{0x4C, 0x9B, 0x8F, 0xB4, 0xA3, 0x82, 0xAA, 0xFE, 0x17, 0xC8, 0x6E, 0x63}}, []byte("\x17\x00\x00\x00\x07test\x00\x4C\x9B\x8F\xB4\xA3\x82\xAA\xFE\x17\xC8\x6E\x63\x00")},
	{&struct{ Test *ObjectID }{&ObjectID{0x4C, 0x9B, 0x8F, 0xB4, 0xA3, 0x82, 0xAA, 0xFE, 0x17, 0xC8, 0x6E, 0x63}}, []byte("\x17\x00\x00\x00\x07Test\x00\x4C\x9B\x8F\xB4\xA3\x82\xAA\xFE\x17\xC8\x6E\x63\x00")},
	{map[string]interface{}{"test": true}, []byte("\x0C\x00\x00\x00\x08test\x00\x01\x00")},
	{map[string]interface{}{"test": false}, []byte("\x0C\x00\x00\x00\x08test\x00\x00\x00")},
	{map[string]bool{"true": true, "false": false}, []byte("\x14\x00\x00\x00\x08false\x00\x00\x08true\x00\x01\x00")},
	{&struct{ False, True bool }{false, true}, []byte("\x14\x00\x00\x00\x08False\x00\x00\x08True\x00\x01\x00")},
	{map[string]interface{}{"test": &time.Time{2008, 9, 17, 20, 4, 26, time.Wednesday, 0, "UTC"}}, []byte("\x13\x00\x00\x00\x09test\x00\xCA\x62\xD1\x48\x00\x00\x00\x00\x00")},
	{&struct{ Test *time.Time }{&time.Time{2008, 9, 17, 20, 4, 26, time.Wednesday, 0, "UTC"}}, []byte("\x13\x00\x00\x00\x09Test\x00\xCA\x62\xD1\x48\x00\x00\x00\x00\x00")},
	{map[string]interface{}{"test": nil}, []byte("\x0B\x00\x00\x00\x0Atest\x00\x00")},
	{&struct{ Test interface{} }{nil}, []byte("\x0B\x00\x00\x00\x0ATest\x00\x00")},
	{&struct{ Test *int }{nil}, []byte("\x0B\x00\x00\x00\x0ATest\x00\x00")},
	{map[string]interface{}{"test": &Regexp{".*", ""}}, []byte("\x0F\x00\x00\x00\x0Btest\x00.*\x00\x00\x00")},
	{&struct{ Test *Regexp }{&Regexp{".*", ""}}, []byte("\x0F\x00\x00\x00\x0BTest\x00.*\x00\x00\x00")},
	{map[string]interface{}{"test": &JavaScript{Code: "function foo(){};"}}, []byte("\x21\x00\x00\x00\x0Dtest\x00\x12\x00\x00\x00function foo(){};\x00\x00")},
	{&struct{ Test *JavaScript }{&JavaScript{Code: "function foo(){};"}}, []byte("\x21\x00\x00\x00\x0DTest\x00\x12\x00\x00\x00function foo(){};\x00\x00")},
	{map[string]interface{}{"test": Symbol("aSymbol")}, []byte("\x17\x00\x00\x00\x0Etest\x00\x08\x00\x00\x00aSymbol\x00\x00")},
	{&struct{ Test Symbol }{"aSymbol"}, []byte("\x17\x00\x00\x00\x0ETest\x00\x08\x00\x00\x00aSymbol\x00\x00")},
	{map[string]interface{}{"test": &JavaScript{"function foo(){};", map[string]interface{}{"hello": "world"}}}, []byte("\x3B\x00\x00\x00\x0Ftest\x00\x30\x00\x00\x00\x12\x00\x00\x00function foo(){};\x00\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00\x00")},
	{&struct{ Test *JavaScript }{&JavaScript{"function foo(){};", map[string]interface{}{"hello": "world"}}}, []byte("\x3B\x00\x00\x00\x0FTest\x00\x30\x00\x00\x00\x12\x00\x00\x00function foo(){};\x00\x16\x00\x00\x00\x02hello\x00\x06\x00\x00\x00world\x00\x00\x00")},
	{map[string]interface{}{"test": int32(10)}, []byte("\x0F\x00\x00\x00\x10test\x00\x0A\x00\x00\x00\x00")},
	{map[string]int32{"a": 1, "b": 2, "c": 3}, []byte("\x1a\x00\x00\x00\x10a\x00\x01\x00\x00\x00\x10c\x00\x03\x00\x00\x00\x10b\x00\x02\x00\x00\x00\x00")},
	{&struct{ A, B, C int32 }{1, 2, 3}, []byte("\x1a\x00\x00\x00\x10A\x00\x01\x00\x00\x00\x10B\x00\x02\x00\x00\x00\x10C\x00\x03\x00\x00\x00\x00")},
	{map[string]interface{}{"test": int64(256)}, []byte("\x13\x00\x00\x00\x12test\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00")},
	{&struct{ Test int64 }{256}, []byte("\x13\x00\x00\x00\x12Test\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00")},
	{map[string]interface{}{"test": MaxKey{}}, []byte("\x0B\x00\x00\x00\x7Ftest\x00\x00")},
	{&struct{ Test MaxKey }{}, []byte("\x0B\x00\x00\x00\x7FTest\x00\x00")},
	{map[string]interface{}{"test": MinKey{}}, []byte("\x0B\x00\x00\x00\xFFtest\x00\x00")},
	{&struct{ Test MinKey }{}, []byte("\x0B\x00\x00\x00\xFFTest\x00\x00")},
	{map[string]interface{}{"BSON": []interface{}{"awesome", float64(5.05), int32(1986)}}, []byte("\x31\x00\x00\x00\x04BSON\x00\x26\x00\x00\x00\x02\x30\x00\x08\x00\x00\x00awesome\x00\x01\x31\x00\x33\x33\x33\x33\x33\x33\x14\x40\x10\x32\x00\xC2\x07\x00\x00\x00\x00")},
	{map[string]interface{}{"BSON": []interface{}{int64(22055360), int64(12688462), int64(212446583), int64(37455565), int64(73465456),
		int64(17133954), int64(14786502), int64(51854974), int64(71727795),
		int64(20146901), int64(167890598)}},
		[]byte("\x8a\x00\x00\x00\x04BSON\x00\x7f\x00\x00\x00\x120\x00\xc0\x89P\x01\x00\x00\x00\x00\x121\x00N\x9c\xc1\x00\x00\x00\x00\x00\x122\x00w\xad\xa9\f\x00\x00\x00\x00\x123\x00\u0346;\x02\x00\x00\x00\x00\x124\x00p\xfe`\x04\x00\x00\x00\x00\x125\x00\x82q\x05\x01\x00\x00\x00\x00\x126\x00\u019f\xe1\x00\x00\x00\x00\x00\x127\x00~>\x17\x03\x00\x00\x00\x00\x128\x00\xb3zF\x04\x00\x00\x00\x00\x129\x00\xd5j3\x01\x00\x00\x00\x00\x1210\x00\xa6\xce\x01\n\x00\x00\x00\x00\x00\x00")},
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
		var doc interface{}
		switch t := reflect.Typeof(test.doc).(type) {
		case *reflect.MapType:
			doc = reflect.MakeMap(t).Interface()
		case *reflect.PtrType:
			// pointer to struct
			ptr := reflect.MakeZero(t).(*reflect.PtrValue)
			val := reflect.MakeZero(t.Elem())
			ptr.PointTo(val)
			doc = ptr.Interface()
		}
		err := Unmarshal(test.bson, doc)
		if err != nil {
			t.Errorf("#%d error: %s", i, err.String())
		}
		if !reflect.DeepEqual(test.doc, doc) {
			t.Errorf("#%d expected\n%+v\ngot\n%+v", i, test.doc, doc)
		}
	}
}

func BenchmarkLargeMapEncode(b *testing.B) {
	b.StopTimer()
	media := map[string]interface{} {
		"uri": "http://javaone.com/keynote.mpg",
		"title": "Javaone Keynote",
		"width": 640,
		"height": 480,
		"format": "video/mpg4",
		"duration": 18000000,
		"size": 58982400,
		"bitrate": 262144,
		"persons": []string{"Bill Gates", "Steve Jobs"},
		"player": "JAVA",
		"copyright": nil,
	}
	images := []map[string]interface{} {
		{
			"uri": "http://javaone.com/keynote_large.jpg",
			"title": "Javaone Keynote",
			"width": 1024,
			"height": 768,
			"large": true,
		},
		{
			"uri": "http://javaone.com/keynote_small.jpg",
			"title": "Javaone Keynote",
			"width": 320,
			"height": 240,
			"large": false,
		},
	}
	doc := map[string]interface{}{"media": media, "images": images}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(doc)
		if err != nil {
			panic(err.String())
		}
	}
}

/*func BenchmarkSmallMapEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
	}
}*/

type mediaType struct {
	Uri string "uri"
	Title string "title"
	W int "width"
	H int "height"
	Format string "format"
	Dur int "duration"
	Size int "size"
	Bitrate int "bitrate"
	Persons []string "persons"
	Player string "player"
	Copyright *string "copyright"
}
type imageType struct {
	Uri string "uri"
	Title string "title"
	W int "width"
	H int "height"
	Large bool "large"
}
type docType struct {
	Media mediaType "media"
	Images []imageType "images"
}

func BenchmarkLargeStructEncode(b *testing.B) {
	media := mediaType{
		"http://javaone.com/keynote.mpg",
		"Javaone Keynote",
		640, 480,
		"video/mpg4",
		18000000,
		58982400,
		262144,
		[]string{"Bill Gates", "Steve Jobs"},
		"JAVA", nil,
	}
	images := []imageType{
		{
			"http://javaone.com/keynote_large.jpg",
			"Javaone Keynote",
			1024, 768, true,
		},
		{
			"http://javaone.com/keynote_small.jpg",
			"Javaone Keynote",
			320, 240, false,
		},
	}
	doc := &docType{media, images}
	for i := 0; i < b.N; i++ {
		_, err := Marshal(doc)
		if err != nil {
			panic(err.String())
		}
	}
}

/*func BenchmarkSmallStructEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
	}
}*/

var encodedBson = []byte("\xe2\x01\x00\x00\x03media\x00\xe9\x00\x00\x00\x02format\x00\v\x00\x00\x00video/mpg4\x00\x10bitrate\x00\x00\x00\x84\x03\ncopyright\x00\x02player\x00\x05\x00\x00\x00JAVA\x00\x10height\x00\xe0\x01\x00\x00\x10width\x00\x80\x02\x00\x00\x02uri\x00\x1f\x00\x00\x00http://javaone.com/keynote.mpg\x00\x02title\x00\x10\x00\x00\x00Javaone Keynote\x00\x10duration\x00\x80\xa8\x12\x01\x04persons\x00)\x00\x00\x00\x020\x00\v\x00\x00\x00Bill Gates\x00\x021\x00\v\x00\x00\x00Steve Jobs\x00\x00\x10size\x00\x00\x00\x84\x03\x00\x04images\x00\xe5\x00\x00\x00\x030\x00m\x00\x00\x00\blarge\x00\x01\x10height\x00\x00\x03\x00\x00\x10width\x00\x00\x04\x00\x00\x02uri\x00%\x00\x00\x00http://javaone.com/keynote_large.jpg\x00\x02title\x00\x10\x00\x00\x00Javaone Keynote\x00\x00\x031\x00m\x00\x00\x00\blarge\x00\x00\x10height\x00\xf0\x00\x00\x00\x10width\x00@\x01\x00\x00\x02uri\x00%\x00\x00\x00http://javaone.com/keynote_small.jpg\x00\x02title\x00\x10\x00\x00\x00Javaone Keynote\x00\x00\x00\x00")

func BenchmarkLargeMapDecode(b *testing.B) {
	doc := map[string]interface{}{}
	for i := 0; i < b.N; i++ {
		err := Unmarshal(encodedBson, doc)
		if err != nil {
			panic(err.String())
		}
	}
}

func BenchmarkLargeStructDecode(b *testing.B) {
	var doc docType
	for i := 0; i < b.N; i++ {
		err := Unmarshal(encodedBson, &doc)
		if err != nil {
			panic(err.String())
		}
	}
}

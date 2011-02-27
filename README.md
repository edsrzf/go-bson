Go-BSON
=======

Go-BSON is a [BSON](http://bsonspec.org/) encoder and decoder package for the [Go
programming language](http://golang.org/). It is primarily intended to be used
with [Mongogo](mongogo).

This project is still in development. It's been tested on Arch and Ubuntu Linux for
the amd64 architecture, but there's no reason it shouldn't work on other platforms
as well.

Dependencies
------------

Go-BSON works with Go release 2011-01-20 or newer, barring any recent language or
library changes.

Usage
-----

Go-BSON has two main functions: one that encodes and one that decodes.

Encode:

    doc := map[string]string{"hello": "world"}
    data, err := bson.Marshal(doc)
    doc2 := struct{Key string}{"value"}
    data2, err := bson.Marshal(&doc2)

You can treat either a struct or a map with a string key type as a document.

Decode:

    // data is a []byte value that contains BSON data
    doc1 := map[string]interface{}
    err := bson.Unmarshal(data, doc1)

    // or, if you have an idea of the data's structure...
    type doc struct {
        Name string
        ID   int
    }
    doc2 := new(doc)
    err = bson.Unmarshal(data, doc2)

The package also provides some types that allow encoding of BSON data that
cannot be represented by Go types, including:

    JavaScript
    MaxKey
    MinKey
    ObjectID
    Regexp
    Symbol

Clients may create additional types that can be BSON-encoded by implementing
the Marshaler interface. See the documentation in the source for more information.

Contributing
------------

Simply use GitHub as usual to create a fork, make your changes, and create a pull
request. Code is expected to be formatted with gofmt and to adhere to the usual Go
conventions -- that is, the conventions used by Go's core libraries.

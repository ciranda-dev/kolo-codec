# kolo-codec

Canonical binary encoding for Kolo.

[![status: experimental](https://img.shields.io/badge/status-experimental-orange)](#status)
[![license: MIT OR Apache-2.0](https://img.shields.io/badge/license-MIT%20OR%20Apache--2.0-blue)](#license)
[![go: 1.26+](https://img.shields.io/badge/go-1.26%2B-00ADD8)](go.mod)

## What it is

Peers sign and hash the bytes they exchange, so a value must always encode to the same bytes. kolo-codec gives every value exactly one byte form: unsigned integers are minimal unsigned LEB128 varints as defined by the [DWARF Version 5 standard](https://dwarfstd.org/dwarf5std.html), signed integers are mapped to unsigned ones by alternating sign (0, -1, 1, -2 become 0, 1, 2, 3), and strings are [UTF-8 (RFC 3629)](https://www.rfc-editor.org/info/rfc3629). It rejects anything else with an error and reads untrusted input without panicking.

It is pure Go and uses only the standard library: no cgo and nothing to install. Most Kolo libraries use it for the values they sign, hash, send or store. kolo-moq and kolo-transport use the wire encodings of their own protocols.

## Install

```sh
go get github.com/ciranda-dev/kolo-codec
```

## Quick start

Encode a number and a string, then decode them and check that nothing is left. Save it outside the repository and run it with `go run .` in a module that requires kolo-codec.

```go
package main

import (
	"fmt"
	"log"

	kolocodec "github.com/ciranda-dev/kolo-codec"
)

func main() {
	var e kolocodec.Encoder
	e.PutUvarint(300)
	e.PutString("hello")

	d := kolocodec.NewDecoder(e.Bytes())
	seq, err := d.Uvarint()
	if err != nil {
		log.Fatal(err)
	}
	body, err := d.String(1024)
	if err != nil {
		log.Fatal(err)
	}
	if err := d.Done(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(seq, body)
}
```

```text
$ go run .
300 hello
```

A complete program that encodes a record and decodes it back is in [examples/basic](examples/basic/main.go).

## API

| Symbol | What it does |
|---|---|
| `Encoder` | Appends values; the zero value is ready to use. `Bytes` returns a copy of the output, `Len` its size and `Reset` empties the encoder for reuse |
| `PutUvarint`, `PutVarint` | Unsigned and signed integers as varints |
| `PutUint32`, `PutUint64` | Fixed-width little-endian integers |
| `PutBool` | One byte, 0 or 1 |
| `PutBytes`, `PutString`, `PutValidString` | A length, then the data; `PutValidString` refuses invalid UTF-8 |
| `PutFixed` | Raw bytes with no length, for keys and hashes |
| `NewDecoder(b)` | Reads the same values back: `Uvarint`, `UvarintMax`, `Varint`, `Uint32`, `Uint64`, `Bool`, `Bytes`, `String`, `Fixed` |
| `Len`, `Done` | Bytes left, and an error if any are |

Every length is checked against a maximum you pass and against the remaining input before anything is allocated. A failed read consumes nothing.

Errors work with `errors.Is`: `ErrShortBuffer`, `ErrNonCanonical`, `ErrTooLarge`, `ErrInvalidUTF8` and `ErrTrailingBytes`. Their messages never include values taken from the input.

## Encodings

| Value | Encoding |
|---|---|
| unsigned integer | minimal unsigned LEB128 varint |
| signed integer | zigzag, then the unsigned varint |
| `uint32`, `uint64` | fixed width, little-endian |
| bool | one byte, 0 or 1 |
| bytes, string | varint length, then the data |
| fixed-size value (key, hash) | the raw bytes, no length |

There is no JSON form. The details, including what canonical means for strings, are in [docs/formats.md](docs/formats.md).

## More

- [docs/guide.md](docs/guide.md): the encoder, the decoder and the errors.
- [docs/formats.md](docs/formats.md): the byte-level encodings and references.
- [docs/limits.md](docs/limits.md): bounds and what the decoder checks.
- [SECURITY.md](SECURITY.md): security policy.

## Status

Experimental. The API may change before v1.0.0.

## License

Licensed under either of the [Apache License, Version 2.0](LICENSE) or the [MIT license](LICENSE-MIT), at your option.

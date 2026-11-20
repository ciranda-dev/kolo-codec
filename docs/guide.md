# Guide

Walkthrough of the API. The byte-level encodings are in [formats.md](formats.md) and the bounds in [limits.md](limits.md). A complete program is in the README, and [examples/basic](../examples/basic/main.go) encodes a record and decodes it back.

## Encoder

`Encoder` writes values with `PutUvarint`, `PutVarint`, `PutUint32`, `PutUint64`, `PutBool`, `PutBytes`, `PutString` and `PutFixed`. The zero value is ready to use.

- `PutValidString` is `PutString` that returns `ErrInvalidUTF8` and writes nothing if the string is not valid UTF-8. `PutString` writes invalid UTF-8 as given, and `Decoder.String` then rejects it.
- `Bytes()` returns a copy of the output. `Len()` is its size without a copy, and `Reset()` empties the encoder so a hot path can reuse its buffer.

## Decoder

`NewDecoder(b)` reads the values back with `Uvarint`, `UvarintMax(max)`, `Varint`, `Uint32`, `Uint64`, `Bool`, `Bytes(max)`, `String(max)` and `Fixed(n)`.

- `Len()` is the number of bytes left, and `Done()` fails if any are.
- A failed read consumes nothing.
- `Bytes` and `Fixed` return copies.
- Use `Bytes` for arbitrary data. `String` accepts only valid UTF-8.

## Errors

For `errors.Is`. Their messages never include values taken from the input.

| Error | When |
|---|---|
| `ErrShortBuffer` | The input ends early |
| `ErrNonCanonical` | A varint with redundant bytes (such as `80 00` for 0), or a bool byte other than 0 or 1 |
| `ErrTooLarge` | A varint over 64 bits, a length over `max`, a negative `max` or `Fixed` size, or a value over the limit in `UvarintMax` |
| `ErrInvalidUTF8` | `String` read bytes that are not valid UTF-8, or `PutValidString` got an invalid string |
| `ErrTrailingBytes` | `Done` found input left |

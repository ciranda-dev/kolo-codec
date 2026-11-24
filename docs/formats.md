# Formats

| Value | Encoding |
|---|---|
| unsigned integer | minimal unsigned LEB128 varint |
| signed integer | zigzag, then the unsigned varint |
| `uint32`, `uint64` | fixed width, little-endian |
| bool | one byte, 0 or 1 |
| bytes, string | varint length, then the data |
| fixed-size value (key, hash) | the raw bytes, no length |

## Canonical

Canonical here means byte-level: each value has one valid encoding, and any input that decodes re-encodes to the same bytes. A varint with redundant bytes, or a bool byte other than 0 or 1, is `ErrNonCanonical`.

## Strings

Strings must be valid UTF-8 but are not normalized, so a precomposed `é` (`c3 a9`) and `e` plus a combining accent (`65 cc 81`) are different values. `PutString` writes invalid UTF-8 as given, and `String` then rejects it; `PutValidString` refuses it instead.

## JSON

There is no JSON form. Most Kolo libraries use these encodings for the values they sign, hash, send or store. kolo-moq and kolo-transport use the wire encodings of their own protocols.

## References

- [DWARF Version 5](https://dwarfstd.org/dwarf5std.html), the Debugging Information Format standard.
  Takes: the unsigned LEB128 encoding, used only in its minimal form.
  Not implemented: signed LEB128; signed integers are mapped to unsigned ones by alternating sign instead.
- [RFC 3629, UTF-8, a transformation format of ISO 10646](https://www.rfc-editor.org/info/rfc3629) (Internet Standard, November 2003).
  Takes: the definition of valid UTF-8 that every string is checked against.
  Not implemented: nothing left out.

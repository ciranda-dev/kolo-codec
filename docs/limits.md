# Limits

kolo-codec has no fixed size limits of its own. The decoder takes its bounds from the caller, so every length read from untrusted input is checked before anything is allocated. This page is reference material: the README does not repeat it.

| Bound | Value | What happens past it |
|---|---|---|
| varint size | 64 bits | `ErrTooLarge` |
| `Bytes(max)`, `String(max)` length | at most `max`, and at most the bytes left in the input | `ErrTooLarge` or `ErrShortBuffer` |
| `UvarintMax(max)` value | at most `max` | `ErrTooLarge` |
| `Fixed(n)` size | at most the bytes left in the input; `n` must not be negative | `ErrShortBuffer` or `ErrTooLarge` |

- A negative `max` or `Fixed` size returns `ErrTooLarge`.
- A failed read consumes nothing, and the error never includes values taken from the input.
- `Bytes` and `Fixed` return copies, so the result does not alias the input.
- Reading untrusted input never panics.

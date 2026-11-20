// Package kolocodec encodes values into canonical bytes for Kolo and decodes
// them without panicking on hostile input. Canonical is byte-level: strings
// must be valid UTF-8 but are not normalized (no NFC).
//
// This package is experimental and its API may change before v1.0.0.
package kolocodec

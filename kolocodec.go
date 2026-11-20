package kolocodec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"unicode/utf8"
)

var (
	// ErrShortBuffer is returned when the input ends before a value does.
	ErrShortBuffer = errors.New("kolocodec: short buffer")
	// ErrNonCanonical is returned when the input is not the one valid encoding of its value.
	ErrNonCanonical = errors.New("kolocodec: non-canonical encoding")
	// ErrTooLarge is returned when a length or value is over the caller's max or a uvarint overflows 64 bits.
	ErrTooLarge = errors.New("kolocodec: value too large")
	// ErrInvalidUTF8 is returned by [Decoder.String] and [Encoder.PutValidString] for invalid UTF-8.
	ErrInvalidUTF8 = errors.New("kolocodec: invalid UTF-8")
	// ErrTrailingBytes is returned by [Decoder.Done] when input is left.
	ErrTrailingBytes = errors.New("kolocodec: trailing bytes")
)

// Encoder appends canonical encodings to a buffer; the zero value is ready to use.
type Encoder struct {
	buf []byte
}

// PutUvarint writes v as a minimal unsigned LEB128 varint.
func (e *Encoder) PutUvarint(v uint64) {
	e.buf = binary.AppendUvarint(e.buf, v)
}

// PutVarint writes v as a zigzag-encoded minimal varint.
func (e *Encoder) PutVarint(v int64) {
	e.PutUvarint(uint64(v<<1) ^ uint64(v>>63))
}

// PutUint32 writes v as 4 bytes, little-endian.
func (e *Encoder) PutUint32(v uint32) {
	e.buf = binary.LittleEndian.AppendUint32(e.buf, v)
}

// PutUint64 writes v as 8 bytes, little-endian.
func (e *Encoder) PutUint64(v uint64) {
	e.buf = binary.LittleEndian.AppendUint64(e.buf, v)
}

// PutBool writes v as one byte, 0 or 1.
func (e *Encoder) PutBool(v bool) {
	if v {
		e.buf = append(e.buf, 1)
	} else {
		e.buf = append(e.buf, 0)
	}
}

// PutBytes writes the length of b as a uvarint, then b.
func (e *Encoder) PutBytes(b []byte) {
	e.PutUvarint(uint64(len(b)))
	e.buf = append(e.buf, b...)
}

// PutString writes s like [Encoder.PutBytes]; use [Encoder.PutValidString] to reject invalid UTF-8.
func (e *Encoder) PutString(s string) {
	e.PutUvarint(uint64(len(s)))
	e.buf = append(e.buf, s...)
}

// PutValidString writes s like [Encoder.PutString], or returns [ErrInvalidUTF8] and writes nothing.
func (e *Encoder) PutValidString(s string) error {
	if !utf8.ValidString(s) {
		return ErrInvalidUTF8
	}
	e.PutString(s)
	return nil
}

// PutFixed writes b with no length, for fixed-size values such as keys and hashes.
func (e *Encoder) PutFixed(b []byte) {
	e.buf = append(e.buf, b...)
}

// Bytes returns a copy of the encoded output.
func (e *Encoder) Bytes() []byte {
	return slices.Clone(e.buf)
}

// Len returns the number of bytes written, without copying them.
func (e *Encoder) Len() int {
	return len(e.buf)
}

// Reset discards the output and keeps the buffer for reuse; earlier results of [Encoder.Bytes] are unaffected.
func (e *Encoder) Reset() {
	e.buf = e.buf[:0]
}

// Decoder reads canonical encodings from a byte slice; a failed read consumes nothing.
type Decoder struct {
	buf []byte
}

// NewDecoder returns a Decoder that reads from b.
func NewDecoder(b []byte) *Decoder {
	return &Decoder{buf: b}
}

// Uvarint reads a minimal unsigned LEB128 varint, rejecting non-minimal encodings.
func (d *Decoder) Uvarint() (uint64, error) {
	v, n, err := uvarint(d.buf)
	if err != nil {
		return 0, err
	}
	d.buf = d.buf[n:]
	return v, nil
}

// UvarintMax reads a uvarint like [Decoder.Uvarint] and returns [ErrTooLarge] if it is over max.
func (d *Decoder) UvarintMax(max uint64) (uint64, error) {
	v, n, err := uvarint(d.buf)
	if err != nil {
		return 0, err
	}
	if v > max {
		return 0, fmt.Errorf("%w: over max %d", ErrTooLarge, max)
	}
	d.buf = d.buf[n:]
	return v, nil
}

// Varint reads a zigzag-encoded minimal varint written by [Encoder.PutVarint].
func (d *Decoder) Varint() (int64, error) {
	u, err := d.Uvarint()
	if err != nil {
		return 0, err
	}
	return int64(u>>1) ^ -int64(u&1), nil
}

// Uint32 reads 4 bytes, little-endian.
func (d *Decoder) Uint32() (uint32, error) {
	if len(d.buf) < 4 {
		return 0, ErrShortBuffer
	}
	v := binary.LittleEndian.Uint32(d.buf)
	d.buf = d.buf[4:]
	return v, nil
}

// Uint64 reads 8 bytes, little-endian.
func (d *Decoder) Uint64() (uint64, error) {
	if len(d.buf) < 8 {
		return 0, ErrShortBuffer
	}
	v := binary.LittleEndian.Uint64(d.buf)
	d.buf = d.buf[8:]
	return v, nil
}

// Bool reads one byte and returns [ErrNonCanonical] if it is not 0 or 1.
func (d *Decoder) Bool() (bool, error) {
	if len(d.buf) < 1 {
		return false, ErrShortBuffer
	}
	b := d.buf[0]
	if b > 1 {
		return false, ErrNonCanonical
	}
	d.buf = d.buf[1:]
	return b == 1, nil
}

// Bytes reads a length-prefixed byte slice of at most max bytes and returns a copy.
func (d *Decoder) Bytes(max int) ([]byte, error) {
	b, n, err := d.prefixed(max)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(b))
	copy(out, b)
	d.buf = d.buf[n:]
	return out, nil
}

// String reads a length-prefixed UTF-8 string of at most max bytes.
func (d *Decoder) String(max int) (string, error) {
	b, n, err := d.prefixed(max)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", ErrInvalidUTF8
	}
	s := string(b)
	d.buf = d.buf[n:]
	return s, nil
}

// Fixed reads exactly n bytes with no length and returns a copy.
func (d *Decoder) Fixed(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("%w: negative length %d", ErrTooLarge, n)
	}
	if len(d.buf) < n {
		return nil, ErrShortBuffer
	}
	out := make([]byte, n)
	copy(out, d.buf)
	d.buf = d.buf[n:]
	return out, nil
}

// Len returns the number of bytes left to read.
func (d *Decoder) Len() int {
	return len(d.buf)
}

// Done returns [ErrTrailingBytes] if any input is left.
func (d *Decoder) Done() error {
	if len(d.buf) > 0 {
		return ErrTrailingBytes
	}
	return nil
}

func (d *Decoder) prefixed(max int) ([]byte, int, error) {
	l, n, err := uvarint(d.buf)
	if err != nil {
		return nil, 0, err
	}
	if max < 0 || l > uint64(max) {
		return nil, 0, ErrTooLarge
	}
	if l > uint64(len(d.buf)-n) {
		return nil, 0, ErrShortBuffer
	}
	end := n + int(l)
	return d.buf[n:end], end, nil
}

// uvarint exists because binary.Uvarint accepts non-minimal encodings.
func uvarint(b []byte) (uint64, int, error) {
	var v uint64
	for i := 0; i < binary.MaxVarintLen64; i++ {
		if i >= len(b) {
			return 0, 0, ErrShortBuffer
		}
		c := b[i]
		if i == binary.MaxVarintLen64-1 && c > 1 {
			return 0, 0, fmt.Errorf("%w: uvarint overflows 64 bits", ErrTooLarge)
		}
		v |= uint64(c&0x7f) << (7 * i)
		if c < 0x80 {
			if c == 0 && i > 0 {
				return 0, 0, ErrNonCanonical
			}
			return v, i + 1, nil
		}
	}
	// Unreachable: the last iteration always returns.
	return 0, 0, fmt.Errorf("%w: uvarint overflows 64 bits", ErrTooLarge)
}

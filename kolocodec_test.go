package kolocodec

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestRoundTrip(t *testing.T) {
	var e Encoder
	uvarints := []uint64{0, 1, 127, 128, 300, 1<<63 - 1, 1 << 63, math.MaxUint64}
	for _, v := range uvarints {
		e.PutUvarint(v)
	}
	e.PutUint32(0)
	e.PutUint32(math.MaxUint32)
	e.PutUint64(0)
	e.PutUint64(math.MaxUint64)
	e.PutBool(false)
	e.PutBool(true)
	e.PutBytes(nil)
	e.PutBytes([]byte{})
	e.PutBytes([]byte("hello"))
	e.PutString("")
	e.PutString("kolo")
	e.PutFixed(nil)
	e.PutFixed([]byte{1, 2, 3})

	d := NewDecoder(e.Bytes())
	for _, want := range uvarints {
		if got, err := d.Uvarint(); err != nil || got != want {
			t.Fatalf("Uvarint() = %d, %v, want %d", got, err, want)
		}
	}
	for _, want := range []uint32{0, math.MaxUint32} {
		if got, err := d.Uint32(); err != nil || got != want {
			t.Fatalf("Uint32() = %d, %v, want %d", got, err, want)
		}
	}
	for _, want := range []uint64{0, math.MaxUint64} {
		if got, err := d.Uint64(); err != nil || got != want {
			t.Fatalf("Uint64() = %d, %v, want %d", got, err, want)
		}
	}
	for _, want := range []bool{false, true} {
		if got, err := d.Bool(); err != nil || got != want {
			t.Fatalf("Bool() = %v, %v, want %v", got, err, want)
		}
	}
	for _, want := range []string{"", "", "hello"} {
		if got, err := d.Bytes(16); err != nil || string(got) != want {
			t.Fatalf("Bytes() = %q, %v, want %q", got, err, want)
		}
	}
	for _, want := range []string{"", "kolo"} {
		if got, err := d.String(16); err != nil || got != want {
			t.Fatalf("String() = %q, %v, want %q", got, err, want)
		}
	}
	if got, err := d.Fixed(0); err != nil || len(got) != 0 {
		t.Fatalf("Fixed(0) = %x, %v, want empty", got, err)
	}
	if got, err := d.Fixed(3); err != nil || !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("Fixed(3) = %x, %v, want 010203", got, err)
	}
	if err := d.Done(); err != nil {
		t.Fatalf("Done() = %v, want nil", err)
	}
}

func TestUvarintBoundaries(t *testing.T) {
	for k := 0; k < 10; k++ {
		for _, v := range []uint64{1<<(7*k) - 1, 1 << (7 * k)} {
			var e Encoder
			e.PutUvarint(v)
			d := NewDecoder(e.Bytes())
			got, err := d.Uvarint()
			if err != nil || got != v {
				t.Fatalf("Uvarint() = %d, %v, want %d", got, err, v)
			}
			if err := d.Done(); err != nil {
				t.Fatalf("Done() after %d = %v", v, err)
			}
		}
	}
}

func TestVectors(t *testing.T) {
	tests := []struct {
		name string
		put  func(*Encoder)
		want string
	}{
		{"uvarint 0", func(e *Encoder) { e.PutUvarint(0) }, "00"},
		{"uvarint 127", func(e *Encoder) { e.PutUvarint(127) }, "7f"},
		{"uvarint 128", func(e *Encoder) { e.PutUvarint(128) }, "8001"},
		{"uvarint 300", func(e *Encoder) { e.PutUvarint(300) }, "ac02"},
		{"uvarint max", func(e *Encoder) { e.PutUvarint(math.MaxUint64) }, "ffffffffffffffffff01"},
		{"varint 0", func(e *Encoder) { e.PutVarint(0) }, "00"},
		{"varint -1", func(e *Encoder) { e.PutVarint(-1) }, "01"},
		{"varint 1", func(e *Encoder) { e.PutVarint(1) }, "02"},
		{"varint -2", func(e *Encoder) { e.PutVarint(-2) }, "03"},
		{"varint 63", func(e *Encoder) { e.PutVarint(63) }, "7e"},
		{"varint -64", func(e *Encoder) { e.PutVarint(-64) }, "7f"},
		{"varint 64", func(e *Encoder) { e.PutVarint(64) }, "8001"},
		{"varint min", func(e *Encoder) { e.PutVarint(math.MinInt64) }, "ffffffffffffffffff01"},
		{"varint max", func(e *Encoder) { e.PutVarint(math.MaxInt64) }, "feffffffffffffffff01"},
		{"uint32 1", func(e *Encoder) { e.PutUint32(1) }, "01000000"},
		{"uint32 max", func(e *Encoder) { e.PutUint32(math.MaxUint32) }, "ffffffff"},
		{"uint64 1", func(e *Encoder) { e.PutUint64(1) }, "0100000000000000"},
		{"uint64 order", func(e *Encoder) { e.PutUint64(0x0102030405060708) }, "0807060504030201"},
		{"bool false", func(e *Encoder) { e.PutBool(false) }, "00"},
		{"bool true", func(e *Encoder) { e.PutBool(true) }, "01"},
		{"bytes empty", func(e *Encoder) { e.PutBytes(nil) }, "00"},
		{"bytes abc", func(e *Encoder) { e.PutBytes([]byte("abc")) }, "03616263"},
		{"string hi", func(e *Encoder) { e.PutString("hi") }, "026869"},
		{"fixed", func(e *Encoder) { e.PutFixed([]byte{0xaa, 0xbb}) }, "aabb"},
		{"fixed empty", func(e *Encoder) { e.PutFixed(nil) }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e Encoder
			tt.put(&e)
			if got := hex.EncodeToString(e.Bytes()); got != tt.want {
				t.Fatalf("got %s, want %s", got, tt.want)
			}
		})
	}
}

// maxInt32Len is a length prefix that fits in an int on every platform.
const maxInt32Len = "ffffffff07"

// maxInt64Len is a length prefix over MaxInt only where int has 32 bits.
const maxInt64Len = "ffffffffffffffff7f"

// over32BitLen is a length of 2^32+3 followed by 3 bytes, which a truncating int(l) would accept.
const over32BitLen = "8380808010616263"

var tooLargeOn32Bit = func() error {
	if strconv.IntSize == 32 {
		return ErrTooLarge
	}
	return ErrShortBuffer
}()

func TestDecodeErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		read func(*Decoder) error
		want error
	}{
		{"uvarint empty", "", readUvarint, ErrShortBuffer},
		{"uvarint cut", "80", readUvarint, ErrShortBuffer},
		{"uvarint cut long", "ffffffffffffffff", readUvarint, ErrShortBuffer},
		{"uvarint overlong 0", "8000", readUvarint, ErrNonCanonical},
		{"uvarint overlong 127", "ff00", readUvarint, ErrNonCanonical},
		{"uvarint overlong 3 bytes", "808000", readUvarint, ErrNonCanonical},
		{"uvarint overlong 10 bytes", "81808080808080808000", readUvarint, ErrNonCanonical},
		{"uvarint overflow last byte 2", "ffffffffffffffffff02", readUvarint, ErrTooLarge},
		{"uvarint overflow last byte 7f", "ffffffffffffffffff7f", readUvarint, ErrTooLarge},
		{"uvarint overflow 11 bytes", "8080808080808080808001", readUvarint, ErrTooLarge},
		{"uvarint max over", "ac02", readUvarintMax(299), ErrTooLarge},
		{"uvarint max zero", "01", readUvarintMax(0), ErrTooLarge},
		{"uvarint max empty", "", readUvarintMax(10), ErrShortBuffer},
		{"uvarint max overlong", "8000", readUvarintMax(10), ErrNonCanonical},
		{"varint empty", "", readVarint, ErrShortBuffer},
		{"varint cut", "ff", readVarint, ErrShortBuffer},
		{"varint overlong", "8100", readVarint, ErrNonCanonical},
		{"varint overflow", "ffffffffffffffffff02", readVarint, ErrTooLarge},
		{"uint32 empty", "", readUint32, ErrShortBuffer},
		{"uint32 short", "010203", readUint32, ErrShortBuffer},
		{"uint64 empty", "", readUint64, ErrShortBuffer},
		{"uint64 short", "01020304050607", readUint64, ErrShortBuffer},
		{"bool empty", "", readBool, ErrShortBuffer},
		{"bool 2", "02", readBool, ErrNonCanonical},
		{"bool ff", "ff", readBool, ErrNonCanonical},
		{"bytes empty", "", readBytes(16), ErrShortBuffer},
		{"bytes bad length", "8000", readBytes(16), ErrNonCanonical},
		{"bytes over remaining", "03aabb", readBytes(16), ErrShortBuffer},
		{"bytes over max", "03aabbcc", readBytes(2), ErrTooLarge},
		{"bytes over max and remaining", "ffffffff0f", readBytes(10), ErrTooLarge},
		{"bytes negative max", "00", readBytes(-1), ErrTooLarge},
		{"bytes huge length", maxInt32Len, readBytes(math.MaxInt), ErrShortBuffer},
		{"bytes length over max int", maxInt64Len, readBytes(math.MaxInt), tooLargeOn32Bit},
		{"bytes length 2^32+3", over32BitLen, readBytes(math.MaxInt), tooLargeOn32Bit},
		{"string length 2^32+3", over32BitLen, readString(math.MaxInt), tooLargeOn32Bit},
		{"bytes length overflow", "ffffffffffffffffff02", readBytes(math.MaxInt), ErrTooLarge},
		{"string empty", "", readString(16), ErrShortBuffer},
		{"string over remaining", "036869", readString(16), ErrShortBuffer},
		{"string over max", "03686969", readString(2), ErrTooLarge},
		{"string huge length", maxInt32Len, readString(math.MaxInt), ErrShortBuffer},
		{"string invalid UTF-8", "02c328", readString(16), ErrInvalidUTF8},
		{"string invalid byte", "01ff", readString(16), ErrInvalidUTF8},
		{"string surrogate", "03eda080", readString(16), ErrInvalidUTF8},
		{"string length over max int", "ffffffffffffffffff01", readString(math.MaxInt), ErrTooLarge},
		{"fixed empty", "", readFixed(1), ErrShortBuffer},
		{"fixed short", "aabbcc", readFixed(4), ErrShortBuffer},
		{"fixed negative", "aabbcc", readFixed(-1), ErrTooLarge},
		{"done trailing", "00", (*Decoder).Done, ErrTrailingBytes},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := hex.DecodeString(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			d := NewDecoder(in)
			if err := tt.read(d); !errors.Is(err, tt.want) {
				t.Fatalf("got error %v, want %v", err, tt.want)
			}
			if d.Len() != len(in) {
				t.Fatalf("failed read consumed %d bytes", len(in)-len(d.buf))
			}
		})
	}
}

func TestDone(t *testing.T) {
	if err := NewDecoder(nil).Done(); err != nil {
		t.Fatalf("Done() on empty input = %v, want nil", err)
	}
	d := NewDecoder([]byte{1, 0, 0, 0, 9})
	if _, err := d.Uint32(); err != nil {
		t.Fatal(err)
	}
	if err := d.Done(); !errors.Is(err, ErrTrailingBytes) {
		t.Fatalf("Done() = %v, want %v", err, ErrTrailingBytes)
	}
}

func TestFailedReadConsumesNothing(t *testing.T) {
	d := NewDecoder([]byte{1, 2, 3})
	if _, err := d.Uint32(); !errors.Is(err, ErrShortBuffer) {
		t.Fatalf("Uint32() error = %v, want %v", err, ErrShortBuffer)
	}
	got, err := d.Fixed(3)
	if err != nil || !bytes.Equal(got, []byte{1, 2, 3}) {
		t.Fatalf("Fixed(3) = %x, %v, want 010203", got, err)
	}
}

func TestDecodedSlicesAreCopies(t *testing.T) {
	in := []byte{3, 'a', 'b', 'c', 'x', 'y'}
	d := NewDecoder(in)
	b, err := d.Bytes(16)
	if err != nil {
		t.Fatal(err)
	}
	f, err := d.Fixed(2)
	if err != nil {
		t.Fatal(err)
	}
	for i := range in {
		in[i] = 0
	}
	if string(b) != "abc" || string(f) != "xy" {
		t.Fatalf("decoded values changed with the input: %q %q", b, f)
	}
	b[0], f[0] = 'z', 'z'
	if in[1] != 0 || in[4] != 0 {
		t.Fatal("writing to a decoded value changed the input")
	}
}

func TestEncoderBytesIsCopy(t *testing.T) {
	var e Encoder
	if len(e.Bytes()) != 0 {
		t.Fatal("zero Encoder is not empty")
	}
	e.PutBool(true)
	out := e.Bytes()
	out[0] = 9
	e.PutBool(false)
	if !bytes.Equal(e.Bytes(), []byte{1, 0}) {
		t.Fatalf("Bytes() = %x after changing an earlier result, want 0100", e.Bytes())
	}
}

func TestEncoderLenAndReset(t *testing.T) {
	var e Encoder
	if e.Len() != 0 {
		t.Fatalf("zero Encoder Len() = %d, want 0", e.Len())
	}
	e.PutUvarint(300)
	e.PutBool(true)
	if e.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", e.Len())
	}
	first := e.Bytes()
	e.Reset()
	if e.Len() != 0 || len(e.Bytes()) != 0 {
		t.Fatalf("after Reset: Len() = %d, Bytes() = %x, want empty", e.Len(), e.Bytes())
	}
	e.PutUvarint(1)
	if !bytes.Equal(e.Bytes(), []byte{1}) {
		t.Fatalf("Bytes() = %x after Reset, want 01", e.Bytes())
	}
	if !bytes.Equal(first, []byte{0xac, 0x02, 1}) {
		t.Fatalf("an earlier Bytes() result changed to %x", first)
	}
}

func TestPutValidString(t *testing.T) {
	var e Encoder
	if err := e.PutValidString("é"); err != nil {
		t.Fatalf("PutValidString(valid) = %v, want nil", err)
	}
	if got := hex.EncodeToString(e.Bytes()); got != "02c3a9" {
		t.Fatalf("Bytes() = %s, want 02c3a9", got)
	}
	for _, s := range []string{"\xc3", "a\xe2\x82", "\xff", "\xed\xa0\x80"} {
		if err := e.PutValidString(s); !errors.Is(err, ErrInvalidUTF8) {
			t.Fatalf("PutValidString(%q) = %v, want %v", s, err, ErrInvalidUTF8)
		}
		if got := hex.EncodeToString(e.Bytes()); got != "02c3a9" {
			t.Fatalf("Bytes() = %s after rejecting %q, want 02c3a9", got, s)
		}
	}
}

func TestVarintRoundTrip(t *testing.T) {
	values := []int64{0, -1, 1, -2, 63, -64, 64, -65, math.MinInt64, math.MaxInt64}
	var e Encoder
	for _, v := range values {
		e.PutVarint(v)
	}
	d := NewDecoder(e.Bytes())
	for _, want := range values {
		if got, err := d.Varint(); err != nil || got != want {
			t.Fatalf("Varint() = %d, %v, want %d", got, err, want)
		}
	}
	if err := d.Done(); err != nil {
		t.Fatalf("Done() = %v, want nil", err)
	}
}

func TestUvarintMax(t *testing.T) {
	d := NewDecoder([]byte{0xac, 0x02, 0xac, 0x02})
	if got, err := d.UvarintMax(300); err != nil || got != 300 {
		t.Fatalf("UvarintMax(300) = %d, %v, want 300", got, err)
	}
	if got, err := d.UvarintMax(math.MaxUint64); err != nil || got != 300 {
		t.Fatalf("UvarintMax(MaxUint64) = %d, %v, want 300", got, err)
	}
	if err := d.Done(); err != nil {
		t.Fatalf("Done() = %v, want nil", err)
	}
}

func TestLen(t *testing.T) {
	d := NewDecoder([]byte{0xac, 0x02, 1})
	if d.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", d.Len())
	}
	if _, err := d.Uvarint(); err != nil {
		t.Fatal(err)
	}
	if d.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", d.Len())
	}
	if _, err := d.Bool(); err != nil {
		t.Fatal(err)
	}
	if d.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", d.Len())
	}
}

func TestStringUTF8(t *testing.T) {
	for _, s := range []string{"\u00e9", "e\u0301", "kolo", ""} {
		var e Encoder
		e.PutString(s)
		d := NewDecoder(e.Bytes())
		if got, err := d.String(16); err != nil || got != s {
			t.Fatalf("String() = %q, %v, want %q", got, err, s)
		}
	}
	var e Encoder
	e.PutString("\xc3(")
	if got := hex.EncodeToString(e.Bytes()); got != "02c328" {
		t.Fatalf("PutString = %s, want 02c328", got)
	}
	d := NewDecoder(e.Bytes())
	if _, err := d.String(16); !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("String() error = %v, want %v", err, ErrInvalidUTF8)
	}
	if b, err := d.Bytes(16); err != nil || string(b) != "\xc3(" {
		t.Fatalf("Bytes() after a rejected String = %x, %v", b, err)
	}
}

// An op byte is a<<4 | kind (kinds 10 to 15 do nothing): a gives size a-1 (UvarintMax: max a), or MaxInt (MaxUint64) at fuzzBig.
const fuzzBig = 15

func fuzzOp(kind, a byte) byte { return a<<4 | kind }

func fuzzOpDecode(op byte) (kind, a byte) { return op & 15, op >> 4 }

type fuzzStep struct{ kind, a byte }

var fuzzDecoderSeeds = []struct {
	steps []fuzzStep
	data  []byte
}{
	{[]fuzzStep{{0, 0}}, []byte{0xac, 0x02}},
	{[]fuzzStep{{0, 0}}, []byte{0x80, 0x00}},
	{[]fuzzStep{{0, 0}}, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01}},
	{[]fuzzStep{{1, 0}, {2, 0}, {3, 0}}, []byte{1, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 1}},
	{[]fuzzStep{{4, 5}, {5, 5}, {6, 3}, {7, 0}}, []byte{3, 'a', 'b', 'c', 2, 'h', 'i', 1, 2}},
	{[]fuzzStep{{4, fuzzBig}}, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}},
	{[]fuzzStep{{5, 5}, {4, 5}}, []byte{0x02, 0xc3, 0x28}},
	{[]fuzzStep{{8, 0}, {8, 0}, {9, 3}, {9, fuzzBig}}, []byte{0x01, 0x80, 0x01, 0x04, 0xac, 0x02}},
}

func TestFuzzOpEncoding(t *testing.T) {
	for _, seed := range fuzzDecoderSeeds {
		for _, st := range seed.steps {
			if kind, a := fuzzOpDecode(fuzzOp(st.kind, st.a)); kind != st.kind || a != st.a {
				t.Errorf("seed op (%d, %d) decodes to (%d, %d)", st.kind, st.a, kind, a)
			}
		}
	}
	for kind := byte(0); kind < 10; kind++ {
		for a := byte(0); a <= fuzzBig; a++ {
			if k, got := fuzzOpDecode(fuzzOp(kind, a)); k != kind || got != a {
				t.Errorf("op (%d, %d) decodes to (%d, %d)", kind, a, k, got)
			}
		}
	}
}

func FuzzDecoder(f *testing.F) {
	for _, seed := range fuzzDecoderSeeds {
		var ops []byte
		for _, st := range seed.steps {
			ops = append(ops, fuzzOp(st.kind, st.a))
		}
		f.Add(ops, seed.data)
	}
	f.Fuzz(func(t *testing.T, ops, data []byte) {
		d := NewDecoder(data)
		var e Encoder
		for _, op := range ops {
			kind, ab := fuzzOpDecode(op)
			a := int(ab)
			arg := a - 1
			umax := uint64(a)
			if a == fuzzBig {
				arg = math.MaxInt
				umax = math.MaxUint64
			}
			before := d.Len()
			var err error
			switch kind {
			case 0:
				var v uint64
				if v, err = d.Uvarint(); err == nil {
					e.PutUvarint(v)
				}
			case 1:
				var v uint32
				if v, err = d.Uint32(); err == nil {
					e.PutUint32(v)
				}
			case 2:
				var v uint64
				if v, err = d.Uint64(); err == nil {
					e.PutUint64(v)
				}
			case 3:
				var v bool
				if v, err = d.Bool(); err == nil {
					e.PutBool(v)
				}
			case 4:
				var v []byte
				if v, err = d.Bytes(arg); err == nil {
					if len(v) > arg {
						t.Fatalf("Bytes(%d) returned %d bytes", arg, len(v))
					}
					e.PutBytes(v)
				}
			case 5:
				var v string
				if v, err = d.String(arg); err == nil {
					if len(v) > arg {
						t.Fatalf("String(%d) returned %d bytes", arg, len(v))
					}
					if !utf8.ValidString(v) {
						t.Fatalf("String(%d) returned invalid UTF-8 %q", arg, v)
					}
					e.PutString(v)
				}
			case 6:
				var v []byte
				if v, err = d.Fixed(arg); err == nil {
					if len(v) != arg {
						t.Fatalf("Fixed(%d) returned %d bytes", arg, len(v))
					}
					e.PutFixed(v)
				}
			case 7:
				err = d.Done()
				if (err == nil) != (d.Len() == 0) {
					t.Fatalf("Done() = %v with %d bytes left", err, d.Len())
				}
			case 8:
				var v int64
				if v, err = d.Varint(); err == nil {
					e.PutVarint(v)
				}
			case 9:
				var v uint64
				if v, err = d.UvarintMax(umax); err == nil {
					if v > umax {
						t.Fatalf("UvarintMax(%d) returned %d", umax, v)
					}
					e.PutUvarint(v)
				}
			}
			if d.Len() != len(d.buf) || d.Len() > before {
				t.Fatalf("Len() = %d, before %d, buffer %d", d.Len(), before, len(d.buf))
			}
			if err != nil {
				if !isSentinel(err) {
					t.Fatalf("op %d returned a non-sentinel error: %v", op, err)
				}
				if d.Len() != before {
					t.Fatalf("op %d failed but consumed %d bytes", op, before-d.Len())
				}
			}
		}
		consumed := data[:len(data)-d.Len()]
		if !bytes.Equal(e.Bytes(), consumed) {
			t.Fatalf("re-encoding gave %x, want %x", e.Bytes(), consumed)
		}
	})
}

func isSentinel(err error) bool {
	return errors.Is(err, ErrShortBuffer) || errors.Is(err, ErrNonCanonical) ||
		errors.Is(err, ErrTooLarge) || errors.Is(err, ErrTrailingBytes) ||
		errors.Is(err, ErrInvalidUTF8)
}

func readUvarint(d *Decoder) error { _, err := d.Uvarint(); return err }
func readVarint(d *Decoder) error  { _, err := d.Varint(); return err }
func readUint32(d *Decoder) error  { _, err := d.Uint32(); return err }
func readUint64(d *Decoder) error  { _, err := d.Uint64(); return err }
func readBool(d *Decoder) error    { _, err := d.Bool(); return err }

func readUvarintMax(max uint64) func(*Decoder) error {
	return func(d *Decoder) error { _, err := d.UvarintMax(max); return err }
}

func readBytes(max int) func(*Decoder) error {
	return func(d *Decoder) error { _, err := d.Bytes(max); return err }
}

func readString(max int) func(*Decoder) error {
	return func(d *Decoder) error { _, err := d.String(max); return err }
}

func readFixed(n int) func(*Decoder) error {
	return func(d *Decoder) error { _, err := d.Fixed(n); return err }
}

func FuzzRoundTrip(f *testing.F) {
	f.Add(uint64(0), uint64(0), int64(0), uint32(0), uint64(0), false, []byte{}, "", []byte{})
	f.Add(uint64(300), uint64(299), int64(-1), uint32(1), uint64(1)<<63, true, []byte("hello"), "kolo", []byte{1, 2, 3})
	f.Add(^uint64(0), ^uint64(0), int64(math.MinInt64), ^uint32(0), ^uint64(0), true, bytes.Repeat([]byte{0xff}, 200), "\xff\xfe", bytes.Repeat([]byte{0}, 32))
	f.Add(uint64(1), uint64(1), int64(math.MaxInt64), uint32(0), uint64(0), false, []byte{}, "\xc3(", []byte{})

	f.Fuzz(func(t *testing.T, u, umax uint64, i int64, u32 uint32, u64 uint64, b bool, bs []byte, s string, fixed []byte) {
		var e Encoder
		e.PutUvarint(u)
		e.PutUvarint(u)
		e.PutVarint(i)
		e.PutUint32(u32)
		e.PutUint64(u64)
		e.PutBool(b)
		e.PutBytes(bs)
		e.PutString(s)
		e.PutFixed(fixed)

		var ve Encoder
		ve.PutBool(b)
		if err := ve.PutValidString(s); utf8.ValidString(s) != (err == nil) {
			t.Fatalf("PutValidString(%q) = %v", s, err)
		} else if err != nil && (!errors.Is(err, ErrInvalidUTF8) || len(ve.Bytes()) != 1) {
			t.Fatalf("PutValidString(%q) = %v and wrote %d bytes", s, err, len(ve.Bytes())-1)
		}

		d := NewDecoder(e.Bytes())
		if got, err := d.Uvarint(); err != nil || got != u {
			t.Fatalf("Uvarint = %d, %v, want %d", got, err, u)
		}
		got, err := d.UvarintMax(umax)
		switch {
		case u > umax && !errors.Is(err, ErrTooLarge):
			t.Fatalf("UvarintMax(%d) of %d = %d, %v, want %v", umax, u, got, err, ErrTooLarge)
		case u > umax:
			if _, err := d.Uvarint(); err != nil {
				t.Fatalf("Uvarint after a rejected UvarintMax = %v", err)
			}
		case err != nil || got != u:
			t.Fatalf("UvarintMax(%d) = %d, %v, want %d", umax, got, err, u)
		}
		if got, err := d.Varint(); err != nil || got != i {
			t.Fatalf("Varint = %d, %v, want %d", got, err, i)
		}
		if got, err := d.Uint32(); err != nil || got != u32 {
			t.Fatalf("Uint32 = %d, %v, want %d", got, err, u32)
		}
		if got, err := d.Uint64(); err != nil || got != u64 {
			t.Fatalf("Uint64 = %d, %v, want %d", got, err, u64)
		}
		if got, err := d.Bool(); err != nil || got != b {
			t.Fatalf("Bool = %v, %v, want %v", got, err, b)
		}
		if got, err := d.Bytes(len(bs)); err != nil || !bytes.Equal(got, bs) {
			t.Fatalf("Bytes = %x, %v, want %x", got, err, bs)
		}
		if utf8.ValidString(s) {
			if got, err := d.String(len(s)); err != nil || got != s {
				t.Fatalf("String = %q, %v, want %q", got, err, s)
			}
		} else {
			if _, err := d.String(len(s)); !errors.Is(err, ErrInvalidUTF8) {
				t.Fatalf("String of invalid UTF-8 %q: error %v, want %v", s, err, ErrInvalidUTF8)
			}
			if got, err := d.Bytes(len(s)); err != nil || string(got) != s {
				t.Fatalf("Bytes after a rejected String = %x, %v, want %x", got, err, s)
			}
		}
		if got, err := d.Fixed(len(fixed)); err != nil || !bytes.Equal(got, fixed) {
			t.Fatalf("Fixed = %x, %v, want %x", got, err, fixed)
		}
		if err := d.Done(); err != nil || d.Len() != 0 {
			t.Fatalf("Done = %v with Len %d", err, d.Len())
		}
	})
}

func TestErrorsCarryNoInputValues(t *testing.T) {
	var e Encoder
	e.PutUvarint(123456789)
	if _, err := NewDecoder(e.Bytes()).UvarintMax(5); !errors.Is(err, ErrTooLarge) || strings.Contains(err.Error(), "123456789") {
		t.Fatalf("UvarintMax error = %v", err)
	}
	d := NewDecoder(make([]byte, 987654))
	if err := d.Done(); !errors.Is(err, ErrTrailingBytes) || strings.Contains(err.Error(), "987654") {
		t.Fatalf("Done error = %v", err)
	}
}

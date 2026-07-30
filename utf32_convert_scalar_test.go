// Copyright 2026 The go-simdutf Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package simdutf

import (
	"bytes"
	"testing"
)

// Test vectors adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// convert_utf32_to_latin1 / convert_utf32_to_utf8 / convert_utf32_to_utf16*
// tests. Canary and short-destination checks are Go-specific slice-contract
// coverage.

func TestUTF32ConvertToLatin1(t *testing.T) {
	cases := []struct {
		name  string
		input []uint32
		want  []byte
		ok    bool
	}{
		{"empty", nil, nil, true},
		{"ascii", []uint32{'a', 'b', 'c'}, []byte("abc"), true},
		{"latin1", []uint32{0x00, 0x7f, 0xff}, []byte{0x00, 0x7f, 0xff}, true},
		{"too_large", []uint32{'a', 0x100, 'b'}, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dst := make([]byte, len(tc.input)+1)
			if len(tc.input) > 0 {
				dst[len(tc.input)] = 0x5a
			}
			n := convertUTF32ToLatin1Scalar(tc.input, dst[:len(tc.input)])
			result := convertUTF32ToLatin1WithErrorsScalar(tc.input, dst[:len(tc.input)])
			if tc.ok {
				if n != len(tc.want) || !bytes.Equal(dst[:n], tc.want) {
					t.Fatalf("convert = %d %x, want %d %x", n, dst[:n], len(tc.want), tc.want)
				}
				if result.Error != Success || result.Count != len(tc.want) {
					t.Fatalf("with_errors = %+v", result)
				}
				if got := convertValidUTF32ToLatin1Scalar(tc.input, dst[:len(tc.input)]); got != len(tc.want) {
					t.Fatalf("valid = %d", got)
				}
			} else {
				if n != 0 {
					t.Fatalf("convert = %d, want 0", n)
				}
				if result.Error != TooLarge {
					t.Fatalf("with_errors = %+v, want TooLarge", result)
				}
				if got := convertValidUTF32ToLatin1Scalar(tc.input, dst[:len(tc.input)]); got != 0 {
					t.Fatalf("valid = %d, want 0", got)
				}
			}
			if len(tc.input) > 0 && dst[len(tc.input)] != 0x5a {
				t.Fatalf("canary overwritten")
			}
		})
	}
}

func TestUTF32ConvertToUTF8(t *testing.T) {
	cases := []struct {
		name  string
		input []uint32
		want  []byte
		err   ErrorCode
		count int
	}{
		{"empty", nil, nil, Success, 0},
		{"ascii", []uint32{'A', 'B'}, []byte("AB"), Success, 2},
		{"bmp", []uint32{0x00e9, 0x20ac}, []byte("\xc3\xa9\xe2\x82\xac"), Success, 5},
		{"supplementary", []uint32{0x1f600}, []byte("\xf0\x9f\x98\x80"), Success, 4},
		{"mixed", []uint32{'A', 0x20ac, 0x1f600}, []byte("A\xe2\x82\xac\xf0\x9f\x98\x80"), Success, 8},
		{"surrogate", []uint32{'A', 0xd800}, nil, Surrogate, 1},
		{"too_large", []uint32{'A', 0x110000}, nil, TooLarge, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			need := utf8LengthFromUTF32Scalar(tc.input)
			dst := make([]byte, need+1)
			dst[need] = 0x5a
			n := convertUTF32ToUTF8Scalar(tc.input, dst[:need])
			result := convertUTF32ToUTF8WithErrorsScalar(tc.input, dst[:need])
			if tc.err == Success {
				if n != len(tc.want) || !bytes.Equal(dst[:n], tc.want) {
					t.Fatalf("convert = %d %q, want %d %q", n, dst[:n], len(tc.want), tc.want)
				}
				if result != (Result{Error: Success, Count: tc.count}) {
					t.Fatalf("with_errors = %+v", result)
				}
				if got := convertValidUTF32ToUTF8Scalar(tc.input, dst[:need]); got != len(tc.want) || !bytes.Equal(dst[:got], tc.want) {
					t.Fatalf("valid = %d %q", got, dst[:got])
				}
			} else {
				if n != 0 {
					t.Fatalf("convert = %d, want 0", n)
				}
				if result != (Result{Error: tc.err, Count: tc.count}) {
					t.Fatalf("with_errors = %+v, want {%v %d}", result, tc.err, tc.count)
				}
			}
			if dst[need] != 0x5a {
				t.Fatalf("canary overwritten")
			}
		})
	}
}

func TestUTF32ConvertToUTF16(t *testing.T) {
	cases := []struct {
		name   string
		input  []uint32
		native []uint16
		err    ErrorCode
		count  int
	}{
		{"empty", nil, nil, Success, 0},
		{"ascii", []uint32{'A', 'B'}, []uint16{'A', 'B'}, Success, 2},
		{"bmp", []uint32{0x00e9, 0x20ac}, []uint16{0x00e9, 0x20ac}, Success, 2},
		{"supplementary", []uint32{0x1f600}, []uint16{0xd83d, 0xde00}, Success, 2},
		{"mixed", []uint32{'A', 0x20ac, 0x1f600}, []uint16{'A', 0x20ac, 0xd83d, 0xde00}, Success, 4},
		{"surrogate", []uint32{'A', 0xd800}, nil, Surrogate, 1},
		{"too_large", []uint32{'A', 0x110000}, nil, TooLarge, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, little := range []bool{true, false} {
				need := utf16LengthFromUTF32Scalar(tc.input)
				dst := make([]uint16, need+1)
				dst[need] = 0x5a5a
				var n int
				var result Result
				if little {
					n = convertUTF32ToUTF16LEScalar(tc.input, dst[:need])
					result = convertUTF32ToUTF16LEWithErrorsScalar(tc.input, dst[:need])
				} else {
					n = convertUTF32ToUTF16BEScalar(tc.input, dst[:need])
					result = convertUTF32ToUTF16BEWithErrorsScalar(tc.input, dst[:need])
				}
				if tc.err == Success {
					want := rawUTF16Words(tc.native, little)
					if n != len(want) || !equalUint16(dst[:n], want) {
						t.Fatalf("little=%v convert = %d %x, want %d %x", little, n, dst[:n], len(want), want)
					}
					if result != (Result{Error: Success, Count: tc.count}) {
						t.Fatalf("little=%v with_errors = %+v", little, result)
					}
					var got int
					if little {
						got = convertValidUTF32ToUTF16LEScalar(tc.input, dst[:need])
					} else {
						got = convertValidUTF32ToUTF16BEScalar(tc.input, dst[:need])
					}
					if got != len(want) || !equalUint16(dst[:got], want) {
						t.Fatalf("little=%v valid = %d %x", little, got, dst[:got])
					}
				} else {
					if n != 0 {
						t.Fatalf("little=%v convert = %d, want 0", little, n)
					}
					if result != (Result{Error: tc.err, Count: tc.count}) {
						t.Fatalf("little=%v with_errors = %+v", little, result)
					}
				}
				if dst[need] != 0x5a5a {
					t.Fatalf("little=%v canary overwritten", little)
				}
			}
		})
	}
}

func equalUint16(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestUTF32ConvertShortDestinationPanics(t *testing.T) {
	input := []uint32{'a', 'b'}
	mustPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			v := recover()
			if v == nil {
				t.Fatalf("%s did not panic", name)
			}
			msg, ok := v.(string)
			if !ok || msg != "simdutf: destination is too short" {
				t.Fatalf("%s panic = %#v, want %q", name, v, "simdutf: destination is too short")
			}
		}()
		fn()
	}
	mustPanic("latin1", func() { convertUTF32ToLatin1Scalar(input, make([]byte, 1)) })
	mustPanic("latin1_errors", func() { convertUTF32ToLatin1WithErrorsScalar(input, make([]byte, 1)) })
	mustPanic("latin1_valid", func() { convertValidUTF32ToLatin1Scalar(input, make([]byte, 1)) })
	mustPanic("utf8", func() { convertUTF32ToUTF8Scalar(input, make([]byte, 0)) })
	mustPanic("utf8_errors", func() { convertUTF32ToUTF8WithErrorsScalar(input, make([]byte, 0)) })
	mustPanic("utf8_valid", func() { convertValidUTF32ToUTF8Scalar(input, make([]byte, 0)) })
	mustPanic("utf16le", func() { convertUTF32ToUTF16LEScalar(input, make([]uint16, 0)) })
	mustPanic("utf16be", func() { convertUTF32ToUTF16BEScalar(input, make([]uint16, 0)) })
	mustPanic("utf16le_errors", func() { convertUTF32ToUTF16LEWithErrorsScalar(input, make([]uint16, 0)) })
	mustPanic("utf16be_errors", func() { convertUTF32ToUTF16BEWithErrorsScalar(input, make([]uint16, 0)) })
	mustPanic("utf16le_valid", func() { convertValidUTF32ToUTF16LEScalar(input, make([]uint16, 0)) })
	mustPanic("utf16be_valid", func() { convertValidUTF32ToUTF16BEScalar(input, make([]uint16, 0)) })
}

func TestUTF32ConvertPublicDispatch(t *testing.T) {
	input := []uint32{'x', 0xff}
	dst := make([]byte, len(input))
	if got := ConvertUTF32ToLatin1(input, dst); got != 2 || string(dst) != "x\xff" {
		t.Fatalf("ConvertUTF32ToLatin1 = %d %q", got, dst)
	}
	if got := ConvertUTF32ToLatin1WithErrors(input, make([]byte, 2)); got != (Result{Error: Success, Count: 2}) {
		t.Fatalf("ConvertUTF32ToLatin1WithErrors = %+v", got)
	}
	if got := ConvertValidUTF32ToLatin1(input, make([]byte, 2)); got != 2 {
		t.Fatalf("ConvertValidUTF32ToLatin1 = %d", got)
	}

	mixed := []uint32{'A', 0x20ac, 0x1f600}
	if got := UTF8LengthFromUTF32(mixed); got != 1+3+4 {
		t.Fatalf("UTF8LengthFromUTF32 = %d", got)
	}
	if got := UTF16LengthFromUTF32(mixed); got != 4 {
		t.Fatalf("UTF16LengthFromUTF32 = %d", got)
	}
	if got := Latin1LengthFromUTF32(7); got != 7 {
		t.Fatalf("Latin1LengthFromUTF32 = %d", got)
	}
	utf8 := make([]byte, UTF8LengthFromUTF32(mixed))
	if got := ConvertUTF32ToUTF8(mixed, utf8); got != len(utf8) || string(utf8) != "A\xe2\x82\xac\xf0\x9f\x98\x80" {
		t.Fatalf("ConvertUTF32ToUTF8 = %d %q", got, utf8)
	}
	if got := ConvertUTF32ToUTF8WithErrors(mixed, make([]byte, len(utf8))); got != (Result{Error: Success, Count: len(utf8)}) {
		t.Fatalf("ConvertUTF32ToUTF8WithErrors = %+v", got)
	}
	if got := ConvertValidUTF32ToUTF8(mixed, make([]byte, len(utf8))); got != len(utf8) {
		t.Fatalf("ConvertValidUTF32ToUTF8 = %d", got)
	}

	wantNative := []uint16{'A', 0x20ac, 0xd83d, 0xde00}
	utf16 := make([]uint16, UTF16LengthFromUTF32(mixed))
	if got := ConvertUTF32ToUTF16LE(mixed, utf16); got != 4 || !equalUint16(utf16, rawUTF16Words(wantNative, true)) {
		t.Fatalf("ConvertUTF32ToUTF16LE = %d %x", got, utf16)
	}
	be := make([]uint16, 4)
	if got := ConvertUTF32ToUTF16BE(mixed, be); got != 4 || !equalUint16(be, rawUTF16Words(wantNative, false)) {
		t.Fatalf("ConvertUTF32ToUTF16BE = %d %x", got, be)
	}
	if got := ConvertUTF32ToUTF16LEWithErrors(mixed, make([]uint16, 4)); got.Error != Success || got.Count != 4 {
		t.Fatalf("ConvertUTF32ToUTF16LEWithErrors = %+v", got)
	}
	if got := ConvertUTF32ToUTF16BEWithErrors(mixed, make([]uint16, 4)); got.Error != Success || got.Count != 4 {
		t.Fatalf("ConvertUTF32ToUTF16BEWithErrors = %+v", got)
	}
	if got := ConvertValidUTF32ToUTF16LE(mixed, make([]uint16, 4)); got != 4 {
		t.Fatalf("ConvertValidUTF32ToUTF16LE = %d", got)
	}
	if got := ConvertValidUTF32ToUTF16BE(mixed, make([]uint16, 4)); got != 4 {
		t.Fatalf("ConvertValidUTF32ToUTF16BE = %d", got)
	}
}

func TestUTF32ConvertErrorCodes(t *testing.T) {
	if got := ConvertUTF32ToUTF8WithErrors([]uint32{0xd800}, make([]byte, 3)); got != (Result{Error: Surrogate, Count: 0}) {
		t.Fatalf("surrogate utf8 = %+v", got)
	}
	if got := ConvertUTF32ToUTF8WithErrors([]uint32{0x110000}, make([]byte, 4)); got != (Result{Error: TooLarge, Count: 0}) {
		t.Fatalf("too large utf8 = %+v", got)
	}
	if got := ConvertUTF32ToUTF16LEWithErrors([]uint32{0xdfff}, make([]uint16, 1)); got != (Result{Error: Surrogate, Count: 0}) {
		t.Fatalf("surrogate utf16 = %+v", got)
	}
	if got := ConvertUTF32ToUTF16BEWithErrors([]uint32{0x110000}, make([]uint16, 2)); got != (Result{Error: TooLarge, Count: 0}) {
		t.Fatalf("too large utf16 = %+v", got)
	}
	if got := ConvertUTF32ToLatin1WithErrors([]uint32{0x100}, make([]byte, 1)); got != (Result{Error: TooLarge, Count: 0}) {
		t.Fatalf("too large latin1 = %+v", got)
	}
}

func TestUTF32HostEndianWrappers(t *testing.T) {
	if !nativeLittleEndian() {
		t.Skip("host-endian wrapper LE-path coverage requires little-endian host")
	}
	input := []uint32{'A', 0x20ac, 0x1f600}
	need := UTF16LengthFromUTF32(input)
	dst := make([]uint16, need)
	if got := ConvertUTF32ToUTF16(input, dst); got != ConvertUTF32ToUTF16LE(input, make([]uint16, need)) {
		t.Fatalf("ConvertUTF32ToUTF16 = %d", got)
	}
	if got := ConvertUTF32ToUTF16WithErrors(input, make([]uint16, need)); got != ConvertUTF32ToUTF16LEWithErrors(input, make([]uint16, need)) {
		t.Fatalf("ConvertUTF32ToUTF16WithErrors = %+v", got)
	}
	if got := ConvertValidUTF32ToUTF16(input, make([]uint16, need)); got != ConvertValidUTF32ToUTF16LE(input, make([]uint16, need)) {
		t.Fatalf("ConvertValidUTF32ToUTF16 = %d", got)
	}
}

func TestUTF32LengthHelpersScalar(t *testing.T) {
	input := []uint32{'A', 0x00e9, 0x20ac, 0x1f600}
	if got := utf8LengthFromUTF32Scalar(input); got != 1+2+3+4 {
		t.Fatalf("utf8LengthFromUTF32Scalar = %d", got)
	}
	if got := utf16LengthFromUTF32Scalar(input); got != 1+1+1+2 {
		t.Fatalf("utf16LengthFromUTF32Scalar = %d", got)
	}
	if got := latin1LengthFromUTF32Scalar(5); got != 5 {
		t.Fatalf("latin1LengthFromUTF32Scalar = %d", got)
	}
}

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
	"math/bits"
	"slices"
	"testing"
	"unicode/utf8"
)

// Test vectors adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// convert_utf8_to_* tests. Canary and short-destination checks are Go-specific
// slice-contract coverage.

func TestUTF8ConvertTable(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		latin1 []byte
		utf16  []uint16
		utf32  []uint32
	}{
		{"nil", "", nil, nil, nil},
		{"ascii", "Hello", []byte("Hello"), []uint16{'H', 'e', 'l', 'l', 'o'}, []uint32{'H', 'e', 'l', 'l', 'o'}},
		{"latin1", "caf\u00e9", []byte{'c', 'a', 'f', 0xe9}, []uint16{'c', 'a', 'f', 0xe9}, []uint32{'c', 'a', 'f', 0xe9}},
		{"emoji", "A\U0001F600B", nil, []uint16{'A', 0xd83d, 0xde00, 'B'}, []uint32{'A', 0x1f600, 'B'}},
		{"arabic", "\u0645\u0631\u062d\u0628\u0627", nil, []uint16{0x0645, 0x0631, 0x062d, 0x0628, 0x0627}, []uint32{0x0645, 0x0631, 0x062d, 0x0628, 0x0627}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := []byte(test.input)
			if test.name == "nil" {
				input = nil
			}

			if test.latin1 != nil {
				dst := make([]byte, Latin1LengthFromUTF8(input))
				if got := ConvertUTF8ToLatin1(input, dst); got != len(test.latin1) || !slices.Equal(dst[:got], test.latin1) {
					t.Fatalf("ConvertUTF8ToLatin1() = %d/%x, want %d/%x", got, dst[:got], len(test.latin1), test.latin1)
				}
				if got := ConvertUTF8ToLatin1WithErrors(input, dst); got != (Result{Error: Success, Count: len(test.latin1)}) {
					t.Fatalf("ConvertUTF8ToLatin1WithErrors() = %#v", got)
				}
				if got := ConvertValidUTF8ToLatin1(input, dst); got != len(test.latin1) || !slices.Equal(dst[:got], test.latin1) {
					t.Fatalf("ConvertValidUTF8ToLatin1() = %d/%x", got, dst[:got])
				}
			} else if len(input) > 0 {
				dst := make([]byte, Latin1LengthFromUTF8(input))
				if got := ConvertUTF8ToLatin1(input, dst); got != 0 {
					t.Fatalf("ConvertUTF8ToLatin1() = %d, want 0", got)
				}
				if got := ConvertUTF8ToLatin1WithErrors(input, dst); got.Error != TooLarge {
					t.Fatalf("ConvertUTF8ToLatin1WithErrors() = %#v, want TooLarge", got)
				}
			}

			dst16 := make([]uint16, UTF16LengthFromUTF8(input))
			if got := ConvertUTF8ToUTF16LE(input, dst16); got != len(test.utf16) || !slices.Equal(dst16[:got], rawUTF16(test.utf16, true)) {
				t.Fatalf("ConvertUTF8ToUTF16LE() = %d/%x, want %d/%x", got, dst16[:got], len(test.utf16), rawUTF16(test.utf16, true))
			}
			if got := ConvertUTF8ToUTF16BE(input, dst16); got != len(test.utf16) || !slices.Equal(dst16[:got], rawUTF16(test.utf16, false)) {
				t.Fatalf("ConvertUTF8ToUTF16BE() = %d/%x", got, dst16[:got])
			}
			if got := ConvertValidUTF8ToUTF16LE(input, dst16); got != len(test.utf16) || !slices.Equal(dst16[:got], rawUTF16(test.utf16, true)) {
				t.Fatalf("ConvertValidUTF8ToUTF16LE() = %d/%x", got, dst16[:got])
			}
			if got := ConvertUTF8ToUTF16LEWithErrors(input, dst16); got != (Result{Error: Success, Count: len(test.utf16)}) {
				t.Fatalf("ConvertUTF8ToUTF16LEWithErrors() = %#v", got)
			}

			dst32 := make([]uint32, UTF32LengthFromUTF8(input))
			if got := ConvertUTF8ToUTF32(input, dst32); got != len(test.utf32) || !slices.Equal(dst32[:got], test.utf32) {
				t.Fatalf("ConvertUTF8ToUTF32() = %d/%x, want %d/%x", got, dst32[:got], len(test.utf32), test.utf32)
			}
			if got := ConvertValidUTF8ToUTF32(input, dst32); got != len(test.utf32) || !slices.Equal(dst32[:got], test.utf32) {
				t.Fatalf("ConvertValidUTF8ToUTF32() = %d/%x", got, dst32[:got])
			}
			if got := ConvertUTF8ToUTF32WithErrors(input, dst32); got != (Result{Error: Success, Count: len(test.utf32)}) {
				t.Fatalf("ConvertUTF8ToUTF32WithErrors() = %#v", got)
			}

			native16 := make([]uint16, len(dst16))
			if got := ConvertUTF8ToUTF16(input, native16); got != len(test.utf16) {
				t.Fatalf("ConvertUTF8ToUTF16() = %d", got)
			}
			explicit := make([]uint16, len(dst16))
			if nativeLittleEndian() {
				ConvertUTF8ToUTF16LE(input, explicit)
			} else {
				ConvertUTF8ToUTF16BE(input, explicit)
			}
			if !slices.Equal(native16, explicit) {
				t.Fatalf("native UTF-16 mismatch")
			}
		})
	}
}

func TestUTF8ConvertErrors(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		err   ErrorCode
		count int
	}{
		{"too_short", []byte{0xc2}, TooShort, 0},
		{"overlong", []byte{0xc0, 0xaf}, Overlong, 0},
		{"surrogate", []byte{0xed, 0xa0, 0x80}, Surrogate, 0},
		{"header", []byte{0xff}, HeaderBits, 0},
		{"too_long", []byte{0x80}, TooLong, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dst16 := make([]uint16, UTF16LengthFromUTF8(test.input)+8)
			got := ConvertUTF8ToUTF16LEWithErrors(test.input, dst16)
			if got.Error != test.err || got.Count != test.count {
				t.Fatalf("UTF16 errors = %#v, want {%v %d}", got, test.err, test.count)
			}
			if ConvertUTF8ToUTF16LE(test.input, dst16) != 0 {
				t.Fatal("ConvertUTF8ToUTF16LE did not return 0")
			}
			dst32 := make([]uint32, UTF32LengthFromUTF8(test.input)+8)
			got = ConvertUTF8ToUTF32WithErrors(test.input, dst32)
			if got.Error != test.err || got.Count != test.count {
				t.Fatalf("UTF32 errors = %#v, want {%v %d}", got, test.err, test.count)
			}
		})
	}
}

func TestUTF8ConvertShortDestinationPanics(t *testing.T) {
	input := []byte("café")
	short := make([]byte, 1)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	ConvertUTF8ToLatin1(input, short)
}

func TestUTF8ConvertRoundTripSample(t *testing.T) {
	input := []byte("ASCII \u00e9 \u0645 \U0001F600")
	if !utf8.Valid(input) {
		t.Fatal("fixture must be valid UTF-8")
	}
	dst32 := make([]uint32, UTF32LengthFromUTF8(input))
	n := ConvertUTF8ToUTF32(input, dst32)
	if n != utf8.RuneCount(input) {
		t.Fatalf("UTF32 count = %d, want %d", n, utf8.RuneCount(input))
	}
	dst16 := make([]uint16, UTF16LengthFromUTF8(input))
	if ConvertUTF8ToUTF16LE(input, dst16) == 0 {
		t.Fatal("UTF16 conversion failed")
	}
}

func rawUTF16(native []uint16, little bool) []uint16 {
	out := make([]uint16, len(native))
	needNative := little == nativeLittleEndian()
	for i, value := range native {
		if needNative {
			out[i] = value
		} else {
			out[i] = bits.ReverseBytes16(value)
		}
	}
	return out
}

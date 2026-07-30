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
	"math/bits"
	"testing"
)

// Test vectors adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// convert_utf16*_to_latin1 / convert_utf16*_to_utf32 / convert_utf16*_to_utf8
// tests. Canary and short-destination checks are Go-specific slice-contract
// coverage.

func rawUTF16Words(native []uint16, little bool) []uint16 {
	out := make([]uint16, len(native))
	nativeLittle := nativeLittleEndian()
	for i, word := range native {
		if little == nativeLittle {
			out[i] = word
		} else {
			out[i] = bits.ReverseBytes16(word)
		}
	}
	return out
}

func TestUTF16ConvertToLatin1(t *testing.T) {
	cases := []struct {
		name   string
		native []uint16
		want   []byte
		ok     bool
	}{
		{"empty", nil, nil, true},
		{"ascii", []uint16{'a', 'b', 'c'}, []byte("abc"), true},
		{"latin1", []uint16{0x00, 0x7f, 0xff}, []byte{0x00, 0x7f, 0xff}, true},
		{"too_large", []uint16{'a', 0x100, 'b'}, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, little := range []bool{true, false} {
				input := rawUTF16Words(tc.native, little)
				dst := make([]byte, len(input)+1)
				dst[len(input)] = 0x5a
				var n int
				var result Result
				if little {
					n = convertUTF16LEToLatin1Scalar(input, dst[:len(input)])
					result = convertUTF16LEToLatin1WithErrorsScalar(input, dst[:len(input)])
				} else {
					n = convertUTF16BEToLatin1Scalar(input, dst[:len(input)])
					result = convertUTF16BEToLatin1WithErrorsScalar(input, dst[:len(input)])
				}
				if tc.ok {
					if n != len(tc.want) || !bytes.Equal(dst[:n], tc.want) {
						t.Fatalf("little=%v convert = %d %x, want %d %x", little, n, dst[:n], len(tc.want), tc.want)
					}
					if result.Error != Success || result.Count != len(tc.want) {
						t.Fatalf("little=%v with_errors = %+v", little, result)
					}
					if little {
						if got := convertValidUTF16LEToLatin1Scalar(input, dst[:len(input)]); got != len(tc.want) {
							t.Fatalf("valid LE = %d", got)
						}
					} else if got := convertValidUTF16BEToLatin1Scalar(input, dst[:len(input)]); got != len(tc.want) {
						t.Fatalf("valid BE = %d", got)
					}
				} else {
					if n != 0 {
						t.Fatalf("little=%v convert = %d, want 0", little, n)
					}
					if result.Error != TooLarge {
						t.Fatalf("little=%v with_errors = %+v, want TooLarge", little, result)
					}
				}
				if dst[len(input)] != 0x5a {
					t.Fatalf("little=%v canary overwritten", little)
				}
			}
		})
	}
}

func TestUTF16ConvertToUTF32(t *testing.T) {
	cases := []struct {
		name   string
		native []uint16
		want   []uint32
		err    ErrorCode
		count  int
	}{
		{"empty", nil, nil, Success, 0},
		{"bmp", []uint16{'A', 0x20ac}, []uint32{'A', 0x20ac}, Success, 2},
		{"pair", []uint16{0xd83d, 0xde00}, []uint32{0x1f600}, Success, 1},
		{"high_only", []uint16{0xd800}, nil, Surrogate, 0},
		{"low_only", []uint16{0xdc00}, nil, Surrogate, 0},
		{"bad_low", []uint16{0xd800, 0x0041}, nil, Surrogate, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, little := range []bool{true, false} {
				input := rawUTF16Words(tc.native, little)
				need := utf32LengthFromUTF16Scalar(input, little == nativeLittleEndian())
				dst := make([]uint32, need+1)
				dst[need] = 0x5a5a5a5a
				var n int
				var result Result
				if little {
					n = convertUTF16LEToUTF32Scalar(input, dst[:need])
					result = convertUTF16LEToUTF32WithErrorsScalar(input, dst[:need])
				} else {
					n = convertUTF16BEToUTF32Scalar(input, dst[:need])
					result = convertUTF16BEToUTF32WithErrorsScalar(input, dst[:need])
				}
				if tc.err == Success {
					if n != len(tc.want) {
						t.Fatalf("little=%v convert = %d, want %d", little, n, len(tc.want))
					}
					for i := range tc.want {
						if dst[i] != tc.want[i] {
							t.Fatalf("little=%v dst[%d]=%#x want %#x", little, i, dst[i], tc.want[i])
						}
					}
					if result.Error != Success || result.Count != len(tc.want) {
						t.Fatalf("little=%v with_errors=%+v", little, result)
					}
					if little {
						if got := convertValidUTF16LEToUTF32Scalar(input, dst[:need]); got != len(tc.want) {
							t.Fatalf("valid LE = %d", got)
						}
					} else if got := convertValidUTF16BEToUTF32Scalar(input, dst[:need]); got != len(tc.want) {
						t.Fatalf("valid BE = %d", got)
					}
				} else {
					if n != 0 {
						t.Fatalf("little=%v convert = %d, want 0", little, n)
					}
					if result.Error != tc.err || result.Count != tc.count {
						t.Fatalf("little=%v with_errors=%+v want {%v %d}", little, result, tc.err, tc.count)
					}
				}
				if dst[need] != 0x5a5a5a5a {
					t.Fatalf("little=%v canary overwritten", little)
				}
			}
		})
	}
}

func TestUTF16ConvertShortDestinationPanics(t *testing.T) {
	input := []uint16{'a', 'b'}
	mustPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Fatalf("%s did not panic", name)
			}
		}()
		fn()
	}
	mustPanic("latin1", func() { convertUTF16LEToLatin1Scalar(input, make([]byte, 1)) })
	mustPanic("utf32", func() { convertUTF16LEToUTF32Scalar(input, make([]uint32, 0)) })
	mustPanic("utf8", func() { convertUTF16LEToUTF8Scalar(input, make([]byte, 0)) })
	mustPanic("utf8_replacement", func() {
		convertUTF16LEToUTF8WithReplacementScalar([]uint16{0xd800}, make([]byte, 2))
	})
}

func TestUTF16ConvertPublicDispatch(t *testing.T) {
	input := rawUTF16Words([]uint16{'x', 0xff}, true)
	dst := make([]byte, len(input))
	if got := ConvertUTF16LEToLatin1(input, dst); got != 2 || string(dst) != "x\xff" {
		t.Fatalf("ConvertUTF16LEToLatin1 = %d %q", got, dst)
	}
	u32 := make([]uint32, 2)
	if got := ConvertUTF16LEToUTF32(rawUTF16Words([]uint16{'A', 0xd83d, 0xde00}, true), u32); got != 2 || u32[0] != 'A' || u32[1] != 0x1f600 {
		t.Fatalf("ConvertUTF16LEToUTF32 = %d %x", got, u32[:got])
	}
	utf16 := rawUTF16Words([]uint16{'A', 0x20ac, 0xd83d, 0xde00}, true)
	if got := UTF8LengthFromUTF16LE(utf16); got != 1+3+4 {
		t.Fatalf("UTF8LengthFromUTF16LE = %d", got)
	}
	if got := UTF32LengthFromUTF16LE(utf16); got != 3 {
		t.Fatalf("UTF32LengthFromUTF16LE = %d", got)
	}
	utf8 := make([]byte, UTF8LengthFromUTF16LE(utf16))
	if got := ConvertUTF16LEToUTF8(utf16, utf8); got != len(utf8) || string(utf8) != "A\xe2\x82\xac\xf0\x9f\x98\x80" {
		t.Fatalf("ConvertUTF16LEToUTF8 = %d %q", got, utf8)
	}
}

func TestUTF16ConvertToUTF8(t *testing.T) {
	cases := []struct {
		name        string
		native      []uint16
		want        []byte
		wantReplace []byte
		err         ErrorCode
		errCount    int
	}{
		{"empty", nil, nil, nil, Success, 0},
		{"ascii", []uint16{'a', 'b', 'c'}, []byte("abc"), []byte("abc"), Success, 0},
		{"two_byte", []uint16{0x00a9}, []byte("\xc2\xa9"), []byte("\xc2\xa9"), Success, 0},
		{"three_byte", []uint16{0x20ac}, []byte("\xe2\x82\xac"), []byte("\xe2\x82\xac"), Success, 0},
		{"pair", []uint16{0xd83d, 0xde00}, []byte("\xf0\x9f\x98\x80"), []byte("\xf0\x9f\x98\x80"), Success, 0},
		{"mixed", []uint16{'A', 0x20ac, 0xd83d, 0xde00}, []byte("A\xe2\x82\xac\xf0\x9f\x98\x80"), []byte("A\xe2\x82\xac\xf0\x9f\x98\x80"), Success, 0},
		{"high_only", []uint16{0xd800}, nil, []byte("\xef\xbf\xbd"), Surrogate, 0},
		{"low_only", []uint16{0xdc00}, nil, []byte("\xef\xbf\xbd"), Surrogate, 0},
		{"bad_low", []uint16{0xd800, 0x0041}, nil, []byte("\xef\xbf\xbdA"), Surrogate, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, little := range []bool{true, false} {
				input := rawUTF16Words(tc.native, little)
				storageNative := little == nativeLittleEndian()
				need := utf8LengthFromUTF16Scalar(input, storageNative)
				needReplace := utf8LengthFromUTF16WithReplacementScalar(input, storageNative)
				if little {
					if got := utf8LengthFromUTF16LEScalar(input); got != need {
						t.Fatalf("utf8 length LE = %d, want %d", got, need)
					}
				} else if got := utf8LengthFromUTF16BEScalar(input); got != need {
					t.Fatalf("utf8 length BE = %d, want %d", got, need)
				}
				dst := make([]byte, need+1)
				dst[need] = 0x5a
				replaceDst := make([]byte, needReplace+1)
				replaceDst[needReplace] = 0x5a
				var n int
				var result Result
				var replaced int
				var valid int
				if little {
					n = convertUTF16LEToUTF8Scalar(input, dst[:need])
					result = convertUTF16LEToUTF8WithErrorsScalar(input, dst[:need])
					replaced = convertUTF16LEToUTF8WithReplacementScalar(input, replaceDst[:needReplace])
					if tc.err == Success {
						valid = convertValidUTF16LEToUTF8Scalar(input, dst[:need])
					}
				} else {
					n = convertUTF16BEToUTF8Scalar(input, dst[:need])
					result = convertUTF16BEToUTF8WithErrorsScalar(input, dst[:need])
					replaced = convertUTF16BEToUTF8WithReplacementScalar(input, replaceDst[:needReplace])
					if tc.err == Success {
						valid = convertValidUTF16BEToUTF8Scalar(input, dst[:need])
					}
				}
				if tc.err == Success {
					if n != len(tc.want) || !bytes.Equal(dst[:n], tc.want) {
						t.Fatalf("little=%v convert = %d %x, want %d %x", little, n, dst[:n], len(tc.want), tc.want)
					}
					if result.Error != Success || result.Count != len(tc.want) {
						t.Fatalf("little=%v with_errors=%+v", little, result)
					}
					if valid != len(tc.want) {
						t.Fatalf("little=%v valid = %d", little, valid)
					}
				} else {
					if n != 0 {
						t.Fatalf("little=%v convert = %d, want 0", little, n)
					}
					if result.Error != tc.err || result.Count != tc.errCount {
						t.Fatalf("little=%v with_errors=%+v want {%v %d}", little, result, tc.err, tc.errCount)
					}
				}
				if replaced != len(tc.wantReplace) || !bytes.Equal(replaceDst[:replaced], tc.wantReplace) {
					t.Fatalf("little=%v replacement = %d %x, want %d %x", little, replaced, replaceDst[:replaced], len(tc.wantReplace), tc.wantReplace)
				}
				if dst[need] != 0x5a {
					t.Fatalf("little=%v convert canary overwritten", little)
				}
				if replaceDst[needReplace] != 0x5a {
					t.Fatalf("little=%v replacement canary overwritten", little)
				}
			}
		})
	}
}

func TestUTF16HelpersScalar(t *testing.T) {
	cases := []struct {
		name        string
		native      []uint16
		count       int
		trim        int
		utf8Replace Result
	}{
		{"empty", nil, 0, 0, Result{Error: Success, Count: 0}},
		{"bmp", []uint16{'A', 0x20ac}, 2, 2, Result{Error: Success, Count: 1 + 3}},
		{"pair", []uint16{0xd83d, 0xde00}, 1, 2, Result{Error: Surrogate, Count: 4}},
		{"high_only", []uint16{0xd800}, 1, 0, Result{Error: Surrogate, Count: 3}},
		{"low_only", []uint16{0xdc00}, 0, 1, Result{Error: Surrogate, Count: 3}},
		{"trim_lead", []uint16{'A', 0xd83d}, 2, 1, Result{Error: Surrogate, Count: 1 + 3}},
		{"unpaired_high_ascii", []uint16{0xd800, 'A'}, 2, 2, Result{Error: Surrogate, Count: 3 + 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, little := range []bool{true, false} {
				input := rawUTF16Words(tc.native, little)
				var count, trim int
				var replace Result
				if little {
					count = countUTF16LEScalar(input)
					trim = trimPartialUTF16LEScalar(input)
					replace = utf8LengthFromUTF16LEWithReplacementScalar(input)
				} else {
					count = countUTF16BEScalar(input)
					trim = trimPartialUTF16BEScalar(input)
					replace = utf8LengthFromUTF16BEWithReplacementScalar(input)
				}
				if count != tc.count {
					t.Fatalf("little=%v count = %d, want %d", little, count, tc.count)
				}
				if trim != tc.trim {
					t.Fatalf("little=%v trim = %d, want %d", little, trim, tc.trim)
				}
				if replace != tc.utf8Replace {
					t.Fatalf("little=%v utf8Replace = %+v, want %+v", little, replace, tc.utf8Replace)
				}
			}
		})
	}
}

func TestChangeEndiannessUTF16Scalar(t *testing.T) {
	input := []uint16{0x1234, 0xd83d, 0xde00}
	want := []uint16{0x3412, 0x3dd8, 0x00de}

	dst := make([]uint16, len(input)+1)
	dst[len(input)] = 0x5a5a
	changeEndiannessUTF16Scalar(input, dst[:len(input)])
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("out-of-place dst[%d]=%#x, want %#x", i, dst[i], want[i])
		}
	}
	if dst[len(input)] != 0x5a5a {
		t.Fatal("canary overwritten")
	}

	inPlace := append([]uint16(nil), input...)
	changeEndiannessUTF16Scalar(inPlace, inPlace)
	for i := range want {
		if inPlace[i] != want[i] {
			t.Fatalf("in-place dst[%d]=%#x, want %#x", i, inPlace[i], want[i])
		}
	}

	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	changeEndiannessUTF16Scalar(input, make([]uint16, len(input)-1))
}

func TestUTF16HelpersPublicAPI(t *testing.T) {
	if got := Latin1LengthFromUTF16(7); got != 7 {
		t.Fatalf("Latin1LengthFromUTF16 = %d", got)
	}

	native := []uint16{'A', 0x20ac, 0xd83d, 0xde00}
	le := rawUTF16Words(native, true)
	be := rawUTF16Words(native, false)

	if got := CountUTF16LE(le); got != 3 {
		t.Fatalf("CountUTF16LE = %d", got)
	}
	if got := CountUTF16BE(be); got != 3 {
		t.Fatalf("CountUTF16BE = %d", got)
	}
	if got := UTF32LengthFromUTF16(native); got != 3 {
		t.Fatalf("UTF32LengthFromUTF16 = %d", got)
	}
	if got := UTF8LengthFromUTF16(native); got != 1+3+4 {
		t.Fatalf("UTF8LengthFromUTF16 = %d", got)
	}

	dst := make([]byte, UTF8LengthFromUTF16(native))
	if got := ConvertValidUTF16ToUTF8(native, dst); got != len(dst) || string(dst) != "A\xe2\x82\xac\xf0\x9f\x98\x80" {
		t.Fatalf("ConvertValidUTF16ToUTF8 = %d %q", got, dst)
	}

	if got := TrimPartialUTF16LE(rawUTF16Words([]uint16{'A', 0xd83d}, true)); got != 1 {
		t.Fatalf("TrimPartialUTF16LE = %d", got)
	}
	if got := TrimPartialUTF16BE(rawUTF16Words([]uint16{'A', 0xd83d}, false)); got != 1 {
		t.Fatalf("TrimPartialUTF16BE = %d", got)
	}
	if got := TrimPartialUTF16([]uint16{'A', 0xd83d}); got != 1 {
		t.Fatalf("TrimPartialUTF16 = %d", got)
	}

	swapped := make([]uint16, len(le))
	ChangeEndiannessUTF16(le, swapped)
	for i := range le {
		if swapped[i] != bits.ReverseBytes16(le[i]) {
			t.Fatalf("ChangeEndiannessUTF16[%d]=%#x, want %#x", i, swapped[i], bits.ReverseBytes16(le[i]))
		}
	}
	ChangeEndiannessUTF16(swapped, swapped)
	for i := range le {
		if swapped[i] != le[i] {
			t.Fatalf("round-trip ChangeEndiannessUTF16[%d]=%#x, want %#x", i, swapped[i], le[i])
		}
	}

	if got := UTF8LengthFromUTF16LEWithReplacement(rawUTF16Words([]uint16{0xd800}, true)); got != (Result{Error: Surrogate, Count: 3}) {
		t.Fatalf("UTF8LengthFromUTF16LEWithReplacement = %+v", got)
	}
	if got := UTF8LengthFromUTF16BEWithReplacement(rawUTF16Words([]uint16{'A'}, false)); got != (Result{Error: Success, Count: 1}) {
		t.Fatalf("UTF8LengthFromUTF16BEWithReplacement = %+v", got)
	}
	if got := UTF8LengthFromUTF16LEWithReplacement(rawUTF16Words([]uint16{0xd83d, 0xde00}, true)); got != (Result{Error: Surrogate, Count: 4}) {
		t.Fatalf("paired surrogate flag = %+v", got)
	}
}

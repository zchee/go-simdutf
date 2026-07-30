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
// convert_utf16*_to_latin1 / convert_utf16*_to_utf32 tests. Canary and
// short-destination checks are Go-specific slice-contract coverage.

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
}

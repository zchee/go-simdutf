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

// Portions Copyright 2021 The simdutf Authors.
//go:build arm64

package simdutf

// Direct differential coverage for the arm64 translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b.

import (
	"math/bits"
	"slices"
	"testing"
)

func rawUTF16NEONTest(words []uint16, little bool) []uint16 {
	out := append([]uint16(nil), words...)
	if !little {
		for i := range out {
			out[i] = bits.ReverseBytes16(out[i])
		}
	}
	return out
}

func TestUTF16NEONDirectAgainstScalar(t *testing.T) {
	semantic := []uint16{0x0061, 0xd800, 0xdc00, 0x0062, 0xdc00, 0x0063, 0xdbff, 0x0064, 0xd800}
	for _, little := range []bool{true, false} {
		for _, n := range []int{0, 1, 15, 16, 17, 31, 32, 33} {
			for _, bad := range []int{-1, 0, n / 2, n - 1} {
				if bad >= n {
					continue
				}
				words := make([]uint16, n)
				for i := range words {
					words[i] = 0x0061
				}
				if bad >= 0 {
					words[bad] = semantic[(bad+4)%len(semantic)]
				}
				input := rawUTF16NEONTest(words, little)
				var gotBool, wantBool bool
				var gotResult, wantResult Result
				if little {
					gotBool, wantBool = validateUTF16LENEON(input), validateUTF16LEScalar(input)
					gotResult, wantResult = validateUTF16LEWithErrorsNEON(input), validateUTF16LEWithErrorsScalar(input)
				} else {
					gotBool, wantBool = validateUTF16BENEON(input), validateUTF16BEScalar(input)
					gotResult, wantResult = validateUTF16BEWithErrorsNEON(input), validateUTF16BEWithErrorsScalar(input)
				}
				if gotBool != wantBool || gotResult != wantResult {
					t.Fatalf("little=%t n=%d bad=%d: bool=%t/%t result=%+v/%+v", little, n, bad, gotBool, wantBool, gotResult, wantResult)
				}
			}
		}
	}
}

func TestToWellFormedUTF16NEONDirect(t *testing.T) {
	semantic := []uint16{0x0061, 0xd800, 0xdc00, 0xdc00, 0xdbff, 0x0062, 0xd800}
	for _, little := range []bool{true, false} {
		input := rawUTF16NEONTest(semantic, little)
		want := make([]uint16, len(input))
		if little {
			toWellFormedUTF16LEScalar(input, want)
		} else {
			toWellFormedUTF16BEScalar(input, want)
		}
		buf := make([]uint16, len(input)+2)
		buf[0], buf[len(buf)-1] = 0xa55a, 0x5aa5
		if little {
			toWellFormedUTF16LENEON(input, buf[1:len(buf)-1])
		} else {
			toWellFormedUTF16BENEON(input, buf[1:len(buf)-1])
		}
		if buf[0] != 0xa55a || buf[len(buf)-1] != 0x5aa5 || !slices.Equal(buf[1:len(buf)-1], want) {
			t.Fatalf("little=%t: canary or output differs: %x want %x", little, buf, want)
		}
		inPlace := append([]uint16(nil), input...)
		if little {
			toWellFormedUTF16LENEON(inPlace, inPlace)
		} else {
			toWellFormedUTF16BENEON(inPlace, inPlace)
		}
		if !slices.Equal(inPlace, want) {
			t.Fatalf("little=%t: in-place output=%x want=%x", little, inPlace, want)
		}
		short := []uint16{0xa55a, 0x5aa5}
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("short destination did not panic")
				}
			}()
			if little {
				toWellFormedUTF16LENEON(input, short)
			} else {
				toWellFormedUTF16BENEON(input, short)
			}
		}()
		if !slices.Equal(short, []uint16{0xa55a, 0x5aa5}) {
			t.Fatalf("little=%t: short destination was modified: %x", little, short)
		}
	}
}

func TestUTF32NEONDirectAgainstScalar(t *testing.T) {
	for _, n := range []int{0, 1, 3, 4, 5, 7, 8, 9} {
		for _, invalid := range []uint32{0, 0xd800, 0xdfff, 0x110000} {
			input := make([]uint32, n)
			for i := range input {
				input[i] = 0x10ffff
			}
			if n > 0 && invalid != 0 {
				input[n/2] = invalid
			}
			if got, want := validateUTF32NEON(input), validateUTF32Scalar(input); got != want {
				t.Fatalf("n=%d invalid=%x: bool=%t want=%t", n, invalid, got, want)
			}
			if got, want := validateUTF32WithErrorsNEON(input), validateUTF32WithErrorsScalar(input); got != want {
				t.Fatalf("n=%d invalid=%x: result=%+v want=%+v", n, invalid, got, want)
			}
		}
	}
}

func FuzzUTFValidationNEONAgainstScalar(f *testing.F) {
	for _, seed := range [][]byte{nil, {0, 0}, {0, 0xd8, 0, 0xdc}, {0, 0, 0x11, 0}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		semantic := fuzzUTF16Words(data)
		for _, little := range []bool{true, false} {
			input := rawUTF16Scalar(semantic, little)
			var gotResult, wantResult Result
			gotDst, wantDst := make([]uint16, len(input)), make([]uint16, len(input))
			if little {
				gotResult = validateUTF16LEWithErrorsNEON(input)
				wantResult = validateUTF16LEWithErrorsScalar(input)
				toWellFormedUTF16LENEON(input, gotDst)
				toWellFormedUTF16LEScalar(input, wantDst)
			} else {
				gotResult = validateUTF16BEWithErrorsNEON(input)
				wantResult = validateUTF16BEWithErrorsScalar(input)
				toWellFormedUTF16BENEON(input, gotDst)
				toWellFormedUTF16BEScalar(input, wantDst)
			}
			if gotResult != wantResult || !slices.Equal(gotDst, wantDst) {
				t.Fatalf("UTF-16 little=%t: result=%+v/%+v output=%x/%x", little, gotResult, wantResult, gotDst, wantDst)
			}
		}

		input32 := make([]uint32, len(data)/4)
		for i := range input32 {
			input32[i] = uint32(data[4*i]) | uint32(data[4*i+1])<<8 |
				uint32(data[4*i+2])<<16 | uint32(data[4*i+3])<<24
		}
		if got, want := validateUTF32WithErrorsNEON(input32), validateUTF32WithErrorsScalar(input32); got != want {
			t.Fatalf("UTF-32 result=%+v want=%+v input=%x", got, want, input32)
		}
	})
}

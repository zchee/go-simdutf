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
)

// Fuzz invariants adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/scalar/utf16.h:20-76,183-213 and
// include/simdutf/scalar/utf32.h:8-50.

func FuzzUTF16ScalarEndianInvariants(f *testing.F) {
	for _, seed := range [][]byte{nil, {0, 0}, {0, 0xd8}, {0, 0xdc}, {0, 0xd8, 0, 0xdc}, {0xff, 0xdb, 0x61, 0}, {0xff, 0xd7, 0, 0xe0}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		semantic := fuzzUTF16Words(data)
		little := rawUTF16Scalar(semantic, true)
		big := rawUTF16Scalar(semantic, false)
		leResult := validateUTF16LEWithErrorsScalar(little)
		beResult := validateUTF16BEWithErrorsScalar(big)
		if leResult != beResult {
			t.Fatalf("endian validation differs: LE=%+v BE=%+v semantic=%x", leResult, beResult, semantic)
		}
		if validateUTF16LEScalar(little) != leResult.IsOK() || validateUTF16BEScalar(big) != beResult.IsOK() {
			t.Fatalf("boolean validator differs from result validator: semantic=%x", semantic)
		}

		leDst := make([]uint16, len(little))
		beDst := make([]uint16, len(big))
		toWellFormedUTF16LEScalar(little, leDst)
		toWellFormedUTF16BEScalar(big, beDst)
		if !validateUTF16LEScalar(leDst) || !validateUTF16BEScalar(beDst) {
			t.Fatalf("repair did not produce valid UTF-16: semantic=%x", semantic)
		}
		if !slices.Equal(fuzzUTF16Semantic(leDst, true), fuzzUTF16Semantic(beDst, false)) {
			t.Fatalf("endian repair differs: LE=%x BE=%x", leDst, beDst)
		}
		if leResult.IsOK() && !slices.Equal(leDst, little) {
			t.Fatalf("repair changed valid UTF-16: input=%x output=%x", little, leDst)
		}

		if len(little) > 0 {
			canary := uint16(0xa55a)
			short := make([]uint16, len(little)-1)
			for i := range short {
				short[i] = canary
			}
			func() {
				defer func() {
					if recover() == nil {
						t.Fatal("short destination did not panic")
					}
				}()
				toWellFormedUTF16LEScalar(little, short)
			}()
			for _, word := range short {
				if word != canary {
					t.Fatalf("short destination was modified: %x", short)
				}
			}
		}
	})
}

func FuzzUTF32ScalarInvariants(f *testing.F) {
	for _, seed := range [][]byte{nil, {0, 0, 0, 0}, {0xff, 0xff, 0x10, 0}, {0, 0xd8, 0, 0}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		input := make([]uint32, len(data)/4)
		for i := range input {
			input[i] = uint32(data[i*4]) | uint32(data[i*4+1])<<8 | uint32(data[i*4+2])<<16 | uint32(data[i*4+3])<<24
		}
		result := validateUTF32WithErrorsScalar(input)
		if validateUTF32Scalar(input) != result.IsOK() {
			t.Fatalf("boolean validator differs from result validator: input=%x result=%+v", input, result)
		}
		if result.IsOK() && result.Count != len(input) {
			t.Fatalf("successful validation count=%d want=%d", result.Count, len(input))
		}
		if result.IsErr() && result.Count >= len(input) {
			t.Fatalf("error index=%d outside input length=%d", result.Count, len(input))
		}
	})
}

func fuzzUTF16Words(data []byte) []uint16 {
	words := make([]uint16, len(data)/2)
	for i := range words {
		words[i] = uint16(data[i*2]) | uint16(data[i*2+1])<<8
	}
	return words
}

func fuzzUTF16Semantic(raw []uint16, little bool) []uint16 {
	semantic := append([]uint16(nil), raw...)
	if little != nativeLittleEndian() {
		for i := range semantic {
			semantic[i] = bits.ReverseBytes16(semantic[i])
		}
	}
	return semantic
}

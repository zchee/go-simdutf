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

//go:build amd64 && goexperiment.simd

package simdutf

import (
	"slices"
	"testing"
)

func TestToWellFormedUTF16ArchsimdDifferential(t *testing.T) {
	cases := [][]uint16{
		nil, make([]uint16, 15), make([]uint16, 16), make([]uint16, 17),
		append(make([]uint16, 15), 0xd800),
		append(append(make([]uint16, 15), 0xd800), 0xdc00),
		{0xdc00, 0x61, 0xd800, 0x62, 0xd800, 0xdc00, 0xdc00},
	}
	for _, semantic := range cases {
		for _, bigEndian := range []bool{false, true} {
			input := append([]uint16(nil), semantic...)
			if bigEndian {
				for i := range input {
					input[i] = input[i]>>8 | input[i]<<8
				}
			}
			want := make([]uint16, len(input))
			got := make([]uint16, len(input)+2)
			got[len(input)], got[len(input)+1] = 0xa55a, 0x5aa5
			if bigEndian {
				toWellFormedUTF16BEScalar(input, want)
				toWellFormedUTF16BEArchsimd(input, got[:len(input)])
			} else {
				toWellFormedUTF16LEScalar(input, want)
				toWellFormedUTF16LEArchsimd(input, got[:len(input)])
			}
			if !slices.Equal(got[:len(input)], want) || got[len(input)] != 0xa55a || got[len(input)+1] != 0x5aa5 {
				t.Fatalf("input=%#v be=%t got=%#v want=%#v", semantic, bigEndian, got, want)
			}
			inPlace := append([]uint16(nil), input...)
			if bigEndian {
				toWellFormedUTF16BEArchsimd(inPlace, inPlace)
			} else {
				toWellFormedUTF16LEArchsimd(inPlace, inPlace)
			}
			if !slices.Equal(inPlace, want) {
				t.Fatalf("in-place input=%#v be=%t got=%#v want=%#v", semantic, bigEndian, inPlace, want)
			}
		}
	}
}

func TestToWellFormedUTF16ArchsimdShortDestinationDoesNotStore(t *testing.T) {
	for _, repair := range []func([]uint16, []uint16){toWellFormedUTF16LEArchsimd, toWellFormedUTF16BEArchsimd} {
		dst := []uint16{0xa55a}
		panicked := false
		func() {
			defer func() { panicked = recover() != nil }()
			repair([]uint16{0xd800, 0x61}, dst)
		}()
		if !panicked || dst[0] != 0xa55a {
			t.Fatalf("panicked=%t dst=%#v", panicked, dst)
		}
	}
}

func FuzzUTFValidationArchsimdAgainstScalar(f *testing.F) {
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
				gotResult = validateUTF16LEWithErrorsArchsimd(input)
				wantResult = validateUTF16LEWithErrorsScalar(input)
				toWellFormedUTF16LEArchsimd(input, gotDst)
				toWellFormedUTF16LEScalar(input, wantDst)
			} else {
				gotResult = validateUTF16BEWithErrorsArchsimd(input)
				wantResult = validateUTF16BEWithErrorsScalar(input)
				toWellFormedUTF16BEArchsimd(input, gotDst)
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
		if got, want := validateUTF32WithErrorsArchsimd(input32), validateUTF32WithErrorsScalar(input32); got != want {
			t.Fatalf("UTF-32 result=%+v want=%+v input=%x", got, want, input32)
		}
	})
}

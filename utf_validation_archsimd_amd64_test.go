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

import "testing"

func TestValidateUTF16ArchsimdDifferential(t *testing.T) {
	cases := [][]uint16{
		nil,
		make([]uint16, 15), make([]uint16, 16), make([]uint16, 17),
		append(make([]uint16, 15), 0xd800),
		append(make([]uint16, 16), 0xdc00),
		append(append(make([]uint16, 15), 0xd800), 0xdc00),
		{0xdc00, 0xd800, 0x61}, {0xd800, 0x61}, {0xd800},
	}
	for _, input := range cases {
		for _, bigEndian := range []bool{false, true} {
			raw := append([]uint16(nil), input...)
			if bigEndian {
				for i := range raw {
					raw[i] = raw[i]>>8 | raw[i]<<8
				}
			}
			var got, want Result
			var gotBool, wantBool bool
			if bigEndian {
				got, want = validateUTF16BEWithErrorsArchsimd(raw), validateUTF16BEWithErrorsScalar(raw)
				gotBool, wantBool = validateUTF16BEArchsimd(raw), validateUTF16BEScalar(raw)
			} else {
				got, want = validateUTF16LEWithErrorsArchsimd(raw), validateUTF16LEWithErrorsScalar(raw)
				gotBool, wantBool = validateUTF16LEArchsimd(raw), validateUTF16LEScalar(raw)
			}
			if got != want || gotBool != wantBool {
				t.Fatalf("input=%#v be=%t got=(%v,%t) want=(%v,%t)", input, bigEndian, got, gotBool, want, wantBool)
			}
		}
	}
}

func TestValidateUTF32ArchsimdDifferential(t *testing.T) {
	cases := [][]uint32{
		nil, make([]uint32, 7), make([]uint32, 8), make([]uint32, 9),
		{0x10ffff, 0x61, 0xd7ff}, {0xd800}, {0xdfff}, {0x110000},
		append(make([]uint32, 7), 0x110000), append(make([]uint32, 8), 0xd800),
	}
	for _, input := range cases {
		got, want := validateUTF32WithErrorsArchsimd(input), validateUTF32WithErrorsScalar(input)
		gotBool, wantBool := validateUTF32Archsimd(input), validateUTF32Scalar(input)
		if got != want || gotBool != wantBool {
			t.Fatalf("input=%#v got=(%v,%t) want=(%v,%t)", input, got, gotBool, want, wantBool)
		}
	}
}

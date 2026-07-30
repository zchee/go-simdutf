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

//go:build arm64

package simdutf

import "testing"

func TestDirectARM64UTF16HelpersAgainstScalar(t *testing.T) {
	cases := [][]uint16{
		{},
		{0x61, 0x62},
		{0x00ff, 0xff00, 0xd800, 0xdc00, 0xd83d, 0xde00},
		make([]uint16, 32),
		make([]uint16, 33),
		make([]uint16, 64),
	}
	for i := range cases[3] {
		cases[3][i] = uint16(0x4e00 + i)
	}
	for i := range cases[4] {
		cases[4][i] = uint16(i * 7)
	}
	for i := range cases[5] {
		if i%4 == 0 {
			cases[5][i] = 0xd800
		} else if i%4 == 1 {
			cases[5][i] = 0xdc00
		} else {
			cases[5][i] = uint16(i)
		}
	}
	for _, input := range cases {
		want := make([]uint16, len(input))
		got := make([]uint16, len(input))
		changeEndiannessUTF16Scalar(input, want)
		changeEndiannessUTF16NEON(input, got)
		if len(got) != len(want) {
			t.Fatalf("changeEndianness NEON length mismatch")
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("changeEndianness NEON mismatch input=%v want=%v got=%v", input, want, got)
			}
		}
		if countUTF16LENEON(input) != countUTF16LEScalar(input) {
			t.Fatalf("countLE NEON mismatch input=%v", input)
		}
		if countUTF16BENEON(input) != countUTF16BEScalar(input) {
			t.Fatalf("countBE NEON mismatch input=%v", input)
		}
		if utf32LengthFromUTF16LENEON(input) != utf32LengthFromUTF16LEScalar(input) {
			t.Fatalf("utf32LengthLE NEON mismatch")
		}
		if utf32LengthFromUTF16BENEON(input) != utf32LengthFromUTF16BEScalar(input) {
			t.Fatalf("utf32LengthBE NEON mismatch")
		}
	}
}

func TestDirectARM64UTF16HelpersPreflightPreservesDestination(t *testing.T) {
	input := []uint16{1, 2, 3, 4}
	dst := []uint16{9, 9}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		changeEndiannessUTF16NEON(input, dst)
	}()
	if dst[0] != 9 || dst[1] != 9 {
		t.Fatal("NEON preflight mutated short destination")
	}
}

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

//go:build amd64

package simdutf

import "testing"

func requireUTF16HelpersAMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skipf("missing required CPU features %#x", feature)
	}
}

func TestDirectAMD64UTF16HelpersAgainstScalar(t *testing.T) {
	cases := [][]uint16{
		{},
		{0},
		{0x61, 0x62, 0x63},
		{0x00ff, 0xff00, 0x1234, 0xd800, 0xdc00, 0xd83d, 0xde00},
		{0xd800}, // lone high
		{0xdc00}, // lone low
		make([]uint16, 31),
		make([]uint16, 32),
		make([]uint16, 33),
		make([]uint16, 64),
		make([]uint16, 65),
	}
	for i := range cases[6] {
		cases[6][i] = uint16(i * 3)
		cases[7][i%32] = uint16(0x100 + i)
	}
	for i := range cases[8] {
		cases[8][i] = uint16(0xd800 + (i % 0x400))
	}
	for i := range cases[9] {
		if i%5 == 0 {
			cases[9][i] = 0xd83d
		} else if i%5 == 1 {
			cases[9][i] = 0xde00
		} else {
			cases[9][i] = uint16(0x4e00 + i)
		}
	}
	for i := range cases[10] {
		cases[10][i] = uint16(i)
	}

	requireUTF16HelpersAMD64Variant(t, cpuSSSE3)
	for _, input := range cases {
		checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Westmere, countUTF16LEWestmere, countUTF16BEWestmere, utf32LengthFromUTF16LEWestmere, utf32LengthFromUTF16BEWestmere)
	}
	if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
		for _, input := range cases {
			checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Haswell, countUTF16LEHaswell, countUTF16BEHaswell, utf32LengthFromUTF16LEHaswell, utf32LengthFromUTF16BEHaswell)
		}
	}
}

func TestDirectAMD64UTF16HelpersPreflightPreservesDestination(t *testing.T) {
	requireUTF16HelpersAMD64Variant(t, cpuSSSE3)
	input := []uint16{1, 2, 3, 4}
	dst := []uint16{9, 9}
	requireUTF16HelpersPanic(t, func() { changeEndiannessUTF16Westmere(input, dst) })
	if dst[0] != 9 || dst[1] != 9 {
		t.Fatal("westmere preflight mutated short destination")
	}
	if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
		dst = []uint16{9, 9}
		requireUTF16HelpersPanic(t, func() { changeEndiannessUTF16Haswell(input, dst) })
		if dst[0] != 9 || dst[1] != 9 {
			t.Fatal("haswell preflight mutated short destination")
		}
	}
}

func requireUTF16HelpersPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func checkUTF16HelpersDirect(
	t *testing.T,
	input []uint16,
	change func([]uint16, []uint16),
	countLE, countBE, lenLE, lenBE func([]uint16) int,
) {
	t.Helper()
	want := make([]uint16, len(input))
	got := make([]uint16, len(input))
	changeEndiannessUTF16Scalar(input, want)
	change(input, got)
	if !equalU16(got, want) {
		t.Fatalf("changeEndianness mismatch input=%v want=%v got=%v", input, want, got)
	}
	if countLE(input) != countUTF16LEScalar(input) {
		t.Fatalf("countLE mismatch input=%v", input)
	}
	if countBE(input) != countUTF16BEScalar(input) {
		t.Fatalf("countBE mismatch input=%v", input)
	}
	if lenLE(input) != utf32LengthFromUTF16LEScalar(input) {
		t.Fatalf("utf32LengthLE mismatch input=%v", input)
	}
	if lenBE(input) != utf32LengthFromUTF16BEScalar(input) {
		t.Fatalf("utf32LengthBE mismatch input=%v", input)
	}
}

func FuzzUTF16HelpersAMD64AgainstScalar(f *testing.F) {
	f.Add([]byte{0, 0x7f, 0xd8, 0x00, 0xdc, 0x00, 0xff, 0xff})
	f.Fuzz(func(t *testing.T, raw []byte) {
		input := make([]uint16, len(raw)/2)
		for i := range input {
			input[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		if detectAMD64Features()&cpuSSSE3 == cpuSSSE3 {
			checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Westmere, countUTF16LEWestmere, countUTF16BEWestmere, utf32LengthFromUTF16LEWestmere, utf32LengthFromUTF16BEWestmere)
		}
		if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
			checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Haswell, countUTF16LEHaswell, countUTF16BEHaswell, utf32LengthFromUTF16LEHaswell, utf32LengthFromUTF16BEHaswell)
		}
	})
}

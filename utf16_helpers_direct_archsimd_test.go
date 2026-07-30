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

func TestDirectArchsimdUTF16HelpersAgainstScalar(t *testing.T) {
	if detectAMD64Features()&cpuAVX2 != cpuAVX2 {
		t.Skip("AVX2 unavailable")
	}
	cases := [][]uint16{
		{},
		{0x20, 0xd800, 0xdc00, 0xffff},
		make([]uint16, 16),
		make([]uint16, 17),
		make([]uint16, 48),
	}
	for i := range cases[2] {
		cases[2][i] = uint16(0x1000 + i)
	}
	for i := range cases[3] {
		cases[3][i] = uint16(i * 11)
	}
	for i := range cases[4] {
		cases[4][i] = uint16(0xdc00 - 8 + i)
	}
	for _, input := range cases {
		checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Archsimd, countUTF16LEArchsimd, countUTF16BEArchsimd, utf32LengthFromUTF16LEArchsimd, utf32LengthFromUTF16BEArchsimd)
	}
}

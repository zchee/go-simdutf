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

//go:build amd64 && goexperiment.simd

package simdutf

import (
	"simd/archsimd"
)

// Independently adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/utf16.h and
// src/generic/utf16/count_code_points_bytemask.h. Uses Uint16x16 Load/Store,
// ShiftAllLeft/ShiftAllRight byte-swap, And/Xor/Min/Add accumulation, and
// ClearAVXUpperBits. Requires AVX2 via the archsimd guard.

func changeEndiannessUTF16Archsimd(input, dst []uint16) {
	if len(dst) < len(input) {
		panic("simdutf: UTF-16 destination too short")
	}
	i := 0
	for ; i+16 <= len(input); i += 16 {
		v := archsimd.LoadUint16x16Slice(input[i:])
		swapped := v.ShiftAllLeft(8).Or(v.ShiftAllRight(8))
		swapped.StoreSlice(dst[i:])
	}
	archsimd.ClearAVXUpperBits()
	if i < len(input) {
		changeEndiannessUTF16Scalar(input[i:], dst[i:])
	}
}

func countUTF16LEArchsimd(input []uint16) int {
	return countUTF16Archsimd(input, false)
}

func countUTF16BEArchsimd(input []uint16) int {
	return countUTF16Archsimd(input, true)
}

func countUTF16Archsimd(input []uint16, bigEndian bool) int {
	if len(input) < 16 {
		if bigEndian {
			return countUTF16BEScalar(input)
		}
		return countUTF16LEScalar(input)
	}
	fc00 := archsimd.BroadcastUint16x16(0xfc00)
	dc00 := archsimd.BroadcastUint16x16(0xdc00)
	one := archsimd.BroadcastUint16x16(1)
	var counters archsimd.Uint16x16
	count := 0
	iterations := 0
	i := 0
	for ; i+16 <= len(input); i += 16 {
		v := archsimd.LoadUint16x16Slice(input[i:])
		if bigEndian {
			v = v.ShiftAllLeft(8).Or(v.ShiftAllRight(8))
		}
		t0 := v.And(fc00)
		t1 := t0.Xor(dc00)
		t2 := t1.Min(one)
		counters = counters.Add(t2)
		iterations++
		if iterations == 65535 {
			count += sumUint16x16(counters)
			counters = archsimd.Uint16x16{}
			iterations = 0
		}
	}
	if iterations != 0 {
		count += sumUint16x16(counters)
	}
	archsimd.ClearAVXUpperBits()
	tail := input[i:]
	if bigEndian {
		return count + countUTF16BEScalar(tail)
	}
	return count + countUTF16LEScalar(tail)
}

func sumUint16x16(v archsimd.Uint16x16) int {
	var lanes [16]uint16
	v.Store(&lanes)
	total := 0
	for _, lane := range lanes {
		total += int(lane)
	}
	return total
}

func utf32LengthFromUTF16LEArchsimd(input []uint16) int {
	return countUTF16LEArchsimd(input)
}

func utf32LengthFromUTF16BEArchsimd(input []uint16) int {
	return countUTF16BEArchsimd(input)
}

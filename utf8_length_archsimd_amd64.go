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
	"math/bits"
	"simd/archsimd"
)

// Translated from simdutf/simdutf at commit
// c7bef0ff14a13fd6ea52e3347da2c659383392de, tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811:
// src/generic/utf8/utf16_length_from_utf8_bytemask.h,
// src/generic/utf8.h:8-20, src/haswell/implementation.cpp:1155-1158,
// src/haswell/implementation.cpp:1288-1291, and
// src/simdutf/haswell/simd.h:99-101,282-286,341-345,353-355.

func latin1LengthFromUTF8Archsimd(input []byte) int {
	return countUTF8Archsimd(input)
}

func utf16LengthFromUTF8Archsimd(input []byte) int {
	if len(input) < 32 {
		return utf16LengthFromUTF8Scalar(input)
	}

	continuationThreshold := archsimd.BroadcastInt8x32(-65)
	fourByteThreshold := archsimd.BroadcastUint8x32(240)
	zero := archsimd.BroadcastUint8x32(0)
	var local archsimd.Uint8x32
	var counters archsimd.Uint64x4
	iterations := 0
	offset := 0
	for ; offset+32 <= len(input); offset += 32 {
		chunk := archsimd.LoadUint8x32Slice(input[offset : offset+32])
		continuation := chunk.AsInt8x32().Greater(continuationThreshold).ToInt8x32().AsUint8x32()
		fourByte := chunk.Min(fourByteThreshold).Equal(fourByteThreshold).ToInt8x32().AsUint8x32()
		local = local.Sub(continuation)
		local = local.Sub(fourByte)

		iterations++
		if iterations == 127 {
			counters = counters.Add(local.SumAbsDiff(zero).AsUint64x4())
			local = zero
			iterations = 0
		}
	}
	if iterations != 0 {
		counters = counters.Add(local.SumAbsDiff(zero).AsUint64x4())
	}

	var lanes [4]uint64
	counters.Store(&lanes)
	archsimd.ClearAVXUpperBits()
	length := lanes[0] + lanes[1] + lanes[2] + lanes[3]
	return int(length) + utf16LengthFromUTF8Scalar(input[offset:])
}

func utf32LengthFromUTF8Archsimd(input []byte) int {
	if len(input) < 64 {
		return utf32LengthFromUTF8Scalar(input)
	}

	threshold := archsimd.BroadcastInt8x32(-65)
	length := 0
	offset := 0
	for ; offset+64 <= len(input); offset += 64 {
		input0 := archsimd.LoadUint8x32Slice(input[offset : offset+32])
		input1 := archsimd.LoadUint8x32Slice(input[offset+32 : offset+64])
		mask0 := input0.AsInt8x32().Greater(threshold).ToBits()
		mask1 := input1.AsInt8x32().Greater(threshold).ToBits()
		mask := uint64(mask0) | uint64(mask1)<<32
		length += bits.OnesCount64(mask)
	}

	archsimd.ClearAVXUpperBits()
	return length + utf32LengthFromUTF8Scalar(input[offset:])
}

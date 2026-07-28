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

import "simd/archsimd"

// Independently adapted from the Haswell count_code_points_bytemask family in
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/utf8.h:21-68 and
// src/haswell/implementation.cpp:1115-1119. Each loop iteration loads four
// Uint8x32 vectors, applies the pinned signed int8 predicate input > -65,
// subtracts its all-ones masks into byte lanes, and widens with SumAbsDiff
// after exactly 63 iterations (255/4) before the lanes can wrap. The remainder
// stays with the pinned arbitrary-byte scalar oracle.
//
// Go 1.26.5 archsimd API provenance:
// src/simd/archsimd/slice_gen_amd64.go:149-162;
// src/simd/archsimd/compare_gen_amd64.go:134-136;
// src/simd/archsimd/ops_amd64.go:342-343,7140-7144,7365-7370,8027-8028,
// 8441-8442,8643-8644; src/simd/archsimd/other_gen_amd64.go:97-102; and
// src/simd/archsimd/extra_amd64.go:9-17.

func countUTF8Archsimd(input []byte) int {
	if len(input) < 128 {
		return countUTF8Scalar(input)
	}

	threshold := archsimd.BroadcastInt8x32(-65)
	var local archsimd.Uint8x32
	var counters archsimd.Uint64x4

	offset := 0
	iterations := 0
	for ; offset+128 <= len(input); offset += 128 {
		input0 := archsimd.LoadUint8x32Slice(input[offset:])
		input1 := archsimd.LoadUint8x32Slice(input[offset+32:])
		input2 := archsimd.LoadUint8x32Slice(input[offset+64:])
		input3 := archsimd.LoadUint8x32Slice(input[offset+96:])

		local = local.Sub(input0.AsInt8x32().Greater(threshold).ToInt8x32().AsUint8x32())
		local = local.Sub(input1.AsInt8x32().Greater(threshold).ToInt8x32().AsUint8x32())
		local = local.Sub(input2.AsInt8x32().Greater(threshold).ToInt8x32().AsUint8x32())
		local = local.Sub(input3.AsInt8x32().Greater(threshold).ToInt8x32().AsUint8x32())

		iterations++
		if iterations == 63 {
			counters = counters.Add(local.SumAbsDiff(archsimd.Uint8x32{}).AsUint64x4())
			local = archsimd.Uint8x32{}
			iterations = 0
		}
	}

	if iterations != 0 {
		counters = counters.Add(local.SumAbsDiff(archsimd.Uint8x32{}).AsUint64x4())
	}

	var lanes [4]uint64
	counters.Store(&lanes)
	archsimd.ClearAVXUpperBits()
	count := lanes[0] + lanes[1] + lanes[2] + lanes[3]
	return int(count) + countUTF8Scalar(input[offset:])
}

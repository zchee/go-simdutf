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
	"unsafe"
)

// Independently adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/find.h. Byte find
// aligns the start pointer to 64 bytes, compares two Uint8x32 vectors per
// iteration (simd8x64), and returns the first match via Mask8x32.ToBits /
// trailing zeroes. UTF-16 find aligns only when the byte misalignment is even,
// compares two Uint16x16 vectors (simd16x32), and extracts the AVX2 byte mask
// through ToInt16x16/AsInt8x32/ToMask/ToBits so each matching lane contributes
// two bits (trailing_zeroes/2), matching the pinned haswell bitmask. Remainders
// reuse the scalar oracle. Requires AVX2 via the archsimd guard.
//
// Go 1.26.5 archsimd API provenance:
// src/simd/archsimd/slice_gen_amd64.go:149-162;
// src/simd/archsimd/other_gen_amd64.go:133-149;
// src/simd/archsimd/ops_amd64.go:2130-2145,8099-8100,8673-8674;
// src/simd/archsimd/types_amd64.go:671; src/simd/archsimd/other_gen_amd64.go:297-300;
// and src/simd/archsimd/extra_amd64.go:9-17.

func findArchsimd(input []byte, value byte) int {
	n := len(input)
	if n == 0 {
		return 0
	}

	start := 0
	misalignment := int(uintptr(unsafe.Pointer(&input[0])) % 64)
	if misalignment != 0 {
		adjustment := 64 - misalignment
		if adjustment > n {
			adjustment = n
		}
		for i := 0; i < adjustment; i++ {
			if input[i] == value {
				return i
			}
		}
		start = adjustment
	}

	if start+64 <= n {
		needle := archsimd.BroadcastUint8x32(value)
		for start+64 <= n {
			v0 := archsimd.LoadUint8x32Slice(input[start:])
			v1 := archsimd.LoadUint8x32Slice(input[start+32:])
			matches := uint64(v0.Equal(needle).ToBits()) | uint64(v1.Equal(needle).ToBits())<<32
			if matches != 0 {
				archsimd.ClearAVXUpperBits()
				return start + bits.TrailingZeros64(matches)
			}
			start += 64
		}
		archsimd.ClearAVXUpperBits()
	}

	return start + findScalar(input[start:], value)
}

func findUTF16Archsimd(input []uint16, value uint16) int {
	n := len(input)
	if n == 0 {
		return 0
	}

	start := 0
	misalignment := int(uintptr(unsafe.Pointer(&input[0])) % 64)
	if misalignment != 0 && misalignment%2 == 0 {
		adjustment := (64 - misalignment) / 2
		if adjustment > n {
			adjustment = n
		}
		for i := 0; i < adjustment; i++ {
			if input[i] == value {
				return i
			}
		}
		start = adjustment
	}

	if start+32 <= n {
		needle := archsimd.BroadcastUint16x16(value)
		for start+32 <= n {
			v0 := archsimd.LoadUint16x16Slice(input[start:])
			v1 := archsimd.LoadUint16x16Slice(input[start+16:])
			// Mask16x16.ToBits is AVX-512-only; reinterpret as bytes for AVX2 VPMOVMSKB.
			m0 := v0.Equal(needle).ToInt16x16().AsInt8x32().ToMask().ToBits()
			m1 := v1.Equal(needle).ToInt16x16().AsInt8x32().ToMask().ToBits()
			matches := uint64(m0) | uint64(m1)<<32
			if matches != 0 {
				archsimd.ClearAVXUpperBits()
				return start + bits.TrailingZeros64(matches)/2
			}
			start += 32
		}
		archsimd.ClearAVXUpperBits()
	}

	return start + findUTF16Scalar(input[start:], value)
}

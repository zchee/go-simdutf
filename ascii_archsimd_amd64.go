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

// Independently adapted from simdutf/simdutf at
// dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// src/generic/ascii_validation.h:6-45 and
// include/simdutf/scalar/ascii.h:15-81 and
// src/generic/validate_utf16.h:128-158 and
// src/haswell/implementation.cpp:278-307. Byte input uses AVX2 for complete
// vectors and the pinned scalar oracle for the remainder. The UTF-16 adaptation
// uses 16-lane AVX2 vectors over raw []uint16 storage: mask 0xff80 for
// little-endian input and 0x80ff for big-endian input, with a zero-filled
// partial tail.
//
// The Go 1.26.5 archsimd API usage is pinned to:
// src/simd/archsimd/slice_gen_amd64.go:149-162,1059-1092;
// src/simd/archsimd/types_amd64.go:549-580,657-687;
// src/simd/archsimd/compare_gen_amd64.go:131-136; and
// src/simd/archsimd/ops_amd64.go:614-617,2142-2145,8099-8100,
// 8342-8343,8673-8674; and
// src/simd/archsimd/other_gen_amd64.go:297-300. Mask16x16.ToBits is
// AVX-512-only in this toolchain, so the equality mask is reinterpreted as 32
// signed bytes and extracted with the AVX2 Mask8x32.ToBits equivalent.
// This is an independent Go adaptation, not a mechanical translation.

func validateASCIIArchsimd(input []byte) bool {
	var zero archsimd.Int8x32
	offset := 0
	for ; offset+32 <= len(input); offset += 32 {
		value := archsimd.LoadUint8x32Slice(input[offset:])
		if value.AsInt8x32().Less(zero).ToBits() != 0 {
			return false
		}
	}
	return validateASCIIScalar(input[offset:])
}

func validateASCIIWithErrorsArchsimd(input []byte) Result {
	var zero archsimd.Int8x32
	offset := 0
	for ; offset+32 <= len(input); offset += 32 {
		value := archsimd.LoadUint8x32Slice(input[offset:])
		mask := value.AsInt8x32().Less(zero).ToBits()
		if mask != 0 {
			return Result{Error: TooLarge, Count: offset + bits.TrailingZeros32(mask)}
		}
	}
	result := validateASCIIWithErrorsScalar(input[offset:])
	result.Count += offset
	return result
}

func validateUTF16LEAsASCIIArchsimd(input []uint16) bool {
	return validateUTF16AsASCIIArchsimd(input, 0xff80)
}

func validateUTF16BEAsASCIIArchsimd(input []uint16) bool {
	return validateUTF16AsASCIIArchsimd(input, 0x80ff)
}

func validateUTF16AsASCIIArchsimd(input []uint16, invalidBits uint16) bool {
	invalid := archsimd.BroadcastUint16x16(invalidBits)
	var zero archsimd.Uint16x16
	offset := 0
	for ; offset+16 <= len(input); offset += 16 {
		value := archsimd.LoadUint16x16Slice(input[offset:])
		valid := value.And(invalid).Equal(zero)
		if valid.ToInt16x16().AsInt8x32().ToMask().ToBits() != 0xffffffff {
			return false
		}
	}
	if offset != len(input) {
		value := archsimd.LoadUint16x16SlicePart(input[offset:])
		valid := value.And(invalid).Equal(zero)
		if valid.ToInt16x16().AsInt8x32().ToMask().ToBits() != 0xffffffff {
			return false
		}
	}
	return true
}

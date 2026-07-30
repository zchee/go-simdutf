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
	"simd/archsimd"
)

// Independently adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/haswell/avx2_convert_utf16_to_utf32.cpp
// and src/westmere/sse_convert_utf16_to_utf32.cpp. Complete 16-code-unit blocks use
// Uint16x16 Load, BE byte-swap via ShiftAllLeft(8).Or(ShiftAllRight(8)), surrogate
// reject with And(0xf800).Equal(0xd800), AVX2 VPMOVZXWD widen via GetLo/GetHi
// ExtendToUint32, and ClearAVXUpperBits. Incomplete tails and any surrogate-bearing
// block re-enter the scalar oracle. Direct callers must satisfy the archsimd AVX2
// guard (no Uint16x16.ExtendToUint32 / AVX512 widen).

func convertUTF16LEToUTF32Archsimd(input []uint16, dst []uint32) int {
	return convertUTF16ToUTF32Archsimd(input, dst, false)
}

func convertUTF16BEToUTF32Archsimd(input []uint16, dst []uint32) int {
	return convertUTF16ToUTF32Archsimd(input, dst, true)
}

func convertUTF16ToUTF32Archsimd(input []uint16, dst []uint32, bigEndian bool) int {
	need := utf32LengthFromUTF16ArchsimdNeed(input, bigEndian)
	if len(dst) < need {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertUTF16BEToUTF32Scalar(input, dst)
		}
		return convertUTF16LEToUTF32Scalar(input, dst)
	}

	f800 := archsimd.BroadcastUint16x16(0xf800)
	d800 := archsimd.BroadcastUint16x16(0xd800)
	i := 0
	for ; i+16 <= len(input); i += 16 {
		v := archsimd.LoadUint16x16Slice(input[i:])
		if bigEndian {
			v = v.ShiftAllLeft(8).Or(v.ShiftAllRight(8))
		}
		surrogate := v.And(f800).Equal(d800)
		if surrogate.ToInt16x16().AsInt8x32().ToMask().ToBits() != 0 {
			break
		}
		// AVX2 VPMOVZXWD: widen each half separately (Uint16x16.ExtendToUint32 is AVX512).
		v.GetLo().ExtendToUint32().StoreSlice(dst[i:])
		v.GetHi().ExtendToUint32().StoreSlice(dst[i+8:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return i
	}
	var n int
	if bigEndian {
		n = convertUTF16BEToUTF32Scalar(input[i:], dst[i:])
	} else {
		n = convertUTF16LEToUTF32Scalar(input[i:], dst[i:])
	}
	if n == 0 {
		return 0
	}
	return i + n
}

func convertUTF16LEToUTF32WithErrorsArchsimd(input []uint16, dst []uint32) Result {
	return convertUTF16ToUTF32WithErrorsArchsimd(input, dst, false)
}

func convertUTF16BEToUTF32WithErrorsArchsimd(input []uint16, dst []uint32) Result {
	return convertUTF16ToUTF32WithErrorsArchsimd(input, dst, true)
}

func convertUTF16ToUTF32WithErrorsArchsimd(input []uint16, dst []uint32, bigEndian bool) Result {
	need := utf32LengthFromUTF16ArchsimdNeed(input, bigEndian)
	if len(dst) < need {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertUTF16BEToUTF32WithErrorsScalar(input, dst)
		}
		return convertUTF16LEToUTF32WithErrorsScalar(input, dst)
	}

	f800 := archsimd.BroadcastUint16x16(0xf800)
	d800 := archsimd.BroadcastUint16x16(0xd800)
	i := 0
	for ; i+16 <= len(input); i += 16 {
		v := archsimd.LoadUint16x16Slice(input[i:])
		if bigEndian {
			v = v.ShiftAllLeft(8).Or(v.ShiftAllRight(8))
		}
		surrogate := v.And(f800).Equal(d800)
		if surrogate.ToInt16x16().AsInt8x32().ToMask().ToBits() != 0 {
			break
		}
		v.GetLo().ExtendToUint32().StoreSlice(dst[i:])
		v.GetHi().ExtendToUint32().StoreSlice(dst[i+8:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return Result{Error: Success, Count: i}
	}
	var res Result
	if bigEndian {
		res = convertUTF16BEToUTF32WithErrorsScalar(input[i:], dst[i:])
	} else {
		res = convertUTF16LEToUTF32WithErrorsScalar(input[i:], dst[i:])
	}
	res.Count += i
	return res
}

func convertValidUTF16LEToUTF32Archsimd(input []uint16, dst []uint32) int {
	return convertValidUTF16ToUTF32Archsimd(input, dst, false)
}

func convertValidUTF16BEToUTF32Archsimd(input []uint16, dst []uint32) int {
	return convertValidUTF16ToUTF32Archsimd(input, dst, true)
}

func convertValidUTF16ToUTF32Archsimd(input []uint16, dst []uint32, bigEndian bool) int {
	need := utf32LengthFromUTF16ArchsimdNeed(input, bigEndian)
	if len(dst) < need {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertValidUTF16BEToUTF32Scalar(input, dst)
		}
		return convertValidUTF16LEToUTF32Scalar(input, dst)
	}

	f800 := archsimd.BroadcastUint16x16(0xf800)
	d800 := archsimd.BroadcastUint16x16(0xd800)
	i := 0
	for ; i+16 <= len(input); i += 16 {
		v := archsimd.LoadUint16x16Slice(input[i:])
		if bigEndian {
			v = v.ShiftAllLeft(8).Or(v.ShiftAllRight(8))
		}
		// Valid UTF-16 may contain paired surrogates; break to the scalar
		// combiner when any surrogate code unit appears in the block.
		surrogate := v.And(f800).Equal(d800)
		if surrogate.ToInt16x16().AsInt8x32().ToMask().ToBits() != 0 {
			break
		}
		v.GetLo().ExtendToUint32().StoreSlice(dst[i:])
		v.GetHi().ExtendToUint32().StoreSlice(dst[i+8:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return i
	}
	var rest int
	if bigEndian {
		rest = convertValidUTF16BEToUTF32Scalar(input[i:], dst[i:])
	} else {
		rest = convertValidUTF16LEToUTF32Scalar(input[i:], dst[i:])
	}
	if rest == 0 {
		return 0
	}
	return i + rest
}

func utf32LengthFromUTF16ArchsimdNeed(input []uint16, bigEndian bool) int {
	if bigEndian {
		return utf32LengthFromUTF16BEScalar(input)
	}
	return utf32LengthFromUTF16LEScalar(input)
}

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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/haswell/avx2_convert_utf16_to_latin1.cpp
// and src/westmere/sse_convert_utf16_to_latin1.cpp. Complete 16-code-unit blocks use
// Uint16x16 Load, BE byte-swap via ShiftAllLeft(8).Or(ShiftAllRight(8)), high-byte
// reject with And(0xff00).IsZero, AVX2 VPSHUFB low-byte pack via PermuteOrZero, and
// ClearAVXUpperBits. Incomplete tails re-enter the scalar oracle. Direct callers must
// satisfy the archsimd AVX2 guard (no VPMOVWB/AVX512 truncate).

// utf16Latin1PackIdx selects the low byte of each uint16 in a 128-bit lane
// (VPSHUFB / PermuteOrZero). Upper 8 outputs are zeroed.
var utf16Latin1PackIdx = [16]int8{
	0, 2, 4, 6, 8, 10, 12, 14,
	-1, -1, -1, -1, -1, -1, -1, -1,
}

func packUTF16Latin1Archsimd(v archsimd.Uint16x16) archsimd.Uint8x16 {
	idx := archsimd.LoadInt8x16(&utf16Latin1PackIdx)
	lo := v.GetLo().AsUint8x16().PermuteOrZero(idx)
	hi := v.GetHi().AsUint8x16().PermuteOrZero(idx)
	// ConcatShiftBytesRight is left-biased: hi.ConcatShiftBytesRight(8, zeros)
	// yields [0..0 | hi[0:8]], which ORs into positions 8..15 beside lo.
	hiPlaced := hi.ConcatShiftBytesRight(8, archsimd.BroadcastUint8x16(0))
	return lo.Or(hiPlaced)
}

func convertUTF16LEToLatin1Archsimd(input []uint16, dst []byte) int {
	return convertUTF16ToLatin1Archsimd(input, dst, false)
}

func convertUTF16BEToLatin1Archsimd(input []uint16, dst []byte) int {
	return convertUTF16ToLatin1Archsimd(input, dst, true)
}

func convertUTF16ToLatin1Archsimd(input []uint16, dst []byte, bigEndian bool) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertUTF16BEToLatin1Scalar(input, dst)
		}
		return convertUTF16LEToLatin1Scalar(input, dst)
	}

	high := archsimd.BroadcastUint16x16(0xff00)
	i := 0
	for ; i+16 <= len(input); i += 16 {
		v := archsimd.LoadUint16x16Slice(input[i:])
		if bigEndian {
			v = v.ShiftAllLeft(8).Or(v.ShiftAllRight(8))
		}
		if !v.And(high).IsZero() {
			break
		}
		packUTF16Latin1Archsimd(v).StoreSlice(dst[i:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return i
	}
	var n int
	if bigEndian {
		n = convertUTF16BEToLatin1Scalar(input[i:], dst[i:])
	} else {
		n = convertUTF16LEToLatin1Scalar(input[i:], dst[i:])
	}
	if n == 0 {
		return 0
	}
	return i + n
}

func convertUTF16LEToLatin1WithErrorsArchsimd(input []uint16, dst []byte) Result {
	return convertUTF16ToLatin1WithErrorsArchsimd(input, dst, false)
}

func convertUTF16BEToLatin1WithErrorsArchsimd(input []uint16, dst []byte) Result {
	return convertUTF16ToLatin1WithErrorsArchsimd(input, dst, true)
}

func convertUTF16ToLatin1WithErrorsArchsimd(input []uint16, dst []byte, bigEndian bool) Result {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertUTF16BEToLatin1WithErrorsScalar(input, dst)
		}
		return convertUTF16LEToLatin1WithErrorsScalar(input, dst)
	}

	high := archsimd.BroadcastUint16x16(0xff00)
	i := 0
	for ; i+16 <= len(input); i += 16 {
		v := archsimd.LoadUint16x16Slice(input[i:])
		if bigEndian {
			v = v.ShiftAllLeft(8).Or(v.ShiftAllRight(8))
		}
		if !v.And(high).IsZero() {
			break
		}
		packUTF16Latin1Archsimd(v).StoreSlice(dst[i:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return Result{Error: Success, Count: i}
	}
	var res Result
	if bigEndian {
		res = convertUTF16BEToLatin1WithErrorsScalar(input[i:], dst[i:])
	} else {
		res = convertUTF16LEToLatin1WithErrorsScalar(input[i:], dst[i:])
	}
	res.Count += i
	return res
}

func convertValidUTF16LEToLatin1Archsimd(input []uint16, dst []byte) int {
	return convertValidUTF16ToLatin1Archsimd(input, dst, false)
}

func convertValidUTF16BEToLatin1Archsimd(input []uint16, dst []byte) int {
	return convertValidUTF16ToLatin1Archsimd(input, dst, true)
}

func convertValidUTF16ToLatin1Archsimd(input []uint16, dst []byte, bigEndian bool) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertValidUTF16BEToLatin1Scalar(input, dst)
		}
		return convertValidUTF16LEToLatin1Scalar(input, dst)
	}

	i := 0
	for ; i+16 <= len(input); i += 16 {
		v := archsimd.LoadUint16x16Slice(input[i:])
		if bigEndian {
			v = v.ShiftAllLeft(8).Or(v.ShiftAllRight(8))
		}
		packUTF16Latin1Archsimd(v).StoreSlice(dst[i:])
	}
	archsimd.ClearAVXUpperBits()
	if bigEndian {
		return i + convertValidUTF16BEToLatin1Scalar(input[i:], dst[i:])
	}
	return i + convertValidUTF16LEToLatin1Scalar(input[i:], dst[i:])
}

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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/haswell/avx2_convert_utf16_to_utf8.cpp
// and src/westmere/sse_convert_utf16_to_utf8.cpp. Complete 16-code-unit ASCII blocks use
// Uint16x16 Load, BE byte-swap via ShiftAllLeft(8).Or(ShiftAllRight(8)), ASCII accept
// with And(0xff80).IsZero, AVX2 low-byte pack via packUTF16Latin1Archsimd / PermuteOrZero,
// and ClearAVXUpperBits. Non-ASCII / surrogate-bearing blocks and incomplete tails
// remount the scalar oracle (no AVX512 widen/pack, no incomplete variable-length UTF-8
// shuffle tables). Empty remaining input after a full SIMD success returns the written
// length directly so Valid/convert do not treat a zero-length scalar tail as failure.
// Direct callers must satisfy the archsimd AVX2 guard.

func convertUTF16LEToUTF8Archsimd(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8Archsimd(input, dst, false)
}

func convertUTF16BEToUTF8Archsimd(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8Archsimd(input, dst, true)
}

func convertUTF16ToUTF8Archsimd(input []uint16, dst []byte, bigEndian bool) int {
	need := utf8LengthFromUTF16ArchsimdNeed(input, bigEndian)
	if len(dst) < need {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertUTF16BEToUTF8Scalar(input, dst)
		}
		return convertUTF16LEToUTF8Scalar(input, dst)
	}

	i := convertUTF16ToUTF8ASCIIBlocksArchsimd(input, dst, bigEndian)
	if i == len(input) {
		return i
	}
	var n int
	if bigEndian {
		n = convertUTF16BEToUTF8Scalar(input[i:], dst[i:])
	} else {
		n = convertUTF16LEToUTF8Scalar(input[i:], dst[i:])
	}
	if n == 0 {
		return 0
	}
	return i + n
}

func convertUTF16LEToUTF8WithErrorsArchsimd(input []uint16, dst []byte) Result {
	return convertUTF16ToUTF8WithErrorsArchsimd(input, dst, false)
}

func convertUTF16BEToUTF8WithErrorsArchsimd(input []uint16, dst []byte) Result {
	return convertUTF16ToUTF8WithErrorsArchsimd(input, dst, true)
}

func convertUTF16ToUTF8WithErrorsArchsimd(input []uint16, dst []byte, bigEndian bool) Result {
	need := utf8LengthFromUTF16ArchsimdNeed(input, bigEndian)
	if len(dst) < need {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertUTF16BEToUTF8WithErrorsScalar(input, dst)
		}
		return convertUTF16LEToUTF8WithErrorsScalar(input, dst)
	}

	i := convertUTF16ToUTF8ASCIIBlocksArchsimd(input, dst, bigEndian)
	if i == len(input) {
		return Result{Error: Success, Count: i}
	}
	var res Result
	if bigEndian {
		res = convertUTF16BEToUTF8WithErrorsScalar(input[i:], dst[i:])
	} else {
		res = convertUTF16LEToUTF8WithErrorsScalar(input[i:], dst[i:])
	}
	res.Count += i
	return res
}

func convertUTF16LEToUTF8WithReplacementArchsimd(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8WithReplacementArchsimd(input, dst, false)
}

func convertUTF16BEToUTF8WithReplacementArchsimd(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8WithReplacementArchsimd(input, dst, true)
}

func convertUTF16ToUTF8WithReplacementArchsimd(input []uint16, dst []byte, bigEndian bool) int {
	need := utf8LengthFromUTF16WithReplacementArchsimdNeed(input, bigEndian)
	if len(dst) < need {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertUTF16BEToUTF8WithReplacementScalar(input, dst)
		}
		return convertUTF16LEToUTF8WithReplacementScalar(input, dst)
	}

	i := convertUTF16ToUTF8ASCIIBlocksArchsimd(input, dst, bigEndian)
	if i == len(input) {
		return i
	}
	if bigEndian {
		return i + convertUTF16BEToUTF8WithReplacementScalar(input[i:], dst[i:])
	}
	return i + convertUTF16LEToUTF8WithReplacementScalar(input[i:], dst[i:])
}

func convertValidUTF16LEToUTF8Archsimd(input []uint16, dst []byte) int {
	return convertValidUTF16ToUTF8Archsimd(input, dst, false)
}

func convertValidUTF16BEToUTF8Archsimd(input []uint16, dst []byte) int {
	return convertValidUTF16ToUTF8Archsimd(input, dst, true)
}

func convertValidUTF16ToUTF8Archsimd(input []uint16, dst []byte, bigEndian bool) int {
	need := utf8LengthFromUTF16ArchsimdNeed(input, bigEndian)
	if len(dst) < need {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		if bigEndian {
			return convertValidUTF16BEToUTF8Scalar(input, dst)
		}
		return convertValidUTF16LEToUTF8Scalar(input, dst)
	}

	i := convertUTF16ToUTF8ASCIIBlocksArchsimd(input, dst, bigEndian)
	if i == len(input) {
		return i
	}
	var rest int
	if bigEndian {
		rest = convertValidUTF16BEToUTF8Scalar(input[i:], dst[i:])
	} else {
		rest = convertValidUTF16LEToUTF8Scalar(input[i:], dst[i:])
	}
	if rest == 0 {
		return 0
	}
	return i + rest
}

func utf8LengthFromUTF16LEArchsimd(input []uint16) int {
	return utf8LengthFromUTF16LEScalar(input)
}

func utf8LengthFromUTF16BEArchsimd(input []uint16) int {
	return utf8LengthFromUTF16BEScalar(input)
}

func utf8LengthFromUTF16LEWithReplacementArchsimd(input []uint16) Result {
	return utf8LengthFromUTF16LEWithReplacementScalar(input)
}

func utf8LengthFromUTF16BEWithReplacementArchsimd(input []uint16) Result {
	return utf8LengthFromUTF16BEWithReplacementScalar(input)
}

// convertUTF16ToUTF8ASCIIBlocksArchsimd packs complete ASCII 16-unit blocks.
// Returns the number of input units (and output bytes) consumed. Non-ASCII
// blocks stop the loop so callers can remount scalar from that offset.
func convertUTF16ToUTF8ASCIIBlocksArchsimd(input []uint16, dst []byte, bigEndian bool) int {
	asciiMask := archsimd.BroadcastUint16x16(0xff80)
	i := 0
	for ; i+16 <= len(input); i += 16 {
		v := archsimd.LoadUint16x16Slice(input[i:])
		if bigEndian {
			v = v.ShiftAllLeft(8).Or(v.ShiftAllRight(8))
		}
		if !v.And(asciiMask).IsZero() {
			break
		}
		packUTF16Latin1Archsimd(v).StoreSlice(dst[i:])
	}
	archsimd.ClearAVXUpperBits()
	return i
}

func utf8LengthFromUTF16ArchsimdNeed(input []uint16, bigEndian bool) int {
	if bigEndian {
		return utf8LengthFromUTF16BEScalar(input)
	}
	return utf8LengthFromUTF16LEScalar(input)
}

func utf8LengthFromUTF16WithReplacementArchsimdNeed(input []uint16, bigEndian bool) int {
	if bigEndian {
		return utf8LengthFromUTF16WithReplacementScalar(input, !nativeLittleEndian())
	}
	return utf8LengthFromUTF16WithReplacementScalar(input, nativeLittleEndian())
}

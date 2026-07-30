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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/haswell/avx2_convert_utf32_to_*.cpp
// and src/westmere/sse_convert_utf32_to_*.cpp. Complete 8-code-unit blocks use
// Uint32x8 Load, Latin-1/ASCII accept with And(0xffffff00)/And(0xffffff80).IsZero,
// BMP accept with And(0xffff0000).IsZero, AVX2 VPSHUFB narrow/pack via
// PermuteOrZero + ConcatShiftBytesRight, and ClearAVXUpperBits. Non-ASCII /
// non-BMP / surrogate-bearing blocks and incomplete tails remount the scalar
// oracle (no AVX512 TruncateToUint8 / TruncateToUint16). Direct callers must
// satisfy the archsimd AVX2 guard.

// utf32Latin1PackIdx selects the low byte of each uint32 in a 128-bit lane
// (VPSHUFB / PermuteOrZero). Upper 12 outputs are zeroed.
var utf32Latin1PackIdx = [16]int8{
	0, 4, 8, 12,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1,
}

// utf32UTF16LEPackIdx selects the low 16 bits of each uint32 as little-endian
// bytes in a 128-bit lane. Upper 8 outputs are zeroed.
var utf32UTF16LEPackIdx = [16]int8{
	0, 1, 4, 5, 8, 9, 12, 13,
	-1, -1, -1, -1, -1, -1, -1, -1,
}

// utf32UTF16BEPackIdx selects the low 16 bits of each uint32 as big-endian
// bytes in a 128-bit lane. Upper 8 outputs are zeroed.
var utf32UTF16BEPackIdx = [16]int8{
	1, 0, 5, 4, 9, 8, 13, 12,
	-1, -1, -1, -1, -1, -1, -1, -1,
}

func packUTF32Latin1Archsimd(v archsimd.Uint32x8) archsimd.Uint8x16 {
	idx := archsimd.LoadInt8x16(&utf32Latin1PackIdx)
	lo := v.GetLo().AsUint8x16().PermuteOrZero(idx)
	hi := v.GetHi().AsUint8x16().PermuteOrZero(idx)
	// ConcatShiftBytesRight is left-biased: hi.ConcatShiftBytesRight(12, zeros)
	// yields [0×4 | hi[0:4] | 0×8], which ORs into positions 4..7 beside lo.
	hiPlaced := hi.ConcatShiftBytesRight(12, archsimd.BroadcastUint8x16(0))
	return lo.Or(hiPlaced)
}

func packUTF32UTF16Archsimd(v archsimd.Uint32x8, bigEndian bool) archsimd.Uint16x8 {
	idx := archsimd.LoadInt8x16(&utf32UTF16LEPackIdx)
	if bigEndian {
		idx = archsimd.LoadInt8x16(&utf32UTF16BEPackIdx)
	}
	lo := v.GetLo().AsUint8x16().PermuteOrZero(idx)
	hi := v.GetHi().AsUint8x16().PermuteOrZero(idx)
	hiPlaced := hi.ConcatShiftBytesRight(8, archsimd.BroadcastUint8x16(0))
	return lo.Or(hiPlaced).AsUint16x8()
}

func convertUTF32ToLatin1Archsimd(input []uint32, dst []byte) int {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToLatin1Scalar(input, dst)
	}

	high := archsimd.BroadcastUint32x8(0xffffff00)
	i := 0
	for ; i+8 <= len(input); i += 8 {
		v := archsimd.LoadUint32x8Slice(input[i:])
		if !v.And(high).IsZero() {
			break
		}
		packUTF32Latin1Archsimd(v).StoreSlicePart(dst[i : i+8])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return i
	}
	n := convertUTF32ToLatin1Scalar(input[i:], dst[i:])
	if n == 0 {
		return 0
	}
	return i + n
}

func convertUTF32ToLatin1WithErrorsArchsimd(input []uint32, dst []byte) Result {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToLatin1WithErrorsScalar(input, dst)
	}

	high := archsimd.BroadcastUint32x8(0xffffff00)
	i := 0
	for ; i+8 <= len(input); i += 8 {
		v := archsimd.LoadUint32x8Slice(input[i:])
		if !v.And(high).IsZero() {
			break
		}
		packUTF32Latin1Archsimd(v).StoreSlicePart(dst[i : i+8])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return Result{Error: Success, Count: i}
	}
	res := convertUTF32ToLatin1WithErrorsScalar(input[i:], dst[i:])
	res.Count += i
	return res
}

func convertValidUTF32ToLatin1Archsimd(input []uint32, dst []byte) int {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF32ToLatin1Scalar(input, dst)
	}

	i := 0
	for ; i+8 <= len(input); i += 8 {
		v := archsimd.LoadUint32x8Slice(input[i:])
		packUTF32Latin1Archsimd(v).StoreSlicePart(dst[i : i+8])
	}
	archsimd.ClearAVXUpperBits()
	return i + convertValidUTF32ToLatin1Scalar(input[i:], dst[i:])
}

func convertUTF32ToUTF8Archsimd(input []uint32, dst []byte) int {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToUTF8Scalar(input, dst)
	}

	i := convertUTF32ToUTF8ASCIIBlocksArchsimd(input, dst)
	if i == len(input) {
		return i
	}
	n := convertUTF32ToUTF8Scalar(input[i:], dst[i:])
	if n == 0 {
		return 0
	}
	return i + n
}

func convertUTF32ToUTF8WithErrorsArchsimd(input []uint32, dst []byte) Result {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToUTF8WithErrorsScalar(input, dst)
	}

	i := convertUTF32ToUTF8ASCIIBlocksArchsimd(input, dst)
	if i == len(input) {
		return Result{Error: Success, Count: i}
	}
	res := convertUTF32ToUTF8WithErrorsScalar(input[i:], dst[i:])
	res.Count += i
	return res
}

func convertValidUTF32ToUTF8Archsimd(input []uint32, dst []byte) int {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF32ToUTF8Scalar(input, dst)
	}

	i := convertUTF32ToUTF8ASCIIBlocksArchsimd(input, dst)
	if i == len(input) {
		return i
	}
	return i + convertValidUTF32ToUTF8Scalar(input[i:], dst[i:])
}

func convertUTF32ToUTF16LEArchsimd(input []uint32, dst []uint16) int {
	return convertUTF32ToUTF16Archsimd(input, dst, false)
}

func convertUTF32ToUTF16BEArchsimd(input []uint32, dst []uint16) int {
	return convertUTF32ToUTF16Archsimd(input, dst, true)
}

func convertUTF32ToUTF16Archsimd(input []uint32, dst []uint16, bigEndian bool) int {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		if bigEndian {
			return convertUTF32ToUTF16BEScalar(input, dst)
		}
		return convertUTF32ToUTF16LEScalar(input, dst)
	}

	i, out := convertUTF32ToUTF16BMPBlocksArchsimd(input, dst, bigEndian, true)
	if i == len(input) {
		return out
	}
	var n int
	if bigEndian {
		n = convertUTF32ToUTF16BEScalar(input[i:], dst[out:])
	} else {
		n = convertUTF32ToUTF16LEScalar(input[i:], dst[out:])
	}
	if n == 0 {
		return 0
	}
	return out + n
}

func convertUTF32ToUTF16LEWithErrorsArchsimd(input []uint32, dst []uint16) Result {
	return convertUTF32ToUTF16WithErrorsArchsimd(input, dst, false)
}

func convertUTF32ToUTF16BEWithErrorsArchsimd(input []uint32, dst []uint16) Result {
	return convertUTF32ToUTF16WithErrorsArchsimd(input, dst, true)
}

func convertUTF32ToUTF16WithErrorsArchsimd(input []uint32, dst []uint16, bigEndian bool) Result {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		if bigEndian {
			return convertUTF32ToUTF16BEWithErrorsScalar(input, dst)
		}
		return convertUTF32ToUTF16LEWithErrorsScalar(input, dst)
	}

	i, out := convertUTF32ToUTF16BMPBlocksArchsimd(input, dst, bigEndian, true)
	if i == len(input) {
		return Result{Error: Success, Count: out}
	}
	var res Result
	if bigEndian {
		res = convertUTF32ToUTF16BEWithErrorsScalar(input[i:], dst[out:])
	} else {
		res = convertUTF32ToUTF16LEWithErrorsScalar(input[i:], dst[out:])
	}
	if res.Error == Success {
		res.Count += out
	} else {
		res.Count += i
	}
	return res
}

func convertValidUTF32ToUTF16LEArchsimd(input []uint32, dst []uint16) int {
	return convertValidUTF32ToUTF16Archsimd(input, dst, false)
}

func convertValidUTF32ToUTF16BEArchsimd(input []uint32, dst []uint16) int {
	return convertValidUTF32ToUTF16Archsimd(input, dst, true)
}

func convertValidUTF32ToUTF16Archsimd(input []uint32, dst []uint16, bigEndian bool) int {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		if bigEndian {
			return convertValidUTF32ToUTF16BEScalar(input, dst)
		}
		return convertValidUTF32ToUTF16LEScalar(input, dst)
	}

	i, out := convertUTF32ToUTF16BMPBlocksArchsimd(input, dst, bigEndian, false)
	if i == len(input) {
		return out
	}
	if bigEndian {
		return out + convertValidUTF32ToUTF16BEScalar(input[i:], dst[out:])
	}
	return out + convertValidUTF32ToUTF16LEScalar(input[i:], dst[out:])
}

func utf8LengthFromUTF32Archsimd(input []uint32) int {
	return utf8LengthFromUTF32Scalar(input)
}

func utf16LengthFromUTF32Archsimd(input []uint32) int {
	return utf16LengthFromUTF32Scalar(input)
}

// convertUTF32ToUTF8ASCIIBlocksArchsimd packs complete ASCII 8-unit blocks.
// Returns the number of input units (and output bytes) consumed. Non-ASCII
// blocks stop the loop so callers can remount scalar from that offset.
func convertUTF32ToUTF8ASCIIBlocksArchsimd(input []uint32, dst []byte) int {
	asciiMask := archsimd.BroadcastUint32x8(0xffffff80)
	i := 0
	for ; i+8 <= len(input); i += 8 {
		v := archsimd.LoadUint32x8Slice(input[i:])
		if !v.And(asciiMask).IsZero() {
			break
		}
		packUTF32Latin1Archsimd(v).StoreSlicePart(dst[i : i+8])
	}
	archsimd.ClearAVXUpperBits()
	return i
}

// convertUTF32ToUTF16BMPBlocksArchsimd packs complete BMP 8-unit blocks.
// When rejectSurrogates is set, surrogate-bearing blocks stop the loop.
// Returns (inputOffset, outputUnits).
func convertUTF32ToUTF16BMPBlocksArchsimd(input []uint32, dst []uint16, bigEndian, rejectSurrogates bool) (int, int) {
	high := archsimd.BroadcastUint32x8(0xffff0000)
	i, out := 0, 0
	for ; i+8 <= len(input); i += 8 {
		v := archsimd.LoadUint32x8Slice(input[i:])
		if !v.And(high).IsZero() {
			break
		}
		if rejectSurrogates {
			surrogate := v.GreaterEqual(archsimd.BroadcastUint32x8(0xd800)).And(
				v.LessEqual(archsimd.BroadcastUint32x8(0xdfff)),
			)
			if surrogate.ToBits() != 0 {
				break
			}
		}
		packUTF32UTF16Archsimd(v, bigEndian).StoreSlice(dst[out:])
		out += 8
	}
	archsimd.ClearAVXUpperBits()
	return i, out
}

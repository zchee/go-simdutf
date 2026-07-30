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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf8_to_*.cpp
// and src/haswell/avx2_convert_utf8_to_*.cpp. Complete ASCII 32-byte blocks use
// archsimd Load/Store/Greater; mixed tails re-enter the scalar oracle.

func convertUTF8ToLatin1Archsimd(input, dst []byte) int {
	return convertUTF8ToLatin1ArchsimdASCII(input, dst, convertUTF8ToLatin1Scalar)
}
func convertValidUTF8ToLatin1Archsimd(input, dst []byte) int {
	return convertUTF8ToLatin1ArchsimdASCII(input, dst, convertValidUTF8ToLatin1Scalar)
}
func convertUTF8ToLatin1ArchsimdASCII(input, dst []byte, tail func([]byte, []byte) int) int {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 32 {
		return tail(input, dst)
	}
	threshold := archsimd.BroadcastUint8x32(0x7f)
	i := 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
		if source.Greater(threshold).ToBits() != 0 {
			break
		}
		source.StoreSlice(dst[i:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return i
	}
	rest := tail(input[i:], dst[i:])
	if rest == 0 {
		return 0
	}
	return i + rest
}

func convertUTF8ToLatin1WithErrorsArchsimd(input, dst []byte) Result {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 32 {
		return convertUTF8ToLatin1WithErrorsScalar(input, dst)
	}
	threshold := archsimd.BroadcastUint8x32(0x7f)
	i := 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
		if source.Greater(threshold).ToBits() != 0 {
			break
		}
		source.StoreSlice(dst[i:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return Result{Error: Success, Count: i}
	}
	res := convertUTF8ToLatin1WithErrorsScalar(input[i:], dst[i:])
	if res.Error != Success {
		res.Count += i
		return res
	}
	res.Count += i
	return res
}

func convertUTF8ToUTF16LEArchsimd(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16Archsimd(input, dst, false, convertUTF8ToUTF16LEScalar)
}
func convertUTF8ToUTF16BEArchsimd(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16Archsimd(input, dst, true, convertUTF8ToUTF16BEScalar)
}
func convertValidUTF8ToUTF16LEArchsimd(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16Archsimd(input, dst, false, convertValidUTF8ToUTF16LEScalar)
}
func convertValidUTF8ToUTF16BEArchsimd(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16Archsimd(input, dst, true, convertValidUTF8ToUTF16BEScalar)
}
func convertUTF8ToUTF16Archsimd(input []byte, dst []uint16, bigEndian bool, tail func([]byte, []uint16) int) int {
	if len(dst) < utf16LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 32 {
		return tail(input, dst)
	}
	threshold := archsimd.BroadcastUint8x32(0x7f)
	i := 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
		if source.Greater(threshold).ToBits() != 0 {
			break
		}
		lo := source.GetLo().ExtendToUint16()
		hi := source.GetHi().ExtendToUint16()
		if bigEndian {
			lo = lo.ShiftAllLeft(8)
			hi = hi.ShiftAllLeft(8)
		}
		lo.StoreSlice(dst[i:])
		hi.StoreSlice(dst[i+16:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return i
	}
	rest := tail(input[i:], dst[i:])
	if rest == 0 {
		return 0
	}
	return i + rest
}

func convertUTF8ToUTF16LEWithErrorsArchsimd(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsArchsimd(input, dst, false, convertUTF8ToUTF16LEWithErrorsScalar)
}
func convertUTF8ToUTF16BEWithErrorsArchsimd(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsArchsimd(input, dst, true, convertUTF8ToUTF16BEWithErrorsScalar)
}
func convertUTF8ToUTF16WithErrorsArchsimd(input []byte, dst []uint16, bigEndian bool, tail func([]byte, []uint16) Result) Result {
	if len(dst) < utf16LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 32 {
		return tail(input, dst)
	}
	threshold := archsimd.BroadcastUint8x32(0x7f)
	i := 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
		if source.Greater(threshold).ToBits() != 0 {
			break
		}
		lo := source.GetLo().ExtendToUint16()
		hi := source.GetHi().ExtendToUint16()
		if bigEndian {
			lo = lo.ShiftAllLeft(8)
			hi = hi.ShiftAllLeft(8)
		}
		lo.StoreSlice(dst[i:])
		hi.StoreSlice(dst[i+16:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return Result{Error: Success, Count: i}
	}
	res := tail(input[i:], dst[i:])
	if res.Error != Success {
		res.Count += i
		return res
	}
	res.Count += i
	return res
}

func convertUTF8ToUTF32Archsimd(input []byte, dst []uint32) int {
	return convertUTF8ToUTF32ArchsimdASCII(input, dst, convertUTF8ToUTF32Scalar)
}
func convertValidUTF8ToUTF32Archsimd(input []byte, dst []uint32) int {
	return convertUTF8ToUTF32ArchsimdASCII(input, dst, convertValidUTF8ToUTF32Scalar)
}
func convertUTF8ToUTF32ArchsimdASCII(input []byte, dst []uint32, tail func([]byte, []uint32) int) int {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 32 {
		return tail(input, dst)
	}
	threshold := archsimd.BroadcastUint8x32(0x7f)
	i := 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
		if source.Greater(threshold).ToBits() != 0 {
			break
		}
		source.GetLo().ExtendLo8ToUint32().StoreSlice(dst[i:])
		archsimd.LoadUint8x16SlicePart(input[i+8:]).ExtendLo8ToUint32().StoreSlice(dst[i+8:])
		archsimd.LoadUint8x16SlicePart(input[i+16:]).ExtendLo8ToUint32().StoreSlice(dst[i+16:])
		archsimd.LoadUint8x16SlicePart(input[i+24:]).ExtendLo8ToUint32().StoreSlice(dst[i+24:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return i
	}
	rest := tail(input[i:], dst[i:])
	if rest == 0 {
		return 0
	}
	return i + rest
}

func convertUTF8ToUTF32WithErrorsArchsimd(input []byte, dst []uint32) Result {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 32 {
		return convertUTF8ToUTF32WithErrorsScalar(input, dst)
	}
	threshold := archsimd.BroadcastUint8x32(0x7f)
	i := 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
		if source.Greater(threshold).ToBits() != 0 {
			break
		}
		source.GetLo().ExtendLo8ToUint32().StoreSlice(dst[i:])
		archsimd.LoadUint8x16SlicePart(input[i+8:]).ExtendLo8ToUint32().StoreSlice(dst[i+8:])
		archsimd.LoadUint8x16SlicePart(input[i+16:]).ExtendLo8ToUint32().StoreSlice(dst[i+16:])
		archsimd.LoadUint8x16SlicePart(input[i+24:]).ExtendLo8ToUint32().StoreSlice(dst[i+24:])
	}
	archsimd.ClearAVXUpperBits()
	if i == len(input) {
		return Result{Error: Success, Count: i}
	}
	res := convertUTF8ToUTF32WithErrorsScalar(input[i:], dst[i:])
	if res.Error != Success {
		res.Count += i
		return res
	}
	res.Count += i
	return res
}

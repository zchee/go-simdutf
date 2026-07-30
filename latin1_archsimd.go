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
	"math/bits"
	"simd/archsimd"
)

// Independently adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/
// sse_convert_latin1_to_utf8.cpp, sse_convert_latin1_to_utf16.cpp, and
// sse_convert_latin1_to_utf32.cpp. The Go 1.26.5 archsimd mapping uses
// LoadUint8x32Slice, Uint8x32.Greater, Mask8x32.ToBits, Uint8x16.ExtendToUint16,
// Uint8x16.ExtendLo8ToUint32, and Uint16x16.ShiftAllLeft, all of which require
// AVX2. Direct callers must therefore satisfy the archsimd AVX2 guard.

func convertLatin1ToUTF8Archsimd(input, dst []byte) int {
	if len(input) < 32 {
		return convertLatin1ToUTF8Scalar(input, dst)
	}
	required := utf8LengthFromLatin1Archsimd(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}

	threshold := archsimd.BroadcastUint8x32(0x7f)
	i, written := 0, 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
		if source.Greater(threshold).ToBits() == 0 {
			source.StoreSlice(dst[written:])
			written += 32
			continue
		}
		for _, value := range input[i : i+32] {
			if value < 0x80 {
				dst[written] = value
				written++
				continue
			}
			dst[written], dst[written+1] = 0xc0|(value>>6), 0x80|(value&0x3f)
			written += 2
		}
	}
	archsimd.ClearAVXUpperBits()
	return written + convertLatin1ToUTF8Scalar(input[i:], dst[written:])
}

func convertLatin1ToUTF16LEArchsimd(input []byte, dst []uint16) int {
	return convertLatin1ToUTF16Archsimd(input, dst, false)
}

func convertLatin1ToUTF16BEArchsimd(input []byte, dst []uint16) int {
	return convertLatin1ToUTF16Archsimd(input, dst, true)
}

func convertLatin1ToUTF16Archsimd(input []byte, dst []uint16, bigEndian bool) int {
	if len(dst) < len(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 32 {
		if bigEndian {
			return convertLatin1ToUTF16BEScalar(input, dst)
		}
		return convertLatin1ToUTF16LEScalar(input, dst)
	}

	i := 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
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
	if bigEndian {
		return i + convertLatin1ToUTF16BEScalar(input[i:], dst[i:])
	}
	return i + convertLatin1ToUTF16LEScalar(input[i:], dst[i:])
}

func convertLatin1ToUTF32Archsimd(input []byte, dst []uint32) int {
	if len(dst) < len(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 32 {
		return convertLatin1ToUTF32Scalar(input, dst)
	}

	i := 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
		source.GetLo().ExtendLo8ToUint32().StoreSlice(dst[i:])
		archsimd.LoadUint8x16SlicePart(input[i+8:]).ExtendLo8ToUint32().StoreSlice(dst[i+8:])
		archsimd.LoadUint8x16SlicePart(input[i+16:]).ExtendLo8ToUint32().StoreSlice(dst[i+16:])
		archsimd.LoadUint8x16SlicePart(input[i+24:]).ExtendLo8ToUint32().StoreSlice(dst[i+24:])
	}
	archsimd.ClearAVXUpperBits()
	return i + convertLatin1ToUTF32Scalar(input[i:], dst[i:])
}

func utf8LengthFromLatin1Archsimd(input []byte) int {
	if len(input) < 32 {
		return utf8LengthFromLatin1Scalar(input)
	}
	threshold := archsimd.BroadcastUint8x32(0x7f)
	i, high := 0, 0
	for ; i+32 <= len(input); i += 32 {
		source := archsimd.LoadUint8x32Slice(input[i:])
		high += bits.OnesCount32(source.Greater(threshold).ToBits())
	}
	archsimd.ClearAVXUpperBits()
	return len(input) + high + utf8LengthFromLatin1Scalar(input[i:]) - len(input[i:])
}

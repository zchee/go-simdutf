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

// Independently adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/scalar/utf16.h:20-76,137-145,183-213 and
// include/simdutf/scalar/utf32.h:8-50. AVX2 vectors skip runs that contain no
// surrogate (UTF-16) or no invalid scalar value (UTF-32); the direct scalar
// remainder preserves the oracle's first-error and repair semantics.

func validateUTF16LEArchsimd(input []uint16) bool {
	if len(input) < 16 {
		return validateUTF16LEScalar(input)
	}
	return validateUTF16Archsimd(input, nativeLittleEndian())
}

func validateUTF16BEArchsimd(input []uint16) bool {
	if len(input) < 16 {
		return validateUTF16BEScalar(input)
	}
	return validateUTF16Archsimd(input, !nativeLittleEndian())
}

func validateUTF16LEWithErrorsArchsimd(input []uint16) Result {
	if len(input) < 16 {
		return validateUTF16LEWithErrorsScalar(input)
	}
	return validateUTF16WithErrorsArchsimd(input, nativeLittleEndian())
}

func validateUTF16BEWithErrorsArchsimd(input []uint16) Result {
	if len(input) < 16 {
		return validateUTF16BEWithErrorsScalar(input)
	}
	return validateUTF16WithErrorsArchsimd(input, !nativeLittleEndian())
}

func validateUTF16Archsimd(input []uint16, storageIsNative bool) bool {
	return validateUTF16WithErrorsArchsimd(input, storageIsNative).Error == Success
}

func validateUTF16WithErrorsArchsimd(input []uint16, storageIsNative bool) Result {
	offset := 0
	for ; offset+16 <= len(input); offset += 16 {
		words := utf16ArchsimdWords(archsimd.LoadUint16x16Slice(input[offset:]), storageIsNative)
		if utf16ArchsimdHasSurrogate(words) {
			break
		}
	}
	for pos := offset; pos < len(input); {
		word := utf16ArchsimdWord(input[pos], storageIsNative)
		if word < 0xd800 || word > 0xdfff {
			pos++
			continue
		}
		if word > 0xdbff || pos+1 == len(input) {
			return Result{Error: Surrogate, Count: pos}
		}
		next := utf16ArchsimdWord(input[pos+1], storageIsNative)
		if next < 0xdc00 || next > 0xdfff {
			return Result{Error: Surrogate, Count: pos}
		}
		pos += 2
	}
	return Result{Error: Success, Count: len(input)}
}

func toWellFormedUTF16LEArchsimd(input, dst []uint16) {
	if len(input) < 16 {
		toWellFormedUTF16LEScalar(input, dst)
		return
	}
	toWellFormedUTF16Archsimd(input, dst, nativeLittleEndian())
}

func toWellFormedUTF16BEArchsimd(input, dst []uint16) {
	if len(input) < 16 {
		toWellFormedUTF16BEScalar(input, dst)
		return
	}
	toWellFormedUTF16Archsimd(input, dst, !nativeLittleEndian())
}

func toWellFormedUTF16Archsimd(input, dst []uint16, storageIsNative bool) {
	if len(dst) < len(input) {
		panic("simdutf: UTF-16 destination too short")
	}
	offset := 0
	for ; offset+16 <= len(input); offset += 16 {
		value := archsimd.LoadUint16x16Slice(input[offset:])
		if utf16ArchsimdHasSurrogate(utf16ArchsimdWords(value, storageIsNative)) {
			break
		}
		value.StoreSlice(dst[offset:])
	}
	replacement := uint16(0xfffd)
	if !storageIsNative {
		replacement = 0xfdff
	}
	for i := offset; i < len(input); i++ {
		word := utf16ArchsimdWord(input[i], storageIsNative)
		switch {
		case word >= 0xd800 && word <= 0xdbff:
			if i+1 < len(input) {
				next := utf16ArchsimdWord(input[i+1], storageIsNative)
				if next >= 0xdc00 && next <= 0xdfff {
					dst[i] = input[i]
					dst[i+1] = input[i+1]
					i++
					continue
				}
			}
			dst[i] = replacement
		case word >= 0xdc00 && word <= 0xdfff:
			dst[i] = replacement
		default:
			dst[i] = input[i]
		}
	}
}

func validateUTF32Archsimd(input []uint32) bool {
	if len(input) < 8 {
		return validateUTF32Scalar(input)
	}
	return validateUTF32WithErrorsArchsimd(input).Error == Success
}

func validateUTF32WithErrorsArchsimd(input []uint32) Result {
	if len(input) < 8 {
		return validateUTF32WithErrorsScalar(input)
	}
	offset := 0
	for ; offset+8 <= len(input); offset += 8 {
		if utf32ArchsimdHasInvalid(archsimd.LoadUint32x8Slice(input[offset:])) {
			break
		}
	}
	for pos := offset; pos < len(input); pos++ {
		word := input[pos]
		if word > 0x10ffff {
			return Result{Error: TooLarge, Count: pos}
		}
		if word >= 0xd800 && word <= 0xdfff {
			return Result{Error: Surrogate, Count: pos}
		}
	}
	return Result{Error: Success, Count: len(input)}
}

func utf16ArchsimdWords(value archsimd.Uint16x16, storageIsNative bool) archsimd.Uint16x16 {
	if storageIsNative {
		return value
	}
	return value.ShiftAllLeft(8).Or(value.ShiftAllRight(8))
}

func utf16ArchsimdHasSurrogate(value archsimd.Uint16x16) bool {
	mask := value.GreaterEqual(archsimd.BroadcastUint16x16(0xd800)).And(
		value.LessEqual(archsimd.BroadcastUint16x16(0xdfff)),
	)
	return mask.ToInt16x16().AsInt8x32().ToMask().ToBits() != 0
}

func utf16ArchsimdWord(raw uint16, storageIsNative bool) uint16 {
	if storageIsNative {
		return raw
	}
	return raw>>8 | raw<<8
}

func utf32ArchsimdHasInvalid(value archsimd.Uint32x8) bool {
	surrogate := value.GreaterEqual(archsimd.BroadcastUint32x8(0xd800)).And(
		value.LessEqual(archsimd.BroadcastUint32x8(0xdfff)),
	)
	return value.Greater(archsimd.BroadcastUint32x8(0x10ffff)).Or(surrogate).ToBits() != 0
}

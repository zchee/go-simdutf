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

package simdutf

import (
	"encoding/binary"
	"math/bits"
)

// Translated and adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/scalar/utf8_to_latin1/{utf8_to_latin1.h,valid_utf8_to_latin1.h},
// include/simdutf/scalar/utf8_to_utf16/{utf8_to_utf16.h,valid_utf8_to_utf16.h},
// include/simdutf/scalar/utf8_to_utf32/{utf8_to_utf32.h,valid_utf8_to_utf32.h},
// and src/fallback/implementation.cpp. Go slices make destination bounds
// explicit and require an all-or-nothing preflight for conversion APIs.

func convertUTF8ToLatin1Scalar(input, dst []byte) int {
	result := convertUTF8ToLatin1WithErrorsScalar(input, dst)
	if result.Error != Success {
		return 0
	}
	return result.Count
}

func convertUTF8ToLatin1WithErrorsScalar(input, dst []byte) Result {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		if pos+16 <= len(input) {
			v1 := binary.LittleEndian.Uint64(input[pos:])
			v2 := binary.LittleEndian.Uint64(input[pos+8:])
			if (v1|v2)&0x8080808080808080 == 0 {
				copy(dst[out:out+16], input[pos:pos+16])
				pos += 16
				out += 16
				continue
			}
		}

		leading := input[pos]
		switch {
		case leading < 0x80:
			dst[out] = leading
			out++
			pos++
		case leading&0xe0 == 0xc0:
			if pos+1 >= len(input) {
				return Result{Error: TooShort, Count: pos}
			}
			if input[pos+1]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(leading&0x1f)<<6 | uint32(input[pos+1]&0x3f)
			if codePoint < 0x80 {
				return Result{Error: Overlong, Count: pos}
			}
			if codePoint > 0xff {
				return Result{Error: TooLarge, Count: pos}
			}
			dst[out] = byte(codePoint)
			out++
			pos += 2
		case leading&0xf0 == 0xe0:
			return Result{Error: TooLarge, Count: pos}
		case leading&0xf8 == 0xf0:
			return Result{Error: TooLarge, Count: pos}
		case leading&0xc0 == 0x80:
			return Result{Error: TooLong, Count: pos}
		default:
			return Result{Error: HeaderBits, Count: pos}
		}
	}
	return Result{Error: Success, Count: out}
}

func convertValidUTF8ToLatin1Scalar(input, dst []byte) int {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
loop:
	for pos < len(input) {
		if pos+16 <= len(input) {
			v1 := binary.LittleEndian.Uint64(input[pos:])
			v2 := binary.LittleEndian.Uint64(input[pos+8:])
			if (v1|v2)&0x8080808080808080 == 0 {
				copy(dst[out:out+16], input[pos:pos+16])
				pos += 16
				out += 16
				continue
			}
		}

		leading := input[pos]
		switch {
		case leading < 0x80:
			dst[out] = leading
			out++
			pos++
		case leading&0xe0 == 0xc0:
			if pos+1 >= len(input) {
				break loop
			}
			if input[pos+1]&0xc0 != 0x80 {
				return 0
			}
			codePoint := uint32(leading&0x1f)<<6 | uint32(input[pos+1]&0x3f)
			dst[out] = byte(codePoint)
			out++
			pos += 2
		default:
			return 0
		}
	}
	return out
}

func convertUTF8ToUTF16LEScalar(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16Scalar(input, dst, nativeLittleEndian())
}

func convertUTF8ToUTF16BEScalar(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16Scalar(input, dst, !nativeLittleEndian())
}

func convertUTF8ToUTF16Scalar(input []byte, dst []uint16, storageIsNative bool) int {
	result := convertUTF8ToUTF16WithErrorsScalar(input, dst, storageIsNative)
	if result.Error != Success {
		return 0
	}
	return result.Count
}

func convertUTF8ToUTF16LEWithErrorsScalar(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsScalar(input, dst, nativeLittleEndian())
}

func convertUTF8ToUTF16BEWithErrorsScalar(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsScalar(input, dst, !nativeLittleEndian())
}

func convertUTF8ToUTF16WithErrorsScalar(input []byte, dst []uint16, storageIsNative bool) Result {
	if len(dst) < utf16LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		if pos+16 <= len(input) {
			v1 := binary.LittleEndian.Uint64(input[pos:])
			v2 := binary.LittleEndian.Uint64(input[pos+8:])
			if (v1|v2)&0x8080808080808080 == 0 {
				for i := range 16 {
					dst[out] = storeUTF16Word(uint16(input[pos+i]), storageIsNative)
					out++
				}
				pos += 16
				continue
			}
		}

		leading := input[pos]
		switch {
		case leading < 0x80:
			dst[out] = storeUTF16Word(uint16(leading), storageIsNative)
			out++
			pos++
		case leading&0xe0 == 0xc0:
			if pos+1 >= len(input) {
				return Result{Error: TooShort, Count: pos}
			}
			if input[pos+1]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(leading&0x1f)<<6 | uint32(input[pos+1]&0x3f)
			if codePoint < 0x80 {
				return Result{Error: Overlong, Count: pos}
			}
			dst[out] = storeUTF16Word(uint16(codePoint), storageIsNative)
			out++
			pos += 2
		case leading&0xf0 == 0xe0:
			if pos+2 >= len(input) {
				return Result{Error: TooShort, Count: pos}
			}
			if input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(leading&0x0f)<<12 |
				uint32(input[pos+1]&0x3f)<<6 |
				uint32(input[pos+2]&0x3f)
			if codePoint < 0x800 {
				return Result{Error: Overlong, Count: pos}
			}
			if codePoint > 0xd7ff && codePoint < 0xe000 {
				return Result{Error: Surrogate, Count: pos}
			}
			dst[out] = storeUTF16Word(uint16(codePoint), storageIsNative)
			out++
			pos += 3
		case leading&0xf8 == 0xf0:
			if pos+3 >= len(input) {
				return Result{Error: TooShort, Count: pos}
			}
			if input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 || input[pos+3]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(leading&0x07)<<18 |
				uint32(input[pos+1]&0x3f)<<12 |
				uint32(input[pos+2]&0x3f)<<6 |
				uint32(input[pos+3]&0x3f)
			if codePoint <= 0xffff {
				return Result{Error: Overlong, Count: pos}
			}
			if codePoint > 0x10ffff {
				return Result{Error: TooLarge, Count: pos}
			}
			codePoint -= 0x10000
			high := uint16(0xd800 + (codePoint >> 10))
			low := uint16(0xdc00 + (codePoint & 0x3ff))
			dst[out] = storeUTF16Word(high, storageIsNative)
			dst[out+1] = storeUTF16Word(low, storageIsNative)
			out += 2
			pos += 4
		case leading&0xc0 == 0x80:
			return Result{Error: TooLong, Count: pos}
		default:
			return Result{Error: HeaderBits, Count: pos}
		}
	}
	return Result{Error: Success, Count: out}
}

func convertValidUTF8ToUTF16LEScalar(input []byte, dst []uint16) int {
	return convertValidUTF8ToUTF16Scalar(input, dst, nativeLittleEndian())
}

func convertValidUTF8ToUTF16BEScalar(input []byte, dst []uint16) int {
	return convertValidUTF8ToUTF16Scalar(input, dst, !nativeLittleEndian())
}

func convertValidUTF8ToUTF16Scalar(input []byte, dst []uint16, storageIsNative bool) int {
	if len(dst) < utf16LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
loop:
	for pos < len(input) {
		if pos+8 <= len(input) {
			v := binary.LittleEndian.Uint64(input[pos:])
			if v&0x8080808080808080 == 0 {
				for i := range 8 {
					dst[out] = storeUTF16Word(uint16(input[pos+i]), storageIsNative)
					out++
				}
				pos += 8
				continue
			}
		}

		leading := input[pos]
		switch {
		case leading < 0x80:
			dst[out] = storeUTF16Word(uint16(leading), storageIsNative)
			out++
			pos++
		case leading&0xe0 == 0xc0:
			if pos+1 >= len(input) {
				break loop
			}
			codePoint := uint16(leading&0x1f)<<6 | uint16(input[pos+1]&0x3f)
			dst[out] = storeUTF16Word(codePoint, storageIsNative)
			out++
			pos += 2
		case leading&0xf0 == 0xe0:
			if pos+2 >= len(input) {
				break loop
			}
			codePoint := uint16(leading&0x0f)<<12 |
				uint16(input[pos+1]&0x3f)<<6 |
				uint16(input[pos+2]&0x3f)
			dst[out] = storeUTF16Word(codePoint, storageIsNative)
			out++
			pos += 3
		case leading&0xf8 == 0xf0:
			if pos+3 >= len(input) {
				break loop
			}
			codePoint := uint32(leading&0x07)<<18 |
				uint32(input[pos+1]&0x3f)<<12 |
				uint32(input[pos+2]&0x3f)<<6 |
				uint32(input[pos+3]&0x3f)
			codePoint -= 0x10000
			high := uint16(0xd800 + (codePoint >> 10))
			low := uint16(0xdc00 + (codePoint & 0x3ff))
			dst[out] = storeUTF16Word(high, storageIsNative)
			dst[out+1] = storeUTF16Word(low, storageIsNative)
			out += 2
			pos += 4
		default:
			return 0
		}
	}
	return out
}

func convertUTF8ToUTF32Scalar(input []byte, dst []uint32) int {
	result := convertUTF8ToUTF32WithErrorsScalar(input, dst)
	if result.Error != Success {
		return 0
	}
	return result.Count
}

func convertUTF8ToUTF32WithErrorsScalar(input []byte, dst []uint32) Result {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		if pos+16 <= len(input) {
			v1 := binary.LittleEndian.Uint64(input[pos:])
			v2 := binary.LittleEndian.Uint64(input[pos+8:])
			if (v1|v2)&0x8080808080808080 == 0 {
				for i := range 16 {
					dst[out] = uint32(input[pos+i])
					out++
				}
				pos += 16
				continue
			}
		}

		leading := input[pos]
		switch {
		case leading < 0x80:
			dst[out] = uint32(leading)
			out++
			pos++
		case leading&0xe0 == 0xc0:
			if pos+1 >= len(input) {
				return Result{Error: TooShort, Count: pos}
			}
			if input[pos+1]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(leading&0x1f)<<6 | uint32(input[pos+1]&0x3f)
			if codePoint < 0x80 {
				return Result{Error: Overlong, Count: pos}
			}
			dst[out] = codePoint
			out++
			pos += 2
		case leading&0xf0 == 0xe0:
			if pos+2 >= len(input) {
				return Result{Error: TooShort, Count: pos}
			}
			if input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(leading&0x0f)<<12 |
				uint32(input[pos+1]&0x3f)<<6 |
				uint32(input[pos+2]&0x3f)
			if codePoint < 0x800 {
				return Result{Error: Overlong, Count: pos}
			}
			if codePoint > 0xd7ff && codePoint < 0xe000 {
				return Result{Error: Surrogate, Count: pos}
			}
			dst[out] = codePoint
			out++
			pos += 3
		case leading&0xf8 == 0xf0:
			if pos+3 >= len(input) {
				return Result{Error: TooShort, Count: pos}
			}
			if input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 || input[pos+3]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(leading&0x07)<<18 |
				uint32(input[pos+1]&0x3f)<<12 |
				uint32(input[pos+2]&0x3f)<<6 |
				uint32(input[pos+3]&0x3f)
			if codePoint <= 0xffff {
				return Result{Error: Overlong, Count: pos}
			}
			if codePoint > 0x10ffff {
				return Result{Error: TooLarge, Count: pos}
			}
			dst[out] = codePoint
			out++
			pos += 4
		case leading&0xc0 == 0x80:
			return Result{Error: TooLong, Count: pos}
		default:
			return Result{Error: HeaderBits, Count: pos}
		}
	}
	return Result{Error: Success, Count: out}
}

func convertValidUTF8ToUTF32Scalar(input []byte, dst []uint32) int {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
loop:
	for pos < len(input) {
		if pos+8 <= len(input) {
			v := binary.LittleEndian.Uint64(input[pos:])
			if v&0x8080808080808080 == 0 {
				for i := range 8 {
					dst[out] = uint32(input[pos+i])
					out++
				}
				pos += 8
				continue
			}
		}

		leading := input[pos]
		switch {
		case leading < 0x80:
			dst[out] = uint32(leading)
			out++
			pos++
		case leading&0xe0 == 0xc0:
			if pos+1 >= len(input) {
				break loop
			}
			dst[out] = uint32(leading&0x1f)<<6 | uint32(input[pos+1]&0x3f)
			out++
			pos += 2
		case leading&0xf0 == 0xe0:
			if pos+2 >= len(input) {
				break loop
			}
			dst[out] = uint32(leading&0x0f)<<12 |
				uint32(input[pos+1]&0x3f)<<6 |
				uint32(input[pos+2]&0x3f)
			out++
			pos += 3
		case leading&0xf8 == 0xf0:
			if pos+3 >= len(input) {
				break loop
			}
			dst[out] = uint32(leading&0x07)<<18 |
				uint32(input[pos+1]&0x3f)<<12 |
				uint32(input[pos+2]&0x3f)<<6 |
				uint32(input[pos+3]&0x3f)
			out++
			pos += 4
		default:
			return 0
		}
	}
	return out
}

func storeUTF16Word(value uint16, storageIsNative bool) uint16 {
	if !storageIsNative {
		return bits.ReverseBytes16(value)
	}
	return value
}

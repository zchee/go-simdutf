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

// Translated and adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/scalar/utf32_to_latin1/{utf32_to_latin1.h,valid_utf32_to_latin1.h},
// include/simdutf/scalar/utf32_to_utf8/{utf32_to_utf8.h,valid_utf32_to_utf8.h},
// include/simdutf/scalar/utf32_to_utf16/{utf32_to_utf16.h,valid_utf32_to_utf16.h},
// include/simdutf/scalar/utf32.h (utf8_length_from_utf32, utf16_length_from_utf32),
// and src/fallback/implementation.cpp. Go slices make destination bounds
// explicit and require an all-or-nothing preflight for conversion APIs.

func latin1LengthFromUTF32Scalar(length int) int {
	return length
}

func utf8LengthFromUTF32Scalar(input []uint32) int {
	counter := 0
	for _, word := range input {
		counter++
		if word > 0x7f {
			counter++
		}
		if word > 0x7ff {
			counter++
		}
		if word > 0xffff {
			counter++
		}
	}
	return counter
}

func utf16LengthFromUTF32Scalar(input []uint32) int {
	counter := 0
	for _, word := range input {
		counter++
		if word > 0xffff {
			counter++
		}
	}
	return counter
}

func convertUTF32ToLatin1Scalar(input []uint32, dst []byte) int {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) == 0 {
		return 0
	}
	var tooLarge uint32
	for i, word := range input {
		tooLarge |= word
		dst[i] = byte(word)
	}
	if tooLarge&0xffffff00 != 0 {
		return 0
	}
	return len(input)
}

func convertUTF32ToLatin1WithErrorsScalar(input []uint32, dst []byte) Result {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	for i, word := range input {
		if word&0xffffff00 != 0 {
			return Result{Error: TooLarge, Count: i}
		}
		dst[i] = byte(word)
	}
	return Result{Error: Success, Count: len(input)}
}

func convertValidUTF32ToLatin1Scalar(input []uint32, dst []byte) int {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	for i, word := range input {
		if word&0xffffff00 != 0 {
			return 0
		}
		dst[i] = byte(word)
	}
	return len(input)
}

func convertUTF32ToUTF8Scalar(input []uint32, dst []byte) int {
	result := convertUTF32ToUTF8WithErrorsScalar(input, dst)
	if result.Error != Success {
		return 0
	}
	return result.Count
}

func convertUTF32ToUTF8WithErrorsScalar(input []uint32, dst []byte) Result {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		word := input[pos]
		switch {
		case word&0xffffff80 == 0:
			dst[out] = byte(word)
			out++
			pos++
		case word&0xfffff800 == 0:
			dst[out] = byte((word >> 6) | 0xc0)
			dst[out+1] = byte((word & 0x3f) | 0x80)
			out += 2
			pos++
		case word&0xffff0000 == 0:
			if word >= 0xd800 && word <= 0xdfff {
				return Result{Error: Surrogate, Count: pos}
			}
			dst[out] = byte((word >> 12) | 0xe0)
			dst[out+1] = byte(((word >> 6) & 0x3f) | 0x80)
			dst[out+2] = byte((word & 0x3f) | 0x80)
			out += 3
			pos++
		default:
			if word > 0x10ffff {
				return Result{Error: TooLarge, Count: pos}
			}
			dst[out] = byte((word >> 18) | 0xf0)
			dst[out+1] = byte(((word >> 12) & 0x3f) | 0x80)
			dst[out+2] = byte(((word >> 6) & 0x3f) | 0x80)
			dst[out+3] = byte((word & 0x3f) | 0x80)
			out += 4
			pos++
		}
	}
	return Result{Error: Success, Count: out}
}

func convertValidUTF32ToUTF8Scalar(input []uint32, dst []byte) int {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		word := input[pos]
		switch {
		case word&0xffffff80 == 0:
			dst[out] = byte(word)
			out++
			pos++
		case word&0xfffff800 == 0:
			dst[out] = byte((word >> 6) | 0xc0)
			dst[out+1] = byte((word & 0x3f) | 0x80)
			out += 2
			pos++
		case word&0xffff0000 == 0:
			dst[out] = byte((word >> 12) | 0xe0)
			dst[out+1] = byte(((word >> 6) & 0x3f) | 0x80)
			dst[out+2] = byte((word & 0x3f) | 0x80)
			out += 3
			pos++
		default:
			dst[out] = byte((word >> 18) | 0xf0)
			dst[out+1] = byte(((word >> 12) & 0x3f) | 0x80)
			dst[out+2] = byte(((word >> 6) & 0x3f) | 0x80)
			dst[out+3] = byte((word & 0x3f) | 0x80)
			out += 4
			pos++
		}
	}
	return out
}

func convertUTF32ToUTF16LEScalar(input []uint32, dst []uint16) int {
	return convertUTF32ToUTF16Scalar(input, dst, nativeLittleEndian())
}

func convertUTF32ToUTF16BEScalar(input []uint32, dst []uint16) int {
	return convertUTF32ToUTF16Scalar(input, dst, !nativeLittleEndian())
}

func convertUTF32ToUTF16Scalar(input []uint32, dst []uint16, storageIsNative bool) int {
	result := convertUTF32ToUTF16WithErrorsScalar(input, dst, storageIsNative)
	if result.Error != Success {
		return 0
	}
	return result.Count
}

func convertUTF32ToUTF16LEWithErrorsScalar(input []uint32, dst []uint16) Result {
	return convertUTF32ToUTF16WithErrorsScalar(input, dst, nativeLittleEndian())
}

func convertUTF32ToUTF16BEWithErrorsScalar(input []uint32, dst []uint16) Result {
	return convertUTF32ToUTF16WithErrorsScalar(input, dst, !nativeLittleEndian())
}

func convertUTF32ToUTF16WithErrorsScalar(input []uint32, dst []uint16, storageIsNative bool) Result {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		word := input[pos]
		if word&0xffff0000 == 0 {
			if word >= 0xd800 && word <= 0xdfff {
				return Result{Error: Surrogate, Count: pos}
			}
			dst[out] = utf16ScalarWord(uint16(word), storageIsNative)
			out++
		} else {
			if word > 0x10ffff {
				return Result{Error: TooLarge, Count: pos}
			}
			word -= 0x10000
			high := uint16(0xd800 + (word >> 10))
			low := uint16(0xdc00 + (word & 0x3ff))
			dst[out] = utf16ScalarWord(high, storageIsNative)
			dst[out+1] = utf16ScalarWord(low, storageIsNative)
			out += 2
		}
		pos++
	}
	return Result{Error: Success, Count: out}
}

func convertValidUTF32ToUTF16LEScalar(input []uint32, dst []uint16) int {
	return convertValidUTF32ToUTF16Scalar(input, dst, nativeLittleEndian())
}

func convertValidUTF32ToUTF16BEScalar(input []uint32, dst []uint16) int {
	return convertValidUTF32ToUTF16Scalar(input, dst, !nativeLittleEndian())
}

func convertValidUTF32ToUTF16Scalar(input []uint32, dst []uint16, storageIsNative bool) int {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		word := input[pos]
		if word&0xffff0000 == 0 {
			dst[out] = utf16ScalarWord(uint16(word), storageIsNative)
			out++
			pos++
			continue
		}
		word -= 0x10000
		high := uint16(0xd800 + (word >> 10))
		low := uint16(0xdc00 + (word & 0x3ff))
		dst[out] = utf16ScalarWord(high, storageIsNative)
		dst[out+1] = utf16ScalarWord(low, storageIsNative)
		out += 2
		pos++
	}
	return out
}

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
// include/simdutf/scalar/utf16_to_latin1/{utf16_to_latin1.h,valid_utf16_to_latin1.h},
// include/simdutf/scalar/utf16_to_utf32/{utf16_to_utf32.h,valid_utf16_to_utf32.h},
// include/simdutf/scalar/utf16_to_utf8/{utf16_to_utf8.h,valid_utf16_to_utf8.h},
// include/simdutf/scalar/utf16.h (utf32_length_from_utf16, utf8_length_from_utf16,
// utf8_length_from_utf16_with_replacement), and
// src/fallback/implementation.cpp. Go slices make destination bounds explicit
// and require an all-or-nothing preflight for conversion APIs.

func latin1LengthFromUTF16Scalar(length int) int {
	return length
}

func utf32LengthFromUTF16LEScalar(input []uint16) int {
	return utf32LengthFromUTF16Scalar(input, nativeLittleEndian())
}

func utf32LengthFromUTF16BEScalar(input []uint16) int {
	return utf32LengthFromUTF16Scalar(input, !nativeLittleEndian())
}

func utf32LengthFromUTF16Scalar(input []uint16, storageIsNative bool) int {
	counter := 0
	for _, raw := range input {
		word := utf16ScalarWord(raw, storageIsNative)
		if word&0xfc00 != 0xdc00 {
			counter++
		}
	}
	return counter
}

func utf8LengthFromUTF16LEScalar(input []uint16) int {
	return utf8LengthFromUTF16Scalar(input, nativeLittleEndian())
}

func utf8LengthFromUTF16BEScalar(input []uint16) int {
	return utf8LengthFromUTF16Scalar(input, !nativeLittleEndian())
}

func utf8LengthFromUTF16Scalar(input []uint16, storageIsNative bool) int {
	counter := 0
	for _, raw := range input {
		word := utf16ScalarWord(raw, storageIsNative)
		counter++
		if word > 0x7f {
			counter++
		}
		if (word > 0x7ff && word <= 0xd7ff) || word >= 0xe000 {
			counter++
		}
	}
	return counter
}

func utf8LengthFromUTF16WithReplacementScalar(input []uint16, storageIsNative bool) int {
	counter := 0
	for i := 0; i < len(input); i++ {
		word := utf16ScalarWord(input[i], storageIsNative)
		if word >= 0xd800 && word <= 0xdbff {
			if i+1 < len(input) {
				next := utf16ScalarWord(input[i+1], storageIsNative)
				if next >= 0xdc00 && next <= 0xdfff {
					counter += 4
					i++
					continue
				}
			}
			counter += 3
			continue
		}
		if word >= 0xdc00 && word <= 0xdfff {
			counter += 3
			continue
		}
		counter++
		if word > 0x7f {
			counter++
		}
		if word > 0x7ff {
			counter++
		}
	}
	return counter
}

func convertUTF16LEToLatin1Scalar(input []uint16, dst []byte) int {
	return convertUTF16ToLatin1Scalar(input, dst, nativeLittleEndian())
}

func convertUTF16BEToLatin1Scalar(input []uint16, dst []byte) int {
	return convertUTF16ToLatin1Scalar(input, dst, !nativeLittleEndian())
}

func convertUTF16ToLatin1Scalar(input []uint16, dst []byte, storageIsNative bool) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) == 0 {
		return 0
	}
	var tooLarge uint16
	for i, raw := range input {
		word := utf16ScalarWord(raw, storageIsNative)
		tooLarge |= word
		dst[i] = byte(word)
	}
	if tooLarge&0xff00 != 0 {
		return 0
	}
	return len(input)
}

func convertUTF16LEToLatin1WithErrorsScalar(input []uint16, dst []byte) Result {
	return convertUTF16ToLatin1WithErrorsScalar(input, dst, nativeLittleEndian())
}

func convertUTF16BEToLatin1WithErrorsScalar(input []uint16, dst []byte) Result {
	return convertUTF16ToLatin1WithErrorsScalar(input, dst, !nativeLittleEndian())
}

func convertUTF16ToLatin1WithErrorsScalar(input []uint16, dst []byte, storageIsNative bool) Result {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	for i, raw := range input {
		word := utf16ScalarWord(raw, storageIsNative)
		if word&0xff00 != 0 {
			return Result{Error: TooLarge, Count: i}
		}
		dst[i] = byte(word)
	}
	return Result{Error: Success, Count: len(input)}
}

func convertValidUTF16LEToLatin1Scalar(input []uint16, dst []byte) int {
	return convertValidUTF16ToLatin1Scalar(input, dst, nativeLittleEndian())
}

func convertValidUTF16BEToLatin1Scalar(input []uint16, dst []byte) int {
	return convertValidUTF16ToLatin1Scalar(input, dst, !nativeLittleEndian())
}

func convertValidUTF16ToLatin1Scalar(input []uint16, dst []byte, storageIsNative bool) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	for i, raw := range input {
		dst[i] = byte(utf16ScalarWord(raw, storageIsNative))
	}
	return len(input)
}

func convertUTF16LEToUTF32Scalar(input []uint16, dst []uint32) int {
	return convertUTF16ToUTF32Scalar(input, dst, nativeLittleEndian())
}

func convertUTF16BEToUTF32Scalar(input []uint16, dst []uint32) int {
	return convertUTF16ToUTF32Scalar(input, dst, !nativeLittleEndian())
}

func convertUTF16ToUTF32Scalar(input []uint16, dst []uint32, storageIsNative bool) int {
	result := convertUTF16ToUTF32WithErrorsScalar(input, dst, storageIsNative)
	if result.Error != Success {
		return 0
	}
	return result.Count
}

func convertUTF16LEToUTF32WithErrorsScalar(input []uint16, dst []uint32) Result {
	return convertUTF16ToUTF32WithErrorsScalar(input, dst, nativeLittleEndian())
}

func convertUTF16BEToUTF32WithErrorsScalar(input []uint16, dst []uint32) Result {
	return convertUTF16ToUTF32WithErrorsScalar(input, dst, !nativeLittleEndian())
}

func convertUTF16ToUTF32WithErrorsScalar(input []uint16, dst []uint32, storageIsNative bool) Result {
	if len(dst) < utf32LengthFromUTF16Scalar(input, storageIsNative) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		word := utf16ScalarWord(input[pos], storageIsNative)
		if word&0xf800 != 0xd800 {
			dst[out] = uint32(word)
			out++
			pos++
			continue
		}
		diff := word - 0xd800
		if diff > 0x3ff {
			return Result{Error: Surrogate, Count: pos}
		}
		if pos+1 >= len(input) {
			return Result{Error: Surrogate, Count: pos}
		}
		next := utf16ScalarWord(input[pos+1], storageIsNative)
		diff2 := next - 0xdc00
		if diff2 > 0x3ff {
			return Result{Error: Surrogate, Count: pos}
		}
		dst[out] = (uint32(diff) << 10) + uint32(diff2) + 0x10000
		out++
		pos += 2
	}
	return Result{Error: Success, Count: out}
}

func convertValidUTF16LEToUTF32Scalar(input []uint16, dst []uint32) int {
	return convertValidUTF16ToUTF32Scalar(input, dst, nativeLittleEndian())
}

func convertValidUTF16BEToUTF32Scalar(input []uint16, dst []uint32) int {
	return convertValidUTF16ToUTF32Scalar(input, dst, !nativeLittleEndian())
}

func convertValidUTF16ToUTF32Scalar(input []uint16, dst []uint32, storageIsNative bool) int {
	if len(dst) < utf32LengthFromUTF16Scalar(input, storageIsNative) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		word := utf16ScalarWord(input[pos], storageIsNative)
		if word&0xf800 != 0xd800 {
			dst[out] = uint32(word)
			out++
			pos++
			continue
		}
		if pos+1 >= len(input) {
			return 0
		}
		diff := word - 0xd800
		next := utf16ScalarWord(input[pos+1], storageIsNative)
		diff2 := next - 0xdc00
		dst[out] = (uint32(diff) << 10) + uint32(diff2) + 0x10000
		out++
		pos += 2
	}
	return out
}

func convertUTF16LEToUTF8Scalar(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8Scalar(input, dst, nativeLittleEndian())
}

func convertUTF16BEToUTF8Scalar(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8Scalar(input, dst, !nativeLittleEndian())
}

func convertUTF16ToUTF8Scalar(input []uint16, dst []byte, storageIsNative bool) int {
	result := convertUTF16ToUTF8WithErrorsScalar(input, dst, storageIsNative)
	if result.Error != Success {
		return 0
	}
	return result.Count
}

func convertUTF16LEToUTF8WithErrorsScalar(input []uint16, dst []byte) Result {
	return convertUTF16ToUTF8WithErrorsScalar(input, dst, nativeLittleEndian())
}

func convertUTF16BEToUTF8WithErrorsScalar(input []uint16, dst []byte) Result {
	return convertUTF16ToUTF8WithErrorsScalar(input, dst, !nativeLittleEndian())
}

func convertUTF16ToUTF8WithErrorsScalar(input []uint16, dst []byte, storageIsNative bool) Result {
	if len(dst) < utf8LengthFromUTF16Scalar(input, storageIsNative) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		word := utf16ScalarWord(input[pos], storageIsNative)
		if word&0xff80 == 0 {
			dst[out] = byte(word)
			out++
			pos++
			continue
		}
		if word&0xf800 == 0 {
			dst[out] = byte((word >> 6) | 0xc0)
			dst[out+1] = byte((word & 0x3f) | 0x80)
			out += 2
			pos++
			continue
		}
		if word&0xf800 != 0xd800 {
			dst[out] = byte((word >> 12) | 0xe0)
			dst[out+1] = byte(((word >> 6) & 0x3f) | 0x80)
			dst[out+2] = byte((word & 0x3f) | 0x80)
			out += 3
			pos++
			continue
		}
		diff := word - 0xd800
		if diff > 0x3ff {
			return Result{Error: Surrogate, Count: pos}
		}
		if pos+1 >= len(input) {
			return Result{Error: Surrogate, Count: pos}
		}
		next := utf16ScalarWord(input[pos+1], storageIsNative)
		diff2 := next - 0xdc00
		if diff2 > 0x3ff {
			return Result{Error: Surrogate, Count: pos}
		}
		value := (uint32(diff) << 10) + uint32(diff2) + 0x10000
		dst[out] = byte((value >> 18) | 0xf0)
		dst[out+1] = byte(((value >> 12) & 0x3f) | 0x80)
		dst[out+2] = byte(((value >> 6) & 0x3f) | 0x80)
		dst[out+3] = byte((value & 0x3f) | 0x80)
		out += 4
		pos += 2
	}
	return Result{Error: Success, Count: out}
}

func convertUTF16LEToUTF8WithReplacementScalar(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8WithReplacementScalar(input, dst, nativeLittleEndian())
}

func convertUTF16BEToUTF8WithReplacementScalar(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8WithReplacementScalar(input, dst, !nativeLittleEndian())
}

func convertUTF16ToUTF8WithReplacementScalar(input []uint16, dst []byte, storageIsNative bool) int {
	if len(dst) < utf8LengthFromUTF16WithReplacementScalar(input, storageIsNative) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		word := utf16ScalarWord(input[pos], storageIsNative)
		if word&0xff80 == 0 {
			dst[out] = byte(word)
			out++
			pos++
			continue
		}
		if word&0xf800 == 0 {
			dst[out] = byte((word >> 6) | 0xc0)
			dst[out+1] = byte((word & 0x3f) | 0x80)
			out += 2
			pos++
			continue
		}
		if word&0xf800 != 0xd800 {
			dst[out] = byte((word >> 12) | 0xe0)
			dst[out+1] = byte(((word >> 6) & 0x3f) | 0x80)
			dst[out+2] = byte((word & 0x3f) | 0x80)
			out += 3
			pos++
			continue
		}
		diff := word - 0xd800
		if diff <= 0x3ff && pos+1 < len(input) {
			next := utf16ScalarWord(input[pos+1], storageIsNative)
			diff2 := next - 0xdc00
			if diff2 <= 0x3ff {
				value := (uint32(diff) << 10) + uint32(diff2) + 0x10000
				dst[out] = byte((value >> 18) | 0xf0)
				dst[out+1] = byte(((value >> 12) & 0x3f) | 0x80)
				dst[out+2] = byte(((value >> 6) & 0x3f) | 0x80)
				dst[out+3] = byte((value & 0x3f) | 0x80)
				out += 4
				pos += 2
				continue
			}
		}
		dst[out] = 0xef
		dst[out+1] = 0xbf
		dst[out+2] = 0xbd
		out += 3
		pos++
	}
	return out
}

func convertValidUTF16LEToUTF8Scalar(input []uint16, dst []byte) int {
	return convertValidUTF16ToUTF8Scalar(input, dst, nativeLittleEndian())
}

func convertValidUTF16BEToUTF8Scalar(input []uint16, dst []byte) int {
	return convertValidUTF16ToUTF8Scalar(input, dst, !nativeLittleEndian())
}

func convertValidUTF16ToUTF8Scalar(input []uint16, dst []byte, storageIsNative bool) int {
	if len(dst) < utf8LengthFromUTF16Scalar(input, storageIsNative) {
		panic("simdutf: destination is too short")
	}
	pos := 0
	out := 0
	for pos < len(input) {
		word := utf16ScalarWord(input[pos], storageIsNative)
		if word&0xff80 == 0 {
			dst[out] = byte(word)
			out++
			pos++
			continue
		}
		if word&0xf800 == 0 {
			dst[out] = byte((word >> 6) | 0xc0)
			dst[out+1] = byte((word & 0x3f) | 0x80)
			out += 2
			pos++
			continue
		}
		if word&0xf800 != 0xd800 {
			dst[out] = byte((word >> 12) | 0xe0)
			dst[out+1] = byte(((word >> 6) & 0x3f) | 0x80)
			dst[out+2] = byte((word & 0x3f) | 0x80)
			out += 3
			pos++
			continue
		}
		if pos+1 >= len(input) {
			return 0
		}
		diff := word - 0xd800
		next := utf16ScalarWord(input[pos+1], storageIsNative)
		diff2 := next - 0xdc00
		value := (uint32(diff) << 10) + uint32(diff2) + 0x10000
		dst[out] = byte((value >> 18) | 0xf0)
		dst[out+1] = byte(((value >> 12) & 0x3f) | 0x80)
		dst[out+2] = byte(((value >> 6) & 0x3f) | 0x80)
		dst[out+3] = byte((value & 0x3f) | 0x80)
		out += 4
		pos += 2
	}
	return out
}

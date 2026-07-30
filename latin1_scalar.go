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

import "math/bits"

// Translated and adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/scalar/latin1.h, include/simdutf/scalar/latin1_to_utf8/
// latin1_to_utf8.h, and src/fallback/implementation.cpp. Go slices make
// destination bounds explicit and require an all-or-nothing preflight for the
// non-safe conversion APIs.

func utf8LengthFromLatin1Scalar(input []byte) int {
	length := len(input)
	for _, value := range input {
		length += int(value >> 7)
	}
	return length
}

func convertLatin1ToUTF8Scalar(input, dst []byte) int {
	required := utf8LengthFromLatin1Scalar(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	written := 0
	for _, value := range input {
		if value < 0x80 {
			dst[written] = value
			written++
			continue
		}
		dst[written] = 0xc0 | value>>6
		dst[written+1] = 0x80 | value&0x3f
		written += 2
	}
	return written
}

func convertLatin1ToUTF8SafeScalar(input, dst []byte) int {
	written := 0
	for _, value := range input {
		width := 1
		if value >= 0x80 {
			width = 2
		}
		if len(dst)-written < width {
			break
		}
		if width == 1 {
			dst[written] = value
			written++
			continue
		}
		dst[written] = 0xc0 | value>>6
		dst[written+1] = 0x80 | value&0x3f
		written += 2
	}
	return written
}

func convertLatin1ToUTF32Scalar(input []byte, dst []uint32) int {
	if len(dst) < len(input) {
		panic("simdutf: destination is too short")
	}
	for index, value := range input {
		dst[index] = uint32(value)
	}
	return len(input)
}

func convertLatin1ToUTF16LEScalar(input []byte, dst []uint16) int {
	return convertLatin1ToUTF16Scalar(input, dst, nativeLittleEndian())
}

func convertLatin1ToUTF16BEScalar(input []byte, dst []uint16) int {
	return convertLatin1ToUTF16Scalar(input, dst, !nativeLittleEndian())
}

func convertLatin1ToUTF16Scalar(input []byte, dst []uint16, storageIsNative bool) int {
	if len(dst) < len(input) {
		panic("simdutf: destination is too short")
	}
	for index, value := range input {
		word := uint16(value)
		if !storageIsNative {
			word = bits.ReverseBytes16(word)
		}
		dst[index] = word
	}
	return len(input)
}

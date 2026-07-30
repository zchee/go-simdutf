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
// include/simdutf/scalar/utf16.h:20-76,137-145,183-213.

func validateUTF16LEScalar(input []uint16) bool {
	return validateUTF16Scalar(input, nativeLittleEndian())
}

func validateUTF16BEScalar(input []uint16) bool {
	return validateUTF16Scalar(input, !nativeLittleEndian())
}

func validateUTF16LEWithErrorsScalar(input []uint16) Result {
	return validateUTF16WithErrorsScalar(input, nativeLittleEndian())
}

func validateUTF16BEWithErrorsScalar(input []uint16) Result {
	return validateUTF16WithErrorsScalar(input, !nativeLittleEndian())
}

func validateUTF16Scalar(input []uint16, storageIsNative bool) bool {
	for pos := 0; pos < len(input); {
		word := utf16ScalarWord(input[pos], storageIsNative)
		if word < 0xd800 || word > 0xdfff {
			pos++
			continue
		}
		if word > 0xdbff || pos+1 == len(input) {
			return false
		}
		if next := utf16ScalarWord(input[pos+1], storageIsNative); next < 0xdc00 || next > 0xdfff {
			return false
		}
		pos += 2
	}
	return true
}

func validateUTF16WithErrorsScalar(input []uint16, storageIsNative bool) Result {
	for pos := 0; pos < len(input); {
		word := utf16ScalarWord(input[pos], storageIsNative)
		if word < 0xd800 || word > 0xdfff {
			pos++
			continue
		}
		if word > 0xdbff || pos+1 == len(input) {
			return Result{Error: Surrogate, Count: pos}
		}
		if next := utf16ScalarWord(input[pos+1], storageIsNative); next < 0xdc00 || next > 0xdfff {
			return Result{Error: Surrogate, Count: pos}
		}
		pos += 2
	}
	return Result{Error: Success, Count: len(input)}
}

// toWellFormedUTF16LEScalar repairs little-endian raw UTF-16 storage. dst must
// have at least len(input) elements. Input and dst may be identical; other
// overlap is not supported.
func toWellFormedUTF16LEScalar(input, dst []uint16) {
	toWellFormedUTF16Scalar(input, dst, nativeLittleEndian())
}

// toWellFormedUTF16BEScalar repairs big-endian raw UTF-16 storage. dst must
// have at least len(input) elements. Input and dst may be identical; other
// overlap is not supported.
func toWellFormedUTF16BEScalar(input, dst []uint16) {
	toWellFormedUTF16Scalar(input, dst, !nativeLittleEndian())
}

func toWellFormedUTF16Scalar(input, dst []uint16, storageIsNative bool) {
	if len(dst) < len(input) {
		panic("simdutf: UTF-16 destination too short")
	}
	words := input
	replacement := uint16(0xfffd)
	if !storageIsNative {
		replacement = bits.ReverseBytes16(replacement)
	}
	for i := 0; i < len(words); i++ {
		word := utf16ScalarWord(words[i], storageIsNative)
		switch {
		case word >= 0xd800 && word <= 0xdbff:
			if i+1 < len(words) {
				next := utf16ScalarWord(words[i+1], storageIsNative)
				if next >= 0xdc00 && next <= 0xdfff {
					dst[i] = words[i]
					dst[i+1] = words[i+1]
					i++
					continue
				}
			}
			dst[i] = replacement
		case word >= 0xdc00 && word <= 0xdfff:
			dst[i] = replacement
		default:
			dst[i] = words[i]
		}
	}
}

func utf16ScalarWord(raw uint16, storageIsNative bool) uint16 {
	if !storageIsNative {
		return bits.ReverseBytes16(raw)
	}
	return raw
}

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

package simdutf

// Public API adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/implementation.h UTF-16 to Latin-1, UTF-16 to UTF-32, and
// UTF-16 to UTF-8 conversion and length entry points. Go slices replace C++
// pointer/length pairs. UTF-16 endian names describe raw []uint16 storage.

// ConvertUTF16LEToLatin1 converts little-endian raw UTF-16 storage to Latin-1.
// It returns 0 when a code unit is outside the Latin-1 range. It panics before
// writing when dst is shorter than len(input).
func ConvertUTF16LEToLatin1(input []uint16, dst []byte) int {
	return activeImplementation.convertUTF16LEToLatin1(input, dst)
}

// ConvertUTF16BEToLatin1 converts big-endian raw UTF-16 storage to Latin-1.
// It returns 0 when a code unit is outside the Latin-1 range. It panics before
// writing when dst is shorter than len(input).
func ConvertUTF16BEToLatin1(input []uint16, dst []byte) int {
	return activeImplementation.convertUTF16BEToLatin1(input, dst)
}

// ConvertUTF16LEToLatin1WithErrors converts little-endian raw UTF-16 storage to
// Latin-1. On success Count is the number of Latin-1 bytes written; on failure
// Count is the input error position. It panics before writing when dst is
// shorter than len(input).
func ConvertUTF16LEToLatin1WithErrors(input []uint16, dst []byte) Result {
	return activeImplementation.convertUTF16LEToLatin1WithErrors(input, dst)
}

// ConvertUTF16BEToLatin1WithErrors converts big-endian raw UTF-16 storage to
// Latin-1. On success Count is the number of Latin-1 bytes written; on failure
// Count is the input error position. It panics before writing when dst is
// shorter than len(input).
func ConvertUTF16BEToLatin1WithErrors(input []uint16, dst []byte) Result {
	return activeImplementation.convertUTF16BEToLatin1WithErrors(input, dst)
}

// ConvertValidUTF16LEToLatin1 converts valid little-endian raw UTF-16 storage
// that is representable as Latin-1. It panics before writing when dst is
// shorter than len(input).
func ConvertValidUTF16LEToLatin1(input []uint16, dst []byte) int {
	return activeImplementation.convertValidUTF16LEToLatin1(input, dst)
}

// ConvertValidUTF16BEToLatin1 converts valid big-endian raw UTF-16 storage that
// is representable as Latin-1. It panics before writing when dst is shorter
// than len(input).
func ConvertValidUTF16BEToLatin1(input []uint16, dst []byte) int {
	return activeImplementation.convertValidUTF16BEToLatin1(input, dst)
}

// ConvertUTF16LEToUTF32 converts little-endian raw UTF-16 storage to UTF-32.
// It returns 0 on unpaired surrogates. It panics before writing when dst is
// shorter than the UTF-32 length required for input.
func ConvertUTF16LEToUTF32(input []uint16, dst []uint32) int {
	return activeImplementation.convertUTF16LEToUTF32(input, dst)
}

// ConvertUTF16BEToUTF32 converts big-endian raw UTF-16 storage to UTF-32.
// It returns 0 on unpaired surrogates. It panics before writing when dst is
// shorter than the UTF-32 length required for input.
func ConvertUTF16BEToUTF32(input []uint16, dst []uint32) int {
	return activeImplementation.convertUTF16BEToUTF32(input, dst)
}

// ConvertUTF16LEToUTF32WithErrors converts little-endian raw UTF-16 storage to
// UTF-32. On success Count is the number of UTF-32 code units written; on
// failure Count is the input error position.
func ConvertUTF16LEToUTF32WithErrors(input []uint16, dst []uint32) Result {
	return activeImplementation.convertUTF16LEToUTF32WithErrors(input, dst)
}

// ConvertUTF16BEToUTF32WithErrors converts big-endian raw UTF-16 storage to
// UTF-32. On success Count is the number of UTF-32 code units written; on
// failure Count is the input error position.
func ConvertUTF16BEToUTF32WithErrors(input []uint16, dst []uint32) Result {
	return activeImplementation.convertUTF16BEToUTF32WithErrors(input, dst)
}

// ConvertValidUTF16LEToUTF32 converts valid little-endian raw UTF-16 storage to
// UTF-32. It panics before writing when dst is shorter than the UTF-32 length required for input.
func ConvertValidUTF16LEToUTF32(input []uint16, dst []uint32) int {
	return activeImplementation.convertValidUTF16LEToUTF32(input, dst)
}

// ConvertValidUTF16BEToUTF32 converts valid big-endian raw UTF-16 storage to
// UTF-32. It panics before writing when dst is shorter than the UTF-32 length required for input.
func ConvertValidUTF16BEToUTF32(input []uint16, dst []uint32) int {
	return activeImplementation.convertValidUTF16BEToUTF32(input, dst)
}

// UTF32LengthFromUTF16LE returns the number of UTF-32 code units required to
// encode little-endian raw UTF-16 storage.
func UTF32LengthFromUTF16LE(input []uint16) int {
	return activeImplementation.utf32LengthFromUTF16LE(input)
}

// UTF32LengthFromUTF16BE returns the number of UTF-32 code units required to
// encode big-endian raw UTF-16 storage.
func UTF32LengthFromUTF16BE(input []uint16) int {
	return activeImplementation.utf32LengthFromUTF16BE(input)
}

// ConvertUTF16LEToUTF8 converts little-endian raw UTF-16 storage to UTF-8.
// It returns 0 on unpaired surrogates. It panics before writing when dst is
// shorter than the UTF-8 length required for input.
func ConvertUTF16LEToUTF8(input []uint16, dst []byte) int {
	return activeImplementation.convertUTF16LEToUTF8(input, dst)
}

// ConvertUTF16BEToUTF8 converts big-endian raw UTF-16 storage to UTF-8.
// It returns 0 on unpaired surrogates. It panics before writing when dst is
// shorter than the UTF-8 length required for input.
func ConvertUTF16BEToUTF8(input []uint16, dst []byte) int {
	return activeImplementation.convertUTF16BEToUTF8(input, dst)
}

// ConvertUTF16LEToUTF8WithErrors converts little-endian raw UTF-16 storage to
// UTF-8. On success Count is the number of UTF-8 bytes written; on failure
// Count is the input error position.
func ConvertUTF16LEToUTF8WithErrors(input []uint16, dst []byte) Result {
	return activeImplementation.convertUTF16LEToUTF8WithErrors(input, dst)
}

// ConvertUTF16BEToUTF8WithErrors converts big-endian raw UTF-16 storage to
// UTF-8. On success Count is the number of UTF-8 bytes written; on failure
// Count is the input error position.
func ConvertUTF16BEToUTF8WithErrors(input []uint16, dst []byte) Result {
	return activeImplementation.convertUTF16BEToUTF8WithErrors(input, dst)
}

// ConvertUTF16LEToUTF8WithReplacement converts little-endian raw UTF-16 storage
// to UTF-8, writing U+FFFD for unpaired surrogates. It panics before writing
// when dst is shorter than the replacement UTF-8 length required for input.
func ConvertUTF16LEToUTF8WithReplacement(input []uint16, dst []byte) int {
	return activeImplementation.convertUTF16LEToUTF8WithReplacement(input, dst)
}

// ConvertUTF16BEToUTF8WithReplacement converts big-endian raw UTF-16 storage
// to UTF-8, writing U+FFFD for unpaired surrogates. It panics before writing
// when dst is shorter than the replacement UTF-8 length required for input.
func ConvertUTF16BEToUTF8WithReplacement(input []uint16, dst []byte) int {
	return activeImplementation.convertUTF16BEToUTF8WithReplacement(input, dst)
}

// ConvertValidUTF16LEToUTF8 converts valid little-endian raw UTF-16 storage to
// UTF-8. It panics before writing when dst is shorter than the UTF-8 length
// required for input.
func ConvertValidUTF16LEToUTF8(input []uint16, dst []byte) int {
	return activeImplementation.convertValidUTF16LEToUTF8(input, dst)
}

// ConvertValidUTF16BEToUTF8 converts valid big-endian raw UTF-16 storage to
// UTF-8. It panics before writing when dst is shorter than the UTF-8 length
// required for input.
func ConvertValidUTF16BEToUTF8(input []uint16, dst []byte) int {
	return activeImplementation.convertValidUTF16BEToUTF8(input, dst)
}

// UTF8LengthFromUTF16LE returns the number of UTF-8 bytes required to encode
// little-endian raw UTF-16 storage.
func UTF8LengthFromUTF16LE(input []uint16) int {
	return activeImplementation.utf8LengthFromUTF16LE(input)
}

// UTF8LengthFromUTF16BE returns the number of UTF-8 bytes required to encode
// big-endian raw UTF-16 storage.
func UTF8LengthFromUTF16BE(input []uint16) int {
	return activeImplementation.utf8LengthFromUTF16BE(input)
}

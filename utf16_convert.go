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
// include/simdutf/implementation.h UTF-16 to Latin-1 and UTF-16 to UTF-32
// conversion entry points. Go slices replace C++ pointer/length pairs. UTF-16
// endian names describe raw []uint16 storage.

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

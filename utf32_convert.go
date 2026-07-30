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
// include/simdutf/implementation.h UTF-32 to Latin-1, UTF-32 to UTF-8, and
// UTF-32 to UTF-16 conversion and length entry points. Go slices replace C++
// pointer/length pairs. UTF-16 endian names describe raw []uint16 storage.

// ConvertUTF32ToLatin1 converts UTF-32 input to Latin-1. It returns 0 when a
// code unit is outside the Latin-1 range. It panics before writing when dst is
// shorter than len(input).
func ConvertUTF32ToLatin1(input []uint32, dst []byte) int {
	return activeImplementation.convertUTF32ToLatin1(input, dst)
}

// ConvertUTF32ToLatin1WithErrors converts UTF-32 input to Latin-1. On success
// Count is the number of Latin-1 bytes written; on failure Count is the input
// error position. It panics before writing when dst is shorter than len(input).
func ConvertUTF32ToLatin1WithErrors(input []uint32, dst []byte) Result {
	return activeImplementation.convertUTF32ToLatin1WithErrors(input, dst)
}

// ConvertValidUTF32ToLatin1 converts valid UTF-32 input that is representable
// as Latin-1. It returns 0 when a code unit is outside the Latin-1 range. It
// panics before writing when dst is shorter than len(input).
func ConvertValidUTF32ToLatin1(input []uint32, dst []byte) int {
	return activeImplementation.convertValidUTF32ToLatin1(input, dst)
}

// ConvertUTF32ToUTF8 converts UTF-32 input to UTF-8. It returns 0 on surrogate
// or too-large code points. It panics before writing when dst is shorter than
// the UTF-8 length required for input.
func ConvertUTF32ToUTF8(input []uint32, dst []byte) int {
	return activeImplementation.convertUTF32ToUTF8(input, dst)
}

// ConvertUTF32ToUTF8WithErrors converts UTF-32 input to UTF-8. On success Count
// is the number of UTF-8 bytes written; on failure Count is the input error
// position. It panics before writing when dst is shorter than the UTF-8 length
// required for input.
func ConvertUTF32ToUTF8WithErrors(input []uint32, dst []byte) Result {
	return activeImplementation.convertUTF32ToUTF8WithErrors(input, dst)
}

// ConvertValidUTF32ToUTF8 converts valid UTF-32 input to UTF-8. It panics
// before writing when dst is shorter than the UTF-8 length required for input.
func ConvertValidUTF32ToUTF8(input []uint32, dst []byte) int {
	return activeImplementation.convertValidUTF32ToUTF8(input, dst)
}

// ConvertUTF32ToUTF16LE converts UTF-32 input to little-endian raw UTF-16
// storage. It returns 0 on surrogate or too-large code points. It panics before
// writing when dst is shorter than the UTF-16 length required for input.
func ConvertUTF32ToUTF16LE(input []uint32, dst []uint16) int {
	return activeImplementation.convertUTF32ToUTF16LE(input, dst)
}

// ConvertUTF32ToUTF16BE converts UTF-32 input to big-endian raw UTF-16
// storage. It returns 0 on surrogate or too-large code points. It panics before
// writing when dst is shorter than the UTF-16 length required for input.
func ConvertUTF32ToUTF16BE(input []uint32, dst []uint16) int {
	return activeImplementation.convertUTF32ToUTF16BE(input, dst)
}

// ConvertUTF32ToUTF16 converts UTF-32 input to host-native UTF-16 storage.
// It returns 0 on surrogate or too-large code points. It panics before writing
// when dst is shorter than the UTF-16 length required for input.
func ConvertUTF32ToUTF16(input []uint32, dst []uint16) int {
	if nativeLittleEndian() {
		return ConvertUTF32ToUTF16LE(input, dst)
	}
	return ConvertUTF32ToUTF16BE(input, dst)
}

// ConvertUTF32ToUTF16LEWithErrors converts UTF-32 input to little-endian raw
// UTF-16 storage. On success Count is the number of UTF-16 code units written;
// on failure Count is the input error position. It panics before writing when
// dst is shorter than the UTF-16 length required for input.
func ConvertUTF32ToUTF16LEWithErrors(input []uint32, dst []uint16) Result {
	return activeImplementation.convertUTF32ToUTF16LEWithErrors(input, dst)
}

// ConvertUTF32ToUTF16BEWithErrors converts UTF-32 input to big-endian raw
// UTF-16 storage. On success Count is the number of UTF-16 code units written;
// on failure Count is the input error position. It panics before writing when
// dst is shorter than the UTF-16 length required for input.
func ConvertUTF32ToUTF16BEWithErrors(input []uint32, dst []uint16) Result {
	return activeImplementation.convertUTF32ToUTF16BEWithErrors(input, dst)
}

// ConvertUTF32ToUTF16WithErrors converts UTF-32 input to host-native UTF-16
// storage. On success Count is the number of UTF-16 code units written; on
// failure Count is the input error position. It panics before writing when dst
// is shorter than the UTF-16 length required for input.
func ConvertUTF32ToUTF16WithErrors(input []uint32, dst []uint16) Result {
	if nativeLittleEndian() {
		return ConvertUTF32ToUTF16LEWithErrors(input, dst)
	}
	return ConvertUTF32ToUTF16BEWithErrors(input, dst)
}

// ConvertValidUTF32ToUTF16LE converts valid UTF-32 input to little-endian raw
// UTF-16 storage. It panics before writing when dst is shorter than the UTF-16
// length required for input.
func ConvertValidUTF32ToUTF16LE(input []uint32, dst []uint16) int {
	return activeImplementation.convertValidUTF32ToUTF16LE(input, dst)
}

// ConvertValidUTF32ToUTF16BE converts valid UTF-32 input to big-endian raw
// UTF-16 storage. It panics before writing when dst is shorter than the UTF-16
// length required for input.
func ConvertValidUTF32ToUTF16BE(input []uint32, dst []uint16) int {
	return activeImplementation.convertValidUTF32ToUTF16BE(input, dst)
}

// ConvertValidUTF32ToUTF16 converts valid UTF-32 input to host-native UTF-16
// storage. It panics before writing when dst is shorter than the UTF-16 length
// required for input.
func ConvertValidUTF32ToUTF16(input []uint32, dst []uint16) int {
	if nativeLittleEndian() {
		return ConvertValidUTF32ToUTF16LE(input, dst)
	}
	return ConvertValidUTF32ToUTF16BE(input, dst)
}

// UTF8LengthFromUTF32 returns the number of UTF-8 bytes required to encode
// UTF-32 input.
func UTF8LengthFromUTF32(input []uint32) int {
	return activeImplementation.utf8LengthFromUTF32(input)
}

// UTF16LengthFromUTF32 returns the number of UTF-16 code units required to
// encode UTF-32 input.
func UTF16LengthFromUTF32(input []uint32) int {
	return activeImplementation.utf16LengthFromUTF32(input)
}

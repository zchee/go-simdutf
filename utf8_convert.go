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
// include/simdutf/implementation.h:1030-1621. Go slices replace C++
// pointer/length pairs. UTF-16 endian names describe raw []uint16 storage.

// ConvertUTF8ToLatin1 converts UTF-8 input to Latin-1. It returns 0 when the
// input is not representable as Latin-1 or is invalid UTF-8. It panics before
// writing when dst is shorter than Latin1LengthFromUTF8(input).
func ConvertUTF8ToLatin1(input, dst []byte) int {
	return activeImplementation.convertUTF8ToLatin1(input, dst)
}

// ConvertUTF8ToLatin1WithErrors converts UTF-8 input to Latin-1. On success
// Count is the number of Latin-1 bytes written; on failure Count is the input
// error position. It panics before writing when dst is shorter than
// Latin1LengthFromUTF8(input).
func ConvertUTF8ToLatin1WithErrors(input, dst []byte) Result {
	return activeImplementation.convertUTF8ToLatin1WithErrors(input, dst)
}

// ConvertValidUTF8ToLatin1 converts valid UTF-8 input that is representable as
// Latin-1. It panics before writing when dst is shorter than
// Latin1LengthFromUTF8(input).
func ConvertValidUTF8ToLatin1(input, dst []byte) int {
	return activeImplementation.convertValidUTF8ToLatin1(input, dst)
}

// ConvertUTF8ToUTF16LE converts UTF-8 input to little-endian raw UTF-16
// storage. It returns 0 on invalid UTF-8. It panics before writing when dst is
// shorter than UTF16LengthFromUTF8(input).
func ConvertUTF8ToUTF16LE(input []byte, dst []uint16) int {
	return activeImplementation.convertUTF8ToUTF16LE(input, dst)
}

// ConvertUTF8ToUTF16BE converts UTF-8 input to big-endian raw UTF-16 storage.
// It returns 0 on invalid UTF-8. It panics before writing when dst is shorter
// than UTF16LengthFromUTF8(input).
func ConvertUTF8ToUTF16BE(input []byte, dst []uint16) int {
	return activeImplementation.convertUTF8ToUTF16BE(input, dst)
}

// ConvertUTF8ToUTF16 converts UTF-8 input to host-native UTF-16 storage.
func ConvertUTF8ToUTF16(input []byte, dst []uint16) int {
	if nativeLittleEndian() {
		return ConvertUTF8ToUTF16LE(input, dst)
	}
	return ConvertUTF8ToUTF16BE(input, dst)
}

// ConvertUTF8ToUTF16LEWithErrors converts UTF-8 input to little-endian raw
// UTF-16 storage. On success Count is the number of UTF-16 code units written;
// on failure Count is the input error position.
func ConvertUTF8ToUTF16LEWithErrors(input []byte, dst []uint16) Result {
	return activeImplementation.convertUTF8ToUTF16LEWithErrors(input, dst)
}

// ConvertUTF8ToUTF16BEWithErrors converts UTF-8 input to big-endian raw UTF-16
// storage. On success Count is the number of UTF-16 code units written; on
// failure Count is the input error position.
func ConvertUTF8ToUTF16BEWithErrors(input []byte, dst []uint16) Result {
	return activeImplementation.convertUTF8ToUTF16BEWithErrors(input, dst)
}

// ConvertUTF8ToUTF16WithErrors converts UTF-8 input to host-native UTF-16
// storage.
func ConvertUTF8ToUTF16WithErrors(input []byte, dst []uint16) Result {
	if nativeLittleEndian() {
		return ConvertUTF8ToUTF16LEWithErrors(input, dst)
	}
	return ConvertUTF8ToUTF16BEWithErrors(input, dst)
}

// ConvertValidUTF8ToUTF16LE converts valid UTF-8 input to little-endian raw
// UTF-16 storage. It panics before writing when dst is shorter than
// UTF16LengthFromUTF8(input).
func ConvertValidUTF8ToUTF16LE(input []byte, dst []uint16) int {
	return activeImplementation.convertValidUTF8ToUTF16LE(input, dst)
}

// ConvertValidUTF8ToUTF16BE converts valid UTF-8 input to big-endian raw UTF-16
// storage. It panics before writing when dst is shorter than
// UTF16LengthFromUTF8(input).
func ConvertValidUTF8ToUTF16BE(input []byte, dst []uint16) int {
	return activeImplementation.convertValidUTF8ToUTF16BE(input, dst)
}

// ConvertValidUTF8ToUTF16 converts valid UTF-8 input to host-native UTF-16
// storage.
func ConvertValidUTF8ToUTF16(input []byte, dst []uint16) int {
	if nativeLittleEndian() {
		return ConvertValidUTF8ToUTF16LE(input, dst)
	}
	return ConvertValidUTF8ToUTF16BE(input, dst)
}

// ConvertUTF8ToUTF32 converts UTF-8 input to UTF-32. It returns 0 on invalid
// UTF-8. It panics before writing when dst is shorter than
// UTF32LengthFromUTF8(input).
func ConvertUTF8ToUTF32(input []byte, dst []uint32) int {
	return activeImplementation.convertUTF8ToUTF32(input, dst)
}

// ConvertUTF8ToUTF32WithErrors converts UTF-8 input to UTF-32. On success Count
// is the number of UTF-32 code units written; on failure Count is the input
// error position.
func ConvertUTF8ToUTF32WithErrors(input []byte, dst []uint32) Result {
	return activeImplementation.convertUTF8ToUTF32WithErrors(input, dst)
}

// ConvertValidUTF8ToUTF32 converts valid UTF-8 input to UTF-32. It panics before
// writing when dst is shorter than UTF32LengthFromUTF8(input).
func ConvertValidUTF8ToUTF32(input []byte, dst []uint32) int {
	return activeImplementation.convertValidUTF8ToUTF32(input, dst)
}

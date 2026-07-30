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
// include/simdutf/implementation.h:830-995,1169,1652,3380. Go slices replace
// C++ pointer/length pairs. UTF-16 endian names describe raw []uint16 storage.

// UTF32LengthFromLatin1 returns the number of UTF-32 code units needed for a
// Latin-1 input of length bytes.
func UTF32LengthFromLatin1(length int) int {
	return length
}

// ConvertLatin1ToUTF8 converts Latin-1 input to UTF-8. It panics before
// writing when dst is shorter than UTF8LengthFromLatin1(input).
func ConvertLatin1ToUTF8(input, dst []byte) int {
	return activeImplementation.convertLatin1ToUTF8(input, dst)
}

// ConvertLatin1ToUTF8Safe converts complete Latin-1 code points which fit in
// dst and returns the number of bytes written.
func ConvertLatin1ToUTF8Safe(input, dst []byte) int {
	return convertLatin1ToUTF8SafeScalar(input, dst)
}

// ConvertLatin1ToUTF32 converts Latin-1 input to UTF-32. It panics before
// writing when dst is shorter than len(input).
func ConvertLatin1ToUTF32(input []byte, dst []uint32) int {
	return activeImplementation.convertLatin1ToUTF32(input, dst)
}

// ConvertLatin1ToUTF16LE converts Latin-1 input to little-endian raw UTF-16
// storage. It panics before writing when dst is shorter than len(input).
func ConvertLatin1ToUTF16LE(input []byte, dst []uint16) int {
	return activeImplementation.convertLatin1ToUTF16LE(input, dst)
}

// ConvertLatin1ToUTF16BE converts Latin-1 input to big-endian raw UTF-16
// storage. It panics before writing when dst is shorter than len(input).
func ConvertLatin1ToUTF16BE(input []byte, dst []uint16) int {
	return activeImplementation.convertLatin1ToUTF16BE(input, dst)
}

// ConvertLatin1ToUTF16 converts Latin-1 input to host-native UTF-16 storage.
func ConvertLatin1ToUTF16(input []byte, dst []uint16) int {
	if nativeLittleEndian() {
		return ConvertLatin1ToUTF16LE(input, dst)
	}
	return ConvertLatin1ToUTF16BE(input, dst)
}

// Latin1LengthFromUTF16 returns the number of Latin-1 bytes needed for a
// UTF-16 input of length code units. It is an identity mapping.
func Latin1LengthFromUTF16(length int) int {
	return latin1LengthFromUTF16Scalar(length)
}

// UTF16LengthFromLatin1 returns the number of UTF-16 code units needed for a
// Latin-1 input of length bytes.
func UTF16LengthFromLatin1(length int) int {
	return length
}

// UTF8LengthFromLatin1 returns the number of UTF-8 bytes needed to encode
// input as Latin-1.
func UTF8LengthFromLatin1(input []byte) int {
	return activeImplementation.utf8LengthFromLatin1(input)
}

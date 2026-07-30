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
// include/simdutf/implementation.h:477-731. Endian names describe the raw
// []uint16 storage encoding.

// ValidateUTF16 reports whether host-native UTF-16 input is well formed.
func ValidateUTF16(input []uint16) bool {
	if nativeLittleEndian() {
		return ValidateUTF16LE(input)
	}
	return ValidateUTF16BE(input)
}

// ValidateUTF16LE reports whether little-endian raw UTF-16 storage is well formed.
func ValidateUTF16LE(input []uint16) bool {
	return activeImplementation.validateUTF16LE(input)
}

// ValidateUTF16BE reports whether big-endian raw UTF-16 storage is well formed.
func ValidateUTF16BE(input []uint16) bool {
	return activeImplementation.validateUTF16BE(input)
}

// ValidateUTF16WithErrors validates host-native UTF-16 and returns the first
// invalid code-unit index, or the input length on success.
func ValidateUTF16WithErrors(input []uint16) Result {
	if nativeLittleEndian() {
		return ValidateUTF16LEWithErrors(input)
	}
	return ValidateUTF16BEWithErrors(input)
}

// ValidateUTF16LEWithErrors validates little-endian raw UTF-16 storage.
func ValidateUTF16LEWithErrors(input []uint16) Result {
	return activeImplementation.validateUTF16LEWithErrors(input)
}

// ValidateUTF16BEWithErrors validates big-endian raw UTF-16 storage.
func ValidateUTF16BEWithErrors(input []uint16) Result {
	return activeImplementation.validateUTF16BEWithErrors(input)
}

// ToWellFormedUTF16 replaces unpaired surrogates in host-native UTF-16 with
// U+FFFD. Input and dst may be identical. It panics before writing when dst is
// shorter than input; other overlap is not supported.
func ToWellFormedUTF16(input, dst []uint16) {
	if nativeLittleEndian() {
		ToWellFormedUTF16LE(input, dst)
		return
	}
	ToWellFormedUTF16BE(input, dst)
}

// ToWellFormedUTF16LE repairs little-endian raw UTF-16 storage.
func ToWellFormedUTF16LE(input, dst []uint16) {
	if len(dst) < len(input) {
		panic("simdutf: UTF-16 destination too short")
	}
	activeImplementation.toWellFormedUTF16LE(input, dst)
}

// ToWellFormedUTF16BE repairs big-endian raw UTF-16 storage.
func ToWellFormedUTF16BE(input, dst []uint16) {
	if len(dst) < len(input) {
		panic("simdutf: UTF-16 destination too short")
	}
	activeImplementation.toWellFormedUTF16BE(input, dst)
}

// TrimPartialUTF16 returns the length of the longest host-native UTF-16 prefix
// that excludes a final truncated high surrogate.
func TrimPartialUTF16(input []uint16) int {
	if nativeLittleEndian() {
		return TrimPartialUTF16LE(input)
	}
	return TrimPartialUTF16BE(input)
}

// ConvertValidUTF16ToUTF8 converts valid host-native UTF-16 storage to UTF-8.
// It panics before writing when dst is shorter than the UTF-8 length required
// for input.
func ConvertValidUTF16ToUTF8(input []uint16, dst []byte) int {
	if nativeLittleEndian() {
		return ConvertValidUTF16LEToUTF8(input, dst)
	}
	return ConvertValidUTF16BEToUTF8(input, dst)
}

// UTF32LengthFromUTF16 returns the number of UTF-32 code units required to
// encode host-native UTF-16 storage.
func UTF32LengthFromUTF16(input []uint16) int {
	if nativeLittleEndian() {
		return UTF32LengthFromUTF16LE(input)
	}
	return UTF32LengthFromUTF16BE(input)
}

// UTF8LengthFromUTF16 returns the number of UTF-8 bytes required to encode
// host-native UTF-16 storage.
func UTF8LengthFromUTF16(input []uint16) int {
	if nativeLittleEndian() {
		return UTF8LengthFromUTF16LE(input)
	}
	return UTF8LengthFromUTF16BE(input)
}

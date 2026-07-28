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

import "encoding/binary"

// Public API adapted from simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
// include/simdutf/implementation.h:315-455. Go slices replace C++ pointer/length
// pairs, and UTF-16 endian names describe the raw []uint16 storage encoding.

// ValidateASCII reports whether every input byte is less than 0x80.
func ValidateASCII(input []byte) bool {
	return activeImplementation.validateASCII(input)
}

// ValidateASCIIWithErrors returns Success with Count equal to the input byte
// length, or TooLarge with Count equal to the first invalid byte index.
func ValidateASCIIWithErrors(input []byte) Result {
	return activeImplementation.validateASCIIWithErrors(input)
}

// ValidateUTF16AsASCII reports whether host-native raw []uint16 storage contains
// only UTF-16 code units less than 0x80.
func ValidateUTF16AsASCII(input []uint16) bool {
	if nativeLittleEndian() {
		return ValidateUTF16LEAsASCII(input)
	}
	return ValidateUTF16BEAsASCII(input)
}

// ValidateUTF16BEAsASCII reports whether big-endian raw []uint16 memory contains
// only UTF-16 code units less than 0x80.
func ValidateUTF16BEAsASCII(input []uint16) bool {
	return activeImplementation.validateUTF16BEAsASCII(input)
}

// ValidateUTF16LEAsASCII reports whether little-endian raw []uint16 memory
// contains only UTF-16 code units less than 0x80.
func ValidateUTF16LEAsASCII(input []uint16) bool {
	return activeImplementation.validateUTF16LEAsASCII(input)
}

func nativeLittleEndian() bool {
	return binary.NativeEndian.Uint16([]byte{1, 0}) == 1
}

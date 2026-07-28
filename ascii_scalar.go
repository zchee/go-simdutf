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

import (
	"encoding/binary"
	"math/bits"
)

// Translated and adapted from simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
// include/simdutf/scalar/ascii.h:15-81, include/simdutf/scalar/utf16.h:8-18,
// and src/fallback/implementation.cpp:49-73. Go uses bounds-checked loads and
// raw []uint16 storage for explicitly named UTF-16 endianness.

func validateASCIIScalar(input []byte) bool {
	pos := 0
	for ; pos+16 <= len(input); pos += 16 {
		v1 := binary.LittleEndian.Uint64(input[pos:])
		v2 := binary.LittleEndian.Uint64(input[pos+8:])
		if (v1|v2)&0x8080808080808080 != 0 {
			return false
		}
	}
	for ; pos < len(input); pos++ {
		if input[pos] >= 0x80 {
			return false
		}
	}
	return true
}

func validateASCIIWithErrorsScalar(input []byte) Result {
	pos := 0
	for ; pos+16 <= len(input); pos += 16 {
		v1 := binary.LittleEndian.Uint64(input[pos:])
		v2 := binary.LittleEndian.Uint64(input[pos+8:])
		if (v1|v2)&0x8080808080808080 != 0 {
			for ; pos < len(input); pos++ {
				if input[pos] >= 0x80 {
					return Result{Error: TooLarge, Count: pos}
				}
			}
		}
	}
	for ; pos < len(input); pos++ {
		if input[pos] >= 0x80 {
			return Result{Error: TooLarge, Count: pos}
		}
	}
	return Result{Error: Success, Count: pos}
}

func validateUTF16LEAsASCIIScalar(input []uint16) bool {
	return validateUTF16AsASCIIScalar(input, nativeLittleEndian())
}

func validateUTF16BEAsASCIIScalar(input []uint16) bool {
	return validateUTF16AsASCIIScalar(input, !nativeLittleEndian())
}

func validateUTF16AsASCIIScalar(input []uint16, storageIsNative bool) bool {
	for _, raw := range input {
		word := raw
		if !storageIsNative {
			word = bits.ReverseBytes16(word)
		}
		if word >= 0x80 {
			return false
		}
	}
	return true
}

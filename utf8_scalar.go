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

import "encoding/binary"

// Translated and adapted from simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// include/simdutf/scalar/utf8.h:9-218,258-268 and
// src/fallback/implementation.cpp:35-48,431-433.
// credit: based on code from Google Fuchsia (Apache Licensed)
// Go uses bounds-checked loads.

func validateUTF8Scalar(input []byte) bool {
	pos := 0
	for pos < len(input) {
		next := pos + 16
		if next <= len(input) {
			v1 := binary.LittleEndian.Uint64(input[pos:])
			v2 := binary.LittleEndian.Uint64(input[pos+8:])
			if (v1|v2)&0x8080808080808080 == 0 {
				pos = next
				continue
			}
		}

		b := input[pos]
		for b < 0x80 {
			pos++
			if pos == len(input) {
				return true
			}
			b = input[pos]
		}

		switch {
		case b&0xe0 == 0xc0:
			next = pos + 2
			if next > len(input) || input[pos+1]&0xc0 != 0x80 {
				return false
			}
			codePoint := uint32(b&0x1f)<<6 | uint32(input[pos+1]&0x3f)
			if codePoint < 0x80 {
				return false
			}
		case b&0xf0 == 0xe0:
			next = pos + 3
			if next > len(input) || input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 {
				return false
			}
			codePoint := uint32(b&0x0f)<<12 | uint32(input[pos+1]&0x3f)<<6 | uint32(input[pos+2]&0x3f)
			if codePoint < 0x800 || codePoint > 0xd7ff && codePoint < 0xe000 {
				return false
			}
		case b&0xf8 == 0xf0:
			next = pos + 4
			if next > len(input) || input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 || input[pos+3]&0xc0 != 0x80 {
				return false
			}
			codePoint := uint32(b&0x07)<<18 | uint32(input[pos+1]&0x3f)<<12 | uint32(input[pos+2]&0x3f)<<6 | uint32(input[pos+3]&0x3f)
			if codePoint <= 0xffff || codePoint > 0x10ffff {
				return false
			}
		default:
			return false
		}
		pos = next
	}
	return true
}

func validateUTF8WithErrorsScalar(input []byte) Result {
	pos := 0
	for pos < len(input) {
		next := pos + 16
		if next <= len(input) {
			v1 := binary.LittleEndian.Uint64(input[pos:])
			v2 := binary.LittleEndian.Uint64(input[pos+8:])
			if (v1|v2)&0x8080808080808080 == 0 {
				pos = next
				continue
			}
		}

		b := input[pos]
		for b < 0x80 {
			pos++
			if pos == len(input) {
				return Result{Error: Success, Count: len(input)}
			}
			b = input[pos]
		}

		switch {
		case b&0xe0 == 0xc0:
			next = pos + 2
			if next > len(input) || input[pos+1]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(b&0x1f)<<6 | uint32(input[pos+1]&0x3f)
			if codePoint < 0x80 {
				return Result{Error: Overlong, Count: pos}
			}
		case b&0xf0 == 0xe0:
			next = pos + 3
			if next > len(input) || input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(b&0x0f)<<12 | uint32(input[pos+1]&0x3f)<<6 | uint32(input[pos+2]&0x3f)
			if codePoint < 0x800 {
				return Result{Error: Overlong, Count: pos}
			}
			if codePoint > 0xd7ff && codePoint < 0xe000 {
				return Result{Error: Surrogate, Count: pos}
			}
		case b&0xf8 == 0xf0:
			next = pos + 4
			if next > len(input) || input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 || input[pos+3]&0xc0 != 0x80 {
				return Result{Error: TooShort, Count: pos}
			}
			codePoint := uint32(b&0x07)<<18 | uint32(input[pos+1]&0x3f)<<12 | uint32(input[pos+2]&0x3f)<<6 | uint32(input[pos+3]&0x3f)
			if codePoint <= 0xffff {
				return Result{Error: Overlong, Count: pos}
			}
			if codePoint > 0x10ffff {
				return Result{Error: TooLarge, Count: pos}
			}
		default:
			if b&0xc0 == 0x80 {
				return Result{Error: TooLong, Count: pos}
			}
			return Result{Error: HeaderBits, Count: pos}
		}
		pos = next
	}
	return Result{Error: Success, Count: len(input)}
}

func countUTF8Scalar(input []byte) int {
	count := 0
	for _, value := range input {
		if int8(value) > -65 {
			count++
		}
	}
	return count
}

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

// Translated and adapted from simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/ascii_validation.h:6-45, src/arm64/implementation.cpp:13-16,
// src/arm64/implementation.cpp:253-298, and
// src/arm64/arm_validate_utf16.cpp:71-91. The assembly routines inspect only
// complete vectors; Go handles the scalar tail and exact error position.

//go:noescape
func validateASCIIPrefixNEON(input []byte) int

//go:noescape
func validateUTF16LEASCIIPrefixNEON(input []uint16) int

//go:noescape
func validateUTF16BEASCIIPrefixNEON(input []uint16) int

func validateASCIINEON(input []byte) bool {
	prefix := validateASCIIPrefixNEON(input)
	if prefix < len(input)&^63 {
		return false
	}
	return validateASCIIScalar(input[prefix:])
}

func validateASCIIWithErrorsNEON(input []byte) Result {
	prefix := validateASCIIPrefixNEON(input)
	result := validateASCIIWithErrorsScalar(input[prefix:])
	result.Count += prefix
	return result
}

func validateUTF16LEAsASCIINEON(input []uint16) bool {
	prefix := validateUTF16LEASCIIPrefixNEON(input)
	if prefix < len(input)&^15 {
		return false
	}
	return validateUTF16LEAsASCIIScalar(input[prefix:])
}

func validateUTF16BEAsASCIINEON(input []uint16) bool {
	prefix := validateUTF16BEASCIIPrefixNEON(input)
	if prefix < len(input)&^15 {
		return false
	}
	return validateUTF16BEAsASCIIScalar(input[prefix:])
}

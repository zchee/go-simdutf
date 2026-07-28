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

//go:build amd64

package simdutf

// Independently translated to Go assembly from
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811):
// src/generic/ascii_validation.h:6-45, src/generic/validate_utf16.h:128-158,
// src/simdutf/westmere/simd.h:168-170,290-297, and
// src/simdutf/haswell/simd.h:177-179,293-300. The assembly routines examine
// complete blocks only; these wrappers use the scalar oracle for the returned
// block or tail.

//go:noescape
func validateASCIIPrefixWestmere(input []byte) int

//go:noescape
func validateASCIIPrefixHaswell(input []byte) int

//go:noescape
func validateUTF16LEASCIIPrefixWestmere(input []uint16) int

//go:noescape
func validateUTF16BEASCIIPrefixWestmere(input []uint16) int

//go:noescape
func validateUTF16LEASCIIPrefixHaswell(input []uint16) int

//go:noescape
func validateUTF16BEASCIIPrefixHaswell(input []uint16) int

func validateASCIIWestmere(input []byte) bool {
	if len(input) < 64 {
		return validateASCIIScalar(input)
	}
	prefix := validateASCIIPrefixWestmere(input)
	if prefix != len(input)&^63 {
		return false
	}
	return validateASCIIScalar(input[prefix:])
}

func validateASCIIWithErrorsWestmere(input []byte) Result {
	if len(input) < 64 {
		return validateASCIIWithErrorsScalar(input)
	}
	prefix := validateASCIIPrefixWestmere(input)
	result := validateASCIIWithErrorsScalar(input[prefix:])
	result.Count += prefix
	return result
}

func validateASCIIHaswell(input []byte) bool {
	if len(input) < 64 {
		return validateASCIIScalar(input)
	}
	prefix := validateASCIIPrefixHaswell(input)
	if prefix != len(input)&^63 {
		return false
	}
	return validateASCIIScalar(input[prefix:])
}

func validateASCIIWithErrorsHaswell(input []byte) Result {
	if len(input) < 64 {
		return validateASCIIWithErrorsScalar(input)
	}
	prefix := validateASCIIPrefixHaswell(input)
	result := validateASCIIWithErrorsScalar(input[prefix:])
	result.Count += prefix
	return result
}

func validateUTF16LEAsASCIIWestmere(input []uint16) bool {
	prefix := validateUTF16LEASCIIPrefixWestmere(input)
	if prefix != len(input)&^31 {
		return false
	}
	return validateUTF16LEAsASCIIScalar(input[prefix:])
}

func validateUTF16BEAsASCIIWestmere(input []uint16) bool {
	prefix := validateUTF16BEASCIIPrefixWestmere(input)
	if prefix != len(input)&^31 {
		return false
	}
	return validateUTF16BEAsASCIIScalar(input[prefix:])
}

func validateUTF16LEAsASCIIHaswell(input []uint16) bool {
	prefix := validateUTF16LEASCIIPrefixHaswell(input)
	if prefix != len(input)&^31 {
		return false
	}
	return validateUTF16LEAsASCIIScalar(input[prefix:])
}

func validateUTF16BEAsASCIIHaswell(input []uint16) bool {
	prefix := validateUTF16BEASCIIPrefixHaswell(input)
	if prefix != len(input)&^31 {
		return false
	}
	return validateUTF16BEAsASCIIScalar(input[prefix:])
}

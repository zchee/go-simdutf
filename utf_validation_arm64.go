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
//go:build arm64

package simdutf

// Translated and adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/arm64/arm_validate_utf16.cpp and src/arm64/arm_validate_utf32.cpp.
// The NEON routines skip complete blocks which cannot contain an invalid code
// unit. Go resumes at the first interesting block so scalar code supplies the
// required cross-block surrogate and exact-error semantics.

//go:noescape
func validateUTF16LEPrefixNEON(input []uint16) int

//go:noescape
func validateUTF16BEPrefixNEON(input []uint16) int

//go:noescape
func validateUTF32PrefixNEON(input []uint32) int

func validateUTF16LENEON(input []uint16) bool {
	if len(input) < 16 {
		return validateUTF16LEScalar(input)
	}
	prefix := validateUTF16LEPrefixNEON(input)
	return validateUTF16LEScalar(input[prefix:])
}

func validateUTF16BENEON(input []uint16) bool {
	if len(input) < 16 {
		return validateUTF16BEScalar(input)
	}
	prefix := validateUTF16BEPrefixNEON(input)
	return validateUTF16BEScalar(input[prefix:])
}

func validateUTF16LEWithErrorsNEON(input []uint16) Result {
	if len(input) < 16 {
		return validateUTF16LEWithErrorsScalar(input)
	}
	prefix := validateUTF16LEPrefixNEON(input)
	result := validateUTF16LEWithErrorsScalar(input[prefix:])
	result.Count += prefix
	return result
}

func validateUTF16BEWithErrorsNEON(input []uint16) Result {
	if len(input) < 16 {
		return validateUTF16BEWithErrorsScalar(input)
	}
	prefix := validateUTF16BEPrefixNEON(input)
	result := validateUTF16BEWithErrorsScalar(input[prefix:])
	result.Count += prefix
	return result
}

// toWellFormedUTF16LENEON repairs little-endian raw UTF-16 storage. dst must
// have at least len(input) elements. Input and dst may be identical; other
// overlap is not supported.
func toWellFormedUTF16LENEON(input, dst []uint16) {
	if len(dst) < len(input) {
		panic("simdutf: UTF-16 destination too short")
	}
	if len(input) < 16 {
		toWellFormedUTF16LEScalar(input, dst)
		return
	}
	// The vector pass is deliberately read-only: stopping at the first block
	// containing a surrogate lets the scalar oracle preserve a pair spanning a
	// vector boundary, including when input and dst are the same slice.
	prefix := validateUTF16LEPrefixNEON(input)
	copy(dst[:prefix], input[:prefix])
	toWellFormedUTF16LEScalar(input[prefix:], dst[prefix:])
}

// toWellFormedUTF16BENEON repairs big-endian raw UTF-16 storage. dst must
// have at least len(input) elements. Input and dst may be identical; other
// overlap is not supported.
func toWellFormedUTF16BENEON(input, dst []uint16) {
	if len(dst) < len(input) {
		panic("simdutf: UTF-16 destination too short")
	}
	if len(input) < 16 {
		toWellFormedUTF16BEScalar(input, dst)
		return
	}
	prefix := validateUTF16BEPrefixNEON(input)
	copy(dst[:prefix], input[:prefix])
	toWellFormedUTF16BEScalar(input[prefix:], dst[prefix:])
}

func validateUTF32NEON(input []uint32) bool {
	if len(input) < 4 {
		return validateUTF32Scalar(input)
	}
	prefix := validateUTF32PrefixNEON(input)
	return validateUTF32Scalar(input[prefix:])
}

func validateUTF32WithErrorsNEON(input []uint32) Result {
	if len(input) < 4 {
		return validateUTF32WithErrorsScalar(input)
	}
	prefix := validateUTF32PrefixNEON(input)
	result := validateUTF32WithErrorsScalar(input[prefix:])
	result.Count += prefix
	return result
}

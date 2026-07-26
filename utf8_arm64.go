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

// Translated and adapted from simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/arm64/implementation.cpp:13-28,
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216, and
// src/generic/utf8_validation/utf8_validator.h:10-80, and
// include/simdutf/scalar/utf8.h:225-251.

//go:noescape
func validateUTF8Lookup4NEON(input []byte, remainder *[64]byte) (count int, hasError uint64)

// The pinned lookup4 validator consumes 64-byte blocks. Measurements on the
// required arm64 host show that smaller inputs can regress the scalar oracle,
// so production dispatch enters NEON only at one complete lookup4 block.
const validateUTF8NEONMinInputSize = 64

func validateUTF8NEON(input []byte) bool {
	if len(input) < validateUTF8NEONMinInputSize {
		return validateUTF8Scalar(input)
	}
	var remainder [64]byte
	copy(remainder[:], input[len(input)&^63:])
	_, hasError := validateUTF8Lookup4NEON(input, &remainder)
	return hasError == 0
}

func validateUTF8WithErrorsNEON(input []byte) Result {
	if len(input) < validateUTF8NEONMinInputSize {
		return validateUTF8WithErrorsScalar(input)
	}
	var remainder [64]byte
	copy(remainder[:], input[len(input)&^63:])
	count, hasError := validateUTF8Lookup4NEON(input, &remainder)
	if hasError == 0 {
		return Result{Error: Success, Count: len(input)}
	}
	return validateUTF8RewindWithErrorsScalar(input, count)
}

// validateUTF8RewindWithErrorsScalar is a direct Go adaptation of the pinned
// scalar::utf8::rewind_and_validate_with_errors helper. The lookup4 checker can
// observe a malformed sequence in the block after its leading byte, so rewind
// from the byte immediately before the reported block and through at most five
// continuation bytes before invoking the scalar oracle.
func validateUTF8RewindWithErrorsScalar(input []byte, count int) Result {
	if count != 0 {
		count--
	}
	start := count
	for range 5 {
		if input[start]&0xc0 != 0x80 {
			break
		}
		if start == 0 {
			break
		}
		start--
	}
	result := validateUTF8WithErrorsScalar(input[start:])
	result.Count += start
	return result
}

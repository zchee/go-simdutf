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
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216,
// src/generic/utf8_validation/utf8_validator.h:10-80,
// src/westmere/implementation.cpp:19-29, and
// src/haswell/implementation.cpp:19-29. The assembly routines examine
// complete 64-byte blocks only. As in the pinned generic driver, these wrappers
// rewind before the first block that reports an error, or before the scalar
// tail when all complete blocks pass, and use the scalar oracle for exact
// public error identity and Count relocation. Rewinding also makes the scalar
// tail observe incomplete sequences at a full-block boundary, so a clean
// lookup4 prefix cannot mask a truncated or invalid final sequence. The rewind
// is bounded to the pinned driver's at most five continuation bytes. The
// Haswell wrapper requires two complete blocks before entering assembly because
// the one-block class regresses against the scalar oracle on the required amd64
// host; lookup4 itself still consumes the pinned 64-byte block shape.

//go:noescape
func validateUTF8PrefixWestmere(input []byte) int

//go:noescape
func validateUTF8PrefixHaswell(input []byte) int

func validateUTF8Westmere(input []byte) bool {
	if len(input) < 64 {
		return validateUTF8Scalar(input)
	}
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixWestmere(input)).Error == Success
}

func validateUTF8WithErrorsWestmere(input []byte) Result {
	if len(input) < 64 {
		return validateUTF8WithErrorsScalar(input)
	}
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixWestmere(input))
}

func validateUTF8Haswell(input []byte) bool {
	if len(input) < 128 {
		return validateUTF8Scalar(input)
	}
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixHaswell(input)).Error == Success
}

func validateUTF8WithErrorsHaswell(input []byte) Result {
	if len(input) < 128 {
		return validateUTF8WithErrorsScalar(input)
	}
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixHaswell(input))
}

func validateUTF8AMD64FromPrefix(input []byte, boundary int) Result {
	if boundary == 0 {
		return validateUTF8WithErrorsScalar(input)
	}
	start := boundary - 1
	for range 5 {
		if input[start]&0xc0 != 0x80 || start == 0 {
			break
		}
		start--
	}
	result := validateUTF8WithErrorsScalar(input[start:])
	result.Count += start
	return result
}

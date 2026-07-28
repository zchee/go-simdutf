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
// include/simdutf/implementation.h:253-306,3931-3938. Go slices replace C++
// pointer/length pairs.

// ValidateUTF8 reports whether input is valid UTF-8.
func ValidateUTF8(input []byte) bool {
	return activeImplementation.validateUTF8(input)
}

// ValidateUTF8WithErrors returns Success with Count equal to the input byte
// length, or the exact validation error with Count equal to its byte position.
func ValidateUTF8WithErrors(input []byte) Result {
	return activeImplementation.validateUTF8WithErrors(input)
}

// CountUTF8 returns the number of Unicode scalar values in valid UTF-8 input.
// Like upstream simdutf, it assumes that input is valid UTF-8 and does not
// validate it.
func CountUTF8(input []byte) int {
	return activeImplementation.countUTF8(input)
}

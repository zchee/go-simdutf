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
// include/simdutf/implementation.h:766-829.

// ValidateUTF32 reports whether every input word is a Unicode scalar value.
func ValidateUTF32(input []uint32) bool {
	return activeImplementation.validateUTF32(input)
}

// ValidateUTF32WithErrors returns the first invalid word index, or the input
// length on success. Values above U+10FFFF report TooLarge; surrogate code
// points report Surrogate.
func ValidateUTF32WithErrors(input []uint32) Result {
	return activeImplementation.validateUTF32WithErrors(input)
}

// Latin1LengthFromUTF32 returns the number of Latin-1 bytes needed for a
// UTF-32 input of length code units. It is an identity mapping.
func Latin1LengthFromUTF32(length int) int {
	return latin1LengthFromUTF32Scalar(length)
}

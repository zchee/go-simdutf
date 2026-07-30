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
// include/simdutf/implementation.h:4158-4184. The pinned pointer return is
// translated to a zero-based index, or len(input) when the value is absent.
// Nil slices are empty input.

// Find returns the index of the first byte equal to value, or len(input).
func Find(input []byte, value byte) int {
	return activeImplementation.find(input, value)
}

// FindUTF16 returns the index of the first UTF-16 code unit equal to value,
// or len(input).
func FindUTF16(input []uint16, value uint16) int {
	return activeImplementation.findUTF16(input, value)
}

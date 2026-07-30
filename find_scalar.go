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

// Translated and adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/fallback/implementation.cpp:575-593. The public Go APIs expose the first
// match as a zero-based index, or len(input) when absent.

func findScalar(input []byte, value byte) int {
	for i, b := range input {
		if b == value {
			return i
		}
	}
	return len(input)
}

func findUTF16Scalar(input []uint16, value uint16) int {
	for i, unit := range input {
		if unit == value {
			return i
		}
	}
	return len(input)
}

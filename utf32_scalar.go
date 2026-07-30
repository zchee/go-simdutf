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
// include/simdutf/scalar/utf32.h:8-50.

func validateUTF32Scalar(input []uint32) bool {
	for _, word := range input {
		if word > 0x10ffff || word >= 0xd800 && word <= 0xdfff {
			return false
		}
	}
	return true
}

func validateUTF32WithErrorsScalar(input []uint32) Result {
	for pos, word := range input {
		if word > 0x10ffff {
			return Result{Error: TooLarge, Count: pos}
		}
		if word >= 0xd800 && word <= 0xdfff {
			return Result{Error: Surrogate, Count: pos}
		}
	}
	return Result{Error: Success, Count: len(input)}
}

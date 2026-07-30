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

// Translated and adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/utf16.h,
// src/generic/utf16/change_endianness.h, src/generic/utf16/count_code_points_bytemask.h,
// src/generic/utf16/utf32_length_from_utf16.h, and src/simdutf/arm64/simd16-inl.h.
// Assembly consumes only complete 32-uint16 NEON groups; wrappers retain scalar tails.

//go:noescape
func changeEndiannessUTF16BlocksNEON(input, dst []uint16) (consumed int)

//go:noescape
func countUTF16LEBlocksNEON(input []uint16) (count int)

//go:noescape
func countUTF16BEBlocksNEON(input []uint16) (count int)

func changeEndiannessUTF16NEON(input, dst []uint16) {
	if len(dst) < len(input) {
		panic("simdutf: UTF-16 destination too short")
	}
	if len(input) == 0 {
		return
	}
	consumed := changeEndiannessUTF16BlocksNEON(input, dst)
	if consumed < len(input) {
		changeEndiannessUTF16Scalar(input[consumed:], dst[consumed:])
	}
}

func countUTF16LENEON(input []uint16) int {
	complete := len(input) &^ 31
	if complete == 0 {
		return countUTF16LEScalar(input)
	}
	return countUTF16LEBlocksNEON(input[:complete]) + countUTF16LEScalar(input[complete:])
}

func countUTF16BENEON(input []uint16) int {
	complete := len(input) &^ 31
	if complete == 0 {
		return countUTF16BEScalar(input)
	}
	return countUTF16BEBlocksNEON(input[:complete]) + countUTF16BEScalar(input[complete:])
}

func utf32LengthFromUTF16LENEON(input []uint16) int {
	return countUTF16LENEON(input)
}

func utf32LengthFromUTF16BENEON(input []uint16) int {
	return countUTF16BENEON(input)
}

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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf16_to_utf32.cpp
// and src/arm64/implementation.cpp. Assembly consumes only complete in-bounds
// 8-uint16 NEON blocks with no surrogates, widening each u16 to u32; on the
// first surrogate-containing vector it returns the consumed prefix and Go
// wrappers remount the scalar oracle for the remainder (and for error paths).

//go:noescape
func convertUTF16LEToUTF32BlocksNEON(input []uint16, dst []uint32) (consumed int)

//go:noescape
func convertUTF16BEToUTF32BlocksNEON(input []uint16, dst []uint32) (consumed int)

func convertUTF16LEToUTF32NEON(input []uint16, dst []uint32) int {
	if len(dst) < utf32LengthFromUTF16LEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16LEToUTF32Scalar(input, dst)
	}
	n := convertUTF16LEToUTF32BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF16LEToUTF32Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF16BEToUTF32NEON(input []uint16, dst []uint32) int {
	if len(dst) < utf32LengthFromUTF16BEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16BEToUTF32Scalar(input, dst)
	}
	n := convertUTF16BEToUTF32BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF16BEToUTF32Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF16LEToUTF32WithErrorsNEON(input []uint16, dst []uint32) Result {
	if len(dst) < utf32LengthFromUTF16LEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16LEToUTF32WithErrorsScalar(input, dst)
	}
	n := convertUTF16LEToUTF32BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF16LEToUTF32WithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertUTF16BEToUTF32WithErrorsNEON(input []uint16, dst []uint32) Result {
	if len(dst) < utf32LengthFromUTF16BEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16BEToUTF32WithErrorsScalar(input, dst)
	}
	n := convertUTF16BEToUTF32BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF16BEToUTF32WithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertValidUTF16LEToUTF32NEON(input []uint16, dst []uint32) int {
	if len(dst) < utf32LengthFromUTF16LEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF16LEToUTF32Scalar(input, dst)
	}
	n := convertUTF16LEToUTF32BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF16LEToUTF32Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertValidUTF16BEToUTF32NEON(input []uint16, dst []uint32) int {
	if len(dst) < utf32LengthFromUTF16BEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF16BEToUTF32Scalar(input, dst)
	}
	n := convertUTF16BEToUTF32BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF16BEToUTF32Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

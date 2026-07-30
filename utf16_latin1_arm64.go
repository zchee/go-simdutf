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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf16_to_latin1.cpp
// and src/arm64/implementation.cpp. Assembly consumes only complete in-bounds
// 8-uint16 NEON blocks and stops before the first non-Latin-1 vector; wrappers
// retain the scalar preflight and tails required by the Go slice contract.

//go:noescape
func convertUTF16LEToLatin1BlocksNEON(input []uint16, dst []byte) (consumed int)

//go:noescape
func convertUTF16BEToLatin1BlocksNEON(input []uint16, dst []byte) (consumed int)

func convertUTF16LEToLatin1NEON(input []uint16, dst []byte) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16LEToLatin1Scalar(input, dst)
	}
	n := convertUTF16LEToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF16LEToLatin1Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF16BEToLatin1NEON(input []uint16, dst []byte) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16BEToLatin1Scalar(input, dst)
	}
	n := convertUTF16BEToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF16BEToLatin1Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF16LEToLatin1WithErrorsNEON(input []uint16, dst []byte) Result {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16LEToLatin1WithErrorsScalar(input, dst)
	}
	n := convertUTF16LEToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF16LEToLatin1WithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertUTF16BEToLatin1WithErrorsNEON(input []uint16, dst []byte) Result {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16BEToLatin1WithErrorsScalar(input, dst)
	}
	n := convertUTF16BEToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF16BEToLatin1WithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertValidUTF16LEToLatin1NEON(input []uint16, dst []byte) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF16LEToLatin1Scalar(input, dst)
	}
	n := convertUTF16LEToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	return n + convertValidUTF16LEToLatin1Scalar(input[n:], dst[n:])
}

func convertValidUTF16BEToLatin1NEON(input []uint16, dst []byte) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF16BEToLatin1Scalar(input, dst)
	}
	n := convertUTF16BEToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	return n + convertValidUTF16BEToLatin1Scalar(input[n:], dst[n:])
}

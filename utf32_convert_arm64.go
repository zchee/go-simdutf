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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf32_to_latin1.cpp,
// src/arm64/arm_convert_utf32_to_utf8.cpp, src/arm64/arm_convert_utf32_to_utf16.cpp, and
// src/arm64/implementation.cpp (utf8/utf16_length_from_utf32). Assembly consumes only
// complete in-bounds NEON blocks for the Latin-1 pack, UTF-8 ASCII 1:1 prefix, and
// UTF-16 BMP (no-surrogate) pack paths; wrappers retain scalar preflight and tails.

//go:noescape
func convertUTF32ToLatin1BlocksNEON(input []uint32, dst []byte) (consumed int)

//go:noescape
func convertUTF32ToUTF8BlocksNEON(input []uint32, dst []byte) (consumed int)

//go:noescape
func convertUTF32ToUTF16LEBlocksNEON(input []uint32, dst []uint16) (consumed int)

//go:noescape
func convertUTF32ToUTF16BEBlocksNEON(input []uint32, dst []uint16) (consumed int)

//go:noescape
func utf8LengthFromUTF32BlocksNEON(input []uint32) (length int)

//go:noescape
func utf16LengthFromUTF32BlocksNEON(input []uint32) (length int)

func convertUTF32ToLatin1NEON(input []uint32, dst []byte) int {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToLatin1Scalar(input, dst)
	}
	n := convertUTF32ToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF32ToLatin1Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF32ToLatin1WithErrorsNEON(input []uint32, dst []byte) Result {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToLatin1WithErrorsScalar(input, dst)
	}
	n := convertUTF32ToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF32ToLatin1WithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertValidUTF32ToLatin1NEON(input []uint32, dst []byte) int {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF32ToLatin1Scalar(input, dst)
	}
	n := convertUTF32ToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF32ToLatin1Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF32ToUTF8NEON(input []uint32, dst []byte) int {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToUTF8Scalar(input, dst)
	}
	n := convertUTF32ToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF32ToUTF8Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF32ToUTF8WithErrorsNEON(input []uint32, dst []byte) Result {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToUTF8WithErrorsScalar(input, dst)
	}
	n := convertUTF32ToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF32ToUTF8WithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertValidUTF32ToUTF8NEON(input []uint32, dst []byte) int {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF32ToUTF8Scalar(input, dst)
	}
	n := convertUTF32ToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF32ToUTF8Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF32ToUTF16LENEON(input []uint32, dst []uint16) int {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToUTF16LEScalar(input, dst)
	}
	n := convertUTF32ToUTF16LEBlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF32ToUTF16LEScalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF32ToUTF16BENEON(input []uint32, dst []uint16) int {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToUTF16BEScalar(input, dst)
	}
	n := convertUTF32ToUTF16BEBlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF32ToUTF16BEScalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF32ToUTF16LEWithErrorsNEON(input []uint32, dst []uint16) Result {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToUTF16LEWithErrorsScalar(input, dst)
	}
	n := convertUTF32ToUTF16LEBlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF32ToUTF16LEWithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertUTF32ToUTF16BEWithErrorsNEON(input []uint32, dst []uint16) Result {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF32ToUTF16BEWithErrorsScalar(input, dst)
	}
	n := convertUTF32ToUTF16BEBlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF32ToUTF16BEWithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertValidUTF32ToUTF16LENEON(input []uint32, dst []uint16) int {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF32ToUTF16LEScalar(input, dst)
	}
	n := convertUTF32ToUTF16LEBlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF32ToUTF16LEScalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertValidUTF32ToUTF16BENEON(input []uint32, dst []uint16) int {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF32ToUTF16BEScalar(input, dst)
	}
	n := convertUTF32ToUTF16BEBlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF32ToUTF16BEScalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func utf8LengthFromUTF32NEON(input []uint32) int {
	const block = 4
	if len(input) < block {
		return utf8LengthFromUTF32Scalar(input)
	}
	n := len(input) &^ (block - 1)
	return utf8LengthFromUTF32BlocksNEON(input[:n]) + utf8LengthFromUTF32Scalar(input[n:])
}

func utf16LengthFromUTF32NEON(input []uint32) int {
	const block = 4
	if len(input) < block {
		return utf16LengthFromUTF32Scalar(input)
	}
	n := len(input) &^ (block - 1)
	return utf16LengthFromUTF32BlocksNEON(input[:n]) + utf16LengthFromUTF32Scalar(input[n:])
}

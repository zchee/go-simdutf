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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf16_to_utf8.cpp
// and src/arm64/implementation.cpp. Assembly consumes only complete in-bounds
// 8-uint16 NEON ASCII blocks (written == consumed, 1:1); on the first non-ASCII
// vector it returns the consumed prefix and Go wrappers remount the scalar
// oracle for the remainder (and for error / replacement / length paths).
// Empty remaining input after a full SIMD success returns the written length
// directly so Valid/convert do not treat a zero-length scalar tail as failure.

//go:noescape
func convertUTF16LEToUTF8BlocksNEON(input []uint16, dst []byte) (consumed int)

//go:noescape
func convertUTF16BEToUTF8BlocksNEON(input []uint16, dst []byte) (consumed int)

func convertUTF16LEToUTF8NEON(input []uint16, dst []byte) int {
	if len(dst) < utf8LengthFromUTF16LEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16LEToUTF8Scalar(input, dst)
	}
	n := convertUTF16LEToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF16LEToUTF8Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF16BEToUTF8NEON(input []uint16, dst []byte) int {
	if len(dst) < utf8LengthFromUTF16BEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16BEToUTF8Scalar(input, dst)
	}
	n := convertUTF16BEToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF16BEToUTF8Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF16LEToUTF8WithErrorsNEON(input []uint16, dst []byte) Result {
	if len(dst) < utf8LengthFromUTF16LEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16LEToUTF8WithErrorsScalar(input, dst)
	}
	n := convertUTF16LEToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF16LEToUTF8WithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertUTF16BEToUTF8WithErrorsNEON(input []uint16, dst []byte) Result {
	if len(dst) < utf8LengthFromUTF16BEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16BEToUTF8WithErrorsScalar(input, dst)
	}
	n := convertUTF16BEToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF16BEToUTF8WithErrorsScalar(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertUTF16LEToUTF8WithReplacementNEON(input []uint16, dst []byte) int {
	if len(dst) < utf8LengthFromUTF16WithReplacementScalar(input, nativeLittleEndian()) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16LEToUTF8WithReplacementScalar(input, dst)
	}
	n := convertUTF16LEToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	return n + convertUTF16LEToUTF8WithReplacementScalar(input[n:], dst[n:])
}

func convertUTF16BEToUTF8WithReplacementNEON(input []uint16, dst []byte) int {
	if len(dst) < utf8LengthFromUTF16WithReplacementScalar(input, !nativeLittleEndian()) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertUTF16BEToUTF8WithReplacementScalar(input, dst)
	}
	n := convertUTF16BEToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	return n + convertUTF16BEToUTF8WithReplacementScalar(input[n:], dst[n:])
}

func convertValidUTF16LEToUTF8NEON(input []uint16, dst []byte) int {
	if len(dst) < utf8LengthFromUTF16LEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF16LEToUTF8Scalar(input, dst)
	}
	n := convertUTF16LEToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF16LEToUTF8Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertValidUTF16BEToUTF8NEON(input []uint16, dst []byte) int {
	if len(dst) < utf8LengthFromUTF16BEScalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 8 {
		return convertValidUTF16BEToUTF8Scalar(input, dst)
	}
	n := convertUTF16BEToUTF8BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF16BEToUTF8Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func utf8LengthFromUTF16LENEON(input []uint16) int {
	return utf8LengthFromUTF16LEScalar(input)
}

func utf8LengthFromUTF16BENEON(input []uint16) int {
	return utf8LengthFromUTF16BEScalar(input)
}

func utf8LengthFromUTF16LEWithReplacementNEON(input []uint16) Result {
	return utf8LengthFromUTF16LEWithReplacementScalar(input)
}

func utf8LengthFromUTF16BEWithReplacementNEON(input []uint16) Result {
	return utf8LengthFromUTF16BEWithReplacementScalar(input)
}

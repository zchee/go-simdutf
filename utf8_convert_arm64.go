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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf8_to_*.cpp
// and src/arm64/implementation.cpp. Assembly consumes only complete ASCII NEON
// blocks; wrappers retain scalar preflight and mixed-tail re-entry.

//go:noescape
func utf8ASCIIToLatin1BlocksNEON(input, dst []byte) (consumed int)

//go:noescape
func utf8ASCIIToUTF16LEBlocksNEON(input []byte, dst []uint16) (consumed int)

//go:noescape
func utf8ASCIIToUTF16BEBlocksNEON(input []byte, dst []uint16) (consumed int)

//go:noescape
func utf8ASCIIToUTF32BlocksNEON(input []byte, dst []uint32) (consumed int)

func convertUTF8ToLatin1NEON(input, dst []byte) int {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertUTF8ToLatin1Scalar(input, dst)
	}
	n := utf8ASCIIToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF8ToLatin1Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF8ToLatin1WithErrorsNEON(input, dst []byte) Result {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertUTF8ToLatin1WithErrorsScalar(input, dst)
	}
	n := utf8ASCIIToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF8ToLatin1WithErrorsScalar(input[n:], dst[n:])
	if res.Error != Success {
		res.Count += n
		return res
	}
	res.Count += n
	return res
}

func convertValidUTF8ToLatin1NEON(input, dst []byte) int {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertValidUTF8ToLatin1Scalar(input, dst)
	}
	n := utf8ASCIIToLatin1BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF8ToLatin1Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF8ToUTF16LENEON(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16NEON(input, dst, utf8ASCIIToUTF16LEBlocksNEON, convertUTF8ToUTF16LEScalar)
}

func convertUTF8ToUTF16BENEON(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16NEON(input, dst, utf8ASCIIToUTF16BEBlocksNEON, convertUTF8ToUTF16BEScalar)
}

func convertUTF8ToUTF16NEON(input []byte, dst []uint16, blocks, tail func([]byte, []uint16) int) int {
	if len(dst) < utf16LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return tail(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	rest := tail(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF8ToUTF16LEWithErrorsNEON(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsNEON(input, dst, utf8ASCIIToUTF16LEBlocksNEON, convertUTF8ToUTF16LEWithErrorsScalar)
}

func convertUTF8ToUTF16BEWithErrorsNEON(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsNEON(input, dst, utf8ASCIIToUTF16BEBlocksNEON, convertUTF8ToUTF16BEWithErrorsScalar)
}

func convertUTF8ToUTF16WithErrorsNEON(input []byte, dst []uint16, blocks func([]byte, []uint16) int, tail func([]byte, []uint16) Result) Result {
	if len(dst) < utf16LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return tail(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := tail(input[n:], dst[n:])
	if res.Error != Success {
		res.Count += n
		return res
	}
	res.Count += n
	return res
}

func convertValidUTF8ToUTF16LENEON(input []byte, dst []uint16) int {
	return convertValidUTF8ToUTF16NEON(input, dst, utf8ASCIIToUTF16LEBlocksNEON, convertValidUTF8ToUTF16LEScalar)
}

func convertValidUTF8ToUTF16BENEON(input []byte, dst []uint16) int {
	return convertValidUTF8ToUTF16NEON(input, dst, utf8ASCIIToUTF16BEBlocksNEON, convertValidUTF8ToUTF16BEScalar)
}

func convertValidUTF8ToUTF16NEON(input []byte, dst []uint16, blocks, tail func([]byte, []uint16) int) int {
	if len(dst) < utf16LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return tail(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	rest := tail(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF8ToUTF32NEON(input []byte, dst []uint32) int {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertUTF8ToUTF32Scalar(input, dst)
	}
	n := utf8ASCIIToUTF32BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF8ToUTF32Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF8ToUTF32WithErrorsNEON(input []byte, dst []uint32) Result {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertUTF8ToUTF32WithErrorsScalar(input, dst)
	}
	n := utf8ASCIIToUTF32BlocksNEON(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := convertUTF8ToUTF32WithErrorsScalar(input[n:], dst[n:])
	if res.Error != Success {
		res.Count += n
		return res
	}
	res.Count += n
	return res
}

func convertValidUTF8ToUTF32NEON(input []byte, dst []uint32) int {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertValidUTF8ToUTF32Scalar(input, dst)
	}
	n := utf8ASCIIToUTF32BlocksNEON(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF8ToUTF32Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

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

//go:build amd64

package simdutf

// Independently translated from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_utf8.cpp,
// src/haswell/avx2_convert_utf16_to_utf8.cpp, and the matching westmere/haswell
// implementation.cpp drivers. Assembly accelerates only complete ASCII vector
// groups (8 uint16 Westmere / 16 uint16 Haswell) where written==consumed; these
// wrappers retain the scalar remount/tail for non-ASCII/surrogate/incomplete
// blocks and the all-or-nothing destination preflight required by Go.
// Variable-length UTF-8 means do not assume written==consumed except on the
// ASCII fast path.

//go:noescape
func utf16LEToUTF8ASCIIBlocksWestmere(input []uint16, dst []byte) (consumed int)

//go:noescape
func utf16BEToUTF8ASCIIBlocksWestmere(input []uint16, dst []byte) (consumed int)

//go:noescape
func utf16LEToUTF8ASCIIBlocksHaswell(input []uint16, dst []byte) (consumed int)

//go:noescape
func utf16BEToUTF8ASCIIBlocksHaswell(input []uint16, dst []byte) (consumed int)

func convertUTF16LEToUTF8Westmere(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8AMD64(input, dst, utf16LEToUTF8ASCIIBlocksWestmere, convertUTF16LEToUTF8Scalar, utf8LengthFromUTF16LEScalar, 8)
}

func convertUTF16BEToUTF8Westmere(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8AMD64(input, dst, utf16BEToUTF8ASCIIBlocksWestmere, convertUTF16BEToUTF8Scalar, utf8LengthFromUTF16BEScalar, 8)
}

func convertUTF16LEToUTF8Haswell(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8AMD64(input, dst, utf16LEToUTF8ASCIIBlocksHaswell, convertUTF16LEToUTF8Scalar, utf8LengthFromUTF16LEScalar, 16)
}

func convertUTF16BEToUTF8Haswell(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8AMD64(input, dst, utf16BEToUTF8ASCIIBlocksHaswell, convertUTF16BEToUTF8Scalar, utf8LengthFromUTF16BEScalar, 16)
}

func convertUTF16ToUTF8AMD64(input []uint16, dst []byte, blocks, tail func([]uint16, []byte) int, length func([]uint16) int, minBlock int) int {
	if len(dst) < length(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < minBlock {
		return tail(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	// ASCII fast path: written == consumed == n.
	rest := tail(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF16LEToUTF8WithErrorsWestmere(input []uint16, dst []byte) Result {
	return convertUTF16ToUTF8WithErrorsAMD64(input, dst, utf16LEToUTF8ASCIIBlocksWestmere, convertUTF16LEToUTF8WithErrorsScalar, utf8LengthFromUTF16LEScalar, 8)
}

func convertUTF16BEToUTF8WithErrorsWestmere(input []uint16, dst []byte) Result {
	return convertUTF16ToUTF8WithErrorsAMD64(input, dst, utf16BEToUTF8ASCIIBlocksWestmere, convertUTF16BEToUTF8WithErrorsScalar, utf8LengthFromUTF16BEScalar, 8)
}

func convertUTF16LEToUTF8WithErrorsHaswell(input []uint16, dst []byte) Result {
	return convertUTF16ToUTF8WithErrorsAMD64(input, dst, utf16LEToUTF8ASCIIBlocksHaswell, convertUTF16LEToUTF8WithErrorsScalar, utf8LengthFromUTF16LEScalar, 16)
}

func convertUTF16BEToUTF8WithErrorsHaswell(input []uint16, dst []byte) Result {
	return convertUTF16ToUTF8WithErrorsAMD64(input, dst, utf16BEToUTF8ASCIIBlocksHaswell, convertUTF16BEToUTF8WithErrorsScalar, utf8LengthFromUTF16BEScalar, 16)
}

func convertUTF16ToUTF8WithErrorsAMD64(input []uint16, dst []byte, blocks func([]uint16, []byte) int, tail func([]uint16, []byte) Result, length func([]uint16) int, minBlock int) Result {
	if len(dst) < length(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < minBlock {
		return tail(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	// Resume from the first non-ASCII/surrogate unit. On Success, Count is
	// UTF-8 bytes written; on error, Count is the failing UTF-16 unit offset.
	// ASCII written==consumed, so adding n is correct for both cases.
	res := tail(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertUTF16LEToUTF8WithReplacementWestmere(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8WithReplacementAMD64(input, dst, utf16LEToUTF8ASCIIBlocksWestmere, convertUTF16LEToUTF8WithReplacementScalar, utf8LengthFromUTF16LEWithReplacementScalar, 8)
}

func convertUTF16BEToUTF8WithReplacementWestmere(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8WithReplacementAMD64(input, dst, utf16BEToUTF8ASCIIBlocksWestmere, convertUTF16BEToUTF8WithReplacementScalar, utf8LengthFromUTF16BEWithReplacementScalar, 8)
}

func convertUTF16LEToUTF8WithReplacementHaswell(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8WithReplacementAMD64(input, dst, utf16LEToUTF8ASCIIBlocksHaswell, convertUTF16LEToUTF8WithReplacementScalar, utf8LengthFromUTF16LEWithReplacementScalar, 16)
}

func convertUTF16BEToUTF8WithReplacementHaswell(input []uint16, dst []byte) int {
	return convertUTF16ToUTF8WithReplacementAMD64(input, dst, utf16BEToUTF8ASCIIBlocksHaswell, convertUTF16BEToUTF8WithReplacementScalar, utf8LengthFromUTF16BEWithReplacementScalar, 16)
}

func convertUTF16ToUTF8WithReplacementAMD64(input []uint16, dst []byte, blocks, tail func([]uint16, []byte) int, length func([]uint16) Result, minBlock int) int {
	need := length(input)
	if len(dst) < need.Count {
		panic("simdutf: destination is too short")
	}
	if len(input) < minBlock {
		return tail(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	return n + tail(input[n:], dst[n:])
}

func convertValidUTF16LEToUTF8Westmere(input []uint16, dst []byte) int {
	return convertValidUTF16ToUTF8AMD64(input, dst, utf16LEToUTF8ASCIIBlocksWestmere, convertValidUTF16LEToUTF8Scalar, utf8LengthFromUTF16LEScalar, 8)
}

func convertValidUTF16BEToUTF8Westmere(input []uint16, dst []byte) int {
	return convertValidUTF16ToUTF8AMD64(input, dst, utf16BEToUTF8ASCIIBlocksWestmere, convertValidUTF16BEToUTF8Scalar, utf8LengthFromUTF16BEScalar, 8)
}

func convertValidUTF16LEToUTF8Haswell(input []uint16, dst []byte) int {
	return convertValidUTF16ToUTF8AMD64(input, dst, utf16LEToUTF8ASCIIBlocksHaswell, convertValidUTF16LEToUTF8Scalar, utf8LengthFromUTF16LEScalar, 16)
}

func convertValidUTF16BEToUTF8Haswell(input []uint16, dst []byte) int {
	return convertValidUTF16ToUTF8AMD64(input, dst, utf16BEToUTF8ASCIIBlocksHaswell, convertValidUTF16BEToUTF8Scalar, utf8LengthFromUTF16BEScalar, 16)
}

func convertValidUTF16ToUTF8AMD64(input []uint16, dst []byte, blocks, tail func([]uint16, []byte) int, length func([]uint16) int, minBlock int) int {
	if len(dst) < length(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < minBlock {
		return tail(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	rest := tail(input[n:], dst[n:])
	if rest == 0 {
		// Empty-tail success: scalar returns 0 on empty input; treat as done.
		if n == len(input) {
			return n
		}
		return 0
	}
	return n + rest
}

func utf8LengthFromUTF16LEWestmere(input []uint16) int {
	return utf8LengthFromUTF16LEScalar(input)
}

func utf8LengthFromUTF16BEWestmere(input []uint16) int {
	return utf8LengthFromUTF16BEScalar(input)
}

func utf8LengthFromUTF16LEHaswell(input []uint16) int {
	return utf8LengthFromUTF16LEScalar(input)
}

func utf8LengthFromUTF16BEHaswell(input []uint16) int {
	return utf8LengthFromUTF16BEScalar(input)
}

func utf8LengthFromUTF16LEWithReplacementWestmere(input []uint16) Result {
	return utf8LengthFromUTF16LEWithReplacementScalar(input)
}

func utf8LengthFromUTF16BEWithReplacementWestmere(input []uint16) Result {
	return utf8LengthFromUTF16BEWithReplacementScalar(input)
}

func utf8LengthFromUTF16LEWithReplacementHaswell(input []uint16) Result {
	return utf8LengthFromUTF16LEWithReplacementScalar(input)
}

func utf8LengthFromUTF16BEWithReplacementHaswell(input []uint16) Result {
	return utf8LengthFromUTF16BEWithReplacementScalar(input)
}

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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_latin1.cpp,
// src/haswell/avx2_convert_utf16_to_latin1.cpp, and the matching westmere/haswell
// implementation.cpp drivers. Assembly converts only complete latin1-only vector
// groups (8 uint16 Westmere / 16 uint16 Haswell); these wrappers retain the
// scalar remount/tail and the all-or-nothing destination preflight required by Go.

//go:noescape
func utf16LEToLatin1BlocksWestmere(input []uint16, dst []byte) (consumed int)

//go:noescape
func utf16BEToLatin1BlocksWestmere(input []uint16, dst []byte) (consumed int)

//go:noescape
func utf16LEToLatin1BlocksHaswell(input []uint16, dst []byte) (consumed int)

//go:noescape
func utf16BEToLatin1BlocksHaswell(input []uint16, dst []byte) (consumed int)

func convertUTF16LEToLatin1Westmere(input []uint16, dst []byte) int {
	return convertUTF16ToLatin1AMD64(input, dst, utf16LEToLatin1BlocksWestmere, convertUTF16LEToLatin1Scalar, 8)
}

func convertUTF16BEToLatin1Westmere(input []uint16, dst []byte) int {
	return convertUTF16ToLatin1AMD64(input, dst, utf16BEToLatin1BlocksWestmere, convertUTF16BEToLatin1Scalar, 8)
}

func convertUTF16LEToLatin1Haswell(input []uint16, dst []byte) int {
	return convertUTF16ToLatin1AMD64(input, dst, utf16LEToLatin1BlocksHaswell, convertUTF16LEToLatin1Scalar, 16)
}

func convertUTF16BEToLatin1Haswell(input []uint16, dst []byte) int {
	return convertUTF16ToLatin1AMD64(input, dst, utf16BEToLatin1BlocksHaswell, convertUTF16BEToLatin1Scalar, 16)
}

func convertUTF16ToLatin1AMD64(input []uint16, dst []byte, blocks, tail func([]uint16, []byte) int, minBlock int) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
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
		return 0
	}
	return n + rest
}

func convertUTF16LEToLatin1WithErrorsWestmere(input []uint16, dst []byte) Result {
	return convertUTF16ToLatin1WithErrorsAMD64(input, dst, utf16LEToLatin1BlocksWestmere, convertUTF16LEToLatin1WithErrorsScalar, 8)
}

func convertUTF16BEToLatin1WithErrorsWestmere(input []uint16, dst []byte) Result {
	return convertUTF16ToLatin1WithErrorsAMD64(input, dst, utf16BEToLatin1BlocksWestmere, convertUTF16BEToLatin1WithErrorsScalar, 8)
}

func convertUTF16LEToLatin1WithErrorsHaswell(input []uint16, dst []byte) Result {
	return convertUTF16ToLatin1WithErrorsAMD64(input, dst, utf16LEToLatin1BlocksHaswell, convertUTF16LEToLatin1WithErrorsScalar, 16)
}

func convertUTF16BEToLatin1WithErrorsHaswell(input []uint16, dst []byte) Result {
	return convertUTF16ToLatin1WithErrorsAMD64(input, dst, utf16BEToLatin1BlocksHaswell, convertUTF16BEToLatin1WithErrorsScalar, 16)
}

func convertUTF16ToLatin1WithErrorsAMD64(input []uint16, dst []byte, blocks func([]uint16, []byte) int, tail func([]uint16, []byte) Result, minBlock int) Result {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
		panic("simdutf: destination is too short")
	}
	if len(input) < minBlock {
		return tail(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return Result{Error: Success, Count: n}
	}
	res := tail(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertValidUTF16LEToLatin1Westmere(input []uint16, dst []byte) int {
	return convertValidUTF16ToLatin1AMD64(input, dst, utf16LEToLatin1BlocksWestmere, convertValidUTF16LEToLatin1Scalar, 8)
}

func convertValidUTF16BEToLatin1Westmere(input []uint16, dst []byte) int {
	return convertValidUTF16ToLatin1AMD64(input, dst, utf16BEToLatin1BlocksWestmere, convertValidUTF16BEToLatin1Scalar, 8)
}

func convertValidUTF16LEToLatin1Haswell(input []uint16, dst []byte) int {
	return convertValidUTF16ToLatin1AMD64(input, dst, utf16LEToLatin1BlocksHaswell, convertValidUTF16LEToLatin1Scalar, 16)
}

func convertValidUTF16BEToLatin1Haswell(input []uint16, dst []byte) int {
	return convertValidUTF16ToLatin1AMD64(input, dst, utf16BEToLatin1BlocksHaswell, convertValidUTF16BEToLatin1Scalar, 16)
}

func convertValidUTF16ToLatin1AMD64(input []uint16, dst []byte, blocks, tail func([]uint16, []byte) int, minBlock int) int {
	if len(dst) < latin1LengthFromUTF16Scalar(len(input)) {
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

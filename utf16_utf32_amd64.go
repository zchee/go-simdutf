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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_utf32.cpp,
// src/haswell/avx2_convert_utf16_to_utf32.cpp, and the matching westmere/haswell
// implementation.cpp drivers. Assembly converts only complete no-surrogate vector
// groups (8 uint16 Westmere / 16 uint16 Haswell) via PMOVZXWD / VPMOVZXWD; these
// wrappers retain the scalar remount/tail and the all-or-nothing destination
// preflight required by Go.

//go:noescape
func utf16LEToUTF32BlocksWestmere(input []uint16, dst []uint32) (consumed int)

//go:noescape
func utf16BEToUTF32BlocksWestmere(input []uint16, dst []uint32) (consumed int)

//go:noescape
func utf16LEToUTF32BlocksHaswell(input []uint16, dst []uint32) (consumed int)

//go:noescape
func utf16BEToUTF32BlocksHaswell(input []uint16, dst []uint32) (consumed int)

func convertUTF16LEToUTF32Westmere(input []uint16, dst []uint32) int {
	return convertUTF16ToUTF32AMD64(input, dst, utf16LEToUTF32BlocksWestmere, convertUTF16LEToUTF32Scalar, utf32LengthFromUTF16LEScalar, 8)
}

func convertUTF16BEToUTF32Westmere(input []uint16, dst []uint32) int {
	return convertUTF16ToUTF32AMD64(input, dst, utf16BEToUTF32BlocksWestmere, convertUTF16BEToUTF32Scalar, utf32LengthFromUTF16BEScalar, 8)
}

func convertUTF16LEToUTF32Haswell(input []uint16, dst []uint32) int {
	return convertUTF16ToUTF32AMD64(input, dst, utf16LEToUTF32BlocksHaswell, convertUTF16LEToUTF32Scalar, utf32LengthFromUTF16LEScalar, 16)
}

func convertUTF16BEToUTF32Haswell(input []uint16, dst []uint32) int {
	return convertUTF16ToUTF32AMD64(input, dst, utf16BEToUTF32BlocksHaswell, convertUTF16BEToUTF32Scalar, utf32LengthFromUTF16BEScalar, 16)
}

func convertUTF16ToUTF32AMD64(input []uint16, dst []uint32, blocks func([]uint16, []uint32) int, tail func([]uint16, []uint32) int, length func([]uint16) int, minBlock int) int {
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
		return 0
	}
	return n + rest
}

func convertUTF16LEToUTF32WithErrorsWestmere(input []uint16, dst []uint32) Result {
	return convertUTF16ToUTF32WithErrorsAMD64(input, dst, utf16LEToUTF32BlocksWestmere, convertUTF16LEToUTF32WithErrorsScalar, utf32LengthFromUTF16LEScalar, 8)
}

func convertUTF16BEToUTF32WithErrorsWestmere(input []uint16, dst []uint32) Result {
	return convertUTF16ToUTF32WithErrorsAMD64(input, dst, utf16BEToUTF32BlocksWestmere, convertUTF16BEToUTF32WithErrorsScalar, utf32LengthFromUTF16BEScalar, 8)
}

func convertUTF16LEToUTF32WithErrorsHaswell(input []uint16, dst []uint32) Result {
	return convertUTF16ToUTF32WithErrorsAMD64(input, dst, utf16LEToUTF32BlocksHaswell, convertUTF16LEToUTF32WithErrorsScalar, utf32LengthFromUTF16LEScalar, 16)
}

func convertUTF16BEToUTF32WithErrorsHaswell(input []uint16, dst []uint32) Result {
	return convertUTF16ToUTF32WithErrorsAMD64(input, dst, utf16BEToUTF32BlocksHaswell, convertUTF16BEToUTF32WithErrorsScalar, utf32LengthFromUTF16BEScalar, 16)
}

func convertUTF16ToUTF32WithErrorsAMD64(input []uint16, dst []uint32, blocks func([]uint16, []uint32) int, tail func([]uint16, []uint32) Result, length func([]uint16) int, minBlock int) Result {
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
	res := tail(input[n:], dst[n:])
	res.Count += n
	return res
}

func convertValidUTF16LEToUTF32Westmere(input []uint16, dst []uint32) int {
	return convertValidUTF16ToUTF32AMD64(input, dst, utf16LEToUTF32BlocksWestmere, convertValidUTF16LEToUTF32Scalar, utf32LengthFromUTF16LEScalar, 8)
}

func convertValidUTF16BEToUTF32Westmere(input []uint16, dst []uint32) int {
	return convertValidUTF16ToUTF32AMD64(input, dst, utf16BEToUTF32BlocksWestmere, convertValidUTF16BEToUTF32Scalar, utf32LengthFromUTF16BEScalar, 8)
}

func convertValidUTF16LEToUTF32Haswell(input []uint16, dst []uint32) int {
	return convertValidUTF16ToUTF32AMD64(input, dst, utf16LEToUTF32BlocksHaswell, convertValidUTF16LEToUTF32Scalar, utf32LengthFromUTF16LEScalar, 16)
}

func convertValidUTF16BEToUTF32Haswell(input []uint16, dst []uint32) int {
	return convertValidUTF16ToUTF32AMD64(input, dst, utf16BEToUTF32BlocksHaswell, convertValidUTF16BEToUTF32Scalar, utf32LengthFromUTF16BEScalar, 16)
}

func convertValidUTF16ToUTF32AMD64(input []uint16, dst []uint32, blocks func([]uint16, []uint32) int, tail func([]uint16, []uint32) int, length func([]uint16) int, minBlock int) int {
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
		return 0
	}
	return n + rest
}

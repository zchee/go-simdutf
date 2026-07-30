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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf32_to_latin1.cpp,
// src/westmere/sse_convert_utf32_to_utf8.cpp, src/westmere/sse_convert_utf32_to_utf16.cpp,
// src/haswell/avx2_convert_utf32_to_latin1.cpp, src/haswell/avx2_convert_utf32_to_utf8.cpp,
// src/haswell/avx2_convert_utf32_to_utf16.cpp, and the matching westmere/haswell
// implementation.cpp length helpers. Assembly accelerates only complete fast-path
// vector groups (4 uint32 Westmere / 8 uint32 Haswell):
//   - Latin-1: all code points ≤ 0xff
//   - UTF-8:   all code points ≤ 0x7f (ASCII; written==consumed)
//   - UTF-16:  all code points are BMP and non-surrogate
// Wrappers retain the scalar remount/tail and the all-or-nothing destination
// preflight required by Go. Length providers currently wrap the scalar oracles;
// forceable Westmere/Haswell slots remain available for later kernel promotion.

//go:noescape
func utf32ToLatin1BlocksWestmere(input []uint32, dst []byte) (consumed int)

//go:noescape
func utf32ToLatin1BlocksHaswell(input []uint32, dst []byte) (consumed int)

//go:noescape
func utf32ToUTF8ASCIIBlocksWestmere(input []uint32, dst []byte) (consumed int)

//go:noescape
func utf32ToUTF8ASCIIBlocksHaswell(input []uint32, dst []byte) (consumed int)

//go:noescape
func utf32ToUTF16LEBlocksWestmere(input []uint32, dst []uint16) (consumed int)

//go:noescape
func utf32ToUTF16BEBlocksWestmere(input []uint32, dst []uint16) (consumed int)

//go:noescape
func utf32ToUTF16LEBlocksHaswell(input []uint32, dst []uint16) (consumed int)

//go:noescape
func utf32ToUTF16BEBlocksHaswell(input []uint32, dst []uint16) (consumed int)

func convertUTF32ToLatin1Westmere(input []uint32, dst []byte) int {
	return convertUTF32ToLatin1AMD64(input, dst, utf32ToLatin1BlocksWestmere, convertUTF32ToLatin1Scalar, 4)
}

func convertUTF32ToLatin1Haswell(input []uint32, dst []byte) int {
	return convertUTF32ToLatin1AMD64(input, dst, utf32ToLatin1BlocksHaswell, convertUTF32ToLatin1Scalar, 8)
}

func convertUTF32ToLatin1WithErrorsWestmere(input []uint32, dst []byte) Result {
	return convertUTF32ToLatin1WithErrorsAMD64(input, dst, utf32ToLatin1BlocksWestmere, convertUTF32ToLatin1WithErrorsScalar, 4)
}

func convertUTF32ToLatin1WithErrorsHaswell(input []uint32, dst []byte) Result {
	return convertUTF32ToLatin1WithErrorsAMD64(input, dst, utf32ToLatin1BlocksHaswell, convertUTF32ToLatin1WithErrorsScalar, 8)
}

func convertValidUTF32ToLatin1Westmere(input []uint32, dst []byte) int {
	return convertValidUTF32ToLatin1AMD64(input, dst, utf32ToLatin1BlocksWestmere, convertValidUTF32ToLatin1Scalar, 4)
}

func convertValidUTF32ToLatin1Haswell(input []uint32, dst []byte) int {
	return convertValidUTF32ToLatin1AMD64(input, dst, utf32ToLatin1BlocksHaswell, convertValidUTF32ToLatin1Scalar, 8)
}

func convertUTF32ToLatin1AMD64(input []uint32, dst []byte, blocks func([]uint32, []byte) int, tail func([]uint32, []byte) int, minBlock int) int {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
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

func convertUTF32ToLatin1WithErrorsAMD64(input []uint32, dst []byte, blocks func([]uint32, []byte) int, tail func([]uint32, []byte) Result, minBlock int) Result {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
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

func convertValidUTF32ToLatin1AMD64(input []uint32, dst []byte, blocks func([]uint32, []byte) int, tail func([]uint32, []byte) int, minBlock int) int {
	if len(dst) < latin1LengthFromUTF32Scalar(len(input)) {
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

func convertUTF32ToUTF8Westmere(input []uint32, dst []byte) int {
	return convertUTF32ToUTF8AMD64(input, dst, utf32ToUTF8ASCIIBlocksWestmere, convertUTF32ToUTF8Scalar, 4)
}

func convertUTF32ToUTF8Haswell(input []uint32, dst []byte) int {
	return convertUTF32ToUTF8AMD64(input, dst, utf32ToUTF8ASCIIBlocksHaswell, convertUTF32ToUTF8Scalar, 8)
}

func convertUTF32ToUTF8WithErrorsWestmere(input []uint32, dst []byte) Result {
	return convertUTF32ToUTF8WithErrorsAMD64(input, dst, utf32ToUTF8ASCIIBlocksWestmere, convertUTF32ToUTF8WithErrorsScalar, 4)
}

func convertUTF32ToUTF8WithErrorsHaswell(input []uint32, dst []byte) Result {
	return convertUTF32ToUTF8WithErrorsAMD64(input, dst, utf32ToUTF8ASCIIBlocksHaswell, convertUTF32ToUTF8WithErrorsScalar, 8)
}

func convertValidUTF32ToUTF8Westmere(input []uint32, dst []byte) int {
	return convertValidUTF32ToUTF8AMD64(input, dst, utf32ToUTF8ASCIIBlocksWestmere, convertValidUTF32ToUTF8Scalar, 4)
}

func convertValidUTF32ToUTF8Haswell(input []uint32, dst []byte) int {
	return convertValidUTF32ToUTF8AMD64(input, dst, utf32ToUTF8ASCIIBlocksHaswell, convertValidUTF32ToUTF8Scalar, 8)
}

func convertUTF32ToUTF8AMD64(input []uint32, dst []byte, blocks func([]uint32, []byte) int, tail func([]uint32, []byte) int, minBlock int) int {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
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

func convertUTF32ToUTF8WithErrorsAMD64(input []uint32, dst []byte, blocks func([]uint32, []byte) int, tail func([]uint32, []byte) Result, minBlock int) Result {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
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

func convertValidUTF32ToUTF8AMD64(input []uint32, dst []byte, blocks func([]uint32, []byte) int, tail func([]uint32, []byte) int, minBlock int) int {
	if len(dst) < utf8LengthFromUTF32Scalar(input) {
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

func convertUTF32ToUTF16LEWestmere(input []uint32, dst []uint16) int {
	return convertUTF32ToUTF16AMD64(input, dst, utf32ToUTF16LEBlocksWestmere, convertUTF32ToUTF16LEScalar, 4)
}

func convertUTF32ToUTF16BEWestmere(input []uint32, dst []uint16) int {
	return convertUTF32ToUTF16AMD64(input, dst, utf32ToUTF16BEBlocksWestmere, convertUTF32ToUTF16BEScalar, 4)
}

func convertUTF32ToUTF16LEHaswell(input []uint32, dst []uint16) int {
	return convertUTF32ToUTF16AMD64(input, dst, utf32ToUTF16LEBlocksHaswell, convertUTF32ToUTF16LEScalar, 8)
}

func convertUTF32ToUTF16BEHaswell(input []uint32, dst []uint16) int {
	return convertUTF32ToUTF16AMD64(input, dst, utf32ToUTF16BEBlocksHaswell, convertUTF32ToUTF16BEScalar, 8)
}

func convertUTF32ToUTF16LEWithErrorsWestmere(input []uint32, dst []uint16) Result {
	return convertUTF32ToUTF16WithErrorsAMD64(input, dst, utf32ToUTF16LEBlocksWestmere, convertUTF32ToUTF16LEWithErrorsScalar, 4)
}

func convertUTF32ToUTF16BEWithErrorsWestmere(input []uint32, dst []uint16) Result {
	return convertUTF32ToUTF16WithErrorsAMD64(input, dst, utf32ToUTF16BEBlocksWestmere, convertUTF32ToUTF16BEWithErrorsScalar, 4)
}

func convertUTF32ToUTF16LEWithErrorsHaswell(input []uint32, dst []uint16) Result {
	return convertUTF32ToUTF16WithErrorsAMD64(input, dst, utf32ToUTF16LEBlocksHaswell, convertUTF32ToUTF16LEWithErrorsScalar, 8)
}

func convertUTF32ToUTF16BEWithErrorsHaswell(input []uint32, dst []uint16) Result {
	return convertUTF32ToUTF16WithErrorsAMD64(input, dst, utf32ToUTF16BEBlocksHaswell, convertUTF32ToUTF16BEWithErrorsScalar, 8)
}

func convertValidUTF32ToUTF16LEWestmere(input []uint32, dst []uint16) int {
	return convertValidUTF32ToUTF16AMD64(input, dst, utf32ToUTF16LEBlocksWestmere, convertValidUTF32ToUTF16LEScalar, 4)
}

func convertValidUTF32ToUTF16BEWestmere(input []uint32, dst []uint16) int {
	return convertValidUTF32ToUTF16AMD64(input, dst, utf32ToUTF16BEBlocksWestmere, convertValidUTF32ToUTF16BEScalar, 4)
}

func convertValidUTF32ToUTF16LEHaswell(input []uint32, dst []uint16) int {
	return convertValidUTF32ToUTF16AMD64(input, dst, utf32ToUTF16LEBlocksHaswell, convertValidUTF32ToUTF16LEScalar, 8)
}

func convertValidUTF32ToUTF16BEHaswell(input []uint32, dst []uint16) int {
	return convertValidUTF32ToUTF16AMD64(input, dst, utf32ToUTF16BEBlocksHaswell, convertValidUTF32ToUTF16BEScalar, 8)
}

func convertUTF32ToUTF16AMD64(input []uint32, dst []uint16, blocks func([]uint32, []uint16) int, tail func([]uint32, []uint16) int, minBlock int) int {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < minBlock {
		return tail(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	// BMP fast path: written == consumed == n.
	rest := tail(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF32ToUTF16WithErrorsAMD64(input []uint32, dst []uint16, blocks func([]uint32, []uint16) int, tail func([]uint32, []uint16) Result, minBlock int) Result {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
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

func convertValidUTF32ToUTF16AMD64(input []uint32, dst []uint16, blocks func([]uint32, []uint16) int, tail func([]uint32, []uint16) int, minBlock int) int {
	if len(dst) < utf16LengthFromUTF32Scalar(input) {
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

func utf8LengthFromUTF32Westmere(input []uint32) int {
	return utf8LengthFromUTF32Scalar(input)
}

func utf8LengthFromUTF32Haswell(input []uint32) int {
	return utf8LengthFromUTF32Scalar(input)
}

func utf16LengthFromUTF32Westmere(input []uint32) int {
	return utf16LengthFromUTF32Scalar(input)
}

func utf16LengthFromUTF32Haswell(input []uint32) int {
	return utf16LengthFromUTF32Scalar(input)
}

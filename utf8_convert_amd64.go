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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf8_to_*.cpp,
// src/haswell/avx2_convert_utf8_to_*.cpp, and the generic validating transcoder drivers.
// Assembly accelerates only complete ASCII runs; Go wrappers retain destination
// preflight and re-enter the pure-Go scalar oracle for the first non-ASCII byte,
// mixed tails, validation errors, and short inputs.

//go:noescape
func utf8ASCIIToLatin1BlocksWestmere(input, dst []byte) (consumed int)

//go:noescape
func utf8ASCIIToLatin1BlocksHaswell(input, dst []byte) (consumed int)

//go:noescape
func utf8ASCIIToUTF16LEBlocksWestmere(input []byte, dst []uint16) (consumed int)

//go:noescape
func utf8ASCIIToUTF16BEBlocksWestmere(input []byte, dst []uint16) (consumed int)

//go:noescape
func utf8ASCIIToUTF16LEBlocksHaswell(input []byte, dst []uint16) (consumed int)

//go:noescape
func utf8ASCIIToUTF16BEBlocksHaswell(input []byte, dst []uint16) (consumed int)

//go:noescape
func utf8ASCIIToUTF32BlocksWestmere(input []byte, dst []uint32) (consumed int)

//go:noescape
func utf8ASCIIToUTF32BlocksHaswell(input []byte, dst []uint32) (consumed int)

func convertUTF8ToLatin1Westmere(input, dst []byte) int {
	return convertUTF8ToLatin1AMD64(input, dst, utf8ASCIIToLatin1BlocksWestmere)
}
func convertUTF8ToLatin1Haswell(input, dst []byte) int {
	return convertUTF8ToLatin1AMD64(input, dst, utf8ASCIIToLatin1BlocksHaswell)
}
func convertUTF8ToLatin1AMD64(input, dst []byte, blocks func([]byte, []byte) int) int {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertUTF8ToLatin1Scalar(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF8ToLatin1Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF8ToLatin1WithErrorsWestmere(input, dst []byte) Result {
	return convertUTF8ToLatin1WithErrorsAMD64(input, dst, utf8ASCIIToLatin1BlocksWestmere)
}
func convertUTF8ToLatin1WithErrorsHaswell(input, dst []byte) Result {
	return convertUTF8ToLatin1WithErrorsAMD64(input, dst, utf8ASCIIToLatin1BlocksHaswell)
}
func convertUTF8ToLatin1WithErrorsAMD64(input, dst []byte, blocks func([]byte, []byte) int) Result {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertUTF8ToLatin1WithErrorsScalar(input, dst)
	}
	n := blocks(input, dst)
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

func convertValidUTF8ToLatin1Westmere(input, dst []byte) int {
	return convertValidUTF8ToLatin1AMD64(input, dst, utf8ASCIIToLatin1BlocksWestmere)
}
func convertValidUTF8ToLatin1Haswell(input, dst []byte) int {
	return convertValidUTF8ToLatin1AMD64(input, dst, utf8ASCIIToLatin1BlocksHaswell)
}
func convertValidUTF8ToLatin1AMD64(input, dst []byte, blocks func([]byte, []byte) int) int {
	if len(dst) < latin1LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertValidUTF8ToLatin1Scalar(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF8ToLatin1Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF8ToUTF16LEWestmere(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16AMD64(input, dst, utf8ASCIIToUTF16LEBlocksWestmere, convertUTF8ToUTF16LEScalar)
}
func convertUTF8ToUTF16BEWestmere(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16AMD64(input, dst, utf8ASCIIToUTF16BEBlocksWestmere, convertUTF8ToUTF16BEScalar)
}
func convertUTF8ToUTF16LEHaswell(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16AMD64(input, dst, utf8ASCIIToUTF16LEBlocksHaswell, convertUTF8ToUTF16LEScalar)
}
func convertUTF8ToUTF16BEHaswell(input []byte, dst []uint16) int {
	return convertUTF8ToUTF16AMD64(input, dst, utf8ASCIIToUTF16BEBlocksHaswell, convertUTF8ToUTF16BEScalar)
}
func convertUTF8ToUTF16AMD64(input []byte, dst []uint16, blocks func([]byte, []uint16) int, tail func([]byte, []uint16) int) int {
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

func convertUTF8ToUTF16LEWithErrorsWestmere(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsAMD64(input, dst, utf8ASCIIToUTF16LEBlocksWestmere, convertUTF8ToUTF16LEWithErrorsScalar)
}
func convertUTF8ToUTF16BEWithErrorsWestmere(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsAMD64(input, dst, utf8ASCIIToUTF16BEBlocksWestmere, convertUTF8ToUTF16BEWithErrorsScalar)
}
func convertUTF8ToUTF16LEWithErrorsHaswell(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsAMD64(input, dst, utf8ASCIIToUTF16LEBlocksHaswell, convertUTF8ToUTF16LEWithErrorsScalar)
}
func convertUTF8ToUTF16BEWithErrorsHaswell(input []byte, dst []uint16) Result {
	return convertUTF8ToUTF16WithErrorsAMD64(input, dst, utf8ASCIIToUTF16BEBlocksHaswell, convertUTF8ToUTF16BEWithErrorsScalar)
}
func convertUTF8ToUTF16WithErrorsAMD64(input []byte, dst []uint16, blocks func([]byte, []uint16) int, tail func([]byte, []uint16) Result) Result {
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

func convertValidUTF8ToUTF16LEWestmere(input []byte, dst []uint16) int {
	return convertValidUTF8ToUTF16AMD64(input, dst, utf8ASCIIToUTF16LEBlocksWestmere, convertValidUTF8ToUTF16LEScalar)
}
func convertValidUTF8ToUTF16BEWestmere(input []byte, dst []uint16) int {
	return convertValidUTF8ToUTF16AMD64(input, dst, utf8ASCIIToUTF16BEBlocksWestmere, convertValidUTF8ToUTF16BEScalar)
}
func convertValidUTF8ToUTF16LEHaswell(input []byte, dst []uint16) int {
	return convertValidUTF8ToUTF16AMD64(input, dst, utf8ASCIIToUTF16LEBlocksHaswell, convertValidUTF8ToUTF16LEScalar)
}
func convertValidUTF8ToUTF16BEHaswell(input []byte, dst []uint16) int {
	return convertValidUTF8ToUTF16AMD64(input, dst, utf8ASCIIToUTF16BEBlocksHaswell, convertValidUTF8ToUTF16BEScalar)
}
func convertValidUTF8ToUTF16AMD64(input []byte, dst []uint16, blocks func([]byte, []uint16) int, tail func([]byte, []uint16) int) int {
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

func convertUTF8ToUTF32Westmere(input []byte, dst []uint32) int {
	return convertUTF8ToUTF32AMD64(input, dst, utf8ASCIIToUTF32BlocksWestmere)
}
func convertUTF8ToUTF32Haswell(input []byte, dst []uint32) int {
	return convertUTF8ToUTF32AMD64(input, dst, utf8ASCIIToUTF32BlocksHaswell)
}
func convertUTF8ToUTF32AMD64(input []byte, dst []uint32, blocks func([]byte, []uint32) int) int {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertUTF8ToUTF32Scalar(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertUTF8ToUTF32Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

func convertUTF8ToUTF32WithErrorsWestmere(input []byte, dst []uint32) Result {
	return convertUTF8ToUTF32WithErrorsAMD64(input, dst, utf8ASCIIToUTF32BlocksWestmere)
}
func convertUTF8ToUTF32WithErrorsHaswell(input []byte, dst []uint32) Result {
	return convertUTF8ToUTF32WithErrorsAMD64(input, dst, utf8ASCIIToUTF32BlocksHaswell)
}
func convertUTF8ToUTF32WithErrorsAMD64(input []byte, dst []uint32, blocks func([]byte, []uint32) int) Result {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertUTF8ToUTF32WithErrorsScalar(input, dst)
	}
	n := blocks(input, dst)
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

func convertValidUTF8ToUTF32Westmere(input []byte, dst []uint32) int {
	return convertValidUTF8ToUTF32AMD64(input, dst, utf8ASCIIToUTF32BlocksWestmere)
}
func convertValidUTF8ToUTF32Haswell(input []byte, dst []uint32) int {
	return convertValidUTF8ToUTF32AMD64(input, dst, utf8ASCIIToUTF32BlocksHaswell)
}
func convertValidUTF8ToUTF32AMD64(input []byte, dst []uint32, blocks func([]byte, []uint32) int) int {
	if len(dst) < utf32LengthFromUTF8Scalar(input) {
		panic("simdutf: destination is too short")
	}
	if len(input) < 16 {
		return convertValidUTF8ToUTF32Scalar(input, dst)
	}
	n := blocks(input, dst)
	if n == len(input) {
		return n
	}
	rest := convertValidUTF8ToUTF32Scalar(input[n:], dst[n:])
	if rest == 0 {
		return 0
	}
	return n + rest
}

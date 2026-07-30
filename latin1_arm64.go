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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/implementation.cpp
// and src/simdutf/arm64/simd.h. The assembly routines consume only complete,
// in-bounds NEON blocks; these wrappers retain the scalar preflight and tails
// required by the Go slice contract.

//go:noescape
func utf8LengthFromLatin1BlocksNEON(input []byte) (length int)

//go:noescape
func convertLatin1ToUTF8ASCIIPrefixNEON(input, dst []byte) (consumed int)

//go:noescape
func convertLatin1ToUTF16LEBlocksNEON(input []byte, dst []uint16) (consumed int)

//go:noescape
func convertLatin1ToUTF16BEBlocksNEON(input []byte, dst []uint16) (consumed int)

//go:noescape
func convertLatin1ToUTF32BlocksNEON(input []byte, dst []uint32) (consumed int)

func utf8LengthFromLatin1NEON(input []byte) int {
	const block = 64
	n := len(input) &^ (block - 1)
	return utf8LengthFromLatin1BlocksNEON(input[:n]) + utf8LengthFromLatin1Scalar(input[n:])
}

func convertLatin1ToUTF8NEON(input, dst []byte) int {
	needed := utf8LengthFromLatin1NEON(input)
	if len(dst) < needed {
		panic("simdutf: Latin-1 to UTF-8 destination too short")
	}
	// A 64-byte ASCII block is already its UTF-8 representation. The NEON
	// kernel copies such blocks and stops before the first non-ASCII block,
	// leaving the variable-width portion to the scalar reference routine.
	n := convertLatin1ToUTF8ASCIIPrefixNEON(input, dst)
	return n + convertLatin1ToUTF8Scalar(input[n:], dst[n:])
}

func convertLatin1ToUTF16LENEON(input []byte, dst []uint16) int {
	if len(dst) < len(input) {
		panic("simdutf: Latin-1 to UTF-16 destination too short")
	}
	const block = 32
	n := len(input) &^ (block - 1)
	consumed := convertLatin1ToUTF16LEBlocksNEON(input[:n], dst[:n])
	return consumed + convertLatin1ToUTF16LEScalar(input[consumed:], dst[consumed:])
}

func convertLatin1ToUTF16BENEON(input []byte, dst []uint16) int {
	if len(dst) < len(input) {
		panic("simdutf: Latin-1 to UTF-16 destination too short")
	}
	const block = 32
	n := len(input) &^ (block - 1)
	consumed := convertLatin1ToUTF16BEBlocksNEON(input[:n], dst[:n])
	return consumed + convertLatin1ToUTF16BEScalar(input[consumed:], dst[consumed:])
}

func convertLatin1ToUTF32NEON(input []byte, dst []uint32) int {
	if len(dst) < len(input) {
		panic("simdutf: Latin-1 to UTF-32 destination too short")
	}
	const block = 16
	n := len(input) &^ (block - 1)
	consumed := convertLatin1ToUTF32BlocksNEON(input[:n], dst[:n])
	return consumed + convertLatin1ToUTF32Scalar(input[consumed:], dst[consumed:])
}

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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/base64lengths.h,
// src/westmere/sse_base64.cpp, src/haswell/avx2_base64.cpp, and the
// westmere/haswell Base64 entry points in src/{westmere,haswell}/implementation.cpp.
//
// Staged hybrid: length and encode use real SSSE3/AVX2 assembly for complete
// vector groups; decode/details remain forceable symbols with scalar bodies
// until a later decode landing. Public selection stays scalar-first.

//go:noescape
func binaryLengthFromBase64BlocksWestmere(input []byte) (count int)

//go:noescape
func binaryLengthFromBase64BlocksHaswell(input []byte) (count int)

//go:noescape
func binaryLengthFromBase64UTF16BlocksWestmere(input []uint16) (count int)

//go:noescape
func binaryLengthFromBase64UTF16BlocksHaswell(input []uint16) (count int)

//go:noescape
func base64EncodeBlocksWestmere(input, dst []byte, url int) (consumed, written int)

//go:noescape
func base64EncodeBlocksHaswell(input, dst []byte, url int) (consumed, written int)

func binaryLengthFromBase64Westmere(input []byte) int {
	return binaryLengthFromBase64AMD64(input, binaryLengthFromBase64BlocksWestmere)
}

func binaryLengthFromBase64Haswell(input []byte) int {
	return binaryLengthFromBase64AMD64(input, binaryLengthFromBase64BlocksHaswell)
}

func binaryLengthFromBase64AMD64(input []byte, blocks func([]byte) int) int {
	complete := len(input) &^ 63
	count := 0
	if complete != 0 {
		count = blocks(input[:complete])
	}
	for _, c := range input[complete:] {
		if c > ' ' {
			count++
		}
	}
	padding := 0
	pos := len(input)
	for pos > 0 && padding < 2 {
		pos--
		c := input[pos]
		if c == '=' {
			padding++
		} else if c > ' ' {
			break
		}
	}
	return ((count - padding) * 3) / 4
}

func binaryLengthFromBase64UTF16Westmere(input []uint16) int {
	return binaryLengthFromBase64UTF16AMD64(input, binaryLengthFromBase64UTF16BlocksWestmere)
}

func binaryLengthFromBase64UTF16Haswell(input []uint16) int {
	return binaryLengthFromBase64UTF16AMD64(input, binaryLengthFromBase64UTF16BlocksHaswell)
}

func binaryLengthFromBase64UTF16AMD64(input []uint16, blocks func([]uint16) int) int {
	complete := len(input) &^ 31
	count := 0
	if complete != 0 {
		count = blocks(input[:complete])
	}
	for _, c := range input[complete:] {
		if c > ' ' {
			count++
		}
	}
	padding := 0
	pos := len(input)
	for pos > 0 && padding < 2 {
		pos--
		c := input[pos]
		if c == '=' {
			padding++
		} else if c > ' ' {
			break
		}
	}
	return ((count - padding) * 3) / 4
}

func base64EncodeURLFlag(options Base64Options) int {
	if options&Base64URL != 0 {
		return 1
	}
	return 0
}

func binaryToBase64Westmere(input, dst []byte, options Base64Options) int {
	return binaryToBase64AMD64(input, dst, options, base64EncodeBlocksWestmere)
}

func binaryToBase64Haswell(input, dst []byte, options Base64Options) int {
	return binaryToBase64AMD64(input, dst, options, base64EncodeBlocksHaswell)
}

func binaryToBase64AMD64(input, dst []byte, options Base64Options, blocks func([]byte, []byte, int) (int, int)) int {
	required := base64LengthFromBinaryScalar(len(input), options)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	consumed, written := blocks(input, dst, base64EncodeURLFlag(options))
	if consumed >= len(input) {
		return written
	}
	return written + tailEncodeBase64(dst[written:], input[consumed:], options, false, 0)
}

func binaryToBase64WithLinesWestmere(input, dst []byte, lineLength int, options Base64Options) int {
	return binaryToBase64WithLinesAMD64(input, dst, lineLength, options, base64EncodeBlocksWestmere)
}

func binaryToBase64WithLinesHaswell(input, dst []byte, lineLength int, options Base64Options) int {
	return binaryToBase64WithLinesAMD64(input, dst, lineLength, options, base64EncodeBlocksHaswell)
}

func binaryToBase64WithLinesAMD64(input, dst []byte, lineLength int, options Base64Options, blocks func([]byte, []byte, int) (int, int)) int {
	required := base64LengthFromBinaryWithLinesScalar(len(input), options, lineLength)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	if lineLength < 4 {
		lineLength = 4
	}
	url := base64EncodeURLFlag(options)
	out := 0
	col := 0
	inOff := 0
	for {
		var buf [128]byte
		consumed, written := blocks(input[inOff:], buf[:], url)
		if consumed == 0 {
			break
		}
		for j := 0; j < written; j++ {
			if col == lineLength {
				dst[out] = '\n'
				out++
				col = 0
			}
			dst[out] = buf[j]
			out++
			col++
		}
		inOff += consumed
	}
	if inOff < len(input) {
		var rem [72]byte
		n := tailEncodeBase64(rem[:], input[inOff:], options, false, 0)
		for j := 0; j < n; j++ {
			if col == lineLength {
				dst[out] = '\n'
				out++
				col = 0
			}
			dst[out] = rem[j]
			out++
			col++
		}
	}
	return out
}

// Decode/details stay forceable and distinct from scalar; SIMD decode lands later.

//go:noinline
func base64ToBinaryWestmere(input []byte, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return base64ToBinaryScalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryHaswell(input []byte, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return base64ToBinaryScalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryUTF16Westmere(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return base64ToBinaryUTF16Scalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryUTF16Haswell(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return base64ToBinaryUTF16Scalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryDetailsWestmere(input []byte, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsScalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryDetailsHaswell(input []byte, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsScalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryDetailsUTF16Westmere(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsUTF16Scalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryDetailsUTF16Haswell(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsUTF16Scalar(input, dst, options, lastChunk)
}

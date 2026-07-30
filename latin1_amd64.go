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

//go:build amd64

package simdutf

// Independently translated from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/
// sse_convert_latin1_to_utf8.cpp, sse_convert_latin1_to_utf16.cpp,
// sse_convert_latin1_to_utf32.cpp, and the matching haswell AVX2 files.
// Assembly handles only complete vector groups; these wrappers retain the
// scalar tail and the all-or-nothing destination preflight required by Go.

//go:noescape
func latin1UTF8ASCIIBlocksWestmere(input, dst []byte) (consumed int)

//go:noescape
func latin1UTF8ASCIIBlocksHaswell(input, dst []byte) (consumed int)

//go:noescape
func latin1UTF16LEBlocksWestmere(input []byte, dst []uint16) (consumed int)

//go:noescape
func latin1UTF16BEBlocksWestmere(input []byte, dst []uint16) (consumed int)

//go:noescape
func latin1UTF16LEBlocksHaswell(input []byte, dst []uint16) (consumed int)

//go:noescape
func latin1UTF16BEBlocksHaswell(input []byte, dst []uint16) (consumed int)

//go:noescape
func latin1UTF32BlocksWestmere(input []byte, dst []uint32) (consumed int)

//go:noescape
func latin1UTF32BlocksHaswell(input []byte, dst []uint32) (consumed int)

//go:noescape
func latin1HighByteBlocksWestmere(input []byte) (count int)

//go:noescape
func latin1HighByteBlocksHaswell(input []byte) (count int)

func convertLatin1ToUTF8Westmere(input, dst []byte) int {
	if len(dst) < utf8LengthFromLatin1Westmere(input) {
		panic("simdutf: destination too short")
	}
	return convertLatin1ToUTF8AMD64(input, dst, latin1UTF8ASCIIBlocksWestmere)
}
func convertLatin1ToUTF8Haswell(input, dst []byte) int {
	if len(dst) < utf8LengthFromLatin1Haswell(input) {
		panic("simdutf: destination too short")
	}
	return convertLatin1ToUTF8AMD64(input, dst, latin1UTF8ASCIIBlocksHaswell)
}
func convertLatin1ToUTF8AMD64(input, dst []byte, blocks func([]byte, []byte) int) int {
	i, written := 0, 0
	for i < len(input) {
		n := blocks(input[i:], dst[written:])
		i, written = i+n, written+n
		if i == len(input) {
			break
		}
		c := input[i]
		i++
		if c < 0x80 {
			dst[written] = c
			written++
		} else {
			dst[written], dst[written+1] = 0xc0|(c>>6), 0x80|(c&0x3f)
			written += 2
		}
	}
	return written
}

func convertLatin1ToUTF16LEWestmere(input []byte, dst []uint16) int {
	return convertLatin1ToUTF16AMD64(input, dst, latin1UTF16LEBlocksWestmere, convertLatin1ToUTF16LEScalar)
}
func convertLatin1ToUTF16BEWestmere(input []byte, dst []uint16) int {
	return convertLatin1ToUTF16AMD64(input, dst, latin1UTF16BEBlocksWestmere, convertLatin1ToUTF16BEScalar)
}
func convertLatin1ToUTF16LEHaswell(input []byte, dst []uint16) int {
	return convertLatin1ToUTF16AMD64(input, dst, latin1UTF16LEBlocksHaswell, convertLatin1ToUTF16LEScalar)
}
func convertLatin1ToUTF16BEHaswell(input []byte, dst []uint16) int {
	return convertLatin1ToUTF16AMD64(input, dst, latin1UTF16BEBlocksHaswell, convertLatin1ToUTF16BEScalar)
}
func convertLatin1ToUTF16AMD64(input []byte, dst []uint16, blocks func([]byte, []uint16) int, tail func([]byte, []uint16) int) int {
	if len(dst) < len(input) {
		panic("simdutf: destination too short")
	}
	n := blocks(input, dst)
	return n + tail(input[n:], dst[n:])
}

func convertLatin1ToUTF32Westmere(input []byte, dst []uint32) int {
	return convertLatin1ToUTF32AMD64(input, dst, latin1UTF32BlocksWestmere)
}
func convertLatin1ToUTF32Haswell(input []byte, dst []uint32) int {
	return convertLatin1ToUTF32AMD64(input, dst, latin1UTF32BlocksHaswell)
}
func convertLatin1ToUTF32AMD64(input []byte, dst []uint32, blocks func([]byte, []uint32) int) int {
	if len(dst) < len(input) {
		panic("simdutf: destination too short")
	}
	n := blocks(input, dst)
	return n + convertLatin1ToUTF32Scalar(input[n:], dst[n:])
}

func utf8LengthFromLatin1Westmere(input []byte) int {
	return latin1LengthAMD64(input, latin1HighByteBlocksWestmere, 64)
}
func utf8LengthFromLatin1Haswell(input []byte) int {
	return latin1LengthAMD64(input, latin1HighByteBlocksHaswell, 128)
}
func latin1LengthAMD64(input []byte, blocks func([]byte) int, block int) int {
	n := len(input) &^ (block - 1)
	high := blocks(input[:n])
	return len(input) + high + utf8LengthFromLatin1Scalar(input[n:]) - len(input[n:])
}

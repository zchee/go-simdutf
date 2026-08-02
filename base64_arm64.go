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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_base64.cpp and
// src/arm64/implementation.cpp Base64 entry points. Assembly owns the NEON
// block kernels; Go wrappers retain scalar tails, padding, and garbage paths.
// Providers stay forceable and scalar-first until qualification promotes.

//go:noescape
func countBase64SignificantBytesNEON(input []byte) int

//go:noescape
func countBase64SignificantUTF16NEON(input []uint16) int

//go:noescape
func binaryToBase64BlocksDefaultNEON(input, dst []byte)

//go:noescape
func binaryToBase64BlocksURLNEON(input, dst []byte)

//go:noescape
func base64DecodeBlocksNEON(input, dst []byte)

func base64URLAlphabet(options Base64Options) bool {
	return options&Base64URL != 0
}

func base64PaddingCountByte(input []byte) int {
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
	return padding
}

func base64PaddingCountUTF16(input []uint16) int {
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
	return padding
}

func binaryLengthFromBase64NEON(input []byte) int {
	const block = 64
	count := 0
	n := len(input) &^ (block - 1)
	if n >= block {
		count = countBase64SignificantBytesNEON(input[:n])
	}
	for _, c := range input[n:] {
		if c > ' ' {
			count++
		}
	}
	padding := base64PaddingCountByte(input)
	return ((count - padding) * 3) / 4
}

func binaryLengthFromBase64UTF16NEON(input []uint16) int {
	const block = 32
	count := 0
	n := len(input) &^ (block - 1)
	if n >= block {
		count = countBase64SignificantUTF16NEON(input[:n])
	}
	for _, c := range input[n:] {
		if c > ' ' {
			count++
		}
	}
	padding := base64PaddingCountUTF16(input)
	return ((count - padding) * 3) / 4
}

func binaryToBase64NEON(input, dst []byte, options Base64Options) int {
	required := base64LengthFromBinaryScalar(len(input), options)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	const block = 48
	out := 0
	n := len(input) / block * block
	if n >= block {
		if base64URLAlphabet(options) {
			binaryToBase64BlocksURLNEON(input[:n], dst)
		} else {
			binaryToBase64BlocksDefaultNEON(input[:n], dst)
		}
		out = n / 3 * 4
		input = input[n:]
		dst = dst[out:]
	}
	return out + tailEncodeBase64(dst, input, options, false, 0)
}

func binaryToBase64WithLinesNEON(input, dst []byte, lineLength int, options Base64Options) int {
	required := base64LengthFromBinaryWithLinesScalar(len(input), options, lineLength)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	if lineLength < 4 {
		lineLength = 4
	}
	// Encode with the NEON block kernel, then insert newlines in Go. This keeps
	// the provider on a real NEON path while matching the scalar line layout
	// (newlines between lines, none after a final short line).
	tmp := make([]byte, base64LengthFromBinaryScalar(len(input), options))
	n := binaryToBase64NEON(input, tmp, options)
	out := 0
	for i := range n {
		if i > 0 && i%lineLength == 0 {
			dst[out] = '\n'
			out++
		}
		dst[out] = tmp[i]
		out++
	}
	return out
}

func base64ToBinaryNEON(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	required := maximalBinaryLengthFromBase64Scalar(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	return base64ToBinaryDetailsNEON(input, dst, options, lastChunk).Result()
}

func base64ToBinaryUTF16NEON(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	required := maximalBinaryLengthFromBase64UTF16Scalar(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	return base64ToBinaryDetailsUTF16NEON(input, dst, options, lastChunk).Result()
}

func base64ToBinaryDetailsNEON(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	if fr, ok := base64ToBinaryDetailsNEONContiguous(input, dst, options, lastChunk); ok {
		return fr
	}
	return base64ToBinaryDetailsScalar(input, dst, options, lastChunk)
}

func base64ToBinaryDetailsUTF16NEON(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	if fr, ok := base64ToBinaryDetailsUTF16NEONContiguous(input, dst, options, lastChunk); ok {
		return fr
	}
	return base64ToBinaryDetailsUTF16Scalar(input, dst, options, lastChunk)
}

// base64ToBinaryDetailsNEONContiguous accelerates contiguous valid Base64 (no
// ignorable bytes in the payload) with NEON 64→48 decode blocks. Whitespace,
// garbage-accept, and mixed payloads fall back to the scalar oracle.
func base64ToBinaryDetailsNEONContiguous(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) (FullResult, bool) {
	if base64IgnoreGarbage(options) || len(input) < 64 {
		return FullResult{}, false
	}
	ri := findEndBase64Byte(input, options)
	equalsigns := ri.equalSigns
	equallocation := ri.equalLocation
	length := ri.srcLen
	fullInputLength := ri.fullInputLength
	if length < 64 {
		return FullResult{}, false
	}
	toBase64 := base64ToValueTable(options)
	for i := range length {
		if toBase64[input[i]] > 63 {
			return FullResult{}, false
		}
	}

	buf := make([]byte, length)
	for i := range length {
		buf[i] = toBase64[input[i]]
	}

	out := 0
	blocks := length &^ 63
	if blocks >= 64 {
		base64DecodeBlocksNEON(buf[:blocks], dst)
		out = blocks / 4 * 3
	}
	r := base64TailDecodeByte(dst[out:], input[blocks:length], equalsigns, options, lastChunk, false)
	r = patchTailResult(r, blocks, out, equallocation, fullInputLength, lastChunk)
	if !IsPartial(lastChunk) && r.Error == Success && equalsigns > 0 {
		if (r.OutputCount%3 == 0) || ((r.OutputCount%3)+1+equalsigns != 4) {
			return FullResult{Error: InvalidBase64Character, InputCount: equallocation, OutputCount: r.OutputCount, PaddingError: true}, true
		}
	}
	if IsPartial(lastChunk) && r.Error == Success && r.InputCount < fullInputLength {
		for r.InputCount < fullInputLength && isIgnorableByte(input[r.InputCount], options) {
			r.InputCount++
		}
		if r.InputCount < fullInputLength {
			for r.InputCount > 0 && isIgnorableByte(input[r.InputCount-1], options) {
				r.InputCount--
			}
		}
	}
	return r, true
}

func base64ToBinaryDetailsUTF16NEONContiguous(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) (FullResult, bool) {
	if base64IgnoreGarbage(options) || len(input) < 64 {
		return FullResult{}, false
	}
	ri := findEndBase64UTF16(input, options)
	equalsigns := ri.equalSigns
	equallocation := ri.equalLocation
	length := ri.srcLen
	fullInputLength := ri.fullInputLength
	if length < 64 {
		return FullResult{}, false
	}
	toBase64 := base64ToValueTable(options)
	for i := range length {
		c := input[i]
		if !isEightByteUTF16(c) || toBase64[byte(c)] > 63 {
			return FullResult{}, false
		}
	}

	buf := make([]byte, length)
	for i := range length {
		buf[i] = toBase64[byte(input[i])]
	}

	out := 0
	blocks := length &^ 63
	if blocks >= 64 {
		base64DecodeBlocksNEON(buf[:blocks], dst)
		out = blocks / 4 * 3
	}
	r := base64TailDecodeUTF16(dst[out:], input[blocks:length], equalsigns, options, lastChunk, false)
	r = patchTailResult(r, blocks, out, equallocation, fullInputLength, lastChunk)
	if !IsPartial(lastChunk) && r.Error == Success && equalsigns > 0 {
		if (r.OutputCount%3 == 0) || ((r.OutputCount%3)+1+equalsigns != 4) {
			return FullResult{Error: InvalidBase64Character, InputCount: equallocation, OutputCount: r.OutputCount, PaddingError: true}, true
		}
	}
	if IsPartial(lastChunk) && r.Error == Success && r.InputCount < fullInputLength {
		for r.InputCount < fullInputLength && isIgnorableUTF16(input[r.InputCount], options) {
			r.InputCount++
		}
		if r.InputCount < fullInputLength {
			for r.InputCount > 0 && isIgnorableUTF16(input[r.InputCount-1], options) {
				r.InputCount--
			}
		}
	}
	return r, true
}

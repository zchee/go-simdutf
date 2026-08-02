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

package simdutf

// Public Base64 API adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/implementation.h and include/simdutf/scalar/base64.h.
// Scalar-first selection via activeImplementation until qualification dispositions promote.

// MaximalBinaryLengthFromBase64 returns a fast upper bound on decoded bytes.
func MaximalBinaryLengthFromBase64(input []byte) int {
	return activeImplementation.maximalBinaryLengthFromBase64(input)
}

// MaximalBinaryLengthFromBase64UTF16 returns a fast upper bound on decoded bytes.
func MaximalBinaryLengthFromBase64UTF16(input []uint16) int {
	return activeImplementation.maximalBinaryLengthFromBase64UTF16(input)
}

// BinaryLengthFromBase64 returns the decoded binary length for base64 input.
func BinaryLengthFromBase64(input []byte) int {
	return activeImplementation.binaryLengthFromBase64(input)
}

// BinaryLengthFromBase64UTF16 returns the decoded binary length for UTF-16 base64 input.
func BinaryLengthFromBase64UTF16(input []uint16) int {
	return activeImplementation.binaryLengthFromBase64UTF16(input)
}

// Base64ToBinary decodes base64 input into dst. It panics before writing when
// dst is shorter than MaximalBinaryLengthFromBase64(input).
func Base64ToBinary(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return activeImplementation.base64ToBinary(input, dst, options, lastChunk)
}

// Base64ToBinaryUTF16 decodes UTF-16 base64 input into dst. It panics before
// writing when dst is shorter than MaximalBinaryLengthFromBase64UTF16(input).
func Base64ToBinaryUTF16(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return activeImplementation.base64ToBinaryUTF16(input, dst, options, lastChunk)
}

// Base64LengthFromBinary returns the base64 length for a binary length.
func Base64LengthFromBinary(length int, options Base64Options) int {
	return base64LengthFromBinaryScalar(length, options)
}

// Base64LengthFromBinaryWithLines returns the base64 length including newlines.
func Base64LengthFromBinaryWithLines(length int, options Base64Options, lineLength int) int {
	return base64LengthFromBinaryWithLinesScalar(length, options, lineLength)
}

// BinaryToBase64 encodes input into dst. It panics before writing when dst is
// shorter than Base64LengthFromBinary(len(input), options).
func BinaryToBase64(input, dst []byte, options Base64Options) int {
	return activeImplementation.binaryToBase64(input, dst, options)
}

// BinaryToBase64WithLines encodes input into dst with line breaks.
func BinaryToBase64WithLines(input, dst []byte, lineLength int, options Base64Options) int {
	return activeImplementation.binaryToBase64WithLines(input, dst, lineLength, options)
}

// Base64ToBinaryDetails decodes base64 input and returns FullResult details.
// It panics before writing when dst is shorter than MaximalBinaryLengthFromBase64(input).
func Base64ToBinaryDetails(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	required := MaximalBinaryLengthFromBase64(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	return activeImplementation.base64ToBinaryDetails(input, dst, options, lastChunk)
}

// Base64ToBinaryDetailsUTF16 decodes UTF-16 base64 input and returns FullResult details.
func Base64ToBinaryDetailsUTF16(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	required := MaximalBinaryLengthFromBase64UTF16(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	return activeImplementation.base64ToBinaryDetailsUTF16(input, dst, options, lastChunk)
}

// Base64Ignorable reports whether value is ignorable under options.
func Base64Ignorable(value byte, options Base64Options) bool {
	return isIgnorableByte(value, options)
}

// Base64IgnorableUTF16 reports whether value is ignorable under options.
func Base64IgnorableUTF16(value uint16, options Base64Options) bool {
	return isIgnorableUTF16(value, options)
}

// Base64Valid reports whether value is a base64 alphabet character.
func Base64Valid(value byte, options Base64Options) bool {
	return isBase64Byte(value, options)
}

// Base64ValidUTF16 reports whether value is a base64 alphabet character.
func Base64ValidUTF16(value uint16, options Base64Options) bool {
	return isBase64UTF16(value, options)
}

// Base64ValidOrPadding reports whether value is base64 or '=' padding.
func Base64ValidOrPadding(value byte, options Base64Options) bool {
	return isBase64OrPaddingByte(value, options)
}

// Base64ValidOrPaddingUTF16 reports whether value is base64 or '=' padding.
func Base64ValidOrPaddingUTF16(value uint16, options Base64Options) bool {
	return isBase64OrPaddingUTF16(value, options)
}

// Base64ToBinarySafe decodes as much as fits in dst without panicking on short output.
func Base64ToBinarySafe(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions, decodeUpToBadChar bool) (Result, int) {
	return base64ToBinarySafeScalar(input, dst, options, lastChunk, decodeUpToBadChar)
}

// Base64ToBinarySafeUTF16 decodes UTF-16 base64 as much as fits in dst.
func Base64ToBinarySafeUTF16(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions, decodeUpToBadChar bool) (Result, int) {
	return base64ToBinarySafeUTF16Scalar(input, dst, options, lastChunk, decodeUpToBadChar)
}

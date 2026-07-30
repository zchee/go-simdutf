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

// Independently adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// arm64 Base64 entry points in src/arm64/implementation.cpp.
// These providers are forceable and stay behind scalar until qualification promotes.

//go:noinline
func binaryLengthFromBase64NEON(input []byte) int {
	return binaryLengthFromBase64Scalar(input)
}

//go:noinline
func binaryLengthFromBase64UTF16NEON(input []uint16) int {
	return binaryLengthFromBase64UTF16Scalar(input)
}

//go:noinline
func base64ToBinaryNEON(input []byte, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return base64ToBinaryScalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryUTF16NEON(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return base64ToBinaryUTF16Scalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryDetailsNEON(input []byte, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsScalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryDetailsUTF16NEON(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsUTF16Scalar(input, dst, options, lastChunk)
}

//go:noinline
func binaryToBase64NEON(input, dst []byte, options Base64Options) int {
	return binaryToBase64Scalar(input, dst, options)
}

//go:noinline
func binaryToBase64WithLinesNEON(input, dst []byte, lineLength int, options Base64Options) int {
	return binaryToBase64WithLinesScalar(input, dst, lineLength, options)
}

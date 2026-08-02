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

//go:build amd64 && !goexperiment.simd

package simdutf

// Archsimd Base64 providers without GOEXPERIMENT=simd: forceable scalar stubs
// that keep the eight dispatch cells distinct until the simd experiment is on.
// With goexperiment.simd, base64_archsimd.go owns Contiguous AVX2 decode/encode.

//go:noinline
func binaryLengthFromBase64Archsimd(input []byte) int {
	return binaryLengthFromBase64Scalar(input)
}

//go:noinline
func binaryLengthFromBase64UTF16Archsimd(input []uint16) int {
	return binaryLengthFromBase64UTF16Scalar(input)
}

//go:noinline
func base64ToBinaryArchsimd(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return base64ToBinaryScalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryUTF16Archsimd(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	return base64ToBinaryUTF16Scalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryDetailsArchsimd(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsScalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryDetailsUTF16Archsimd(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsUTF16Scalar(input, dst, options, lastChunk)
}

//go:noinline
func binaryToBase64Archsimd(input, dst []byte, options Base64Options) int {
	return binaryToBase64Scalar(input, dst, options)
}

//go:noinline
func binaryToBase64WithLinesArchsimd(input, dst []byte, lineLength int, options Base64Options) int {
	return binaryToBase64WithLinesScalar(input, dst, lineLength, options)
}

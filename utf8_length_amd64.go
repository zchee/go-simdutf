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

const (
	utf16LengthFromUTF8DispatchCutoff = 16
	utf32LengthFromUTF8DispatchCutoff = 64
)

// Independently translated from
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811):
// src/generic/utf8/utf16_length_from_utf8_bytemask.h,
// src/generic/utf8.h:8-20, src/westmere/implementation.cpp:1142-1154,
// 1244-1247,1319-1323, src/haswell/implementation.cpp:1115-1127,
// 1155-1158,1288-1292, and the corresponding target simd.h files. Raw
// kernels mask their lengths internally; wrappers preserve the pinned scalar
// tails and return 64-bit totals through Go's amd64 int.

func latin1LengthFromUTF8Westmere(input []byte) int {
	return countUTF8Westmere(input)
}

func latin1LengthFromUTF8Haswell(input []byte) int {
	return countUTF8Haswell(input)
}

//go:noescape
func utf16LengthFromUTF8BlocksWestmere(input []byte) (length int)

//go:noescape
func utf16LengthFromUTF8BlocksHaswell(input []byte) (length int)

func utf16LengthFromUTF8Westmere(input []byte) int {
	complete := len(input) &^ 15
	if complete == 0 {
		return utf16LengthFromUTF8Scalar(input)
	}
	return utf16LengthFromUTF8BlocksWestmere(input[:complete]) + utf16LengthFromUTF8Scalar(input[complete:])
}

func utf16LengthFromUTF8Haswell(input []byte) int {
	complete := len(input) &^ 31
	if complete == 0 {
		return utf16LengthFromUTF8Scalar(input)
	}
	return utf16LengthFromUTF8BlocksHaswell(input[:complete]) + utf16LengthFromUTF8Scalar(input[complete:])
}

//go:noescape
func utf32LengthFromUTF8BlocksWestmere(input []byte) (length int)

//go:noescape
func utf32LengthFromUTF8BlocksHaswell(input []byte) (length int)

func utf32LengthFromUTF8Westmere(input []byte) int {
	complete := len(input) &^ 63
	if complete == 0 {
		return utf32LengthFromUTF8Scalar(input)
	}
	return utf32LengthFromUTF8BlocksWestmere(input[:complete]) + utf32LengthFromUTF8Scalar(input[complete:])
}

func utf32LengthFromUTF8Haswell(input []byte) int {
	complete := len(input) &^ 63
	if complete == 0 {
		return utf32LengthFromUTF8Scalar(input)
	}
	return utf32LengthFromUTF8BlocksHaswell(input[:complete]) + utf32LengthFromUTF8Scalar(input[complete:])
}

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

// Translated and adapted from
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811): src/generic/utf8.h:72-86,
// src/arm64/implementation.cpp:1121-1124,1178-1181,1292-1295, and
// src/simdutf/arm64/simd.h:420-529. The Latin-1 and UTF-32 routes are exactly
// the pinned count_utf8/count_code_points route. The UTF-16 assembly kernel
// reads complete 64-byte blocks only, and the Go wrapper uses the pinned
// scalar oracle for the tail.

func latin1LengthFromUTF8NEON(input []byte) int {
	return countUTF8NEON(input)
}

//go:noescape
func utf16LengthFromUTF8BlocksNEON(input []byte) (length int)

func utf16LengthFromUTF8NEON(input []byte) int {
	complete := len(input) &^ 63
	if complete == 0 {
		return utf16LengthFromUTF8Scalar(input)
	}
	return utf16LengthFromUTF8BlocksNEON(input[:complete]) + utf16LengthFromUTF8Scalar(input[complete:])
}

func utf32LengthFromUTF8NEON(input []byte) int {
	return countUTF8NEON(input)
}

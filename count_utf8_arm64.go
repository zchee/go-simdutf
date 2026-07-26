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

// Translated and adapted from simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/utf8.h:8-17, src/arm64/implementation.cpp:1113-1117, and
// src/simdutf/arm64/simd.h:420-529. The assembly kernel reads complete
// 64-byte blocks only; the Go wrapper preserves the pinned scalar tail.

//go:noescape
func countUTF8BlocksNEON(input []byte) (count int)

func countUTF8NEON(input []byte) int {
	complete := len(input) &^ 63
	if complete == 0 {
		return countUTF8Scalar(input)
	}
	return countUTF8BlocksNEON(input[:complete]) + countUTF8Scalar(input[complete:])
}

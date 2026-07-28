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

// Independently translated from the two separate count_utf8 families in
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/utf8.h:8-68,
// src/westmere/implementation.cpp:1142-1146, and
// src/haswell/implementation.cpp:1115-1119. The raw kernels read complete
// 64-byte (Westmere) or 128-byte (Haswell) groups only; the wrappers preserve
// the pinned scalar tail and return 64-bit totals through Go's amd64 int.

//go:noescape
func countUTF8BlocksWestmere(input []byte) (count int)

//go:noescape
func countUTF8BlocksHaswell(input []byte) (count int)

func countUTF8Westmere(input []byte) int {
	complete := len(input) &^ 63
	if complete == 0 {
		return countUTF8Scalar(input)
	}
	return countUTF8BlocksWestmere(input[:complete]) + countUTF8Scalar(input[complete:])
}

func countUTF8Haswell(input []byte) int {
	complete := len(input) &^ 127
	if complete == 0 {
		return countUTF8Scalar(input)
	}
	return countUTF8BlocksHaswell(input[:complete]) + countUTF8Scalar(input[complete:])
}

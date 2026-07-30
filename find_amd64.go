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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/find.h and the
// westmere/haswell util::find entry points in src/{westmere,haswell}/implementation.cpp.
// Assembly owns the 64-byte align prologue, simd8x64/simd16x32 equality scan, and
// scalar tail; wrappers are thin linkage over the nosplit kernels.

//go:noescape
func findWestmere(input []byte, value byte) int

//go:noescape
func findHaswell(input []byte, value byte) int

//go:noescape
func findUTF16Westmere(input []uint16, value uint16) int

//go:noescape
func findUTF16Haswell(input []uint16, value uint16) int

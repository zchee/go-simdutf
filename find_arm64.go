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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_find.cpp util_find.
// The public Go APIs expose the first match as a zero-based index, or len(input)
// when absent. Assembly owns the NEON search; Go 1.26 Plan 9 arm64 lacks VSHRN,
// so matching chunks are refined with a scalar scan of the proven hit window.

//go:noescape
func findNEON(input []byte, value byte) int

//go:noescape
func findUTF16NEON(input []uint16, value uint16) int

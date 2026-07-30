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

//go:build amd64 && goexperiment.simd

package simdutf

// detectEncodingsArchsimd follows the fallback control-flow semantics from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// src/fallback/implementation.cpp:8-32 by composing the existing archsimd
// validateUTF8 / validateUTF16LE / validateUTF32 providers. Direct callers
// must satisfy the archsimd AVX2 guard. Public selection remains scalar-first;
// this symbol is forceable / direct-only until qualification dispositions
// promote it.

func detectEncodingsArchsimd(input []byte) Encoding {
	if bom := CheckBOM(input); bom != Unspecified {
		return bom
	}
	var out Encoding
	if validateUTF8Archsimd(input) {
		out |= UTF8
	}
	if len(input)%2 == 0 {
		if validateUTF16LEArchsimd(nativeUTF16UnitsFromBytes(input)) {
			out |= UTF16LE
		}
	}
	if len(input)%4 == 0 {
		if validateUTF32Archsimd(nativeUTF32UnitsFromBytes(input)) {
			out |= UTF32LE
		}
	}
	return out
}

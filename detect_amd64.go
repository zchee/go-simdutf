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

// detectEncodingsWestmere / detectEncodingsHaswell follow the fallback
// control-flow semantics from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// src/fallback/implementation.cpp:8-32 (also mirrored by arm64). This batch
// composes the existing Westmere/Haswell validateUTF8 / validateUTF16LE /
// validateUTF32 providers rather than porting the large one-pass westmere/
// haswell detect loops. Public selection remains scalar-first; these symbols
// are forceable / direct-only until qualification dispositions promote them.

func detectEncodingsWestmere(input []byte) Encoding {
	return detectEncodingsAMD64(input, validateUTF8Westmere, validateUTF16LEWestmere, validateUTF32Westmere)
}

func detectEncodingsHaswell(input []byte) Encoding {
	return detectEncodingsAMD64(input, validateUTF8Haswell, validateUTF16LEHaswell, validateUTF32Haswell)
}

func detectEncodingsAMD64(
	input []byte,
	validateUTF8 func([]byte) bool,
	validateUTF16LE func([]uint16) bool,
	validateUTF32 func([]uint32) bool,
) Encoding {
	if bom := CheckBOM(input); bom != Unspecified {
		return bom
	}
	var out Encoding
	if validateUTF8(input) {
		out |= UTF8
	}
	if len(input)%2 == 0 {
		if validateUTF16LE(nativeUTF16UnitsFromBytes(input)) {
			out |= UTF16LE
		}
	}
	if len(input)%4 == 0 {
		if validateUTF32(nativeUTF32UnitsFromBytes(input)) {
			out |= UTF32LE
		}
	}
	return out
}

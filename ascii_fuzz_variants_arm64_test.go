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

//go:build arm64

package simdutf

// Hand-authored Go-only direct fuzz registration for the assembly port pinned
// to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/ascii_validation.h:6-45 and src/arm64/arm_validate_utf16.cpp:71-91.
// It registers test functions only and adds no product behavior.

func init() {
	registerASCIIFuzzVariant(asciiFuzzVariant{
		name: "neon",
		validate: variant[func([]byte) bool]{
			value: validateASCIINEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
		withErrors: variant[func([]byte) Result]{
			value: validateASCIIWithErrorsNEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
	})
	registerUTF16ASCIIFuzzVariant(utf16ASCIIFuzzVariant{
		name: "neon",
		le: variant[func([]uint16) bool]{
			value: validateUTF16LEAsASCIINEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
		be: variant[func([]uint16) bool]{
			value: validateUTF16BEAsASCIINEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
	})
}

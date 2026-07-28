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

//go:build amd64

package simdutf

// Hand-authored Go-only direct fuzz registration for the assembly port pinned
// to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.
// It registers test functions only and adds no product behavior.

func init() {
	registerASCIIFuzzVariant(asciiFuzzVariant{
		name: "westmere",
		validate: variant[func([]byte) bool]{
			value: validateASCIIWestmere, kind: implementationWestmere,
			available: true,
		},
		withErrors: variant[func([]byte) Result]{
			value: validateASCIIWithErrorsWestmere, kind: implementationWestmere,
			available: true,
		},
	})
	registerUTF16ASCIIFuzzVariant(utf16ASCIIFuzzVariant{
		name: "westmere",
		le: variant[func([]uint16) bool]{
			value: validateUTF16LEAsASCIIWestmere, kind: implementationWestmere,
			available: true,
		},
		be: variant[func([]uint16) bool]{
			value: validateUTF16BEAsASCIIWestmere, kind: implementationWestmere,
			available: true,
		},
	})

	registerASCIIFuzzVariant(asciiFuzzVariant{
		name: "haswell",
		validate: variant[func([]byte) bool]{
			value: validateASCIIHaswell, kind: implementationHaswell,
			required: cpuAVX2, available: true,
		},
		withErrors: variant[func([]byte) Result]{
			value: validateASCIIWithErrorsHaswell, kind: implementationHaswell,
			required: cpuAVX2, available: true,
		},
	})
	registerUTF16ASCIIFuzzVariant(utf16ASCIIFuzzVariant{
		name: "haswell",
		le: variant[func([]uint16) bool]{
			value: validateUTF16LEAsASCIIHaswell, kind: implementationHaswell,
			required: cpuAVX2, available: true,
		},
		be: variant[func([]uint16) bool]{
			value: validateUTF16BEAsASCIIHaswell, kind: implementationHaswell,
			required: cpuAVX2, available: true,
		},
	})
}

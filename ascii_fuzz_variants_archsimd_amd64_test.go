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

// Hand-authored Go-only direct fuzz registration for the archsimd adaptation
// pinned to simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.
// It registers test functions only and adds no product behavior.

func init() {
	registerASCIIFuzzVariant(asciiFuzzVariant{
		name: "archsimd",
		validate: variant[func([]byte) bool]{
			value: validateASCIIArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
		withErrors: variant[func([]byte) Result]{
			value: validateASCIIWithErrorsArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
	})
	registerUTF16ASCIIFuzzVariant(utf16ASCIIFuzzVariant{
		name: "archsimd",
		le: variant[func([]uint16) bool]{
			value: validateUTF16LEAsASCIIArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
		be: variant[func([]uint16) bool]{
			value: validateUTF16BEAsASCIIArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
	})
}

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

package simdutf

// Hand-authored Go-only direct differential fuzz registry scaffolding for the
// port pinned to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.
// It defines test metadata only and adds no product behavior.

type asciiFuzzVariant struct {
	name       string
	validate   variant[func([]byte) bool]
	withErrors variant[func([]byte) Result]
}

type utf16ASCIIFuzzVariant struct {
	name string
	le   variant[func([]uint16) bool]
	be   variant[func([]uint16) bool]
}

var (
	asciiFuzzVariants      []asciiFuzzVariant
	utf16ASCIIFuzzVariants []utf16ASCIIFuzzVariant
)

func registerASCIIFuzzVariant(candidate asciiFuzzVariant) {
	if candidate.name == "" || candidate.validate.value == nil || candidate.withErrors.value == nil {
		panic("simdutf: invalid direct ASCII fuzz variant")
	}
	for _, registered := range asciiFuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct ASCII fuzz variant " + candidate.name)
		}
	}
	asciiFuzzVariants = append(asciiFuzzVariants, candidate)
}

func registerUTF16ASCIIFuzzVariant(candidate utf16ASCIIFuzzVariant) {
	if candidate.name == "" || candidate.le.value == nil || candidate.be.value == nil {
		panic("simdutf: invalid direct UTF-16 ASCII fuzz variant")
	}
	for _, registered := range utf16ASCIIFuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct UTF-16 ASCII fuzz variant " + candidate.name)
		}
	}
	utf16ASCIIFuzzVariants = append(utf16ASCIIFuzzVariants, candidate)
}

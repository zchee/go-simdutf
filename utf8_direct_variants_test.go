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

// Hand-authored Go-only direct UTF-8 benchmark registry scaffolding for the
// port pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b.
// It defines test-only variant slots and adds no product behavior.

type utf8DirectVariant struct {
	name       string
	validate   variant[func([]byte) bool]
	withErrors variant[func([]byte) Result]
}

var utf8DirectVariants []utf8DirectVariant

func registerUTF8DirectVariant(candidate utf8DirectVariant) {
	if candidate.name == "" || candidate.validate.value == nil || candidate.withErrors.value == nil {
		panic("simdutf: invalid direct UTF-8 benchmark variant")
	}
	for _, registered := range utf8DirectVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct UTF-8 benchmark variant " + candidate.name)
		}
	}
	utf8DirectVariants = append(utf8DirectVariants, candidate)
}

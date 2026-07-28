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

// Hand-authored Go-only direct UTF-8 differential fuzz registry scaffolding
// for simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de. It defines
// test metadata only and adds no product behavior.

type utf8FuzzVariant struct {
	name       string
	validate   variant[func([]byte) bool]
	withErrors variant[func([]byte) Result]
}

var utf8FuzzVariants []utf8FuzzVariant

func registerUTF8FuzzVariant(candidate utf8FuzzVariant) {
	if candidate.name == "" || candidate.validate.value == nil || candidate.withErrors.value == nil {
		panic("simdutf: invalid direct UTF-8 fuzz variant")
	}
	for _, registered := range utf8FuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct UTF-8 fuzz variant " + candidate.name)
		}
	}
	utf8FuzzVariants = append(utf8FuzzVariants, candidate)
}

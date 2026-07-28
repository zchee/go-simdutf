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

import "testing"

// Hand-authored Go-only direct CountUTF8 differential fuzz registry
// scaffolding for
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b. It defines test
// metadata only and adds no product behavior.

type countUTF8FuzzVariant struct {
	name string
	variant[func([]byte) int]
}

var countUTF8FuzzVariants []countUTF8FuzzVariant

func registerCountUTF8FuzzVariant(candidate countUTF8FuzzVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct CountUTF8 fuzz variant")
	}
	for _, registered := range countUTF8FuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct CountUTF8 fuzz variant " + candidate.name)
		}
	}
	countUTF8FuzzVariants = append(countUTF8FuzzVariants, candidate)
}

func TestRegisterCountUTF8FuzzVariant(t *testing.T) {
	saved := countUTF8FuzzVariants
	defer func() { countUTF8FuzzVariants = saved }()
	registerCountUTF8FuzzVariant(countUTF8FuzzVariant{
		name: "test-scalar",
		variant: variant[func([]byte) int]{
			value:     countUTF8Scalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := countUTF8FuzzVariants[len(countUTF8FuzzVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, countUTF8Scalar) {
		t.Fatalf("registered fuzz variant = %q %p, want test-scalar %p", got.name, got.value, countUTF8Scalar)
	}
}

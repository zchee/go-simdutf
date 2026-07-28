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

// Hand-authored Go-only direct CountUTF8 benchmark registry scaffolding for
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b. It defines
// test-only variant slots and adds no product behavior.

type countUTF8DirectVariant struct {
	name string
	variant[func([]byte) int]
}

var countUTF8DirectVariants []countUTF8DirectVariant

func registerCountUTF8DirectVariant(candidate countUTF8DirectVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct CountUTF8 benchmark variant")
	}
	for _, registered := range countUTF8DirectVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct CountUTF8 benchmark variant " + candidate.name)
		}
	}
	countUTF8DirectVariants = append(countUTF8DirectVariants, candidate)
}

func TestRegisterCountUTF8DirectVariant(t *testing.T) {
	saved := countUTF8DirectVariants
	defer func() { countUTF8DirectVariants = saved }()
	registerCountUTF8DirectVariant(countUTF8DirectVariant{
		name: "test-scalar",
		variant: variant[func([]byte) int]{
			value:     countUTF8Scalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := countUTF8DirectVariants[len(countUTF8DirectVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, countUTF8Scalar) {
		t.Fatalf("registered direct variant = %q %p, want test-scalar %p", got.name, got.value, countUTF8Scalar)
	}
}

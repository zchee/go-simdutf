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

// Hand-authored Go-only direct UTF-8 length benchmark registry scaffolding for
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// benchmarks/shortbench.cpp:29-65,419-422,493-497,520-526 and
// benchmarks/src/benchmark.cpp:167-169,999-1011. It defines test-only named
// variant slots and adds no product behavior or mutable dispatch override.

type utf8LengthDirectVariant struct {
	name   string
	latin1 variant[func([]byte) int]
	utf16  variant[func([]byte) int]
	utf32  variant[func([]byte) int]
}

var utf8LengthDirectVariants []utf8LengthDirectVariant

func registerUTF8LengthDirectVariant(candidate utf8LengthDirectVariant) {
	if candidate.name == "" || !validUTF8LengthVariantCells(candidate.latin1, candidate.utf16, candidate.utf32) {
		panic("simdutf: invalid direct UTF-8 length benchmark variant")
	}
	for _, registered := range utf8LengthDirectVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct UTF-8 length benchmark variant " + candidate.name)
		}
	}
	utf8LengthDirectVariants = append(utf8LengthDirectVariants, candidate)
}

func validUTF8LengthVariantCells(cells ...variant[func([]byte) int]) bool {
	implemented := false
	for _, cell := range cells {
		if cell.value == nil {
			if cell.available || cell.kind != implementationScalar || cell.required != 0 {
				panic("simdutf: inconsistent absent UTF-8 length variant cell")
			}
			continue
		}
		if !cell.available {
			panic("simdutf: unavailable UTF-8 length variant cell has a function")
		}
		implemented = true
	}
	return implemented
}

func TestRegisterUTF8LengthDirectVariant(t *testing.T) {
	saved := utf8LengthDirectVariants
	defer func() { utf8LengthDirectVariants = saved }()

	tests := []struct {
		name          string
		backend       string
		implemented   uint8
		availableNil  uint8
		unavailableFn uint8
		duplicate     bool
		wantPanic     bool
	}{
		{name: "partial UTF16 only", backend: "test-utf16", implemented: 1 << 1},
		{name: "partial UTF32 only", backend: "test-utf32", implemented: 1 << 2},
		{name: "mixed implemented and not applicable", backend: "test-mixed", implemented: 1<<0 | 1<<2},
		{name: "empty name", implemented: 1 << 1, wantPanic: true},
		{name: "no implemented cells", backend: "test-empty", wantPanic: true},
		{name: "available operation with nil value", backend: "test-nil", availableNil: 1 << 1, wantPanic: true},
		{name: "unavailable operation with value", backend: "test-unavailable", unavailableFn: 1 << 2, wantPanic: true},
		{name: "duplicate name", backend: "test-duplicate", implemented: 1 << 1, duplicate: true, wantPanic: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			utf8LengthDirectVariants = nil
			var invoked [3]int
			candidate := makeUTF8LengthDirectTestVariant(test.backend, test.implemented, &invoked)
			setInvalidUTF8LengthDirectTestCell(&candidate, test.availableNil, true)
			setInvalidUTF8LengthDirectTestCell(&candidate, test.unavailableFn, false)
			if test.duplicate {
				registerUTF8LengthDirectVariant(candidate)
			}

			panicked := didPanic(func() { registerUTF8LengthDirectVariant(candidate) })
			if panicked != test.wantPanic {
				t.Fatalf("register panic = %t, want %t", panicked, test.wantPanic)
			}
			if test.wantPanic {
				return
			}

			got := utf8LengthDirectVariants[len(utf8LengthDirectVariants)-1]
			input := []byte("a\xf0\x90\x8d\x88")
			want := [...]int{
				latin1LengthFromUTF8Scalar(input),
				utf16LengthFromUTF8Scalar(input),
				utf32LengthFromUTF8Scalar(input),
			}
			cells := [...]variant[func([]byte) int]{got.latin1, got.utf16, got.utf32}
			for i, cell := range cells {
				if !cell.supportedBy(selectionInput{}) {
					if cell.value != nil {
						t.Fatalf("unsupported cell %d has a function", i)
					}
					continue
				}
				if result := cell.value(input); result != want[i] {
					t.Errorf("cell %d result = %d, scalar = %d", i, result, want[i])
				}
			}
			for i := range invoked {
				wantInvoked := 0
				if test.implemented&(1<<i) != 0 {
					wantInvoked = 1
				}
				if invoked[i] != wantInvoked {
					t.Errorf("cell %d invoked %d times, want %d", i, invoked[i], wantInvoked)
				}
			}
		})
	}
}

func makeUTF8LengthDirectTestVariant(name string, implemented uint8, invoked *[3]int) utf8LengthDirectVariant {
	functions := [...]func([]byte) int{
		func(input []byte) int { invoked[0]++; return latin1LengthFromUTF8Scalar(input) },
		func(input []byte) int { invoked[1]++; return utf16LengthFromUTF8Scalar(input) },
		func(input []byte) int { invoked[2]++; return utf32LengthFromUTF8Scalar(input) },
	}
	cells := [3]variant[func([]byte) int]{}
	for i := range cells {
		if implemented&(1<<i) != 0 {
			cells[i] = variant[func([]byte) int]{value: functions[i], available: true}
		}
	}
	return utf8LengthDirectVariant{name: name, latin1: cells[0], utf16: cells[1], utf32: cells[2]}
}

func setInvalidUTF8LengthDirectTestCell(candidate *utf8LengthDirectVariant, mask uint8, available bool) {
	if mask == 0 {
		return
	}
	cell := variant[func([]byte) int]{available: available}
	if !available {
		cell.value = utf32LengthFromUTF8Scalar
	}
	switch mask {
	case 1 << 0:
		candidate.latin1 = cell
	case 1 << 1:
		candidate.utf16 = cell
	case 1 << 2:
		candidate.utf32 = cell
	}
}

func didPanic(run func()) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	run()
	return false
}

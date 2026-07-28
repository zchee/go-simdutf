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

import (
	"bytes"
	"os"
	"testing"
)

// Hand-authored Go-only test scaffolding for the port pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b. This file defines
// test guards, direct-variant invocation, and provenance enforcement only; it
// does not define product behavior or port upstream algorithm vectors.

type guardedElement interface {
	~byte | ~uint16 | ~uint32
}

type guardedSlice[T guardedElement] struct {
	storage    []T
	body       []T
	prefixSize int
	suffixSize int
	sentinel   T
}

func newGuardedSlice[T guardedElement](prefixSize, bodySize, suffixSize int, sentinel T) guardedSlice[T] {
	storage := make([]T, prefixSize+bodySize+suffixSize)
	for i := range storage {
		storage[i] = sentinel
	}
	return guardedSlice[T]{
		storage:    storage,
		body:       storage[prefixSize : prefixSize+bodySize],
		prefixSize: prefixSize,
		suffixSize: suffixSize,
		sentinel:   sentinel,
	}
}

func (guard guardedSlice[T]) canariesIntact() bool {
	for _, value := range guard.storage[:guard.prefixSize] {
		if value != guard.sentinel {
			return false
		}
	}
	for _, value := range guard.storage[len(guard.storage)-guard.suffixSize:] {
		if value != guard.sentinel {
			return false
		}
	}
	return true
}

func (guard guardedSlice[T]) requireCanariesIntact(tb testing.TB) {
	tb.Helper()
	if !guard.canariesIntact() {
		tb.Error("guarded slice prefix or suffix canary was modified")
	}
}

type directVariant struct {
	name string
	variant[func()]
}

func invokeRunnableVariants(tb testing.TB, input selectionInput, variants ...directVariant) {
	tb.Helper()
	seen := make(map[string]struct{}, len(variants))
	runnable := 0
	for _, candidate := range variants {
		if candidate.name == "" {
			tb.Fatal("direct variant has an empty name")
		}
		if _, exists := seen[candidate.name]; exists {
			tb.Fatalf("duplicate direct variant name %q", candidate.name)
		}
		seen[candidate.name] = struct{}{}
		if !candidate.variant.supportedBy(input) {
			continue
		}
		if candidate.value == nil {
			tb.Fatalf("runnable direct variant %q has no function", candidate.name)
		}
		candidate.value()
		runnable++
	}
	if runnable == 0 {
		tb.Fatal("direct variant table has no runnable entry")
	}
}

type provenanceExpectation struct {
	name              string
	requiredFragments []string
}

func requireProvenance(tb testing.TB, expectations ...provenanceExpectation) {
	tb.Helper()
	for _, file := range expectations {
		contents, err := os.ReadFile(file.name)
		if err != nil {
			tb.Errorf("read %s: %v", file.name, err)
			continue
		}
		for _, fragment := range file.requiredFragments {
			if !bytes.Contains(contents, []byte(fragment)) {
				tb.Errorf("%s does not record required provenance fragment %q", file.name, fragment)
			}
		}
	}
}

func TestGuardedSliceCanaries(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{name: "byte", run: testGuardedSliceCanaries[byte](0xa5)},
		{name: "uint16", run: testGuardedSliceCanaries[uint16](0xa55a)},
		{name: "uint32", run: testGuardedSliceCanaries[uint32](0xa55aa55a)},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func testGuardedSliceCanaries[T guardedElement](sentinel T) func(*testing.T) {
	return func(t *testing.T) {
		guard := newGuardedSlice(2, 3, 2, sentinel)
		if !guard.canariesIntact() {
			t.Fatal("fresh guarded slice has corrupt canaries")
		}
		guard.requireCanariesIntact(t)

		guard.storage[0] = 0
		if guard.canariesIntact() {
			t.Error("prefix corruption was not detected")
		}
		guard.storage[0] = sentinel

		guard.storage[len(guard.storage)-1] = 0
		if guard.canariesIntact() {
			t.Error("suffix corruption was not detected")
		}
	}
}

func TestInvokeRunnableVariants(t *testing.T) {
	var invoked []string
	invokeRunnableVariants(
		t, selectionInput{features: cpuSSE42},
		directVariant{name: "first", variant: variant[func()]{
			value:     func() { invoked = append(invoked, "first") },
			kind:      implementationWestmere,
			required:  cpuSSE42,
			available: true,
		}},
		directVariant{name: "unavailable", variant: variant[func()]{
			value:     func() { panic("unavailable direct variant was invoked") },
			kind:      implementationScalar,
			available: false,
		}},
		directVariant{name: "unsupported", variant: variant[func()]{
			value:     func() { panic("unsupported direct variant was invoked") },
			kind:      implementationHaswell,
			required:  cpuAVX2,
			available: true,
		}},
		directVariant{name: "second", variant: variant[func()]{
			value:     func() { invoked = append(invoked, "second") },
			kind:      implementationScalar,
			available: true,
		}},
	)

	want := []string{"first", "second"}
	if len(invoked) != len(want) {
		t.Fatalf("invoked variants = %v, want %v", invoked, want)
	}
	for i := range want {
		if invoked[i] != want[i] {
			t.Fatalf("invoked variants = %v, want %v", invoked, want)
		}
	}
}

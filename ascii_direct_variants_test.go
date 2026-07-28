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

// Hand-authored Go-only benchmark registry scaffolding for the port pinned to
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de and
// docs/porting/benchmark-contract.md. This file provides test-only direct
// variant slots; it defines no product behavior and translates no upstream
// algorithm.

type asciiBoolBenchmarkVariant struct {
	name string
	variant[func([]byte) bool]
}

type asciiResultBenchmarkVariant struct {
	name string
	variant[func([]byte) Result]
}

var asciiBenchmarkSelectionInput = detectSelectionInput()

var validateASCIIBenchmarkVariants = [...]asciiBoolBenchmarkVariant{
	{name: "public", variant: variant[func([]byte) bool]{
		value:     ValidateASCII,
		kind:      implementationScalar,
		available: true,
	}},
	{name: "scalar", variant: variant[func([]byte) bool]{
		value:     validateASCIIScalar,
		kind:      implementationScalar,
		available: true,
	}},
	{name: "westmere", variant: variant[func([]byte) bool]{kind: implementationWestmere}},
	{name: "haswell", variant: variant[func([]byte) bool]{kind: implementationHaswell}},
	{name: "neon", variant: variant[func([]byte) bool]{kind: implementationNEON}},
	{name: "archsimd", variant: variant[func([]byte) bool]{kind: implementationArchsimd}},
}

var validateASCIIWithErrorsBenchmarkVariants = [...]asciiResultBenchmarkVariant{
	{name: "public", variant: variant[func([]byte) Result]{
		value:     ValidateASCIIWithErrors,
		kind:      implementationScalar,
		available: true,
	}},
	{name: "scalar", variant: variant[func([]byte) Result]{
		value:     validateASCIIWithErrorsScalar,
		kind:      implementationScalar,
		available: true,
	}},
	{name: "westmere", variant: variant[func([]byte) Result]{kind: implementationWestmere}},
	{name: "haswell", variant: variant[func([]byte) Result]{kind: implementationHaswell}},
	{name: "neon", variant: variant[func([]byte) Result]{kind: implementationNEON}},
	{name: "archsimd", variant: variant[func([]byte) Result]{kind: implementationArchsimd}},
}

func registerASCIIDirectBenchmarkVariants(
	name string,
	validate variant[func([]byte) bool],
	validateWithErrors variant[func([]byte) Result],
) {
	for i := 2; i < len(validateASCIIBenchmarkVariants); i++ {
		if validateASCIIBenchmarkVariants[i].name != name {
			continue
		}
		if validate.value == nil || validateWithErrors.value == nil {
			panic("simdutf: direct ASCII benchmark variant has a nil function")
		}
		if validate.kind != validateASCIIBenchmarkVariants[i].kind ||
			validateWithErrors.kind != validateASCIIWithErrorsBenchmarkVariants[i].kind {
			panic("simdutf: direct ASCII benchmark variant has the wrong implementation kind")
		}
		if validateASCIIBenchmarkVariants[i].available ||
			validateASCIIWithErrorsBenchmarkVariants[i].available {
			panic("simdutf: direct ASCII benchmark variant registered twice")
		}
		validateASCIIBenchmarkVariants[i].variant = validate
		validateASCIIWithErrorsBenchmarkVariants[i].variant = validateWithErrors
		return
	}
	panic("simdutf: unknown direct ASCII benchmark variant " + name)
}

func TestASCIIBenchmarkVariantRegistry(t *testing.T) {
	wantNames := [...]string{"public", "scalar", "westmere", "haswell", "neon", "archsimd"}
	wantKinds := [...]implementationKind{
		implementationScalar,
		implementationScalar,
		implementationWestmere,
		implementationHaswell,
		implementationNEON,
		implementationArchsimd,
	}
	if len(validateASCIIBenchmarkVariants) != len(wantNames) ||
		len(validateASCIIWithErrorsBenchmarkVariants) != len(wantNames) {
		t.Fatalf("ASCII benchmark registry lengths = (%d, %d), want (%d, %d)",
			len(validateASCIIBenchmarkVariants), len(validateASCIIWithErrorsBenchmarkVariants),
			len(wantNames), len(wantNames))
	}
	for i := range wantNames {
		boolVariant := validateASCIIBenchmarkVariants[i]
		resultVariant := validateASCIIWithErrorsBenchmarkVariants[i]
		if boolVariant.name != wantNames[i] || resultVariant.name != wantNames[i] {
			t.Errorf("ASCII benchmark registry names at %d = (%q, %q), want %q",
				i, boolVariant.name, resultVariant.name, wantNames[i])
		}
		if boolVariant.kind != wantKinds[i] || resultVariant.kind != wantKinds[i] {
			t.Errorf("ASCII benchmark registry kinds at %d = (%d, %d), want %d",
				i, boolVariant.kind, resultVariant.kind, wantKinds[i])
		}
		if boolVariant.supportedBy(asciiBenchmarkSelectionInput) && boolVariant.value == nil {
			t.Errorf("runnable ValidateASCII variant %q has a nil function", wantNames[i])
		}
		if resultVariant.supportedBy(asciiBenchmarkSelectionInput) && resultVariant.value == nil {
			t.Errorf("runnable ValidateASCIIWithErrors variant %q has a nil function", wantNames[i])
		}
	}
}

func TestRegisterASCIIDirectBenchmarkVariantsRejectsUnknownName(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("unknown direct ASCII benchmark variant name did not panic")
		}
	}()
	registerASCIIDirectBenchmarkVariants(
		"unknown",
		variant[func([]byte) bool]{value: validateASCIIScalar},
		variant[func([]byte) Result]{value: validateASCIIWithErrorsScalar},
	)
}

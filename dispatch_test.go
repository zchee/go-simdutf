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

// Hand-authored Go-only tests for immutable dispatch selection. The tested
// first-supported priority contract comes from
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; these are not
// upstream algorithm vectors.

func TestSelectVariantScalarWithZeroFeatures(t *testing.T) {
	got := selectVariant(selectionInput{}, variant[string]{
		value:     "scalar",
		kind:      implementationScalar,
		available: true,
	})
	if got != "scalar" {
		t.Fatalf("selectVariant() = %q, want scalar", got)
	}
}

func TestSelectVariantFirstEligibleWins(t *testing.T) {
	got := selectVariant(
		selectionInput{features: cpuSSE42},
		variant[int]{value: 1, kind: implementationWestmere, required: cpuSSE42, available: true},
		variant[int]{value: 2, kind: implementationWestmere, required: cpuSSE42, available: true},
		variant[int]{value: 3, kind: implementationScalar, available: true},
	)
	if got != 1 {
		t.Fatalf("selectVariant() = %d, want first eligible value 1", got)
	}
}

func TestSelectVariantSkipsUnavailableHigherPriority(t *testing.T) {
	got := selectVariant(
		selectionInput{features: cpuAVX2 | cpuSSE42},
		variant[string]{value: "haswell", kind: implementationHaswell, required: cpuAVX2, available: false},
		variant[string]{value: "westmere", kind: implementationWestmere, required: cpuSSE42, available: true},
		variant[string]{value: "scalar", kind: implementationScalar, available: true},
	)
	if got != "westmere" {
		t.Fatalf("selectVariant() = %q, want westmere", got)
	}
}

func TestSelectVariantRejectsOneMissingRequiredBit(t *testing.T) {
	required := cpuSSE42 | cpuPOPCNT | cpuAVX2 | cpuBMI1 | cpuBMI2 | cpuLZCNT
	for _, test := range []struct {
		name    string
		missing cpuFeatures
	}{
		{name: "SSE4.2", missing: cpuSSE42},
		{name: "POPCNT", missing: cpuPOPCNT},
		{name: "AVX2", missing: cpuAVX2},
		{name: "BMI1", missing: cpuBMI1},
		{name: "BMI2", missing: cpuBMI2},
		{name: "LZCNT", missing: cpuLZCNT},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := selectVariant(
				selectionInput{features: required &^ test.missing},
				variant[string]{value: "haswell", kind: implementationHaswell, required: required, available: true},
				variant[string]{value: "scalar", kind: implementationScalar, available: true},
			)
			if got != "scalar" {
				t.Fatalf("selectVariant() = %q, want scalar", got)
			}
		})
	}
}

func TestSelectVariantSyntheticWestmereFallback(t *testing.T) {
	got := selectVariant(
		selectionInput{features: cpuSSE42 | cpuPOPCNT},
		variant[string]{value: "haswell", kind: implementationHaswell, required: cpuAVX2 | cpuBMI1, available: true},
		variant[string]{value: "westmere", kind: implementationWestmere, required: cpuSSE42 | cpuPOPCNT, available: true},
		variant[string]{value: "scalar", kind: implementationScalar, available: true},
	)
	if got != "westmere" {
		t.Fatalf("selectVariant() = %q, want westmere", got)
	}
}

func TestSelectVariantHaswellWestmereScalarFallthrough(t *testing.T) {
	haswellFeatures := cpuAVX2 | cpuBMI1 | cpuBMI2 | cpuLZCNT | cpuPOPCNT
	westmereFeatures := cpuSSE42 | cpuPOPCNT
	candidates := []variant[string]{
		{value: "haswell", kind: implementationHaswell, required: haswellFeatures, available: true},
		{value: "westmere", kind: implementationWestmere, required: westmereFeatures, available: true},
		{value: "scalar", kind: implementationScalar, available: true},
	}

	for _, test := range []struct {
		name     string
		features cpuFeatures
		want     string
	}{
		{name: "haswell", features: haswellFeatures | westmereFeatures, want: "haswell"},
		{name: "westmere", features: westmereFeatures, want: "westmere"},
		{name: "scalar", want: "scalar"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := selectVariant(selectionInput{features: test.features}, candidates...); got != test.want {
				t.Fatalf("selectVariant() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSelectVariantNEONRequiresNEON(t *testing.T) {
	candidates := []variant[string]{
		{value: "neon", kind: implementationNEON, required: cpuNEON, available: true},
		{value: "scalar", kind: implementationScalar, available: true},
	}
	if got := selectVariant(selectionInput{}, candidates...); got != "scalar" {
		t.Fatalf("without NEON selectVariant() = %q, want scalar", got)
	}
	if got := selectVariant(selectionInput{features: cpuNEON}, candidates...); got != "neon" {
		t.Fatalf("with NEON selectVariant() = %q, want neon", got)
	}
}

func TestSelectVariantArchsimdRequiresRuntimeAVX2(t *testing.T) {
	candidates := []variant[string]{
		{value: "archsimd", kind: implementationArchsimd, required: cpuAVX2 | cpuBMI1, available: true},
		{value: "scalar", kind: implementationScalar, available: true},
	}
	if got := selectVariant(selectionInput{features: cpuAVX2 | cpuBMI1}, candidates...); got != "scalar" {
		t.Fatalf("without archsimd AVX2 selectVariant() = %q, want scalar", got)
	}
	if got := selectVariant(selectionInput{features: cpuAVX2, archsimdAVX2: true}, candidates...); got != "scalar" {
		t.Fatalf("without BMI1 selectVariant() = %q, want scalar", got)
	}
	if got := selectVariant(selectionInput{features: cpuAVX2 | cpuBMI1, archsimdAVX2: true}, candidates...); got != "archsimd" {
		t.Fatalf("with archsimd AVX2 selectVariant() = %q, want archsimd", got)
	}
}

func TestSelectVariantPanicsWithoutScalarFallback(t *testing.T) {
	defer func() {
		got := recover()
		if got != "simdutf: internal dispatch has no available implementation" {
			t.Fatalf("panic = %v, want stable internal dispatch panic", got)
		}
	}()

	selectVariant(selectionInput{}, variant[int]{
		value:     1,
		kind:      implementationHaswell,
		required:  cpuAVX2,
		available: true,
	})
}

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

//go:build arm64

package simdutf

import "testing"

// Hand-authored Go-only tests for arm64 NEON feature gating, irrelevant-bit
// handling, UTF-8 and ASCII-family fallback, and live table identity. The
// dispatch contract is pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; these are not
// upstream test vectors.

func TestMakeImplementationARM64SyntheticNEONGate(t *testing.T) {
	for _, input := range []selectionInput{{}, {features: ^cpuNEON}} {
		checkUTF8ImplementationFunctions(t, makeImplementation(input))
		if got := makeImplementation(input).countUTF8; !sameFunction(got, countUTF8Scalar) {
			t.Errorf("without exact NEON countUTF8 selected %p, want scalar %p", got, countUTF8Scalar)
		}
		checkImplementationFunctions(t, makeImplementation(input),
			validateASCIIScalar, validateASCIIWithErrorsScalar,
			validateUTF16LEAsASCIIScalar, validateUTF16BEAsASCIIScalar)
	}
	for _, input := range []selectionInput{{features: cpuNEON}, {features: ^cpuFeatures(0)}} {
		checkUTF8ImplementationFunctions(t, makeImplementation(input))
		if got := makeImplementation(input).countUTF8; !sameFunction(got, countUTF8NEON) {
			t.Errorf("with NEON countUTF8 selected %p, want NEON %p", got, countUTF8NEON)
		}
		checkImplementationFunctions(t, makeImplementation(input),
			validateASCIINEON, validateASCIIWithErrorsNEON,
			validateUTF16LEAsASCIINEON, validateUTF16BEAsASCIINEON)
	}
}

func TestMakeImplementationARM64Live(t *testing.T) {
	checkUTF8ImplementationFunctions(t, activeImplementation)
	if got := activeImplementation.countUTF8; !sameFunction(got, countUTF8NEON) {
		t.Errorf("live countUTF8 selected %p, want NEON %p", got, countUTF8NEON)
	}
	checkImplementationFunctions(t, activeImplementation,
		validateASCIINEON, validateASCIIWithErrorsNEON,
		validateUTF16LEAsASCIINEON, validateUTF16BEAsASCIINEON)
}

func TestARM64DirectRegistriesRetainNEONValidators(t *testing.T) {
	check := func(name string, candidates []utf8DirectVariant) {
		t.Helper()
		for _, candidate := range candidates {
			if candidate.name != "neon" {
				continue
			}
			if !sameFunction(candidate.validate.value, validateUTF8NEON) {
				t.Errorf("%s neon validate = %p, want %p", name, candidate.validate.value, validateUTF8NEON)
			}
			if !sameFunction(candidate.withErrors.value, validateUTF8WithErrorsNEON) {
				t.Errorf("%s neon with-errors = %p, want %p", name, candidate.withErrors.value, validateUTF8WithErrorsNEON)
			}
			return
		}
		t.Errorf("%s registry has no neon variant", name)
	}

	check("direct", utf8DirectVariants)
	fuzz := make([]utf8DirectVariant, len(utf8FuzzVariants))
	for i, candidate := range utf8FuzzVariants {
		fuzz[i] = utf8DirectVariant(candidate)
	}
	check("fuzz", fuzz)
}

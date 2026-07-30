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
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/implementation.cpp
// and the per-symbol ISA/object-proof policy in
// docs/porting/provenance.md; these are not
// upstream test vectors.

func TestMakeImplementationARM64SyntheticNEONGate(t *testing.T) {
	if utf16LengthFromUTF8DispatchCutoff != 0 || utf32LengthFromUTF8DispatchCutoff != 0 {
		t.Fatalf("arm64 length cutoffs = (%d, %d), want zero", utf16LengthFromUTF8DispatchCutoff, utf32LengthFromUTF8DispatchCutoff)
	}
	for _, input := range []selectionInput{{}, {features: ^cpuNEON}} {
		got := makeImplementation(input)
		checkUTF8ImplementationFunctions(t, got)
		if gotCount := got.countUTF8; !sameFunction(gotCount, countUTF8Scalar) {
			t.Errorf("without exact NEON countUTF8 selected %p, want scalar %p", gotCount, countUTF8Scalar)
		}
		checkUTF8LengthImplementationFunctions(t, got, utf16LengthFromUTF8Scalar, utf32LengthFromUTF8Scalar)
		checkImplementationFunctions(t, got,
			validateASCIIScalar, validateASCIIWithErrorsScalar,
			validateUTF16LEAsASCIIScalar, validateUTF16BEAsASCIIScalar)
	}
	for _, input := range []selectionInput{{features: cpuNEON}, {features: ^cpuFeatures(0)}} {
		got := makeImplementation(input)
		checkUTF8ImplementationFunctions(t, got)
		if gotCount := got.countUTF8; !sameFunction(gotCount, countUTF8NEON) {
			t.Errorf("with NEON countUTF8 selected %p, want NEON %p", gotCount, countUTF8NEON)
		}
		checkUTF8LengthImplementationFunctions(t, got, utf16LengthFromUTF8Scalar, utf32LengthFromUTF8Scalar)
		checkImplementationFunctions(t, got,
			validateASCIINEON, validateASCIIWithErrorsNEON,
			validateUTF16LEAsASCIINEON, validateUTF16BEAsASCIINEON)
	}
}

func TestMakeImplementationARM64Live(t *testing.T) {
	checkUTF8ImplementationFunctions(t, activeImplementation)
	checkUTF8LengthImplementationFunctions(t, activeImplementation, utf16LengthFromUTF8Scalar, utf32LengthFromUTF8Scalar)
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

func TestMakeImplementationARM64W01ProvidersRemainDirectOnly(t *testing.T) {
	got := makeImplementation(selectionInput{features: cpuNEON})
	if !sameFunction(got.validateUTF16LE, validateUTF16LEScalar) ||
		!sameFunction(got.validateUTF16BE, validateUTF16BEScalar) ||
		!sameFunction(got.validateUTF16LEWithErrors, validateUTF16LEWithErrorsScalar) ||
		!sameFunction(got.validateUTF16BEWithErrors, validateUTF16BEWithErrorsScalar) ||
		!sameFunction(got.toWellFormedUTF16LE, toWellFormedUTF16LEScalar) ||
		!sameFunction(got.toWellFormedUTF16BE, toWellFormedUTF16BEScalar) ||
		!sameFunction(got.validateUTF32, validateUTF32Scalar) ||
		!sameFunction(got.validateUTF32WithErrors, validateUTF32WithErrorsScalar) {
		t.Fatal("unqualified NEON W01 provider leaked into selection")
	}
}

func TestMakeImplementationARM64Latin1QualificationSelection(t *testing.T) {
	got := makeImplementation(selectionInput{features: cpuNEON})
	// Selected after Q-latin1-ramp qualification on darwin-arm64-apple-m3-max.
	if !sameFunction(got.convertLatin1ToUTF16LE, convertLatin1ToUTF16LENEON) {
		t.Fatal("ConvertLatin1ToUTF16LE must select NEON after qualification")
	}
	if !sameFunction(got.convertLatin1ToUTF32, convertLatin1ToUTF32NEON) {
		t.Fatal("ConvertLatin1ToUTF32 must select NEON after qualification")
	}
	// Direct-only Latin-1 providers remain registered after scalar for force/tests.
	if !sameFunction(got.utf8LengthFromLatin1, utf8LengthFromLatin1Scalar) ||
		!sameFunction(got.convertLatin1ToUTF8, convertLatin1ToUTF8Scalar) ||
		!sameFunction(got.convertLatin1ToUTF16BE, convertLatin1ToUTF16BEScalar) {
		t.Fatal("direct-only Latin-1 providers leaked ahead of scalar")
	}
	live := activeImplementation
	if !sameFunction(live.convertLatin1ToUTF16LE, convertLatin1ToUTF16LENEON) ||
		!sameFunction(live.convertLatin1ToUTF32, convertLatin1ToUTF32NEON) ||
		!sameFunction(live.utf8LengthFromLatin1, utf8LengthFromLatin1Scalar) ||
		!sameFunction(live.convertLatin1ToUTF8, convertLatin1ToUTF8Scalar) ||
		!sameFunction(live.convertLatin1ToUTF16BE, convertLatin1ToUTF16BEScalar) {
		t.Fatal("live Latin-1 selection mismatched qualification dispositions")
	}
}

func TestMakeImplementationARM64UTF8QualificationSelection(t *testing.T) {
	// FC-v1-utf8-source qualification dispositions are all direct_only; public UTF-8
	// convert selection stays scalar-first; NEON remains forceable/direct-only.
	got := makeImplementation(selectionInput{features: cpuNEON})
	if !sameFunction(got.convertUTF8ToLatin1, convertUTF8ToLatin1Scalar) ||
		!sameFunction(got.convertUTF8ToLatin1WithErrors, convertUTF8ToLatin1WithErrorsScalar) ||
		!sameFunction(got.convertValidUTF8ToLatin1, convertValidUTF8ToLatin1Scalar) ||
		!sameFunction(got.convertUTF8ToUTF16LE, convertUTF8ToUTF16LEScalar) ||
		!sameFunction(got.convertUTF8ToUTF16BE, convertUTF8ToUTF16BEScalar) ||
		!sameFunction(got.convertUTF8ToUTF16LEWithErrors, convertUTF8ToUTF16LEWithErrorsScalar) ||
		!sameFunction(got.convertUTF8ToUTF16BEWithErrors, convertUTF8ToUTF16BEWithErrorsScalar) ||
		!sameFunction(got.convertValidUTF8ToUTF16LE, convertValidUTF8ToUTF16LEScalar) ||
		!sameFunction(got.convertValidUTF8ToUTF16BE, convertValidUTF8ToUTF16BEScalar) ||
		!sameFunction(got.convertUTF8ToUTF32, convertUTF8ToUTF32Scalar) ||
		!sameFunction(got.convertUTF8ToUTF32WithErrors, convertUTF8ToUTF32WithErrorsScalar) ||
		!sameFunction(got.convertValidUTF8ToUTF32, convertValidUTF8ToUTF32Scalar) {
		t.Fatal("UTF-8 convert providers leaked ahead of scalar before qualification")
	}
	live := activeImplementation
	if !sameFunction(live.convertUTF8ToLatin1, convertUTF8ToLatin1Scalar) ||
		!sameFunction(live.convertUTF8ToUTF16LE, convertUTF8ToUTF16LEScalar) ||
		!sameFunction(live.convertUTF8ToUTF16BE, convertUTF8ToUTF16BEScalar) ||
		!sameFunction(live.convertUTF8ToUTF32, convertUTF8ToUTF32Scalar) {
		t.Fatal("live UTF-8 convert selection leaked ahead of scalar before qualification")
	}
}

func TestMakeImplementationARM64UTF16HelpersQualificationSelection(t *testing.T) {
	// Helper providers stay scalar-first pending qualification dispositions.
	for _, input := range []selectionInput{
		{},
		{features: cpuNEON},
	} {
		got := makeImplementation(input)
		if !sameFunction(got.changeEndiannessUTF16, changeEndiannessUTF16Scalar) ||
			!sameFunction(got.countUTF16LE, countUTF16LEScalar) ||
			!sameFunction(got.countUTF16BE, countUTF16BEScalar) ||
			!sameFunction(got.utf32LengthFromUTF16LE, utf32LengthFromUTF16LEScalar) ||
			!sameFunction(got.utf32LengthFromUTF16BE, utf32LengthFromUTF16BEScalar) {
			t.Fatalf("UTF-16 helper providers leaked ahead of scalar for %#v", input)
		}
	}
}

func TestMakeImplementationARM64UTF16Latin1QualificationSelection(t *testing.T) {
	// UTF-16→Latin-1 NEON providers stay scalar-first pending qualification.
	for _, input := range []selectionInput{
		{},
		{features: cpuNEON},
	} {
		got := makeImplementation(input)
		if !sameFunction(got.convertUTF16LEToLatin1, convertUTF16LEToLatin1Scalar) ||
			!sameFunction(got.convertUTF16BEToLatin1, convertUTF16BEToLatin1Scalar) ||
			!sameFunction(got.convertUTF16LEToLatin1WithErrors, convertUTF16LEToLatin1WithErrorsScalar) ||
			!sameFunction(got.convertUTF16BEToLatin1WithErrors, convertUTF16BEToLatin1WithErrorsScalar) ||
			!sameFunction(got.convertValidUTF16LEToLatin1, convertValidUTF16LEToLatin1Scalar) ||
			!sameFunction(got.convertValidUTF16BEToLatin1, convertValidUTF16BEToLatin1Scalar) {
			t.Fatalf("UTF-16→Latin-1 providers leaked ahead of scalar for %#v", input)
		}
	}
}

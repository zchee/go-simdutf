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

//go:build amd64

package simdutf

import "testing"

// Hand-authored Go-only tests for amd64 implementation-table priority, UTF-8
// and ASCII feature gates, and live function identity. The dispatch contract is pinned
// to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/implementation.cpp
// and the per-symbol ISA/object-proof policy in
// docs/porting/provenance.md; these are not
// upstream test vectors.

func TestMakeImplementationAMD64SyntheticPriority(t *testing.T) {
	if utf16LengthFromUTF8DispatchCutoff != 16 || utf32LengthFromUTF8DispatchCutoff != 64 {
		t.Fatalf("amd64 length cutoffs = (%d, %d), want (16, 64)", utf16LengthFromUTF8DispatchCutoff, utf32LengthFromUTF8DispatchCutoff)
	}
	t.Run("count UTF-8", func(t *testing.T) {
		zero := makeImplementation(selectionInput{})
		if got := zero.countUTF8; !sameFunction(got, countUTF8Westmere) {
			t.Errorf("zero features selected %p, want Westmere %p", got, countUTF8Westmere)
		}
		if !sameFunction(zero.latin1LengthFromUTF8, zero.countUTF8) {
			t.Errorf("zero features selected Latin-1 %p and CountUTF8 %p", zero.latin1LengthFromUTF8, zero.countUTF8)
		}
		avx2 := makeImplementation(selectionInput{features: cpuAVX2})
		if got := avx2.countUTF8; !sameFunction(got, countUTF8Haswell) {
			t.Errorf("AVX2 selected %p, want Haswell %p", got, countUTF8Haswell)
		}
		if !sameFunction(avx2.latin1LengthFromUTF8, avx2.countUTF8) {
			t.Errorf("AVX2 selected Latin-1 %p and CountUTF8 %p", avx2.latin1LengthFromUTF8, avx2.countUTF8)
		}
	})
	t.Run("UTF-8 lengths", func(t *testing.T) {
		for _, input := range []selectionInput{
			{},
			{features: cpuAVX2},
			{features: cpuPOPCNT},
			{features: cpuAVX2 | cpuPOPCNT},
			{features: ^cpuFeatures(0), archsimdAVX2: true},
		} {
			wantUTF32 := utf32LengthFromUTF8Scalar
			if input.features&cpuPOPCNT == cpuPOPCNT {
				wantUTF32 = utf32LengthFromUTF8Westmere
			}
			checkUTF8LengthImplementationFunctions(t, makeImplementation(input), utf16LengthFromUTF8Westmere, wantUTF32)
		}
	})
	checkUTF8ImplementationFunctions(t, makeImplementation(selectionInput{}))
	t.Run("UTF-8 Westmere remains direct-only", func(t *testing.T) {
		checkUTF8ImplementationFunctions(t, makeImplementation(selectionInput{features: cpuSSSE3}))
	})
	checkUTF8ImplementationFunctionsWant(t,
		makeImplementation(selectionInput{features: cpuAVX2}),
		validateUTF8Haswell, validateUTF8WithErrorsHaswell)
	if archsimdValidateUTF8() != nil {
		checkUTF8ImplementationFunctionsWant(t,
			makeImplementation(selectionInput{features: ^cpuFeatures(0), archsimdAVX2: true}),
			archsimdValidateUTF8(), archsimdValidateUTF8WithErrors())
	} else {
		checkUTF8ImplementationFunctionsWant(t,
			makeImplementation(selectionInput{features: ^cpuFeatures(0), archsimdAVX2: true}),
			validateUTF8Haswell, validateUTF8WithErrorsHaswell)
	}

	checkUTF8ImplementationFunctions(t, makeImplementation(selectionInput{features: cpuSSE42 | cpuPOPCNT}))
	t.Run("westmere requires no feature bits", func(t *testing.T) {
		checkImplementationFunctions(t, makeImplementation(selectionInput{}),
			validateASCIIWestmere, validateASCIIWithErrorsWestmere,
			validateUTF16LEAsASCIIWestmere, validateUTF16BEAsASCIIWestmere)
	})

	t.Run("haswell requires AVX2", func(t *testing.T) {
		checkImplementationFunctions(t, makeImplementation(selectionInput{features: cpuAVX2}),
			validateASCIIHaswell, validateASCIIWithErrorsHaswell,
			validateUTF16LEAsASCIIHaswell, validateUTF16BEAsASCIIHaswell)
	})

	t.Run("one missing AVX2 gate falls through", func(t *testing.T) {
		checkImplementationFunctions(t, makeImplementation(selectionInput{archsimdAVX2: true}),
			validateASCIIWestmere, validateASCIIWithErrorsWestmere,
			validateUTF16LEAsASCIIWestmere, validateUTF16BEAsASCIIWestmere)
	})
}

func TestMakeImplementationAMD64Live(t *testing.T) {
	input := detectSelectionInput()
	got := activeImplementation
	wantUTF32 := utf32LengthFromUTF8Scalar
	if input.features&cpuPOPCNT == cpuPOPCNT {
		wantUTF32 = utf32LengthFromUTF8Westmere
	}
	checkUTF8LengthImplementationFunctions(t, got, utf16LengthFromUTF8Westmere, wantUTF32)
	if input.archsimdAVX2 && input.features&cpuAVX2 == cpuAVX2 && archsimdCountUTF8() != nil {
		if !sameFunction(got.countUTF8, archsimdCountUTF8()) {
			t.Errorf("live countUTF8 selected %p, want archsimd %p", got.countUTF8, archsimdCountUTF8())
		}
	} else if input.features&cpuAVX2 == cpuAVX2 {
		if !sameFunction(got.countUTF8, countUTF8Haswell) {
			t.Errorf("live countUTF8 selected %p, want Haswell %p", got.countUTF8, countUTF8Haswell)
		}
	} else if !sameFunction(got.countUTF8, countUTF8Westmere) {
		t.Errorf("live countUTF8 selected %p, want Westmere %p", got.countUTF8, countUTF8Westmere)
	}
	switch {
	case input.archsimdAVX2 && input.features&cpuAVX2 == cpuAVX2 && archsimdValidateUTF8() != nil:
		checkUTF8ImplementationFunctionsWant(t, got, archsimdValidateUTF8(), archsimdValidateUTF8WithErrors())
	case input.features&cpuAVX2 == cpuAVX2:
		checkUTF8ImplementationFunctionsWant(t, got, validateUTF8Haswell, validateUTF8WithErrorsHaswell)
	default:
		checkUTF8ImplementationFunctions(t, got)
	}
	if input.archsimdAVX2 && input.features&cpuAVX2 != 0 && archsimdValidateASCII() != nil {
		checkImplementationFunctions(t, got,
			archsimdValidateASCII(), archsimdValidateASCIIWithErrors(),
			archsimdValidateUTF16LEAsASCII(), archsimdValidateUTF16BEAsASCII())
		return
	}
	if input.features&cpuAVX2 != 0 {
		checkImplementationFunctions(t, got,
			validateASCIIHaswell, validateASCIIWithErrorsHaswell,
			validateUTF16LEAsASCIIHaswell, validateUTF16BEAsASCIIHaswell)
		return
	}
	checkImplementationFunctions(t, got,
		validateASCIIWestmere, validateASCIIWithErrorsWestmere,
		validateUTF16LEAsASCIIWestmere, validateUTF16BEAsASCIIWestmere)
}

func TestMakeImplementationAMD64W01QualifiedSelection(t *testing.T) {
	scalar := makeImplementation(selectionInput{})
	if !sameFunction(scalar.validateUTF16LE, validateUTF16LEScalar) ||
		!sameFunction(scalar.validateUTF16BE, validateUTF16BEScalar) ||
		!sameFunction(scalar.validateUTF16LEWithErrors, validateUTF16LEWithErrorsScalar) ||
		!sameFunction(scalar.validateUTF16BEWithErrors, validateUTF16BEWithErrorsScalar) ||
		!sameFunction(scalar.toWellFormedUTF16LE, toWellFormedUTF16LEScalar) ||
		!sameFunction(scalar.toWellFormedUTF16BE, toWellFormedUTF16BEScalar) ||
		!sameFunction(scalar.validateUTF32, validateUTF32Scalar) ||
		!sameFunction(scalar.validateUTF32WithErrors, validateUTF32WithErrorsScalar) {
		t.Fatal("feature-free W01 selection is not scalar")
	}

	qualified := makeImplementation(selectionInput{features: cpuSSE42 | cpuAVX2})
	if !sameFunction(qualified.toWellFormedUTF16BE, toWellFormedUTF16BEWestmere) {
		t.Fatal("qualified amd64 selection did not retain Westmere UTF-16BE repair")
	}
	if !sameFunction(qualified.validateUTF32WithErrors, validateUTF32WithErrorsHaswell) {
		t.Fatal("qualified amd64 selection did not select Haswell UTF-32 errors")
	}
	if !sameFunction(qualified.validateUTF16LE, validateUTF16LEScalar) ||
		!sameFunction(qualified.validateUTF16BE, validateUTF16BEScalar) ||
		!sameFunction(qualified.validateUTF16BEWithErrors, validateUTF16BEWithErrorsScalar) ||
		!sameFunction(qualified.toWellFormedUTF16LE, toWellFormedUTF16LEScalar) {
		t.Fatal("direct-only W01 providers leaked into default selection")
	}

	withSIMD := makeImplementation(selectionInput{features: cpuSSE42 | cpuAVX2, archsimdAVX2: true})
	if archsimdValidateUTF16LEWithErrors() != nil {
		if !sameFunction(withSIMD.validateUTF16LEWithErrors, archsimdValidateUTF16LEWithErrors()) ||
			!sameFunction(withSIMD.validateUTF32, archsimdValidateUTF32()) {
			t.Fatal("qualified archsimd W01 providers were not selected")
		}
	} else if !sameFunction(withSIMD.validateUTF16LEWithErrors, validateUTF16LEWithErrorsScalar) ||
		!sameFunction(withSIMD.validateUTF32, validateUTF32Scalar) {
		t.Fatal("archsimd providers leaked into a non-SIMD build")
	}
}

func TestMakeImplementationAMD64Latin1QualificationSelection(t *testing.T) {
	// After Q-latin1-ramp on linux-amd64-debian-13-xeon-platinum-8481c:
	// westmere/haswell Latin-1 remain direct_only; archsimd UTF8LengthFromLatin1
	// is selected. Other Latin-1-source ops stay scalar-first.
	for _, input := range []selectionInput{
		{},
		{features: cpuSSSE3},
		{features: cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2},
	} {
		got := makeImplementation(input)
		if !sameFunction(got.utf8LengthFromLatin1, utf8LengthFromLatin1Scalar) ||
			!sameFunction(got.convertLatin1ToUTF8, convertLatin1ToUTF8Scalar) ||
			!sameFunction(got.convertLatin1ToUTF16LE, convertLatin1ToUTF16LEScalar) ||
			!sameFunction(got.convertLatin1ToUTF16BE, convertLatin1ToUTF16BEScalar) ||
			!sameFunction(got.convertLatin1ToUTF32, convertLatin1ToUTF32Scalar) {
			t.Fatalf("Latin-1 public selection leaked accelerated provider for %#v", input)
		}
	}

	withSIMD := makeImplementation(selectionInput{features: cpuSSSE3 | cpuAVX2, archsimdAVX2: true})
	if archsimdUTF8LengthFromLatin1() != nil {
		if !sameFunction(withSIMD.utf8LengthFromLatin1, archsimdUTF8LengthFromLatin1()) {
			t.Fatal("qualified archsimd UTF8LengthFromLatin1 was not selected")
		}
	} else if !sameFunction(withSIMD.utf8LengthFromLatin1, utf8LengthFromLatin1Scalar) {
		t.Fatal("archsimd Latin-1 length leaked into a non-SIMD build")
	}
	if !sameFunction(withSIMD.convertLatin1ToUTF8, convertLatin1ToUTF8Scalar) ||
		!sameFunction(withSIMD.convertLatin1ToUTF16LE, convertLatin1ToUTF16LEScalar) ||
		!sameFunction(withSIMD.convertLatin1ToUTF16BE, convertLatin1ToUTF16BEScalar) ||
		!sameFunction(withSIMD.convertLatin1ToUTF32, convertLatin1ToUTF32Scalar) {
		t.Fatal("direct-only Latin-1 providers leaked ahead of scalar under SIMD")
	}
}

func TestMakeImplementationAMD64UTF8QualificationSelection(t *testing.T) {
	// FC-v1-utf8-source qualification dispositions are all direct_only; public UTF-8
	// convert selection stays scalar-first across westmere/haswell/archsimd.
	for _, input := range []selectionInput{
		{},
		{features: cpuSSSE3},
		{features: cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2, archsimdAVX2: true},
	} {
		got := makeImplementation(input)
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
			t.Fatalf("UTF-8 convert providers leaked ahead of scalar for %#v", input)
		}
	}
}

func TestMakeImplementationAMD64UTF16HelpersQualificationSelection(t *testing.T) {
	// FC-v1-utf16-source helper providers are forceable but remain scalar-first
	// until qualification dispositions promote selected backends.
	for _, input := range []selectionInput{
		{},
		{features: cpuSSSE3},
		{features: cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2, archsimdAVX2: true},
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

func TestMakeImplementationAMD64UTF16Latin1QualificationSelection(t *testing.T) {
	// UTF-16→Latin-1 providers are forceable but remain scalar-first until
	// qualification dispositions promote selected backends.
	for _, input := range []selectionInput{
		{},
		{features: cpuSSSE3},
		{features: cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2, archsimdAVX2: true},
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

func TestMakeImplementationAMD64UTF16UTF32QualificationSelection(t *testing.T) {
	// UTF-16→UTF-32 providers are forceable but remain scalar-first until
	// qualification dispositions promote selected backends.
	for _, input := range []selectionInput{
		{},
		{features: cpuSSSE3},
		{features: cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2, archsimdAVX2: true},
	} {
		got := makeImplementation(input)
		if !sameFunction(got.convertUTF16LEToUTF32, convertUTF16LEToUTF32Scalar) ||
			!sameFunction(got.convertUTF16BEToUTF32, convertUTF16BEToUTF32Scalar) ||
			!sameFunction(got.convertUTF16LEToUTF32WithErrors, convertUTF16LEToUTF32WithErrorsScalar) ||
			!sameFunction(got.convertUTF16BEToUTF32WithErrors, convertUTF16BEToUTF32WithErrorsScalar) ||
			!sameFunction(got.convertValidUTF16LEToUTF32, convertValidUTF16LEToUTF32Scalar) ||
			!sameFunction(got.convertValidUTF16BEToUTF32, convertValidUTF16BEToUTF32Scalar) {
			t.Fatalf("UTF-16→UTF-32 providers leaked ahead of scalar-first qualification gate for %#v", input)
		}
	}
}

func TestMakeImplementationAMD64UTF16UTF8QualificationSelection(t *testing.T) {
	// UTF-16→UTF-8 providers are forceable but remain scalar-first until
	// qualification dispositions promote selected backends.
	for _, input := range []selectionInput{
		{},
		{features: cpuSSSE3},
		{features: cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2, archsimdAVX2: true},
	} {
		got := makeImplementation(input)
		if !sameFunction(got.convertUTF16LEToUTF8, convertUTF16LEToUTF8Scalar) ||
			!sameFunction(got.convertUTF16BEToUTF8, convertUTF16BEToUTF8Scalar) ||
			!sameFunction(got.convertUTF16LEToUTF8WithErrors, convertUTF16LEToUTF8WithErrorsScalar) ||
			!sameFunction(got.convertUTF16BEToUTF8WithErrors, convertUTF16BEToUTF8WithErrorsScalar) ||
			!sameFunction(got.convertUTF16LEToUTF8WithReplacement, convertUTF16LEToUTF8WithReplacementScalar) ||
			!sameFunction(got.convertUTF16BEToUTF8WithReplacement, convertUTF16BEToUTF8WithReplacementScalar) ||
			!sameFunction(got.convertValidUTF16LEToUTF8, convertValidUTF16LEToUTF8Scalar) ||
			!sameFunction(got.convertValidUTF16BEToUTF8, convertValidUTF16BEToUTF8Scalar) ||
			!sameFunction(got.utf8LengthFromUTF16LE, utf8LengthFromUTF16LEScalar) ||
			!sameFunction(got.utf8LengthFromUTF16BE, utf8LengthFromUTF16BEScalar) ||
			!sameFunction(got.utf8LengthFromUTF16LEWithReplacement, utf8LengthFromUTF16LEWithReplacementScalar) ||
			!sameFunction(got.utf8LengthFromUTF16BEWithReplacement, utf8LengthFromUTF16BEWithReplacementScalar) {
			t.Fatalf("UTF-16→UTF-8 providers leaked ahead of scalar-first qualification gate for %#v", input)
		}
	}
}

func TestMakeImplementationAMD64UTF32SourceQualificationSelection(t *testing.T) {
	// UTF-32-source convert providers remain scalar-first until qualification
	// dispositions promote selected backends.
	for _, input := range []selectionInput{
		{},
		{features: cpuSSSE3},
		{features: cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2},
		{features: cpuSSSE3 | cpuAVX2, archsimdAVX2: true},
	} {
		got := makeImplementation(input)
		if !sameFunction(got.convertUTF32ToLatin1, convertUTF32ToLatin1Scalar) ||
			!sameFunction(got.convertUTF32ToLatin1WithErrors, convertUTF32ToLatin1WithErrorsScalar) ||
			!sameFunction(got.convertValidUTF32ToLatin1, convertValidUTF32ToLatin1Scalar) ||
			!sameFunction(got.convertUTF32ToUTF8, convertUTF32ToUTF8Scalar) ||
			!sameFunction(got.convertUTF32ToUTF8WithErrors, convertUTF32ToUTF8WithErrorsScalar) ||
			!sameFunction(got.convertValidUTF32ToUTF8, convertValidUTF32ToUTF8Scalar) ||
			!sameFunction(got.convertUTF32ToUTF16LE, convertUTF32ToUTF16LEScalar) ||
			!sameFunction(got.convertUTF32ToUTF16BE, convertUTF32ToUTF16BEScalar) ||
			!sameFunction(got.convertUTF32ToUTF16LEWithErrors, convertUTF32ToUTF16LEWithErrorsScalar) ||
			!sameFunction(got.convertUTF32ToUTF16BEWithErrors, convertUTF32ToUTF16BEWithErrorsScalar) ||
			!sameFunction(got.convertValidUTF32ToUTF16LE, convertValidUTF32ToUTF16LEScalar) ||
			!sameFunction(got.convertValidUTF32ToUTF16BE, convertValidUTF32ToUTF16BEScalar) ||
			!sameFunction(got.utf8LengthFromUTF32, utf8LengthFromUTF32Scalar) ||
			!sameFunction(got.utf16LengthFromUTF32, utf16LengthFromUTF32Scalar) {
			t.Fatalf("UTF-32-source providers leaked ahead of scalar for %#v", input)
		}
	}
}

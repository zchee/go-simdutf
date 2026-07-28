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

//go:build !amd64 && !arm64

package simdutf

import "testing"

// Hand-authored Go-only tests for generic-target scalar dispatch under live
// and synthetic feature inputs. The dispatch contract is pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/implementation.cpp
// and the per-symbol ISA/object-proof policy in
// docs/porting/provenance.md; these are not
// upstream test vectors.

func TestMakeImplementationGenericSyntheticAndLive(t *testing.T) {
	if utf16LengthFromUTF8DispatchCutoff != 0 || utf32LengthFromUTF8DispatchCutoff != 0 {
		t.Fatalf("generic length cutoffs = (%d, %d), want zero", utf16LengthFromUTF8DispatchCutoff, utf32LengthFromUTF8DispatchCutoff)
	}
	checkUTF8ImplementationFunctions(t, activeImplementation)
	checkUTF8LengthImplementationFunctions(t, activeImplementation, utf16LengthFromUTF8Scalar, utf32LengthFromUTF8Scalar)
	checkImplementationFunctions(t, activeImplementation,
		validateASCIIScalar, validateASCIIWithErrorsScalar,
		validateUTF16LEAsASCIIScalar, validateUTF16BEAsASCIIScalar)
	for _, input := range []selectionInput{{}, {features: ^cpuFeatures(0), archsimdAVX2: true}} {
		got := makeImplementation(input)
		checkUTF8ImplementationFunctions(t, got)
		checkUTF8LengthImplementationFunctions(t, got, utf16LengthFromUTF8Scalar, utf32LengthFromUTF8Scalar)
		checkImplementationFunctions(t, got,
			validateASCIIScalar, validateASCIIWithErrorsScalar,
			validateUTF16LEAsASCIIScalar, validateUTF16BEAsASCIIScalar)
	}
}

func TestCountUTF8DispatchZeroFeaturesSelectsScalar(t *testing.T) {
	got := makeImplementation(selectionInput{})
	if !sameFunction(got.countUTF8, countUTF8Scalar) {
		t.Errorf("countUTF8 selected %p, want scalar %p", got.countUTF8, countUTF8Scalar)
	}
}

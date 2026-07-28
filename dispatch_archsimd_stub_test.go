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

//go:build amd64 && !goexperiment.simd

package simdutf

import "testing"

// Hand-authored Go-only tests for non-SIMD archsimd provider absence and
// fallback selection without references to tagged backend symbols. The
// dispatch contract is pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/implementation.cpp
// and the per-symbol ISA/object-proof policy in
// docs/porting/provenance.md; these are not
// upstream test vectors.

func archsimdUTF8DirectVariantsExpected() bool {
	return false
}

func TestArchsimdProvidersUnavailableWithoutExperiment(t *testing.T) {
	if archsimdAVX2Available() ||
		archsimdCountUTF8() != nil ||
		archsimdValidateASCII() != nil ||
		archsimdValidateASCIIWithErrors() != nil ||
		archsimdValidateUTF16LEAsASCII() != nil ||
		archsimdValidateUTF16BEAsASCII() != nil ||
		archsimdValidateUTF8() != nil ||
		archsimdValidateUTF8WithErrors() != nil {
		t.Fatal("archsimd provider is available without GOEXPERIMENT=simd")
	}
	got := makeImplementation(selectionInput{features: cpuAVX2, archsimdAVX2: true})
	if !sameFunction(got.countUTF8, countUTF8Haswell) {
		t.Errorf("countUTF8 selected %p, want Haswell %p", got.countUTF8, countUTF8Haswell)
	}
	checkUTF8LengthImplementationFunctions(t, got, utf16LengthFromUTF8Westmere, utf32LengthFromUTF8Scalar)
	checkUTF8ImplementationFunctionsWant(t, got, validateUTF8Haswell, validateUTF8WithErrorsHaswell)
	checkImplementationFunctions(t, got,
		validateASCIIHaswell, validateASCIIWithErrorsHaswell,
		validateUTF16LEAsASCIIHaswell, validateUTF16BEAsASCIIHaswell)

	withPOPCNT := makeImplementation(selectionInput{features: cpuAVX2 | cpuPOPCNT, archsimdAVX2: true})
	checkUTF8LengthImplementationFunctions(t, withPOPCNT, utf16LengthFromUTF8Westmere, utf32LengthFromUTF8Westmere)
}

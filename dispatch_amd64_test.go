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
// to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; these are not
// upstream test vectors.

func TestMakeImplementationAMD64SyntheticPriority(t *testing.T) {
	checkUTF8ImplementationFunctions(t, makeImplementation(selectionInput{}))
	checkUTF8ImplementationFunctionsWant(t,
		makeImplementation(selectionInput{features: cpuSSSE3}),
		validateUTF8Westmere, validateUTF8WithErrorsWestmere)
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
	switch {
	case input.archsimdAVX2 && input.features&cpuAVX2 == cpuAVX2 && archsimdValidateUTF8() != nil:
		checkUTF8ImplementationFunctionsWant(t, got, archsimdValidateUTF8(), archsimdValidateUTF8WithErrors())
	case input.features&cpuAVX2 == cpuAVX2:
		checkUTF8ImplementationFunctionsWant(t, got, validateUTF8Haswell, validateUTF8WithErrorsHaswell)
	case input.features&cpuSSSE3 == cpuSSSE3:
		checkUTF8ImplementationFunctionsWant(t, got, validateUTF8Westmere, validateUTF8WithErrorsWestmere)
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

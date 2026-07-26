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

// Hand-authored Go-only tests for amd64 ASCII implementation-table priority,
// feature gates, and live function identity. The dispatch contract is pinned
// to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; these are not
// upstream test vectors.

func TestMakeImplementationAMD64SyntheticPriority(t *testing.T) {
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

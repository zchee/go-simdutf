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

//go:build amd64 && goexperiment.simd

package simdutf

import "testing"

// Hand-authored Go-only tests for the amd64 archsimd provider identities and
// the independent compile-time, CPU-feature, and runtime dispatch gates. The
// dispatch contract is pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; these are not
// upstream test vectors.

func TestMakeImplementationArchsimdSyntheticRuntimeGate(t *testing.T) {
	withoutRuntimeGate := makeImplementation(selectionInput{features: cpuAVX2})
	checkImplementationFunctions(t, withoutRuntimeGate,
		validateASCIIHaswell, validateASCIIWithErrorsHaswell,
		validateUTF16LEAsASCIIHaswell, validateUTF16BEAsASCIIHaswell)

	withoutCPUFeature := makeImplementation(selectionInput{archsimdAVX2: true})
	checkImplementationFunctions(t, withoutCPUFeature,
		validateASCIIWestmere, validateASCIIWithErrorsWestmere,
		validateUTF16LEAsASCIIWestmere, validateUTF16BEAsASCIIWestmere)

	withBothGates := makeImplementation(selectionInput{features: cpuAVX2, archsimdAVX2: true})
	checkImplementationFunctions(t, withBothGates,
		validateASCIIArchsimd, validateASCIIWithErrorsArchsimd,
		validateUTF16LEAsASCIIArchsimd, validateUTF16BEAsASCIIArchsimd)
}

func TestArchsimdProvidersMatchBackends(t *testing.T) {
	checkImplementationFunctions(t, implementation{
		validateASCII:           archsimdValidateASCII(),
		validateASCIIWithErrors: archsimdValidateASCIIWithErrors(),
		validateUTF16LEAsASCII:  archsimdValidateUTF16LEAsASCII(),
		validateUTF16BEAsASCII:  archsimdValidateUTF16BEAsASCII(),
	}, validateASCIIArchsimd, validateASCIIWithErrorsArchsimd,
		validateUTF16LEAsASCIIArchsimd, validateUTF16BEAsASCIIArchsimd)
}

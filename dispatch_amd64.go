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

// Go-only dispatch glue based on the first-supported priority semantics in
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; this is not an
// algorithm translation.

func detectHostFeatures() cpuFeatures {
	return detectAMD64Features()
}

func makeImplementation(input selectionInput) implementation {
	archsimdUTF8 := archsimdValidateUTF8()
	archsimdUTF8WithErrors := archsimdValidateUTF8WithErrors()
	archsimdASCII := archsimdValidateASCII()
	archsimdASCIIWithErrors := archsimdValidateASCIIWithErrors()
	archsimdUTF16LE := archsimdValidateUTF16LEAsASCII()
	archsimdUTF16BE := archsimdValidateUTF16BEAsASCII()

	return implementation{
		validateUTF8: selectVariant(input,
			variant[func([]byte) bool]{value: archsimdUTF8, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8 != nil},
			variant[func([]byte) bool]{value: validateUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) bool]{value: validateUTF8Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
			variant[func([]byte) bool]{value: validateUTF8Scalar, kind: implementationScalar, available: true},
		),
		validateUTF8WithErrors: selectVariant(input,
			variant[func([]byte) Result]{value: archsimdUTF8WithErrors, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8WithErrors != nil},
			variant[func([]byte) Result]{value: validateUTF8WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) Result]{value: validateUTF8WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
			variant[func([]byte) Result]{value: validateUTF8WithErrorsScalar, kind: implementationScalar, available: true},
		),
		validateASCII: selectVariant(input,
			variant[func([]byte) bool]{value: archsimdASCII, kind: implementationArchsimd, required: cpuAVX2, available: archsimdASCII != nil},
			variant[func([]byte) bool]{value: validateASCIIHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) bool]{value: validateASCIIWestmere, kind: implementationWestmere, available: true},
			variant[func([]byte) bool]{value: validateASCIIScalar, kind: implementationScalar, available: true},
		),
		validateASCIIWithErrors: selectVariant(input,
			variant[func([]byte) Result]{value: archsimdASCIIWithErrors, kind: implementationArchsimd, required: cpuAVX2, available: archsimdASCIIWithErrors != nil},
			variant[func([]byte) Result]{value: validateASCIIWithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) Result]{value: validateASCIIWithErrorsWestmere, kind: implementationWestmere, available: true},
			variant[func([]byte) Result]{value: validateASCIIWithErrorsScalar, kind: implementationScalar, available: true},
		),
		validateUTF16LEAsASCII: selectVariant(input,
			variant[func([]uint16) bool]{value: archsimdUTF16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16LE != nil},
			variant[func([]uint16) bool]{value: validateUTF16LEAsASCIIHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) bool]{value: validateUTF16LEAsASCIIWestmere, kind: implementationWestmere, available: true},
			variant[func([]uint16) bool]{value: validateUTF16LEAsASCIIScalar, kind: implementationScalar, available: true},
		),
		validateUTF16BEAsASCII: selectVariant(input,
			variant[func([]uint16) bool]{value: archsimdUTF16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16BE != nil},
			variant[func([]uint16) bool]{value: validateUTF16BEAsASCIIHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) bool]{value: validateUTF16BEAsASCIIWestmere, kind: implementationWestmere, available: true},
			variant[func([]uint16) bool]{value: validateUTF16BEAsASCIIScalar, kind: implementationScalar, available: true},
		),
	}
}

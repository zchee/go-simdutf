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

const (
	utf16LengthFromUTF8DispatchCutoff = 0
	utf32LengthFromUTF8DispatchCutoff = 0
)

// Go-only dispatch glue based on the first-supported priority semantics in
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; this is not an
// algorithm translation.

func detectHostFeatures() cpuFeatures {
	return 0
}

func makeImplementation(input selectionInput) implementation {
	countUTF8 := selectVariant(input,
		variant[func([]byte) int]{value: countUTF8Scalar, kind: implementationScalar, available: true},
	)
	return implementation{
		validateUTF8: selectVariant(input,
			variant[func([]byte) bool]{value: validateUTF8Scalar, kind: implementationScalar, available: true},
		),
		validateUTF8WithErrors: selectVariant(input,
			variant[func([]byte) Result]{value: validateUTF8WithErrorsScalar, kind: implementationScalar, available: true},
		),
		countUTF8:            countUTF8,
		latin1LengthFromUTF8: countUTF8,
		utf16LengthFromUTF8: selectVariant(input,
			variant[func([]byte) int]{value: utf16LengthFromUTF8Scalar, kind: implementationScalar, available: true},
		),
		utf32LengthFromUTF8: selectVariant(input,
			variant[func([]byte) int]{value: utf32LengthFromUTF8Scalar, kind: implementationScalar, available: true},
		),
		validateASCII: selectVariant(input,
			variant[func([]byte) bool]{value: validateASCIIScalar, kind: implementationScalar, available: true},
		),
		validateASCIIWithErrors: selectVariant(input,
			variant[func([]byte) Result]{value: validateASCIIWithErrorsScalar, kind: implementationScalar, available: true},
		),
		validateUTF16LEAsASCII: selectVariant(input,
			variant[func([]uint16) bool]{value: validateUTF16LEAsASCIIScalar, kind: implementationScalar, available: true},
		),
		validateUTF16BEAsASCII: selectVariant(input,
			variant[func([]uint16) bool]{value: validateUTF16BEAsASCIIScalar, kind: implementationScalar, available: true},
		),
	}
}

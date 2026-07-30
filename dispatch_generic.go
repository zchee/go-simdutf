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
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/implementation.cpp
// and the per-symbol ISA/object-proof policy in
// docs/porting/provenance.md; this is not an
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
		validateUTF16LE:           validateUTF16LEScalar,
		validateUTF16BE:           validateUTF16BEScalar,
		validateUTF16LEWithErrors: validateUTF16LEWithErrorsScalar,
		validateUTF16BEWithErrors: validateUTF16BEWithErrorsScalar,
		toWellFormedUTF16LE:       toWellFormedUTF16LEScalar,
		toWellFormedUTF16BE:       toWellFormedUTF16BEScalar,
		validateUTF32:             validateUTF32Scalar,
		validateUTF32WithErrors:   validateUTF32WithErrorsScalar,
		utf8LengthFromLatin1:      utf8LengthFromLatin1Scalar,
		convertLatin1ToUTF8:       convertLatin1ToUTF8Scalar,
		convertLatin1ToUTF16LE:    convertLatin1ToUTF16LEScalar,
		convertLatin1ToUTF16BE:    convertLatin1ToUTF16BEScalar,
		convertLatin1ToUTF32:      convertLatin1ToUTF32Scalar,
	}
}

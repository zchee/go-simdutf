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
		validateUTF16LE:                     validateUTF16LEScalar,
		validateUTF16BE:                     validateUTF16BEScalar,
		validateUTF16LEWithErrors:           validateUTF16LEWithErrorsScalar,
		validateUTF16BEWithErrors:           validateUTF16BEWithErrorsScalar,
		toWellFormedUTF16LE:                 toWellFormedUTF16LEScalar,
		toWellFormedUTF16BE:                 toWellFormedUTF16BEScalar,
		validateUTF32:                       validateUTF32Scalar,
		validateUTF32WithErrors:             validateUTF32WithErrorsScalar,
		utf8LengthFromLatin1:                utf8LengthFromLatin1Scalar,
		convertLatin1ToUTF8:                 convertLatin1ToUTF8Scalar,
		convertLatin1ToUTF16LE:              convertLatin1ToUTF16LEScalar,
		convertLatin1ToUTF16BE:              convertLatin1ToUTF16BEScalar,
		convertLatin1ToUTF32:                convertLatin1ToUTF32Scalar,
		convertUTF8ToLatin1:                 convertUTF8ToLatin1Scalar,
		convertUTF8ToLatin1WithErrors:       convertUTF8ToLatin1WithErrorsScalar,
		convertValidUTF8ToLatin1:            convertValidUTF8ToLatin1Scalar,
		convertUTF8ToUTF16LE:                convertUTF8ToUTF16LEScalar,
		convertUTF8ToUTF16BE:                convertUTF8ToUTF16BEScalar,
		convertUTF8ToUTF16LEWithErrors:      convertUTF8ToUTF16LEWithErrorsScalar,
		convertUTF8ToUTF16BEWithErrors:      convertUTF8ToUTF16BEWithErrorsScalar,
		convertValidUTF8ToUTF16LE:           convertValidUTF8ToUTF16LEScalar,
		convertValidUTF8ToUTF16BE:           convertValidUTF8ToUTF16BEScalar,
		convertUTF8ToUTF32:                  convertUTF8ToUTF32Scalar,
		convertUTF8ToUTF32WithErrors:        convertUTF8ToUTF32WithErrorsScalar,
		convertValidUTF8ToUTF32:             convertValidUTF8ToUTF32Scalar,
		convertUTF16LEToLatin1:              convertUTF16LEToLatin1Scalar,
		convertUTF16BEToLatin1:              convertUTF16BEToLatin1Scalar,
		convertUTF16LEToLatin1WithErrors:    convertUTF16LEToLatin1WithErrorsScalar,
		convertUTF16BEToLatin1WithErrors:    convertUTF16BEToLatin1WithErrorsScalar,
		convertValidUTF16LEToLatin1:         convertValidUTF16LEToLatin1Scalar,
		convertValidUTF16BEToLatin1:         convertValidUTF16BEToLatin1Scalar,
		convertUTF16LEToUTF32:               convertUTF16LEToUTF32Scalar,
		convertUTF16BEToUTF32:               convertUTF16BEToUTF32Scalar,
		convertUTF16LEToUTF32WithErrors:     convertUTF16LEToUTF32WithErrorsScalar,
		convertUTF16BEToUTF32WithErrors:     convertUTF16BEToUTF32WithErrorsScalar,
		convertValidUTF16LEToUTF32:          convertValidUTF16LEToUTF32Scalar,
		convertValidUTF16BEToUTF32:          convertValidUTF16BEToUTF32Scalar,
		utf32LengthFromUTF16LE:              utf32LengthFromUTF16LEScalar,
		utf32LengthFromUTF16BE:              utf32LengthFromUTF16BEScalar,
		convertUTF16LEToUTF8:                convertUTF16LEToUTF8Scalar,
		convertUTF16BEToUTF8:                convertUTF16BEToUTF8Scalar,
		convertUTF16LEToUTF8WithErrors:      convertUTF16LEToUTF8WithErrorsScalar,
		convertUTF16BEToUTF8WithErrors:      convertUTF16BEToUTF8WithErrorsScalar,
		convertUTF16LEToUTF8WithReplacement: convertUTF16LEToUTF8WithReplacementScalar,
		convertUTF16BEToUTF8WithReplacement: convertUTF16BEToUTF8WithReplacementScalar,
		convertValidUTF16LEToUTF8:           convertValidUTF16LEToUTF8Scalar,
		convertValidUTF16BEToUTF8:           convertValidUTF16BEToUTF8Scalar,
		utf8LengthFromUTF16LE:               utf8LengthFromUTF16LEScalar,
		utf8LengthFromUTF16BE:               utf8LengthFromUTF16BEScalar,
	}
}

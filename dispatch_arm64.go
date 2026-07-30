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

//go:build arm64

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
	return cpuNEON
}

func makeImplementation(input selectionInput) implementation {
	countUTF8 := selectVariant(
		input,
		variant[func([]byte) int]{value: countUTF8NEON, kind: implementationNEON, required: cpuNEON, available: true},
		variant[func([]byte) int]{value: countUTF8Scalar, kind: implementationScalar, available: true},
	)

	// Keep the direct NEON validators available to tests, fuzzing, and benchmarks.
	// Production stays scalar because all-length NEON and every tested Go cutoff
	// significantly regress short inputs; pinned upstream has no short path.
	return implementation{
		validateUTF8: selectVariant(
			input,
			variant[func([]byte) bool]{value: validateUTF8Scalar, kind: implementationScalar, available: true},
		),
		validateUTF8WithErrors: selectVariant(
			input,
			variant[func([]byte) Result]{value: validateUTF8WithErrorsScalar, kind: implementationScalar, available: true},
		),
		countUTF8:            countUTF8,
		latin1LengthFromUTF8: countUTF8,
		utf16LengthFromUTF8: selectVariant(
			input,
			variant[func([]byte) int]{value: utf16LengthFromUTF8Scalar, kind: implementationScalar, available: true},
		),
		utf32LengthFromUTF8: selectVariant(
			input,
			variant[func([]byte) int]{value: utf32LengthFromUTF8Scalar, kind: implementationScalar, available: true},
		),
		validateASCII: selectVariant(
			input,
			variant[func([]byte) bool]{value: validateASCIINEON, kind: implementationNEON, required: cpuNEON, available: true},
			variant[func([]byte) bool]{value: validateASCIIScalar, kind: implementationScalar, available: true},
		),
		validateASCIIWithErrors: selectVariant(
			input,
			variant[func([]byte) Result]{value: validateASCIIWithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
			variant[func([]byte) Result]{value: validateASCIIWithErrorsScalar, kind: implementationScalar, available: true},
		),
		validateUTF16LEAsASCII: selectVariant(
			input,
			variant[func([]uint16) bool]{value: validateUTF16LEAsASCIINEON, kind: implementationNEON, required: cpuNEON, available: true},
			variant[func([]uint16) bool]{value: validateUTF16LEAsASCIIScalar, kind: implementationScalar, available: true},
		),
		validateUTF16BEAsASCII: selectVariant(
			input,
			variant[func([]uint16) bool]{value: validateUTF16BEAsASCIINEON, kind: implementationNEON, required: cpuNEON, available: true},
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
		utf8LengthFromLatin1: selectVariant(
			input,
			variant[func([]byte) int]{value: utf8LengthFromLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]byte) int]{value: utf8LengthFromLatin1NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertLatin1ToUTF8: selectVariant(
			input,
			variant[func([]byte, []byte) int]{value: convertLatin1ToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte) int]{value: convertLatin1ToUTF8NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertLatin1ToUTF16LE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16LENEON, kind: implementationNEON, required: cpuNEON, available: true},
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16LEScalar, kind: implementationScalar, available: true},
		),
		convertLatin1ToUTF16BE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16BENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertLatin1ToUTF32: selectVariant(
			input,
			variant[func([]byte, []uint32) int]{value: convertLatin1ToUTF32NEON, kind: implementationNEON, required: cpuNEON, available: true},
			variant[func([]byte, []uint32) int]{value: convertLatin1ToUTF32Scalar, kind: implementationScalar, available: true},
		),
		convertUTF8ToLatin1: selectVariant(
			input,
			variant[func([]byte, []byte) int]{value: convertUTF8ToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte) int]{value: convertUTF8ToLatin1NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF8ToLatin1WithErrors: selectVariant(
			input,
			variant[func([]byte, []byte) Result]{value: convertUTF8ToLatin1WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte) Result]{value: convertUTF8ToLatin1WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF8ToLatin1: selectVariant(
			input,
			variant[func([]byte, []byte) int]{value: convertValidUTF8ToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte) int]{value: convertValidUTF8ToLatin1NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF8ToUTF16LE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16LENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF8ToUTF16BE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16BENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF8ToUTF16LEWithErrors: selectVariant(
			input,
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16LEWithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16LEWithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF8ToUTF16BEWithErrors: selectVariant(
			input,
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16BEWithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16BEWithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF8ToUTF16LE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16LENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF8ToUTF16BE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16BENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF8ToUTF32: selectVariant(
			input,
			variant[func([]byte, []uint32) int]{value: convertUTF8ToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint32) int]{value: convertUTF8ToUTF32NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF8ToUTF32WithErrors: selectVariant(
			input,
			variant[func([]byte, []uint32) Result]{value: convertUTF8ToUTF32WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint32) Result]{value: convertUTF8ToUTF32WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF8ToUTF32: selectVariant(
			input,
			variant[func([]byte, []uint32) int]{value: convertValidUTF8ToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint32) int]{value: convertValidUTF8ToUTF32NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16LEToLatin1: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToLatin1NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16BEToLatin1: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToLatin1NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16LEToLatin1WithErrors: selectVariant(
			input,
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToLatin1WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToLatin1WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16BEToLatin1WithErrors: selectVariant(
			input,
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToLatin1WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToLatin1WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF16LEToLatin1: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToLatin1NEON, kind: implementationNEON, required: cpuNEON, available: true},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToLatin1Scalar, kind: implementationScalar, available: true},
		),
		convertValidUTF16BEToLatin1: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToLatin1NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16LEToUTF32: selectVariant(
			input,
			variant[func([]uint16, []uint32) int]{value: convertUTF16LEToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) int]{value: convertUTF16LEToUTF32NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16BEToUTF32: selectVariant(
			input,
			variant[func([]uint16, []uint32) int]{value: convertUTF16BEToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) int]{value: convertUTF16BEToUTF32NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16LEToUTF32WithErrors: selectVariant(
			input,
			variant[func([]uint16, []uint32) Result]{value: convertUTF16LEToUTF32WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) Result]{value: convertUTF16LEToUTF32WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16BEToUTF32WithErrors: selectVariant(
			input,
			variant[func([]uint16, []uint32) Result]{value: convertUTF16BEToUTF32WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) Result]{value: convertUTF16BEToUTF32WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF16LEToUTF32: selectVariant(
			input,
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16LEToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16LEToUTF32NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF16BEToUTF32: selectVariant(
			input,
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16BEToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16BEToUTF32NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		utf32LengthFromUTF16LE: selectVariant(
			input,
			variant[func([]uint16) int]{value: utf32LengthFromUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: utf32LengthFromUTF16LENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		utf32LengthFromUTF16BE: selectVariant(
			input,
			variant[func([]uint16) int]{value: utf32LengthFromUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: utf32LengthFromUTF16BENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16LEToUTF8: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16BEToUTF8: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16LEToUTF8WithErrors: selectVariant(
			input,
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToUTF8WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToUTF8WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16BEToUTF8WithErrors: selectVariant(
			input,
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToUTF8WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToUTF8WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16LEToUTF8WithReplacement: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8WithReplacementScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8WithReplacementNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF16BEToUTF8WithReplacement: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8WithReplacementScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8WithReplacementNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF16LEToUTF8: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToUTF8NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF16BEToUTF8: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToUTF8NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		utf8LengthFromUTF16LE: selectVariant(
			input,
			variant[func([]uint16) int]{value: utf8LengthFromUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: utf8LengthFromUTF16LENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		utf8LengthFromUTF16BE: selectVariant(
			input,
			variant[func([]uint16) int]{value: utf8LengthFromUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: utf8LengthFromUTF16BENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		countUTF16LE: selectVariant(
			input,
			variant[func([]uint16) int]{value: countUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: countUTF16LENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		countUTF16BE: selectVariant(
			input,
			variant[func([]uint16) int]{value: countUTF16BENEON, kind: implementationNEON, required: cpuNEON, available: true},
			variant[func([]uint16) int]{value: countUTF16BEScalar, kind: implementationScalar, available: true},
		),
		changeEndiannessUTF16: selectVariant(
			input,
			variant[func([]uint16, []uint16)]{value: changeEndiannessUTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint16)]{value: changeEndiannessUTF16NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		utf8LengthFromUTF16LEWithReplacement: selectVariant(
			input,
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16LEWithReplacementScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16LEWithReplacementNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		utf8LengthFromUTF16BEWithReplacement: selectVariant(
			input,
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16BEWithReplacementScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16BEWithReplacementNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),

		convertUTF32ToLatin1: selectVariant(
			input,
			variant[func([]uint32, []byte) int]{value: convertUTF32ToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) int]{value: convertUTF32ToLatin1NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF32ToLatin1WithErrors: selectVariant(
			input,
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToLatin1WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToLatin1WithErrorsScalar, kind: implementationScalar, available: true},
		),
		convertValidUTF32ToLatin1: selectVariant(
			input,
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToLatin1NEON, kind: implementationNEON, required: cpuNEON, available: true},
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToLatin1Scalar, kind: implementationScalar, available: true},
		),
		convertUTF32ToUTF8: selectVariant(
			input,
			variant[func([]uint32, []byte) int]{value: convertUTF32ToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) int]{value: convertUTF32ToUTF8NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF32ToUTF8WithErrors: selectVariant(
			input,
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToUTF8WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToUTF8WithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF32ToUTF8: selectVariant(
			input,
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToUTF8NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF32ToUTF16LE: selectVariant(
			input,
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16LENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF32ToUTF16BE: selectVariant(
			input,
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16BENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF32ToUTF16LEWithErrors: selectVariant(
			input,
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16LEWithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16LEWithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertUTF32ToUTF16BEWithErrors: selectVariant(
			input,
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16BEWithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16BEWithErrorsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF32ToUTF16LE: selectVariant(
			input,
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16LENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		convertValidUTF32ToUTF16BE: selectVariant(
			input,
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16BENEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		utf8LengthFromUTF32: selectVariant(
			input,
			variant[func([]uint32) int]{value: utf8LengthFromUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32) int]{value: utf8LengthFromUTF32NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		utf16LengthFromUTF32: selectVariant(
			input,
			variant[func([]uint32) int]{value: utf16LengthFromUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32) int]{value: utf16LengthFromUTF32NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		detectEncodings: selectVariant(
			input,
			variant[func([]byte) Encoding]{value: detectEncodingsScalar, kind: implementationScalar, available: true},
			variant[func([]byte) Encoding]{value: detectEncodingsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		find: selectVariant(
			input,
			variant[func([]byte, byte) int]{value: findScalar, kind: implementationScalar, available: true},
			variant[func([]byte, byte) int]{value: findNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		findUTF16: selectVariant(
			input,
			variant[func([]uint16, uint16) int]{value: findUTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, uint16) int]{value: findUTF16NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		maximalBinaryLengthFromBase64: selectVariant(
			input,
			variant[func([]byte) int]{value: maximalBinaryLengthFromBase64Scalar, kind: implementationScalar, available: true},
		),
		maximalBinaryLengthFromBase64UTF16: selectVariant(
			input,
			variant[func([]uint16) int]{value: maximalBinaryLengthFromBase64UTF16Scalar, kind: implementationScalar, available: true},
		),
		binaryLengthFromBase64: selectVariant(
			input,
			variant[func([]byte) int]{value: binaryLengthFromBase64Scalar, kind: implementationScalar, available: true},
			variant[func([]byte) int]{value: binaryLengthFromBase64NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		binaryLengthFromBase64UTF16: selectVariant(
			input,
			variant[func([]uint16) int]{value: binaryLengthFromBase64UTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: binaryLengthFromBase64UTF16NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		base64ToBinary: selectVariant(
			input,
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		base64ToBinaryUTF16: selectVariant(
			input,
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryUTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryUTF16NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		base64ToBinaryDetails: selectVariant(
			input,
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		base64ToBinaryDetailsUTF16: selectVariant(
			input,
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsUTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsUTF16NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		base64ToBinarySafe:                   base64ToBinarySafeScalar,
		base64ToBinarySafeUTF16:              base64ToBinarySafeUTF16Scalar,
		binaryToBase64: selectVariant(
			input,
			variant[func([]byte, []byte, Base64Options) int]{value: binaryToBase64Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte, Base64Options) int]{value: binaryToBase64NEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
		binaryToBase64WithLines: selectVariant(
			input,
			variant[func([]byte, []byte, int, Base64Options) int]{value: binaryToBase64WithLinesScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte, int, Base64Options) int]{value: binaryToBase64WithLinesNEON, kind: implementationNEON, required: cpuNEON, available: true},
		),
	}
}

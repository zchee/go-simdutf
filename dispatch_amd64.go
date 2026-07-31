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
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/implementation.cpp
// and the per-symbol ISA/object-proof policy in
// docs/porting/provenance.md; this is not an
// algorithm translation.

func detectHostFeatures() cpuFeatures {
	return detectAMD64Features()
}

func makeImplementation(input selectionInput) implementation {
	archsimdUTF8 := archsimdValidateUTF8()
	archsimdUTF8WithErrors := archsimdValidateUTF8WithErrors()
	archsimdCount := archsimdCountUTF8()
	archsimdASCII := archsimdValidateASCII()
	archsimdASCIIWithErrors := archsimdValidateASCIIWithErrors()
	archsimdUTF16LE := archsimdValidateUTF16LEAsASCII()
	archsimdUTF16BE := archsimdValidateUTF16BEAsASCII()
	archsimdValidate16LEWithErrors := archsimdValidateUTF16LEWithErrors()
	archsimdValidate32 := archsimdValidateUTF32()
	archsimdDetectFn := archsimdDetectEncodings()
	archsimdLatin1Length := archsimdUTF8LengthFromLatin1()
	archsimdLatin1UTF8 := archsimdConvertLatin1ToUTF8()
	archsimdLatin1UTF16LE := archsimdConvertLatin1ToUTF16LE()
	archsimdLatin1UTF16BE := archsimdConvertLatin1ToUTF16BE()
	archsimdLatin1UTF32 := archsimdConvertLatin1ToUTF32()
	archsimdUTF8Latin1 := archsimdConvertUTF8ToLatin1()
	archsimdUTF8Latin1Err := archsimdConvertUTF8ToLatin1WithErrors()
	archsimdValidUTF8Latin1 := archsimdConvertValidUTF8ToLatin1()
	archsimdUTF8UTF16LE := archsimdConvertUTF8ToUTF16LE()
	archsimdUTF8UTF16BE := archsimdConvertUTF8ToUTF16BE()
	archsimdUTF8UTF16LEErr := archsimdConvertUTF8ToUTF16LEWithErrors()
	archsimdUTF8UTF16BEErr := archsimdConvertUTF8ToUTF16BEWithErrors()
	archsimdValidUTF8UTF16LE := archsimdConvertValidUTF8ToUTF16LE()
	archsimdValidUTF8UTF16BE := archsimdConvertValidUTF8ToUTF16BE()
	archsimdUTF8UTF32 := archsimdConvertUTF8ToUTF32()
	archsimdUTF8UTF32Err := archsimdConvertUTF8ToUTF32WithErrors()
	archsimdValidUTF8UTF32 := archsimdConvertValidUTF8ToUTF32()
	archsimdChangeEndian := archsimdChangeEndiannessUTF16()
	archsimdCount16LE := archsimdCountUTF16LE()
	archsimdCount16BE := archsimdCountUTF16BE()
	archsimdUTF32Len16LE := archsimdUTF32LengthFromUTF16LE()
	archsimdUTF32Len16BE := archsimdUTF32LengthFromUTF16BE()
	archsimdUTF16Latin1LE := archsimdConvertUTF16LEToLatin1()
	archsimdUTF16Latin1BE := archsimdConvertUTF16BEToLatin1()
	archsimdUTF16Latin1LEErr := archsimdConvertUTF16LEToLatin1WithErrors()
	archsimdUTF16Latin1BEErr := archsimdConvertUTF16BEToLatin1WithErrors()
	archsimdValidUTF16Latin1LE := archsimdConvertValidUTF16LEToLatin1()
	archsimdValidUTF16Latin1BE := archsimdConvertValidUTF16BEToLatin1()
	archsimdUTF16UTF32LE := archsimdConvertUTF16LEToUTF32()
	archsimdUTF16UTF32BE := archsimdConvertUTF16BEToUTF32()
	archsimdUTF16UTF32LEErr := archsimdConvertUTF16LEToUTF32WithErrors()
	archsimdUTF16UTF32BEErr := archsimdConvertUTF16BEToUTF32WithErrors()
	archsimdValidUTF16UTF32LE := archsimdConvertValidUTF16LEToUTF32()
	archsimdValidUTF16UTF32BE := archsimdConvertValidUTF16BEToUTF32()
	archsimdUTF16UTF8LE := archsimdConvertUTF16LEToUTF8()
	archsimdUTF16UTF8BE := archsimdConvertUTF16BEToUTF8()
	archsimdUTF16UTF8LEErr := archsimdConvertUTF16LEToUTF8WithErrors()
	archsimdUTF16UTF8BEErr := archsimdConvertUTF16BEToUTF8WithErrors()
	archsimdUTF16UTF8LERepl := archsimdConvertUTF16LEToUTF8WithReplacement()
	archsimdUTF16UTF8BERepl := archsimdConvertUTF16BEToUTF8WithReplacement()
	archsimdValidUTF16UTF8LE := archsimdConvertValidUTF16LEToUTF8()
	archsimdValidUTF16UTF8BE := archsimdConvertValidUTF16BEToUTF8()
	archsimdUTF8Len16LE := archsimdUTF8LengthFromUTF16LE()
	archsimdUTF8Len16BE := archsimdUTF8LengthFromUTF16BE()
	archsimdUTF8Len16LERepl := archsimdUTF8LengthFromUTF16LEWithReplacement()
	archsimdUTF8Len16BERepl := archsimdUTF8LengthFromUTF16BEWithReplacement()
	archsimdUTF32Latin1 := archsimdConvertUTF32ToLatin1()
	archsimdUTF32Latin1Err := archsimdConvertUTF32ToLatin1WithErrors()
	archsimdValidUTF32Latin1 := archsimdConvertValidUTF32ToLatin1()
	archsimdUTF32UTF8 := archsimdConvertUTF32ToUTF8()
	archsimdUTF32UTF8Err := archsimdConvertUTF32ToUTF8WithErrors()
	archsimdValidUTF32UTF8 := archsimdConvertValidUTF32ToUTF8()
	archsimdUTF32UTF16LE := archsimdConvertUTF32ToUTF16LE()
	archsimdUTF32UTF16BE := archsimdConvertUTF32ToUTF16BE()
	archsimdUTF32UTF16LEErr := archsimdConvertUTF32ToUTF16LEWithErrors()
	archsimdUTF32UTF16BEErr := archsimdConvertUTF32ToUTF16BEWithErrors()
	archsimdValidUTF32UTF16LE := archsimdConvertValidUTF32ToUTF16LE()
	archsimdValidUTF32UTF16BE := archsimdConvertValidUTF32ToUTF16BE()
	archsimdUTF8Len32 := archsimdUTF8LengthFromUTF32()
	archsimdUTF16Len32 := archsimdUTF16LengthFromUTF32()
	archsimdFindFn := archsimdFind()
	archsimdFindUTF16Fn := archsimdFindUTF16()
	countUTF8 := selectVariant(
		input,
		variant[func([]byte) int]{value: archsimdCount, kind: implementationArchsimd, required: cpuAVX2, available: archsimdCount != nil},
		variant[func([]byte) int]{value: countUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
		variant[func([]byte) int]{value: countUTF8Westmere, kind: implementationWestmere, available: true},
		variant[func([]byte) int]{value: countUTF8Scalar, kind: implementationScalar, available: true},
	)

	// The translated Westmere UTF-8 validators remain available to direct tests,
	// fuzzing, and benchmarks. They are not production candidates because they
	// regress both the complete-block short class and the pinned bulk corpus
	// against the scalar oracle on the required amd64 host.
	return implementation{
		validateUTF8: selectVariant(
			input,
			variant[func([]byte) bool]{value: archsimdUTF8, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8 != nil},
			variant[func([]byte) bool]{value: validateUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) bool]{value: validateUTF8Scalar, kind: implementationScalar, available: true},
		),
		validateUTF8WithErrors: selectVariant(
			input,
			variant[func([]byte) Result]{value: archsimdUTF8WithErrors, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8WithErrors != nil},
			variant[func([]byte) Result]{value: validateUTF8WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) Result]{value: validateUTF8WithErrorsScalar, kind: implementationScalar, available: true},
		),
		countUTF8:            countUTF8,
		latin1LengthFromUTF8: countUTF8,
		utf16LengthFromUTF8: selectVariant(
			input,
			variant[func([]byte) int]{value: utf16LengthFromUTF8Westmere, kind: implementationWestmere, available: true},
			variant[func([]byte) int]{value: utf16LengthFromUTF8Scalar, kind: implementationScalar, available: true},
		),
		utf32LengthFromUTF8: selectVariant(
			input,
			variant[func([]byte) int]{value: utf32LengthFromUTF8Westmere, kind: implementationWestmere, required: cpuPOPCNT, available: true},
			variant[func([]byte) int]{value: utf32LengthFromUTF8Scalar, kind: implementationScalar, available: true},
		),
		validateASCII: selectVariant(
			input,
			variant[func([]byte) bool]{value: archsimdASCII, kind: implementationArchsimd, required: cpuAVX2, available: archsimdASCII != nil},
			variant[func([]byte) bool]{value: validateASCIIHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) bool]{value: validateASCIIWestmere, kind: implementationWestmere, available: true},
			variant[func([]byte) bool]{value: validateASCIIScalar, kind: implementationScalar, available: true},
		),
		validateASCIIWithErrors: selectVariant(
			input,
			variant[func([]byte) Result]{value: archsimdASCIIWithErrors, kind: implementationArchsimd, required: cpuAVX2, available: archsimdASCIIWithErrors != nil},
			variant[func([]byte) Result]{value: validateASCIIWithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) Result]{value: validateASCIIWithErrorsWestmere, kind: implementationWestmere, available: true},
			variant[func([]byte) Result]{value: validateASCIIWithErrorsScalar, kind: implementationScalar, available: true},
		),
		validateUTF16LEAsASCII: selectVariant(
			input,
			variant[func([]uint16) bool]{value: archsimdUTF16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16LE != nil},
			variant[func([]uint16) bool]{value: validateUTF16LEAsASCIIHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) bool]{value: validateUTF16LEAsASCIIWestmere, kind: implementationWestmere, available: true},
			variant[func([]uint16) bool]{value: validateUTF16LEAsASCIIScalar, kind: implementationScalar, available: true},
		),
		validateUTF16BEAsASCII: selectVariant(
			input,
			variant[func([]uint16) bool]{value: archsimdUTF16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16BE != nil},
			variant[func([]uint16) bool]{value: validateUTF16BEAsASCIIHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) bool]{value: validateUTF16BEAsASCIIWestmere, kind: implementationWestmere, available: true},
			variant[func([]uint16) bool]{value: validateUTF16BEAsASCIIScalar, kind: implementationScalar, available: true},
		),
		validateUTF16LE: validateUTF16LEScalar,
		validateUTF16BE: validateUTF16BEScalar,
		validateUTF16LEWithErrors: selectVariant(
			input,
			variant[func([]uint16) Result]{value: archsimdValidate16LEWithErrors, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidate16LEWithErrors != nil},
			variant[func([]uint16) Result]{value: validateUTF16LEWithErrorsScalar, kind: implementationScalar, available: true},
		),
		validateUTF16BEWithErrors: validateUTF16BEWithErrorsScalar,
		toWellFormedUTF16LE:       toWellFormedUTF16LEScalar,
		toWellFormedUTF16BE: selectVariant(
			input,
			variant[func([]uint16, []uint16)]{value: toWellFormedUTF16BEWestmere, kind: implementationWestmere, required: cpuSSE42, available: true},
			variant[func([]uint16, []uint16)]{value: toWellFormedUTF16BEScalar, kind: implementationScalar, available: true},
		),
		validateUTF32: selectVariant(
			input,
			variant[func([]uint32) bool]{value: archsimdValidate32, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidate32 != nil},
			variant[func([]uint32) bool]{value: validateUTF32Scalar, kind: implementationScalar, available: true},
		),
		validateUTF32WithErrors: selectVariant(
			input,
			variant[func([]uint32) Result]{value: validateUTF32WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32) Result]{value: validateUTF32WithErrorsScalar, kind: implementationScalar, available: true},
		),
		utf8LengthFromLatin1: selectVariant(
			input,
			variant[func([]byte) int]{value: archsimdLatin1Length, kind: implementationArchsimd, required: cpuAVX2, available: archsimdLatin1Length != nil},
			variant[func([]byte) int]{value: utf8LengthFromLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]byte) int]{value: utf8LengthFromLatin1Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) int]{value: utf8LengthFromLatin1Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertLatin1ToUTF8: selectVariant(
			input,
			variant[func([]byte, []byte) int]{value: convertLatin1ToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte) int]{value: archsimdLatin1UTF8, kind: implementationArchsimd, required: cpuAVX2, available: archsimdLatin1UTF8 != nil},
			variant[func([]byte, []byte) int]{value: convertLatin1ToUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []byte) int]{value: convertLatin1ToUTF8Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertLatin1ToUTF16LE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: archsimdLatin1UTF16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdLatin1UTF16LE != nil},
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16LEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16LEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertLatin1ToUTF16BE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: archsimdLatin1UTF16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdLatin1UTF16BE != nil},
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16BEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint16) int]{value: convertLatin1ToUTF16BEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertLatin1ToUTF32: selectVariant(
			input,
			variant[func([]byte, []uint32) int]{value: convertLatin1ToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint32) int]{value: archsimdLatin1UTF32, kind: implementationArchsimd, required: cpuAVX2, available: archsimdLatin1UTF32 != nil},
			variant[func([]byte, []uint32) int]{value: convertLatin1ToUTF32Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint32) int]{value: convertLatin1ToUTF32Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF8ToLatin1: selectVariant(
			input,
			variant[func([]byte, []byte) int]{value: convertUTF8ToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte) int]{value: archsimdUTF8Latin1, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8Latin1 != nil},
			variant[func([]byte, []byte) int]{value: convertUTF8ToLatin1Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []byte) int]{value: convertUTF8ToLatin1Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF8ToLatin1WithErrors: selectVariant(
			input,
			variant[func([]byte, []byte) Result]{value: convertUTF8ToLatin1WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte) Result]{value: archsimdUTF8Latin1Err, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8Latin1Err != nil},
			variant[func([]byte, []byte) Result]{value: convertUTF8ToLatin1WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []byte) Result]{value: convertUTF8ToLatin1WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF8ToLatin1: selectVariant(
			input,
			variant[func([]byte, []byte) int]{value: convertValidUTF8ToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte) int]{value: archsimdValidUTF8Latin1, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF8Latin1 != nil},
			variant[func([]byte, []byte) int]{value: convertValidUTF8ToLatin1Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []byte) int]{value: convertValidUTF8ToLatin1Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF8ToUTF16LE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: archsimdUTF8UTF16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8UTF16LE != nil},
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16LEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16LEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF8ToUTF16BE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: archsimdUTF8UTF16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8UTF16BE != nil},
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16BEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint16) int]{value: convertUTF8ToUTF16BEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF8ToUTF16LEWithErrors: selectVariant(
			input,
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16LEWithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) Result]{value: archsimdUTF8UTF16LEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8UTF16LEErr != nil},
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16LEWithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16LEWithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF8ToUTF16BEWithErrors: selectVariant(
			input,
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16BEWithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) Result]{value: archsimdUTF8UTF16BEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8UTF16BEErr != nil},
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16BEWithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint16) Result]{value: convertUTF8ToUTF16BEWithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF8ToUTF16LE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: archsimdValidUTF8UTF16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF8UTF16LE != nil},
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16LEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16LEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF8ToUTF16BE: selectVariant(
			input,
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint16) int]{value: archsimdValidUTF8UTF16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF8UTF16BE != nil},
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16BEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint16) int]{value: convertValidUTF8ToUTF16BEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF8ToUTF32: selectVariant(
			input,
			variant[func([]byte, []uint32) int]{value: convertUTF8ToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint32) int]{value: archsimdUTF8UTF32, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8UTF32 != nil},
			variant[func([]byte, []uint32) int]{value: convertUTF8ToUTF32Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint32) int]{value: convertUTF8ToUTF32Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF8ToUTF32WithErrors: selectVariant(
			input,
			variant[func([]byte, []uint32) Result]{value: convertUTF8ToUTF32WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint32) Result]{value: archsimdUTF8UTF32Err, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8UTF32Err != nil},
			variant[func([]byte, []uint32) Result]{value: convertUTF8ToUTF32WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint32) Result]{value: convertUTF8ToUTF32WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF8ToUTF32: selectVariant(
			input,
			variant[func([]byte, []uint32) int]{value: convertValidUTF8ToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []uint32) int]{value: archsimdValidUTF8UTF32, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF8UTF32 != nil},
			variant[func([]byte, []uint32) int]{value: convertValidUTF8ToUTF32Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []uint32) int]{value: convertValidUTF8ToUTF32Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16LEToLatin1: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdUTF16Latin1LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16Latin1LE != nil},
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToLatin1Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToLatin1Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16BEToLatin1: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdUTF16Latin1BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16Latin1BE != nil},
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToLatin1Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToLatin1Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16LEToLatin1WithErrors: selectVariant(
			input,
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToLatin1WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) Result]{value: archsimdUTF16Latin1LEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16Latin1LEErr != nil},
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToLatin1WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToLatin1WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16BEToLatin1WithErrors: selectVariant(
			input,
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToLatin1WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) Result]{value: archsimdUTF16Latin1BEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16Latin1BEErr != nil},
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToLatin1WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToLatin1WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF16LEToLatin1: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdValidUTF16Latin1LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF16Latin1LE != nil},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToLatin1Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToLatin1Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF16BEToLatin1: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdValidUTF16Latin1BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF16Latin1BE != nil},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToLatin1Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToLatin1Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16LEToUTF32: selectVariant(
			input,
			variant[func([]uint16, []uint32) int]{value: convertUTF16LEToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) int]{value: archsimdUTF16UTF32LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF32LE != nil},
			variant[func([]uint16, []uint32) int]{value: convertUTF16LEToUTF32Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []uint32) int]{value: convertUTF16LEToUTF32Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16BEToUTF32: selectVariant(
			input,
			variant[func([]uint16, []uint32) int]{value: convertUTF16BEToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) int]{value: archsimdUTF16UTF32BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF32BE != nil},
			variant[func([]uint16, []uint32) int]{value: convertUTF16BEToUTF32Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []uint32) int]{value: convertUTF16BEToUTF32Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16LEToUTF32WithErrors: selectVariant(
			input,
			variant[func([]uint16, []uint32) Result]{value: convertUTF16LEToUTF32WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) Result]{value: archsimdUTF16UTF32LEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF32LEErr != nil},
			variant[func([]uint16, []uint32) Result]{value: convertUTF16LEToUTF32WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []uint32) Result]{value: convertUTF16LEToUTF32WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16BEToUTF32WithErrors: selectVariant(
			input,
			variant[func([]uint16, []uint32) Result]{value: convertUTF16BEToUTF32WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) Result]{value: archsimdUTF16UTF32BEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF32BEErr != nil},
			variant[func([]uint16, []uint32) Result]{value: convertUTF16BEToUTF32WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []uint32) Result]{value: convertUTF16BEToUTF32WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF16LEToUTF32: selectVariant(
			input,
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16LEToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) int]{value: archsimdValidUTF16UTF32LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF16UTF32LE != nil},
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16LEToUTF32Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16LEToUTF32Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF16BEToUTF32: selectVariant(
			input,
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16BEToUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint32) int]{value: archsimdValidUTF16UTF32BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF16UTF32BE != nil},
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16BEToUTF32Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []uint32) int]{value: convertValidUTF16BEToUTF32Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		utf32LengthFromUTF16LE: selectVariant(
			input,
			variant[func([]uint16) int]{value: utf32LengthFromUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: archsimdUTF32Len16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32Len16LE != nil},
			variant[func([]uint16) int]{value: utf32LengthFromUTF16LEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) int]{value: utf32LengthFromUTF16LEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		utf32LengthFromUTF16BE: selectVariant(
			input,
			variant[func([]uint16) int]{value: utf32LengthFromUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: archsimdUTF32Len16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32Len16BE != nil},
			variant[func([]uint16) int]{value: utf32LengthFromUTF16BEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) int]{value: utf32LengthFromUTF16BEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16LEToUTF8: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdUTF16UTF8LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF8LE != nil},
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16BEToUTF8: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdUTF16UTF8BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF8BE != nil},
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16LEToUTF8WithErrors: selectVariant(
			input,
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToUTF8WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) Result]{value: archsimdUTF16UTF8LEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF8LEErr != nil},
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToUTF8WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) Result]{value: convertUTF16LEToUTF8WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16BEToUTF8WithErrors: selectVariant(
			input,
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToUTF8WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) Result]{value: archsimdUTF16UTF8BEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF8BEErr != nil},
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToUTF8WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) Result]{value: convertUTF16BEToUTF8WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16LEToUTF8WithReplacement: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8WithReplacementScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdUTF16UTF8LERepl, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF8LERepl != nil},
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8WithReplacementHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16LEToUTF8WithReplacementWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF16BEToUTF8WithReplacement: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8WithReplacementScalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdUTF16UTF8BERepl, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16UTF8BERepl != nil},
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8WithReplacementHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertUTF16BEToUTF8WithReplacementWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF16LEToUTF8: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdValidUTF16UTF8LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF16UTF8LE != nil},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16LEToUTF8Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF16BEToUTF8: selectVariant(
			input,
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte) int]{value: archsimdValidUTF16UTF8BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF16UTF8BE != nil},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte) int]{value: convertValidUTF16BEToUTF8Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		utf8LengthFromUTF16LE: selectVariant(
			input,
			variant[func([]uint16) int]{value: utf8LengthFromUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: archsimdUTF8Len16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8Len16LE != nil},
			variant[func([]uint16) int]{value: utf8LengthFromUTF16LEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) int]{value: utf8LengthFromUTF16LEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		utf8LengthFromUTF16BE: selectVariant(
			input,
			variant[func([]uint16) int]{value: utf8LengthFromUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: archsimdUTF8Len16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8Len16BE != nil},
			variant[func([]uint16) int]{value: utf8LengthFromUTF16BEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) int]{value: utf8LengthFromUTF16BEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		countUTF16LE: selectVariant(
			input,
			variant[func([]uint16) int]{value: countUTF16LEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
			variant[func([]uint16) int]{value: countUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: archsimdCount16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdCount16LE != nil},
			variant[func([]uint16) int]{value: countUTF16LEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
		),
		countUTF16BE: selectVariant(
			input,
			variant[func([]uint16) int]{value: countUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: archsimdCount16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdCount16BE != nil},
			variant[func([]uint16) int]{value: countUTF16BEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) int]{value: countUTF16BEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		changeEndiannessUTF16: selectVariant(
			input,
			variant[func([]uint16, []uint16)]{value: changeEndiannessUTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []uint16)]{value: archsimdChangeEndian, kind: implementationArchsimd, required: cpuAVX2, available: archsimdChangeEndian != nil},
			variant[func([]uint16, []uint16)]{value: changeEndiannessUTF16Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []uint16)]{value: changeEndiannessUTF16Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		utf8LengthFromUTF16LEWithReplacement: selectVariant(
			input,
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16LEWithReplacementScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) Result]{value: archsimdUTF8Len16LERepl, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8Len16LERepl != nil},
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16LEWithReplacementHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16LEWithReplacementWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		utf8LengthFromUTF16BEWithReplacement: selectVariant(
			input,
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16BEWithReplacementScalar, kind: implementationScalar, available: true},
			variant[func([]uint16) Result]{value: archsimdUTF8Len16BERepl, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8Len16BERepl != nil},
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16BEWithReplacementHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) Result]{value: utf8LengthFromUTF16BEWithReplacementWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),

		convertUTF32ToLatin1: selectVariant(
			input,
			variant[func([]uint32, []byte) int]{value: convertUTF32ToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) int]{value: archsimdUTF32Latin1, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32Latin1 != nil},
			variant[func([]uint32, []byte) int]{value: convertUTF32ToLatin1Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []byte) int]{value: convertUTF32ToLatin1Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF32ToLatin1WithErrors: selectVariant(
			input,
			variant[func([]uint32, []byte) Result]{value: archsimdUTF32Latin1Err, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32Latin1Err != nil},
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToLatin1WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToLatin1WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToLatin1WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF32ToLatin1: selectVariant(
			input,
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToLatin1Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) int]{value: archsimdValidUTF32Latin1, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF32Latin1 != nil},
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToLatin1Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToLatin1Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF32ToUTF8: selectVariant(
			input,
			variant[func([]uint32, []byte) int]{value: convertUTF32ToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) int]{value: archsimdUTF32UTF8, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32UTF8 != nil},
			variant[func([]uint32, []byte) int]{value: convertUTF32ToUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []byte) int]{value: convertUTF32ToUTF8Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF32ToUTF8WithErrors: selectVariant(
			input,
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToUTF8WithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) Result]{value: archsimdUTF32UTF8Err, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32UTF8Err != nil},
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToUTF8WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []byte) Result]{value: convertUTF32ToUTF8WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF32ToUTF8: selectVariant(
			input,
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToUTF8Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []byte) int]{value: archsimdValidUTF32UTF8, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF32UTF8 != nil},
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []byte) int]{value: convertValidUTF32ToUTF8Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF32ToUTF16LE: selectVariant(
			input,
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) int]{value: archsimdUTF32UTF16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32UTF16LE != nil},
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16LEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16LEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF32ToUTF16BE: selectVariant(
			input,
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) int]{value: archsimdUTF32UTF16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32UTF16BE != nil},
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16BEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []uint16) int]{value: convertUTF32ToUTF16BEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF32ToUTF16LEWithErrors: selectVariant(
			input,
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16LEWithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) Result]{value: archsimdUTF32UTF16LEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32UTF16LEErr != nil},
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16LEWithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16LEWithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertUTF32ToUTF16BEWithErrors: selectVariant(
			input,
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16BEWithErrorsScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) Result]{value: archsimdUTF32UTF16BEErr, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF32UTF16BEErr != nil},
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16BEWithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []uint16) Result]{value: convertUTF32ToUTF16BEWithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF32ToUTF16LE: selectVariant(
			input,
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16LEScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) int]{value: archsimdValidUTF32UTF16LE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF32UTF16LE != nil},
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16LEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16LEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		convertValidUTF32ToUTF16BE: selectVariant(
			input,
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16BEScalar, kind: implementationScalar, available: true},
			variant[func([]uint32, []uint16) int]{value: archsimdValidUTF32UTF16BE, kind: implementationArchsimd, required: cpuAVX2, available: archsimdValidUTF32UTF16BE != nil},
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16BEHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32, []uint16) int]{value: convertValidUTF32ToUTF16BEWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		utf8LengthFromUTF32: selectVariant(
			input,
			variant[func([]uint32) int]{value: utf8LengthFromUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32) int]{value: archsimdUTF8Len32, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF8Len32 != nil},
			variant[func([]uint32) int]{value: utf8LengthFromUTF32Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32) int]{value: utf8LengthFromUTF32Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		utf16LengthFromUTF32: selectVariant(
			input,
			variant[func([]uint32) int]{value: utf16LengthFromUTF32Scalar, kind: implementationScalar, available: true},
			variant[func([]uint32) int]{value: archsimdUTF16Len32, kind: implementationArchsimd, required: cpuAVX2, available: archsimdUTF16Len32 != nil},
			variant[func([]uint32) int]{value: utf16LengthFromUTF32Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint32) int]{value: utf16LengthFromUTF32Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		detectEncodings: selectVariant(
			input,
			variant[func([]byte) Encoding]{value: detectEncodingsScalar, kind: implementationScalar, available: true},
			variant[func([]byte) Encoding]{value: archsimdDetectFn, kind: implementationArchsimd, required: cpuAVX2, available: archsimdDetectFn != nil},
			variant[func([]byte) Encoding]{value: detectEncodingsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) Encoding]{value: detectEncodingsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		find: selectVariant(
			input,
			variant[func([]byte, byte) int]{value: findHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, byte) int]{value: findWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
			variant[func([]byte, byte) int]{value: findScalar, kind: implementationScalar, available: true},
			variant[func([]byte, byte) int]{value: archsimdFindFn, kind: implementationArchsimd, required: cpuAVX2, available: archsimdFindFn != nil},
		),
		findUTF16: selectVariant(
			input,
			variant[func([]uint16, uint16) int]{value: findUTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, uint16) int]{value: archsimdFindUTF16Fn, kind: implementationArchsimd, required: cpuAVX2, available: archsimdFindUTF16Fn != nil},
			variant[func([]uint16, uint16) int]{value: findUTF16Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, uint16) int]{value: findUTF16Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
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
			variant[func([]byte) int]{value: binaryLengthFromBase64Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte) int]{value: binaryLengthFromBase64Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
			variant[func([]byte) int]{value: binaryLengthFromBase64Scalar, kind: implementationScalar, available: true},
			variant[func([]byte) int]{value: binaryLengthFromBase64Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
		),
		binaryLengthFromBase64UTF16: selectVariant(
			input,
			variant[func([]uint16) int]{value: binaryLengthFromBase64UTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16) int]{value: binaryLengthFromBase64UTF16Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
			variant[func([]uint16) int]{value: binaryLengthFromBase64UTF16Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16) int]{value: binaryLengthFromBase64UTF16Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		base64ToBinary: selectVariant(
			input,
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryArchsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		base64ToBinaryUTF16: selectVariant(
			input,
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryUTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryUTF16Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryUTF16Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) Result]{value: base64ToBinaryUTF16Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		base64ToBinaryDetails: selectVariant(
			input,
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsArchsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		base64ToBinaryDetailsUTF16: selectVariant(
			input,
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsUTF16Scalar, kind: implementationScalar, available: true},
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsUTF16Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsUTF16Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]uint16, []byte, Base64Options, LastChunkHandlingOptions) FullResult]{value: base64ToBinaryDetailsUTF16Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		base64ToBinarySafe:      base64ToBinarySafeScalar,
		base64ToBinarySafeUTF16: base64ToBinarySafeUTF16Scalar,
		binaryToBase64: selectVariant(
			input,
			variant[func([]byte, []byte, Base64Options) int]{value: binaryToBase64Scalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte, Base64Options) int]{value: binaryToBase64Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
			variant[func([]byte, []byte, Base64Options) int]{value: binaryToBase64Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []byte, Base64Options) int]{value: binaryToBase64Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
		binaryToBase64WithLines: selectVariant(
			input,
			variant[func([]byte, []byte, int, Base64Options) int]{value: binaryToBase64WithLinesScalar, kind: implementationScalar, available: true},
			variant[func([]byte, []byte, int, Base64Options) int]{value: binaryToBase64WithLinesArchsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
			variant[func([]byte, []byte, int, Base64Options) int]{value: binaryToBase64WithLinesHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
			variant[func([]byte, []byte, int, Base64Options) int]{value: binaryToBase64WithLinesWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		),
	}
}

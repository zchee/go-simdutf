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
	archsimdLatin1Length := archsimdUTF8LengthFromLatin1()
	archsimdLatin1UTF8 := archsimdConvertLatin1ToUTF8()
	archsimdLatin1UTF16LE := archsimdConvertLatin1ToUTF16LE()
	archsimdLatin1UTF16BE := archsimdConvertLatin1ToUTF16BE()
	archsimdLatin1UTF32 := archsimdConvertLatin1ToUTF32()
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
		utf16LengthFromUTF8: selectVariant(input,
			variant[func([]byte) int]{value: utf16LengthFromUTF8Westmere, kind: implementationWestmere, available: true},
			variant[func([]byte) int]{value: utf16LengthFromUTF8Scalar, kind: implementationScalar, available: true},
		),
		utf32LengthFromUTF8: selectVariant(input,
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
	}
}

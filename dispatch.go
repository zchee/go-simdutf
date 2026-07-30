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

package simdutf

import (
	"os"
	"strings"
)

// Go-only dispatch glue based on the first-supported priority semantics in
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/implementation.cpp
// and the per-symbol ISA/object-proof policy in
// docs/porting/provenance.md; this is not an
// algorithm translation.

type cpuFeatures uint16

const (
	cpuSSE42 cpuFeatures = 1 << iota
	cpuPOPCNT
	cpuAVX2
	cpuBMI1
	cpuBMI2
	cpuLZCNT
	cpuNEON
)

type implementationKind uint8

const (
	implementationScalar implementationKind = iota
	implementationWestmere
	implementationHaswell
	implementationNEON
	implementationArchsimd
)

type selectionInput struct {
	features     cpuFeatures
	archsimdAVX2 bool
}

type variant[T any] struct {
	value     T
	kind      implementationKind
	required  cpuFeatures
	available bool
}

func (candidate variant[T]) supportedBy(input selectionInput) bool {
	if !candidate.available {
		return false
	}
	if candidate.kind == implementationArchsimd && !input.archsimdAVX2 {
		return false
	}
	return input.features&candidate.required == candidate.required
}

func selectVariant[T any](input selectionInput, candidates ...variant[T]) T {
	if force, ok := forcedImplementationKind(); ok {
		for _, candidate := range candidates {
			if candidate.kind == force && candidate.supportedBy(input) {
				return candidate.value
			}
		}
		// Operations without the forced provider keep first-supported selection.
	}
	for _, candidate := range candidates {
		if candidate.supportedBy(input) {
			return candidate.value
		}
	}
	panic("simdutf: internal dispatch has no available implementation")
}

func forcedImplementationKind() (implementationKind, bool) {
	// Qualification campaigns set SIMDUTF_BENCH_EXPECT_TIER; FORCE_PROVIDER is a local alias.
	raw := strings.TrimSpace(os.Getenv("SIMDUTF_FORCE_PROVIDER"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("SIMDUTF_BENCH_EXPECT_TIER"))
	}
	switch strings.ToLower(raw) {
	case "":
		return 0, false
	case "scalar":
		return implementationScalar, true
	case "westmere":
		return implementationWestmere, true
	case "haswell":
		return implementationHaswell, true
	case "neon":
		return implementationNEON, true
	case "archsimd":
		return implementationArchsimd, true
	default:
		if strings.TrimSpace(os.Getenv("SIMDUTF_FORCE_PROVIDER")) != "" {
			panic("simdutf: unsupported SIMDUTF_FORCE_PROVIDER value")
		}
		// Unknown EXPECT_TIER values are ignored for selection; the qualification
		// guard remains responsible for rejecting mismatched providers.
		return 0, false
	}
}

type implementation struct {
	validateUTF8                     func([]byte) bool
	validateUTF8WithErrors           func([]byte) Result
	countUTF8                        func([]byte) int
	latin1LengthFromUTF8             func([]byte) int
	utf8LengthFromLatin1             func([]byte) int
	utf16LengthFromUTF8              func([]byte) int
	utf32LengthFromUTF8              func([]byte) int
	validateASCII                    func([]byte) bool
	validateASCIIWithErrors          func([]byte) Result
	validateUTF16LEAsASCII           func([]uint16) bool
	validateUTF16BEAsASCII           func([]uint16) bool
	validateUTF16LE                  func([]uint16) bool
	validateUTF16BE                  func([]uint16) bool
	validateUTF16LEWithErrors        func([]uint16) Result
	validateUTF16BEWithErrors        func([]uint16) Result
	toWellFormedUTF16LE              func([]uint16, []uint16)
	toWellFormedUTF16BE              func([]uint16, []uint16)
	validateUTF32                    func([]uint32) bool
	validateUTF32WithErrors          func([]uint32) Result
	convertLatin1ToUTF8              func([]byte, []byte) int
	convertLatin1ToUTF16LE           func([]byte, []uint16) int
	convertLatin1ToUTF16BE           func([]byte, []uint16) int
	convertLatin1ToUTF32             func([]byte, []uint32) int
	convertUTF8ToLatin1              func([]byte, []byte) int
	convertUTF8ToLatin1WithErrors    func([]byte, []byte) Result
	convertValidUTF8ToLatin1         func([]byte, []byte) int
	convertUTF8ToUTF16LE             func([]byte, []uint16) int
	convertUTF8ToUTF16BE             func([]byte, []uint16) int
	convertUTF8ToUTF16LEWithErrors   func([]byte, []uint16) Result
	convertUTF8ToUTF16BEWithErrors   func([]byte, []uint16) Result
	convertValidUTF8ToUTF16LE        func([]byte, []uint16) int
	convertValidUTF8ToUTF16BE        func([]byte, []uint16) int
	convertUTF8ToUTF32               func([]byte, []uint32) int
	convertUTF8ToUTF32WithErrors     func([]byte, []uint32) Result
	convertValidUTF8ToUTF32          func([]byte, []uint32) int
	convertUTF16LEToLatin1           func([]uint16, []byte) int
	convertUTF16BEToLatin1           func([]uint16, []byte) int
	convertUTF16LEToLatin1WithErrors func([]uint16, []byte) Result
	convertUTF16BEToLatin1WithErrors func([]uint16, []byte) Result
	convertValidUTF16LEToLatin1      func([]uint16, []byte) int
	convertValidUTF16BEToLatin1      func([]uint16, []byte) int
	convertUTF16LEToUTF32            func([]uint16, []uint32) int
	convertUTF16BEToUTF32            func([]uint16, []uint32) int
	convertUTF16LEToUTF32WithErrors  func([]uint16, []uint32) Result
	convertUTF16BEToUTF32WithErrors  func([]uint16, []uint32) Result
	convertValidUTF16LEToUTF32       func([]uint16, []uint32) int
	convertValidUTF16BEToUTF32       func([]uint16, []uint32) int
}

var activeImplementation = makeImplementation(detectSelectionInput())

func detectSelectionInput() selectionInput {
	return selectionInput{
		features:     detectHostFeatures(),
		archsimdAVX2: archsimdAVX2Available(),
	}
}

func archsimdProvidersAvailable() bool {
	available := archsimdValidateASCII() != nil
	if (archsimdValidateASCIIWithErrors() != nil) != available ||
		(archsimdValidateUTF16LEAsASCII() != nil) != available ||
		(archsimdValidateUTF16BEAsASCII() != nil) != available {
		panic("simdutf: internal dispatch has incomplete ASCII archsimd providers")
	}
	return available
}

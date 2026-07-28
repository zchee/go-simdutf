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

import (
	"bytes"
	"encoding/binary"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Hand-authored Go-only direct differential and complete-block contract
// coverage for the lookup4 assembly ports pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216 and
// src/generic/utf8_validation/utf8_validator.h:10-80.

func TestValidateUTF8AMD64Lookup4RODATA(t *testing.T) {
	// Exact lookup bytes and masks derive from the pinned
	// src/generic/utf8_validation/utf8_lookup4_algorithm.h:16-108. The 0x60
	// and 0x70 subtraction constants derive from the pinned
	// src/westmere/implementation.cpp:19-28 and
	// src/haswell/implementation.cpp:19-28 continuation predicates. Haswell
	// VPSHUFB requires each 16-byte lookup table in both 128-bit lanes.
	tables := []struct {
		name string
		want [32]byte
	}{
		{
			name: "utf8LookupHigh",
			want: [32]byte{
				0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
				0x80, 0x80, 0x80, 0x80, 0x21, 0x01, 0x15, 0x49,
				0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
				0x80, 0x80, 0x80, 0x80, 0x21, 0x01, 0x15, 0x49,
			},
		},
		{
			name: "utf8LookupLow",
			want: [32]byte{
				0xe7, 0xa3, 0x83, 0x83, 0x8b, 0xcb, 0xcb, 0xcb,
				0xcb, 0xcb, 0xcb, 0xcb, 0xcb, 0xdb, 0xcb, 0xcb,
				0xe7, 0xa3, 0x83, 0x83, 0x8b, 0xcb, 0xcb, 0xcb,
				0xcb, 0xcb, 0xcb, 0xcb, 0xcb, 0xdb, 0xcb, 0xcb,
			},
		},
		{
			name: "utf8LookupInput",
			want: [32]byte{
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0xe6, 0xae, 0xba, 0xba, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0xe6, 0xae, 0xba, 0xba, 0x01, 0x01, 0x01, 0x01,
			},
		},
		{name: "utf8NibbleMask", want: repeatedUTF8AMD64TableByte(0x0f)},
		{name: "utf8Sub60", want: repeatedUTF8AMD64TableByte(0x60)},
		{name: "utf8Sub70", want: repeatedUTF8AMD64TableByte(0x70)},
		{name: "utf8Bit80", want: repeatedUTF8AMD64TableByte(0x80)},
	}

	source, err := os.ReadFile("utf8_amd64.s")
	if err != nil {
		t.Fatal(err)
	}
	dataPattern := regexp.MustCompile(`^DATA ·([[:alnum:]]+)<>\+([0-9]+)\(SB\)/8, \$(0x[0-9a-fA-F]{16})$`)
	globlPattern := regexp.MustCompile(`^GLOBL ·([[:alnum:]]+)<>\(SB\), RODATA\|NOPTR, \$32$`)
	var dataRecords, globlRecords [][]string
	for lineNumber, line := range strings.Split(string(source), "\n") {
		switch {
		case strings.HasPrefix(line, "DATA "):
			match := dataPattern.FindStringSubmatch(line)
			if match == nil {
				t.Fatalf("utf8_amd64.s:%d: malformed DATA declaration %q", lineNumber+1, line)
			}
			dataRecords = append(dataRecords, match)
		case strings.HasPrefix(line, "GLOBL "):
			match := globlPattern.FindStringSubmatch(line)
			if match == nil {
				t.Fatalf("utf8_amd64.s:%d: malformed GLOBL declaration %q", lineNumber+1, line)
			}
			globlRecords = append(globlRecords, match)
		}
	}

	if got, want := len(dataRecords), len(tables)*4; got != want {
		t.Fatalf("DATA /8 declaration count = %d, want %d", got, want)
	}
	if got, want := len(globlRecords), len(tables); got != want {
		t.Fatalf("GLOBL RODATA|NOPTR, $32 declaration count = %d, want %d", got, want)
	}
	for tableIndex, table := range tables {
		var got [32]byte
		for word := 0; word < 4; word++ {
			recordIndex := tableIndex*4 + word
			record := dataRecords[recordIndex]
			if record[1] != table.name {
				t.Fatalf("DATA declaration %d symbol = %q, want %q", recordIndex, record[1], table.name)
			}
			wantOffset := strconv.Itoa(word * 8)
			if record[2] != wantOffset {
				t.Fatalf("DATA declaration %d offset = %q, want %q", recordIndex, record[2], wantOffset)
			}
			literal, err := strconv.ParseUint(record[3], 0, 64)
			if err != nil {
				t.Fatalf("DATA declaration %d literal %q: %v", recordIndex, record[3], err)
			}
			binary.LittleEndian.PutUint64(got[word*8:], literal)
		}
		if got != table.want {
			t.Errorf("%s bytes = % x, want % x", table.name, got, table.want)
		}
		if gotName := globlRecords[tableIndex][1]; gotName != table.name {
			t.Errorf("GLOBL declaration %d symbol = %q, want exact declaration for %q", tableIndex, gotName, table.name)
		}
	}
}

func TestValidateUTF8AMD64ScalarCutoffSourceContract(t *testing.T) {
	// The pinned generic driver only enters lookup4 while it has a complete
	// 64-byte block. Westmere preserves that structural cutoff. Haswell requires
	// two complete blocks because the one-block class regresses against scalar on
	// the required amd64 host. Lock both Go wrapper policies before an ABI0 prefix
	// symbol can be invoked.
	source, err := os.ReadFile("utf8_amd64.go")
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"ValidateUTF8/Westmere": `func validateUTF8Westmere(input []byte) bool {
	if len(input) < 64 {
		return validateUTF8Scalar(input)
	}
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixWestmere(input)).Error == Success
}`,
		"ValidateUTF8WithErrors/Westmere": `func validateUTF8WithErrorsWestmere(input []byte) Result {
	if len(input) < 64 {
		return validateUTF8WithErrorsScalar(input)
	}
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixWestmere(input))
}`,
		"ValidateUTF8/Haswell": `func validateUTF8Haswell(input []byte) bool {
	if len(input) < 128 {
		return validateUTF8Scalar(input)
	}
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixHaswell(input)).Error == Success
}`,
		"ValidateUTF8WithErrors/Haswell": `func validateUTF8WithErrorsHaswell(input []byte) Result {
	if len(input) < 128 {
		return validateUTF8WithErrorsScalar(input)
	}
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixHaswell(input))
}`,
	} {
		t.Run(name, func(t *testing.T) {
			if count := strings.Count(string(source), want); count != 1 {
				t.Fatalf("exact short-input scalar cutoff contract occurs %d times, want 1\n%s", count, want)
			}
		})
	}
}

func repeatedUTF8AMD64TableByte(value byte) (table [32]byte) {
	for i := range table {
		table[i] = value
	}
	return table
}

func TestValidateUTF8AMD64VariantRegistries(t *testing.T) {
	want := map[string]struct {
		kind     implementationKind
		required cpuFeatures
	}{
		"westmere": {implementationWestmere, cpuSSSE3},
		"haswell":  {implementationHaswell, cpuAVX2},
	}
	// The direct archsimd implementation remains registered for benchmarks and
	// scalar-differential fuzzing even when the performance no-go keeps its
	// production provider unavailable.
	if archsimdUTF8DirectVariantsExpected() {
		want["archsimd"] = struct {
			kind     implementationKind
			required cpuFeatures
		}{implementationArchsimd, cpuAVX2}
	}
	check := func(name string, variants []utf8DirectVariant) {
		t.Helper()
		if len(variants) != len(want) {
			t.Fatalf("%s registry has %d variants, want %d", name, len(variants), len(want))
		}
		for _, candidate := range variants {
			expected, ok := want[candidate.name]
			if !ok {
				t.Errorf("%s registry contains unexpected variant %q", name, candidate.name)
				continue
			}
			if candidate.validate.value == nil || !candidate.validate.available || candidate.validate.kind != expected.kind || candidate.validate.required != expected.required {
				t.Errorf("%s %s ValidateUTF8 registration = {kind:%v required:%#x available:%t}, want {kind:%v required:%#x available:true}", name, candidate.name, candidate.validate.kind, candidate.validate.required, candidate.validate.available, expected.kind, expected.required)
			}
			if candidate.withErrors.value == nil || !candidate.withErrors.available || candidate.withErrors.kind != expected.kind || candidate.withErrors.required != expected.required {
				t.Errorf("%s %s ValidateUTF8WithErrors registration = {kind:%v required:%#x available:%t}, want {kind:%v required:%#x available:true}", name, candidate.name, candidate.withErrors.kind, candidate.withErrors.required, candidate.withErrors.available, expected.kind, expected.required)
			}
		}
	}

	check("direct", utf8DirectVariants)
	fuzz := make([]utf8DirectVariant, len(utf8FuzzVariants))
	for i, candidate := range utf8FuzzVariants {
		fuzz[i] = utf8DirectVariant(candidate)
	}
	check("fuzz", fuzz)
}

func TestValidateUTF8WestmereVariantFeatureGate(t *testing.T) {
	for _, candidate := range utf8DirectVariants {
		if candidate.name != "westmere" {
			continue
		}
		withSSSE3 := selectionInput{features: cpuSSSE3}
		if !candidate.validate.supportedBy(withSSSE3) {
			t.Error("ValidateUTF8 Westmere cell rejected SSSE3")
		}
		if !candidate.withErrors.supportedBy(withSSSE3) {
			t.Error("ValidateUTF8WithErrors Westmere cell rejected SSSE3")
		}
		if candidate.validate.supportedBy(selectionInput{}) {
			t.Error("ValidateUTF8 Westmere cell accepted missing SSSE3")
		}
		if candidate.withErrors.supportedBy(selectionInput{}) {
			t.Error("ValidateUTF8WithErrors Westmere cell accepted missing SSSE3")
		}
		return
	}
	t.Fatal("direct registry has no Westmere variant")
}

func TestValidateUTF8AMD64ScalarParity(t *testing.T) {
	inputs := [][]byte{nil, {}}
	for _, length := range []int{15, 16, 17, 31, 32, 33, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129} {
		inputs = append(inputs, bytes.Repeat([]byte{'a'}, length))
	}
	valid := [][]byte{{0xc2, 0x80}, {0xe0, 0xa0, 0x80}, {0xed, 0x9f, 0xbf}, {0xf0, 0x90, 0x80, 0x80}, {0xf4, 0x8f, 0xbf, 0xbf}}
	for _, boundary := range []int{16, 32, 48, 64, 80, 96, 128} {
		for _, sequence := range valid {
			for split := 1; split < len(sequence); split++ {
				input := bytes.Repeat([]byte{'a'}, boundary-split)
				input = append(input, sequence...)
				input = append(input, bytes.Repeat([]byte{'b'}, 67)...)
				inputs = append(inputs, input)
			}
		}
	}
	invalid := [][]byte{
		{0x80},
		{0xff},
		{0xc0, 0x80},
		{0xe0, 0x80, 0x80},
		{0xed, 0xa0, 0x80},
		{0xf0, 0x80, 0x80, 0x80},
		{0xf4, 0x90, 0x80, 0x80},
		{0xf5, 0x80, 0x80, 0x80},
		{0xc2},
		{0xe1, 0x80},
		{0xf0, 0x90, 0x80},
		{0xe1, 0x80, 'x'},
	}
	for _, prefix := range []int{0, 15, 16, 31, 32, 61, 62, 63, 64, 65, 81, 126, 127, 128} {
		for _, suffix := range invalid {
			input := bytes.Repeat([]byte{'a'}, prefix)
			inputs = append(inputs, append(input, suffix...))
		}
	}
	for i, input := range inputs {
		t.Run(strconv.Itoa(i)+"/length="+strconv.Itoa(len(input)), func(t *testing.T) {
			checkUTF8AMD64Variants(t, input)
		})
	}
}

func TestValidateUTF8AMD64PrefixStopsAtFirstFailingBlock(t *testing.T) {
	for _, test := range []struct {
		name string
		pos  int
		want int
	}{
		{"first block", 30, 0},
		{"second block", 64 + 30, 64},
		{"third block", 128 + 30, 128},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := bytes.Repeat([]byte{'a'}, 192)
			input[test.pos] = 0x80
			for _, candidate := range utf8AMD64RawVariants() {
				if !candidate.supported {
					continue
				}
				if got := candidate.prefix(input); got != test.want {
					t.Errorf("%s prefix = %d, want %d", candidate.name, got, test.want)
				}
			}
			checkUTF8AMD64Variants(t, input)
		})
	}
}

func TestValidateUTF8AMD64PrefixAcceleratesNonASCIIBlocks(t *testing.T) {
	sequence := []byte{0xc2, 0x80, 0xe0, 0xa0, 0x80, 0xf0, 0x90, 0x80, 0x80}
	input := make([]byte, 0, 128)
	for len(input)+len(sequence) <= 128 {
		input = append(input, sequence...)
	}
	input = append(input, bytes.Repeat([]byte{'a'}, 128-len(input))...)
	for _, candidate := range utf8AMD64RawVariants() {
		if !candidate.supported {
			continue
		}
		if got := candidate.prefix(input); got != len(input) {
			t.Errorf("%s non-ASCII prefix = %d, want %d", candidate.name, got, len(input))
		}
	}
	checkUTF8AMD64Variants(t, input)
}

func TestValidateUTF8AMD64IncompleteFullBlockAndNextBlock(t *testing.T) {
	for _, position := range []int{61, 62, 63, 125, 126, 127} {
		for _, tail := range [][]byte{{0xc2}, {0xe1, 0x80}, {0xf1, 0x80, 0x80}, {0xe1, 0x80, 'x'}} {
			input := bytes.Repeat([]byte{'a'}, position)
			input = append(input, tail...)
			checkUTF8AMD64Variants(t, input)
		}
	}
}

func TestValidateUTF8AMD64DoesNotWriteInput(t *testing.T) {
	backing := make([]byte, 259)
	backing[0], backing[len(backing)-1] = 0xa5, 0x5a
	for i := 1; i < len(backing)-1; i++ {
		backing[i] = byte(i & 0x7f)
	}
	backing[64] = 0xf0
	before := slices.Clone(backing)
	input := backing[1 : len(backing)-1]
	checkUTF8AMD64Variants(t, input)
	if !slices.Equal(backing, before) {
		t.Fatal("amd64 UTF-8 validators modified input or canaries")
	}
}

type utf8AMD64RawVariant struct {
	name      string
	supported bool
	prefix    func([]byte) int
	validate  func([]byte) bool
	errors    func([]byte) Result
}

func utf8AMD64RawVariants() []utf8AMD64RawVariant {
	features := detectSelectionInput().features
	return []utf8AMD64RawVariant{
		{"westmere", features&cpuSSSE3 == cpuSSSE3, validateUTF8PrefixWestmere, validateUTF8Westmere, validateUTF8WithErrorsWestmere},
		{"haswell", features&cpuAVX2 == cpuAVX2, validateUTF8PrefixHaswell, validateUTF8Haswell, validateUTF8WithErrorsHaswell},
	}
}

func checkUTF8AMD64Variants(t *testing.T, input []byte) {
	t.Helper()
	wantBool := validateUTF8Scalar(input)
	wantResult := validateUTF8WithErrorsScalar(input)
	for _, candidate := range utf8AMD64RawVariants() {
		if !candidate.supported {
			continue
		}
		if got := candidate.validate(input); got != wantBool {
			t.Errorf("%s validate = %t, scalar = %t for %x", candidate.name, got, wantBool, input)
		}
		if got := candidate.errors(input); got != wantResult {
			t.Errorf("%s with errors = %+v, scalar = %+v for %x", candidate.name, got, wantResult, input)
		}
	}
}

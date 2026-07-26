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

//go:build amd64 && (darwin || linux)

package simdutf

import (
	"math/bits"
	"os"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

// Hand-authored Go differential and block-contract tests for the independent
// assembly translation pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.

func TestValidateASCIIAMD64Direct(t *testing.T) {
	variants := []struct {
		name       string
		available  bool
		prefix     func([]byte) int
		validate   func([]byte) bool
		withErrors func([]byte) Result
	}{
		{name: "westmere", available: true, prefix: validateASCIIPrefixWestmere, validate: validateASCIIWestmere, withErrors: validateASCIIWithErrorsWestmere},
		{name: "haswell", available: detectSelectionInput().features&cpuAVX2 != 0, prefix: validateASCIIPrefixHaswell, validate: validateASCIIHaswell, withErrors: validateASCIIWithErrorsHaswell},
	}
	lengths := [...]int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129}

	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			if !variant.available {
				t.Skip("AVX2 is unavailable")
			}
			for _, length := range lengths {
				invalidPositions := [][]int{nil}
				if length != 0 {
					invalidPositions = append(invalidPositions,
						[]int{0},
						[]int{length / 2},
						[]int{length - 1},
						[]int{0, length / 2, length - 1},
					)
				}
				for caseIndex, positions := range invalidPositions {
					input := make([]byte, length)
					for _, position := range positions {
						input[position] = 0x80
					}
					before := slices.Clone(input)
					wantPrefix := len(input) &^ 63
					for block := 0; block < wantPrefix; block += 64 {
						if !validateASCIIScalar(input[block : block+64]) {
							wantPrefix = block
							break
						}
					}
					if got := variant.prefix(input); got != wantPrefix {
						t.Errorf("length %d case %d prefix = %d, want %d", length, caseIndex, got, wantPrefix)
					}
					if got, want := variant.validate(input), validateASCIIScalar(input); got != want {
						t.Errorf("length %d case %d validate = %v, want %v", length, caseIndex, got, want)
					}
					if got, want := variant.withErrors(input), validateASCIIWithErrorsScalar(input); got != want {
						t.Errorf("length %d case %d result = %+v, want %+v", length, caseIndex, got, want)
					}
					if !slices.Equal(input, before) {
						t.Fatalf("length %d case %d modified input", length, caseIndex)
					}
				}
			}
		})
	}
}

func TestValidateUTF16AsASCIIAMD64Direct(t *testing.T) {
	type endianVariant struct {
		name     string
		little   bool
		prefix   func([]uint16) int
		validate func([]uint16) bool
		scalar   func([]uint16) bool
	}
	tiers := []struct {
		name      string
		available bool
		variants  []endianVariant
	}{
		{name: "westmere", available: true, variants: []endianVariant{
			{name: "little-endian", little: true, prefix: validateUTF16LEASCIIPrefixWestmere, validate: validateUTF16LEAsASCIIWestmere, scalar: validateUTF16LEAsASCIIScalar},
			{name: "big-endian", prefix: validateUTF16BEASCIIPrefixWestmere, validate: validateUTF16BEAsASCIIWestmere, scalar: validateUTF16BEAsASCIIScalar},
		}},
		{name: "haswell", available: detectSelectionInput().features&cpuAVX2 != 0, variants: []endianVariant{
			{name: "little-endian", little: true, prefix: validateUTF16LEASCIIPrefixHaswell, validate: validateUTF16LEAsASCIIHaswell, scalar: validateUTF16LEAsASCIIScalar},
			{name: "big-endian", prefix: validateUTF16BEASCIIPrefixHaswell, validate: validateUTF16BEAsASCIIHaswell, scalar: validateUTF16BEAsASCIIScalar},
		}},
	}
	lengths := [...]int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65}

	for _, tier := range tiers {
		t.Run(tier.name, func(t *testing.T) {
			if !tier.available {
				t.Skip("AVX2 is unavailable")
			}
			for _, variant := range tier.variants {
				t.Run(variant.name, func(t *testing.T) {
					for _, length := range lengths {
						positions := []int{-1}
						if length != 0 {
							positions = append(positions, 0, length/2, length-1)
						}
						for caseIndex, position := range positions {
							input := make([]uint16, length)
							for i := range input {
								input[i] = encodeASCIIWord(0x7f, variant.little)
							}
							if position >= 0 {
								input[position] = encodeASCIIWord(0x80, variant.little)
							}
							before := slices.Clone(input)
							wantPrefix := len(input) &^ 31
							for block := 0; block < wantPrefix; block += 32 {
								if !variant.scalar(input[block : block+32]) {
									wantPrefix = block
									break
								}
							}
							if got := variant.prefix(input); got != wantPrefix {
								t.Errorf("length %d case %d prefix = %d, want %d", length, caseIndex, got, wantPrefix)
							}
							if got, want := variant.validate(input), variant.scalar(input); got != want {
								t.Errorf("length %d case %d validate = %v, want %v", length, caseIndex, got, want)
							}
							if !slices.Equal(input, before) {
								t.Fatalf("length %d case %d modified input", length, caseIndex)
							}
						}
					}
				})
			}
		})
	}
}

func TestValidateUTF16AsASCIIAMD64RawSemanticValues(t *testing.T) {
	values := [...]struct {
		semantic uint16
		little   uint16
		big      uint16
	}{
		{semantic: 0x0000, little: 0x0000, big: 0x0000},
		{semantic: 0x007f, little: 0x007f, big: 0x7f00},
		{semantic: 0x0080, little: 0x0080, big: 0x8000},
		{semantic: 0x0100, little: 0x0100, big: 0x0001},
		{semantic: 0x7fff, little: 0x7fff, big: 0xff7f},
		{semantic: 0x8000, little: 0x8000, big: 0x0080},
		{semantic: 0xffff, little: 0xffff, big: 0xffff},
	}
	positions := [...]struct {
		name  string
		index int
	}{
		{name: "first-block-start", index: 0},
		{name: "first-block-middle", index: 15},
		{name: "first-block-end", index: 31},
		{name: "second-block-start", index: 32},
		{name: "second-block-middle", index: 47},
		{name: "second-block-end", index: 63},
		{name: "tail-start", index: 64},
		{name: "tail-middle", index: 67},
		{name: "tail-end", index: 70},
	}
	tiers := [...]struct {
		name      string
		available bool
		lePrefix  func([]uint16) int
		bePrefix  func([]uint16) int
		le        func([]uint16) bool
		be        func([]uint16) bool
	}{
		{
			name: "westmere", available: true,
			lePrefix: validateUTF16LEASCIIPrefixWestmere, bePrefix: validateUTF16BEASCIIPrefixWestmere,
			le: validateUTF16LEAsASCIIWestmere, be: validateUTF16BEAsASCIIWestmere,
		},
		{
			name: "haswell", available: amd64HasAVX2(),
			lePrefix: validateUTF16LEASCIIPrefixHaswell, bePrefix: validateUTF16BEASCIIPrefixHaswell,
			le: validateUTF16LEAsASCIIHaswell, be: validateUTF16BEAsASCIIHaswell,
		},
	}

	for _, tier := range tiers {
		t.Run(tier.name, func(t *testing.T) {
			if !tier.available {
				t.Skip("AVX2 is unavailable")
			}
			for _, value := range values {
				if got := encodeASCIIWord(value.semantic, true); got != value.little {
					t.Fatalf("semantic %#04x LE raw = %#04x, want %#04x", value.semantic, got, value.little)
				}
				if got := encodeASCIIWord(value.semantic, false); got != value.big {
					t.Fatalf("semantic %#04x BE raw = %#04x, want %#04x", value.semantic, got, value.big)
				}
				for _, position := range positions {
					t.Run("value="+strconv.FormatUint(uint64(value.semantic), 16)+"/"+position.name, func(t *testing.T) {
						leInput := make([]uint16, 71)
						beInput := make([]uint16, 71)
						leInput[position.index] = value.little
						beInput[position.index] = value.big
						checkUTF16AMD64AgainstScalar(t, leInput, tier.lePrefix, tier.le, validateUTF16LEAsASCIIScalar)
						checkUTF16AMD64AgainstScalar(t, beInput, tier.bePrefix, tier.be, validateUTF16BEAsASCIIScalar)
					})
				}
			}
		})
	}
}

func TestValidateASCIIAMD64TypedNilDirect(t *testing.T) {
	var byteInput []byte
	var wordInput []uint16
	tiers := [...]struct {
		name            string
		available       bool
		bytePrefix      func([]byte) int
		validateByte    func([]byte) bool
		withErrors      func([]byte) Result
		lePrefix        func([]uint16) int
		bePrefix        func([]uint16) int
		validateUTF16LE func([]uint16) bool
		validateUTF16BE func([]uint16) bool
	}{
		{
			name: "westmere", available: true,
			bytePrefix: validateASCIIPrefixWestmere, validateByte: validateASCIIWestmere, withErrors: validateASCIIWithErrorsWestmere,
			lePrefix: validateUTF16LEASCIIPrefixWestmere, bePrefix: validateUTF16BEASCIIPrefixWestmere,
			validateUTF16LE: validateUTF16LEAsASCIIWestmere, validateUTF16BE: validateUTF16BEAsASCIIWestmere,
		},
		{
			name: "haswell", available: amd64HasAVX2(),
			bytePrefix: validateASCIIPrefixHaswell, validateByte: validateASCIIHaswell, withErrors: validateASCIIWithErrorsHaswell,
			lePrefix: validateUTF16LEASCIIPrefixHaswell, bePrefix: validateUTF16BEASCIIPrefixHaswell,
			validateUTF16LE: validateUTF16LEAsASCIIHaswell, validateUTF16BE: validateUTF16BEAsASCIIHaswell,
		},
	}

	for _, tier := range tiers {
		t.Run(tier.name, func(t *testing.T) {
			if !tier.available {
				t.Skip("AVX2 is unavailable")
			}
			if byteInput != nil || wordInput != nil {
				t.Fatal("typed-nil test inputs unexpectedly became non-nil")
			}
			if got := tier.bytePrefix(byteInput); got != 0 {
				t.Errorf("byte prefix = %d, want 0", got)
			}
			if !tier.validateByte(byteInput) {
				t.Error("byte validator rejected typed nil")
			}
			if got, want := tier.withErrors(byteInput), (Result{Error: Success, Count: 0}); got != want {
				t.Errorf("byte result = %+v, want %+v", got, want)
			}
			if got := tier.lePrefix(wordInput); got != 0 {
				t.Errorf("UTF-16 LE prefix = %d, want 0", got)
			}
			if got := tier.bePrefix(wordInput); got != 0 {
				t.Errorf("UTF-16 BE prefix = %d, want 0", got)
			}
			if !tier.validateUTF16LE(wordInput) {
				t.Error("UTF-16 LE validator rejected typed nil")
			}
			if !tier.validateUTF16BE(wordInput) {
				t.Error("UTF-16 BE validator rejected typed nil")
			}
		})
	}
}

func TestValidateASCIIAMD64GuardPageNoOverread(t *testing.T) {
	variants := []string{"westmere"}
	if amd64HasAVX2() {
		variants = append(variants, "haswell")
	}
	for _, kind := range []string{"byte", "utf16"} {
		lengths := []int{63, 64, 65, 127, 128, 129}
		if kind == "utf16" {
			lengths = []int{31, 32, 33, 63, 64, 65}
		}
		for _, variant := range variants {
			for _, length := range lengths {
				name := kind + "/" + variant + "/length=" + strconv.Itoa(length)
				t.Run(name, func(t *testing.T) {
					runPageGuardSubprocess(t, "TestValidateASCIIAMD64GuardPageHelper", "SIMDUTF_AMD64_GUARD",
						kind+","+variant+","+strconv.Itoa(length))
				})
			}
		}
	}
}

func TestValidateASCIIAMD64GuardPageHelper(t *testing.T) {
	guardCase := os.Getenv("SIMDUTF_AMD64_GUARD")
	if guardCase == "" {
		return
	}
	parts := strings.Split(guardCase, ",")
	if len(parts) != 3 {
		t.Fatalf("invalid guard case %q", guardCase)
	}
	length, err := strconv.Atoi(parts[2])
	if err != nil {
		t.Fatalf("invalid guard length %q: %v", parts[2], err)
	}
	pageSize := os.Getpagesize()
	mapped, err := syscall.Mmap(-1, 0, 2*pageSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := syscall.Munmap(mapped); err != nil {
			t.Errorf("munmap: %v", err)
		}
	}()
	if err := syscall.Mprotect(mapped[pageSize:], syscall.PROT_NONE); err != nil {
		t.Fatal(err)
	}

	switch parts[0] {
	case "byte":
		input := mapped[pageSize-length : pageSize]
		for i := range input {
			input[i] = byte(i & 0x7f)
		}
		switch parts[1] {
		case "westmere":
			checkASCIIAMD64GuardCalls(t, input, validateASCIIPrefixWestmere, validateASCIIWestmere, validateASCIIWithErrorsWestmere)
		case "haswell":
			checkASCIIAMD64GuardCalls(t, input, validateASCIIPrefixHaswell, validateASCIIHaswell, validateASCIIWithErrorsHaswell)
		default:
			t.Fatalf("invalid byte variant %q", parts[1])
		}
	case "utf16":
		// Test-only unsafe conversion is confined to the accessible mmap page;
		// product code remains unsafe-free. The logical slice ends exactly where
		// the PROT_NONE page begins.
		words := unsafe.Slice((*uint16)(unsafe.Pointer(&mapped[0])), pageSize/2)
		input := words[len(words)-length:]
		switch parts[1] {
		case "westmere":
			checkUTF16AMD64GuardCalls(t, input,
				validateUTF16LEASCIIPrefixWestmere, validateUTF16BEASCIIPrefixWestmere,
				validateUTF16LEAsASCIIWestmere, validateUTF16BEAsASCIIWestmere)
		case "haswell":
			checkUTF16AMD64GuardCalls(t, input,
				validateUTF16LEASCIIPrefixHaswell, validateUTF16BEASCIIPrefixHaswell,
				validateUTF16LEAsASCIIHaswell, validateUTF16BEAsASCIIHaswell)
		default:
			t.Fatalf("invalid UTF-16 variant %q", parts[1])
		}
	default:
		t.Fatalf("invalid guard kind %q", parts[0])
	}
}

func checkASCIIAMD64GuardCalls(
	t *testing.T,
	input []byte,
	prefix func([]byte) int,
	validate func([]byte) bool,
	withErrors func([]byte) Result,
) {
	t.Helper()
	if got, want := prefix(input), len(input)&^63; got != want {
		t.Fatalf("prefix = %d, want %d", got, want)
	}
	if !validate(input) {
		t.Fatal("valid guard-page byte input rejected")
	}
	if got, want := withErrors(input), (Result{Error: Success, Count: len(input)}); got != want {
		t.Fatalf("result = %+v, want %+v", got, want)
	}
}

func checkUTF16AMD64GuardCalls(
	t *testing.T,
	input []uint16,
	lePrefix func([]uint16) int,
	bePrefix func([]uint16) int,
	le func([]uint16) bool,
	be func([]uint16) bool,
) {
	t.Helper()
	if got, want := lePrefix(input), len(input)&^31; got != want {
		t.Fatalf("LE prefix = %d, want %d", got, want)
	}
	if got, want := bePrefix(input), len(input)&^31; got != want {
		t.Fatalf("BE prefix = %d, want %d", got, want)
	}
	if !le(input) || !be(input) {
		t.Fatal("valid guard-page UTF-16 input rejected")
	}
}

func checkUTF16AMD64AgainstScalar(
	t *testing.T,
	input []uint16,
	prefix func([]uint16) int,
	validate func([]uint16) bool,
	scalar func([]uint16) bool,
) {
	t.Helper()
	wantPrefix := len(input) &^ 31
	for block := 0; block < wantPrefix; block += 32 {
		if !scalar(input[block : block+32]) {
			wantPrefix = block
			break
		}
	}
	if got := prefix(input); got != wantPrefix {
		t.Fatalf("prefix = %d, want %d", got, wantPrefix)
	}
	if got, want := validate(input), scalar(input); got != want {
		t.Fatalf("validate = %v, want %v", got, want)
	}
}

func amd64HasAVX2() bool {
	return detectSelectionInput().features&cpuAVX2 != 0
}

func encodeASCIIWord(value uint16, little bool) uint16 {
	if little {
		return value
	}
	return bits.ReverseBytes16(value)
}

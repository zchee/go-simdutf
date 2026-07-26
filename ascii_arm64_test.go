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
	"math/bits"
	"slices"
	"strconv"
	"testing"
)

// Hand-authored Go-only tests for arm64 assembly slice boundaries, raw-storage
// endian handling, exact scalar fallback errors, and input immutability. The
// algorithm under test is independently translated from
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/ascii_validation.h:6-45, src/arm64/implementation.cpp:13-16,
// and src/arm64/arm_validate_utf16.cpp:71-91.

func TestValidateASCIINEONBoundaries(t *testing.T) {
	lengths := [...]int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129}
	for _, length := range lengths {
		t.Run(testNameForLength(length), func(t *testing.T) {
			valid := make([]byte, length)
			for i := range valid {
				valid[i] = byte(i % 0x80)
			}
			checkASCIINEON(t, valid)

			positions := uniquePositions(length)
			for _, pos := range positions {
				input := slices.Clone(valid)
				input[pos] = 0x80
				checkASCIINEON(t, input)
			}

			if length > 1 {
				input := slices.Clone(valid)
				input[0], input[length/2], input[length-1] = 0x80, 0xff, 0x81
				checkASCIINEON(t, input)
			}
		})
	}
}

func TestValidateASCIIPrefixNEON(t *testing.T) {
	for _, length := range [...]int{0, 1, 63, 64, 65, 127, 128, 129} {
		valid := make([]byte, length)
		if got, want := validateASCIIPrefixNEON(valid), length&^63; got != want {
			t.Errorf("valid prefix length %d = %d, want %d", length, got, want)
		}
		for _, pos := range uniquePositions(length) {
			input := slices.Clone(valid)
			input[pos] = 0x80
			want := length &^ 63
			if pos < want {
				want = pos &^ 63
			}
			if got := validateASCIIPrefixNEON(input); got != want {
				t.Errorf("invalid prefix length %d position %d = %d, want %d", length, pos, got, want)
			}
		}
	}
}

func TestValidateUTF16AsASCIINEONBoundaries(t *testing.T) {
	lengths := [...]int{0, 1, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65}
	variants := [...]struct {
		name   string
		little bool
		prefix func([]uint16) int
		neon   func([]uint16) bool
		scalar func([]uint16) bool
	}{
		{"little-endian", true, validateUTF16LEASCIIPrefixNEON, validateUTF16LEAsASCIINEON, validateUTF16LEAsASCIIScalar},
		{"big-endian", false, validateUTF16BEASCIIPrefixNEON, validateUTF16BEAsASCIINEON, validateUTF16BEAsASCIIScalar},
	}

	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			for _, length := range lengths {
				t.Run(testNameForLength(length), func(t *testing.T) {
					valid := make([]uint16, length)
					for i := range valid {
						valid[i] = rawUTF16ASCIIWord(uint16(i%0x80), variant.little)
					}
					checkUTF16ASCIINEON(t, valid, variant.prefix, variant.neon, variant.scalar)

					for _, pos := range uniquePositions(length) {
						for _, semantic := range [...]uint16{0x7f, 0x80} {
							input := slices.Clone(valid)
							input[pos] = rawUTF16ASCIIWord(semantic, variant.little)
							checkUTF16ASCIINEON(t, input, variant.prefix, variant.neon, variant.scalar)
						}
					}
				})
			}
		})
	}
}

func TestASCIINEONDoesNotWriteInput(t *testing.T) {
	bytes := make([]byte, 129)
	bytes[63], bytes[128] = 0x80, 0xff
	wantBytes := slices.Clone(bytes)
	validateASCIIPrefixNEON(bytes)
	validateASCIINEON(bytes)
	validateASCIIWithErrorsNEON(bytes)
	if !slices.Equal(bytes, wantBytes) {
		t.Fatal("byte validators modified input")
	}

	words := make([]uint16, 65)
	words[15], words[64] = 0x80, 0xffff
	wantWords := slices.Clone(words)
	validateUTF16LEASCIIPrefixNEON(words)
	validateUTF16BEASCIIPrefixNEON(words)
	validateUTF16LEAsASCIINEON(words)
	validateUTF16BEAsASCIINEON(words)
	if !slices.Equal(words, wantWords) {
		t.Fatal("UTF-16 validators modified input")
	}
}

func FuzzValidateASCIINEONAgainstScalar(f *testing.F) {
	for _, seed := range [][]byte{nil, {}, make([]byte, 63), make([]byte, 64), make([]byte, 65), {0x7f, 0x80, 0xff}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if got, want := validateASCIINEON(input), validateASCIIScalar(input); got != want {
			t.Fatalf("validateASCIINEON = %v, want %v", got, want)
		}
		if got, want := validateASCIIWithErrorsNEON(input), validateASCIIWithErrorsScalar(input); got != want {
			t.Fatalf("validateASCIIWithErrorsNEON = %+v, want %+v", got, want)
		}
	})
}

func FuzzValidateUTF16AsASCIINEONAgainstScalar(f *testing.F) {
	for _, seed := range [][]byte{nil, {}, make([]byte, 30), make([]byte, 32), make([]byte, 34), {0, 0, 0x80, 0}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, bytes []byte) {
		input := make([]uint16, len(bytes)/2)
		for i := range input {
			input[i] = uint16(bytes[2*i]) | uint16(bytes[2*i+1])<<8
		}
		if got, want := validateUTF16LEAsASCIINEON(input), validateUTF16LEAsASCIIScalar(input); got != want {
			t.Fatalf("validateUTF16LEAsASCIINEON = %v, want %v", got, want)
		}
		if got, want := validateUTF16BEAsASCIINEON(input), validateUTF16BEAsASCIIScalar(input); got != want {
			t.Fatalf("validateUTF16BEAsASCIINEON = %v, want %v", got, want)
		}
	})
}

func checkASCIINEON(t *testing.T, input []byte) {
	t.Helper()
	if got, want := validateASCIINEON(input), validateASCIIScalar(input); got != want {
		t.Errorf("validateASCIINEON = %v, want %v", got, want)
	}
	if got, want := validateASCIIWithErrorsNEON(input), validateASCIIWithErrorsScalar(input); got != want {
		t.Errorf("validateASCIIWithErrorsNEON = %+v, want %+v", got, want)
	}
}

func checkUTF16ASCIINEON(
	t *testing.T,
	input []uint16,
	prefix func([]uint16) int,
	neon func([]uint16) bool,
	scalar func([]uint16) bool,
) {
	t.Helper()
	wantValid := scalar(input)
	if got := neon(input); got != wantValid {
		t.Errorf("NEON validator = %v, want %v for %#x", got, wantValid, input)
	}
	wantPrefix := len(input) &^ 15
	if !wantValid {
		for i := 0; i < wantPrefix; i += 16 {
			if !scalar(input[i : i+16]) {
				wantPrefix = i
				break
			}
		}
	}
	if got := prefix(input); got != wantPrefix {
		t.Errorf("NEON prefix = %d, want %d for %#x", got, wantPrefix, input)
	}
}

func rawUTF16ASCIIWord(semantic uint16, little bool) uint16 {
	if little == nativeLittleEndian() {
		return semantic
	}
	return bits.ReverseBytes16(semantic)
}

func uniquePositions(length int) []int {
	if length == 0 {
		return nil
	}
	positions := []int{0, length / 2, length - 1}
	return slices.Compact(positions)
}

func testNameForLength(length int) string {
	return "length_" + strconv.Itoa(length)
}

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
	"slices"
	"testing"
)

// Hand-authored Go-only family differential fuzz coverage for the port pinned
// to simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.
// Scalar functions remain the explicit oracle; this adds no product behavior.

func FuzzValidateASCII(f *testing.F) {
	f.Add(true, uint8(0), []byte(nil))
	f.Add(false, uint8(1), []byte{})
	for index, length := range [...]int{15, 16, 17, 31, 32, 33, 63, 64, 65, 95, 96, 97, 127, 128, 129} {
		valid := make([]byte, length)
		for i := range valid {
			valid[i] = byte((i*29 + 7) & 0x7f)
		}
		f.Add(false, uint8(index*3+2), valid)
		invalid := slices.Clone(valid)
		invalid[length-1] = 0x80 | byte(index)
		f.Add(false, uint8(index*3+3), invalid)
	}
	f.Add(false, uint8(31), []byte{0x7f, 0x80, 0xff, 0x00})

	f.Fuzz(func(t *testing.T, forceNil bool, alignment uint8, fuzzInput []byte) {
		selection := detectSelectionInput()
		if forceNil {
			checkASCIIFuzzVariants(t, selection, nil)
		}

		prefix := int(alignment%32) + 1
		guard := newGuardedSlice(prefix, len(fuzzInput), 33-prefix, byte(0xa5))
		copy(guard.body, fuzzInput)
		before := slices.Clone(guard.storage)
		checkASCIIFuzzVariants(t, selection, guard.body)
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatal("direct ASCII validators modified guarded storage")
		}
	})
}

func FuzzValidateUTF16AsASCII(f *testing.F) {
	f.Add(true, uint8(0), []byte(nil))
	f.Add(false, uint8(1), []byte{})
	for index, length := range [...]int{15, 16, 17, 31, 32, 33, 63, 64, 65} {
		valid := make([]uint16, length)
		for i := range valid {
			valid[i] = uint16((i*29 + 7) & 0x7f)
		}
		f.Add(false, uint8(index*3+2), encodeASCIIFuzzWords(valid))
		invalid := slices.Clone(valid)
		invalid[length-1] = [...]uint16{0x0080, 0x8000, 0xffff}[index%3]
		f.Add(false, uint8(index*3+3), encodeASCIIFuzzWords(invalid))
	}
	f.Add(false, uint8(15), encodeASCIIFuzzWords([]uint16{0x0000, 0x007f, 0x0080, 0x7f00, 0x8000, 0xffff}))

	f.Fuzz(func(t *testing.T, forceNil bool, alignment uint8, raw []byte) {
		selection := detectSelectionInput()
		if forceNil {
			checkUTF16ASCIIFuzzVariants(t, selection, nil)
		}

		input := decodeASCIIFuzzWords(raw)
		prefix := int(alignment%16) + 1
		guard := newGuardedSlice(prefix, len(input), 17-prefix, uint16(0xa55a))
		copy(guard.body, input)
		before := slices.Clone(guard.storage)
		checkUTF16ASCIIFuzzVariants(t, selection, guard.body)
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatal("direct UTF-16 ASCII validators modified guarded storage")
		}
	})
}

func checkASCIIFuzzVariants(t *testing.T, selection selectionInput, input []byte) {
	t.Helper()
	wantBool := validateASCIIScalar(input)
	wantResult := validateASCIIWithErrorsScalar(input)
	for _, candidate := range asciiFuzzVariants {
		if candidate.validate.supportedBy(selection) {
			if got := candidate.validate.value(input); got != wantBool {
				t.Errorf("%s ValidateASCII = %v, want scalar %v", candidate.name, got, wantBool)
			}
		}
		if candidate.withErrors.supportedBy(selection) {
			if got := candidate.withErrors.value(input); got != wantResult {
				t.Errorf("%s ValidateASCIIWithErrors = %+v, want scalar %+v", candidate.name, got, wantResult)
			}
		}
	}
}

func checkUTF16ASCIIFuzzVariants(t *testing.T, selection selectionInput, input []uint16) {
	t.Helper()
	wantLE := validateUTF16LEAsASCIIScalar(input)
	wantBE := validateUTF16BEAsASCIIScalar(input)
	for _, candidate := range utf16ASCIIFuzzVariants {
		if candidate.le.supportedBy(selection) {
			if got := candidate.le.value(input); got != wantLE {
				t.Errorf("%s ValidateUTF16LEAsASCII = %v, want scalar %v", candidate.name, got, wantLE)
			}
		}
		if candidate.be.supportedBy(selection) {
			if got := candidate.be.value(input); got != wantBE {
				t.Errorf("%s ValidateUTF16BEAsASCII = %v, want scalar %v", candidate.name, got, wantBE)
			}
		}
	}
}

func encodeASCIIFuzzWords(words []uint16) []byte {
	encoded := make([]byte, len(words)*2)
	for i, word := range words {
		encoded[2*i] = byte(word)
		encoded[2*i+1] = byte(word >> 8)
	}
	return encoded
}

func decodeASCIIFuzzWords(raw []byte) []uint16 {
	words := make([]uint16, (len(raw)+1)/2)
	for i := range words {
		words[i] = uint16(raw[2*i])
		if 2*i+1 < len(raw) {
			words[i] |= uint16(raw[2*i+1]) << 8
		}
	}
	return words
}

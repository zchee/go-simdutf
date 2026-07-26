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

import (
	"bytes"
	"slices"
	"strconv"
	"testing"
)

// Hand-authored Go-only direct differential coverage for the lookup4 assembly
// port pinned to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216 and
// src/generic/utf8_validation/utf8_validator.h:10-80.

func TestValidateUTF8NEONScalarParity(t *testing.T) {
	inputs := [][]byte{nil, {}}
	for _, length := range []int{15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129} {
		inputs = append(inputs, bytes.Repeat([]byte{'a'}, length))
	}
	validSequences := [][]byte{{0xc2, 0x80}, {0xe0, 0xa0, 0x80}, {0xf0, 0x90, 0x80, 0x80}}
	for _, boundary := range []int{16, 32, 48, 64, 80, 128} {
		for _, sequence := range validSequences {
			for split := 1; split < len(sequence); split++ {
				start := boundary - split
				input := bytes.Repeat([]byte{'a'}, start)
				input = append(input, sequence...)
				input = append(input, bytes.Repeat([]byte{'b'}, 17)...)
				inputs = append(inputs, input)
			}
		}
	}
	invalid := [][]byte{
		{0x80}, {0xff}, {0xc0, 0x80}, {0xe0, 0x80, 0x80},
		{0xed, 0xa0, 0x80}, {0xf0, 0x80, 0x80, 0x80}, {0xf4, 0x90, 0x80, 0x80},
		{0xc2}, {0xe1, 0x80}, {0xf0, 0x90, 0x80},
	}
	for _, prefix := range []int{0, 19, 64, 81, 128} {
		for _, suffix := range invalid {
			input := bytes.Repeat([]byte{'a'}, prefix)
			inputs = append(inputs, append(input, suffix...))
		}
	}
	for i, input := range inputs {
		t.Run(strconv.Itoa(i)+"/length="+strconv.Itoa(len(input)), func(t *testing.T) {
			checkUTF8NEON(t, input)
		})
	}
}

func TestValidateUTF8NEONIncompleteAndNextBlockErrors(t *testing.T) {
	for _, position := range []int{63, 64, 127} {
		for _, lead := range []byte{0xc2, 0xe1, 0xf1} {
			input := bytes.Repeat([]byte{'a'}, position)
			input = append(input, lead)
			checkUTF8NEON(t, input)
		}
	}
	for _, position := range []int{63, 127} {
		input := bytes.Repeat([]byte{'a'}, position)
		input = append(input, 0xe1, 0x80, 'x')
		var remainder [64]byte
		copy(remainder[:], input[len(input)&^63:])
		count, hasError := validateUTF8Lookup4NEON(input, &remainder)
		wantCount := (position + 1) &^ 63
		if hasError == 0 || count != wantCount {
			t.Errorf("error at %d raw result = (%d, %#x), want observing block %d", position, count, hasError, wantCount)
		}
		checkUTF8NEON(t, input)
	}
}

func TestValidateUTF8NEONDoesNotWriteInput(t *testing.T) {
	backing := make([]byte, 131)
	backing[0], backing[len(backing)-1] = 0xa5, 0x5a
	for i := 1; i < len(backing)-1; i++ {
		backing[i] = byte(i & 0x7f)
	}
	backing[64] = 0xf0
	before := slices.Clone(backing)
	input := backing[1 : len(backing)-1]
	validateUTF8NEON(input)
	validateUTF8WithErrorsNEON(input)
	if !slices.Equal(backing, before) {
		t.Fatal("NEON UTF-8 validators modified input or canaries")
	}
}

func checkUTF8NEON(t *testing.T, input []byte) {
	t.Helper()
	if got, want := validateUTF8NEON(input), validateUTF8Scalar(input); got != want {
		t.Errorf("validateUTF8NEON = %t, scalar = %t for %x", got, want, input)
	}
	if got, want := validateUTF8WithErrorsNEON(input), validateUTF8WithErrorsScalar(input); got != want {
		t.Errorf("validateUTF8WithErrorsNEON = %+v, scalar = %+v for %x", got, want, input)
	}
}

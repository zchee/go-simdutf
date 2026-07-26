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

//go:build amd64 && goexperiment.simd

package simdutf

import (
	"bytes"
	"fmt"
	"slices"
	"testing"
)

// Direct differential coverage for the lookup4 algorithm at
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216 and
// src/generic/utf8_validation/utf8_validator.h:10-80.

func requireUTF8ArchsimdAVX2(t *testing.T) {
	t.Helper()
	candidate := variant[func()]{value: func() {}, kind: implementationArchsimd, required: cpuAVX2, available: true}
	if !candidate.supportedBy(detectSelectionInput()) {
		t.Skip("direct UTF-8 archsimd AVX2 implementation is unsupported")
	}
}

func TestValidateUTF8ArchsimdMatchesScalar(t *testing.T) {
	requireUTF8ArchsimdAVX2(t)

	validSequences := [][]byte{
		{'a'}, {0xc2, 0x80}, {0xe0, 0xa0, 0x80}, {0xed, 0x9f, 0xbf},
		{0xf0, 0x90, 0x80, 0x80}, {0xf4, 0x8f, 0xbf, 0xbf},
	}
	invalidSequences := [][]byte{
		{0x80}, {0xff}, {0xc0, 0x80}, {0xe0, 0x80, 0x80},
		{0xed, 0xa0, 0x80}, {0xf0, 0x80, 0x80, 0x80},
		{0xf4, 0x90, 0x80, 0x80}, {0xf5, 0x80, 0x80, 0x80},
		{0xc2}, {0xe1, 0x80}, {0xf0, 0x90, 0x80}, {0xe1, 0x80, 'x'},
	}

	inputs := [][]byte{nil, {}}
	for _, length := range []int{1, 15, 16, 17, 31, 32, 33, 61, 62, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129, 191, 192, 193} {
		inputs = append(inputs, bytes.Repeat([]byte{'a'}, length))
		for _, invalid := range invalidSequences {
			for _, position := range []int{0, length / 2, length} {
				input := bytes.Repeat([]byte{'a'}, position)
				input = append(input, invalid...)
				input = append(input, bytes.Repeat([]byte{'b'}, length-position)...)
				inputs = append(inputs, input)
			}
		}
	}
	for _, boundary := range []int{16, 32, 48, 63, 64, 65, 80, 96, 127, 128} {
		for _, sequence := range validSequences[1:] {
			for split := 1; split < len(sequence); split++ {
				input := bytes.Repeat([]byte{'a'}, boundary-split)
				input = append(input, sequence...)
				input = append(input, bytes.Repeat([]byte{'b'}, 67)...)
				inputs = append(inputs, input)
			}
		}
	}

	for i, input := range inputs {
		t.Run(fmt.Sprintf("case=%d/length=%d", i, len(input)), func(t *testing.T) {
			backing := make([]byte, len(input)+2)
			backing[0], backing[len(backing)-1] = 0xa5, 0x5a
			copy(backing[1:], input)
			before := slices.Clone(backing)
			guarded := backing[1 : len(backing)-1]
			if got, want := validateUTF8Archsimd(guarded), validateUTF8Scalar(guarded); got != want {
				t.Fatalf("validateUTF8Archsimd() = %t, scalar = %t for %x", got, want, guarded)
			}
			if got, want := validateUTF8WithErrorsArchsimd(guarded), validateUTF8WithErrorsScalar(guarded); got != want {
				t.Fatalf("validateUTF8WithErrorsArchsimd() = %+v, scalar = %+v for %x", got, want, guarded)
			}
			if !slices.Equal(backing, before) {
				t.Fatal("archsimd UTF-8 validation modified input or canaries")
			}
		})
	}
}

func TestValidateUTF8PrefixArchsimdStopsAtFirstFailingBlock(t *testing.T) {
	requireUTF8ArchsimdAVX2(t)
	for _, test := range []struct {
		position int
		want     int
	}{{30, 0}, {94, 64}, {158, 128}} {
		input := bytes.Repeat([]byte{'a'}, 192)
		input[test.position] = 0x80
		if got := validateUTF8PrefixArchsimd(input); got != test.want {
			t.Errorf("error at %d: prefix = %d, want %d", test.position, got, test.want)
		}
	}
}

func TestValidateUTF8ArchsimdLaneBridgeOrientation(t *testing.T) {
	requireUTF8ArchsimdAVX2(t)
	for _, position := range []int{16, 32, 48} {
		input := bytes.Repeat([]byte{'a'}, 64)
		input[position] = 0xff
		if got := validateUTF8PrefixArchsimd(input); got != 0 {
			t.Errorf("invalid byte at lane boundary %d: prefix = %d, want 0", position, got)
		}
		if got, want := validateUTF8WithErrorsArchsimd(input), validateUTF8WithErrorsScalar(input); got != want {
			t.Errorf("invalid byte at lane boundary %d: with errors = %+v, scalar = %+v", position, got, want)
		}
	}
	for _, boundary := range []int{16, 32, 48} {
		input := bytes.Repeat([]byte{'a'}, boundary-2)
		input = append(input, 0xf0, 0x90, 0x80, 0x80)
		input = append(input, bytes.Repeat([]byte{'b'}, 66-len(input))...)
		if got, want := validateUTF8WithErrorsArchsimd(input), validateUTF8WithErrorsScalar(input); got != want {
			t.Errorf("valid sequence across lane boundary %d: with errors = %+v, scalar = %+v", boundary, got, want)
		}
	}
}

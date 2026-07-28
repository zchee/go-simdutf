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
	"fmt"
	"testing"
)

// Independently adapted direct differential coverage for the algorithms at
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
// src/generic/ascii_validation.h:6-45 and
// src/haswell/implementation.cpp:278-307. The direct archsimd invocation guard
// follows Go 1.26.5 src/simd/archsimd/cpu_amd64.go:7-61.

var asciiArchsimdTestLengths = [...]int{
	0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129,
}

var utf16ArchsimdTestLengths = [...]int{
	0, 1, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65,
}

func requireASCIIArchsimdAVX2(t *testing.T) {
	t.Helper()
	candidate := variant[func()]{
		value:     func() {},
		kind:      implementationArchsimd,
		required:  cpuAVX2,
		available: true,
	}
	if !candidate.supportedBy(detectSelectionInput()) {
		t.Skip("direct archsimd AVX2 implementation is unsupported")
	}
}

func TestValidateASCIIArchsimdMatchesScalar(t *testing.T) {
	requireASCIIArchsimdAVX2(t)

	for _, length := range asciiArchsimdTestLengths {
		t.Run(fmt.Sprintf("length=%d/valid", length), func(t *testing.T) {
			input := makeASCIIArchsimdInput(length)
			if got, want := validateASCIIArchsimd(input), validateASCIIScalar(input); got != want {
				t.Fatalf("validateASCIIArchsimd() = %t, want %t", got, want)
			}
			if got, want := validateASCIIWithErrorsArchsimd(input), validateASCIIWithErrorsScalar(input); got != want {
				t.Fatalf("validateASCIIWithErrorsArchsimd() = %+v, want %+v", got, want)
			}
		})

		for position := 0; position < length; position++ {
			t.Run(fmt.Sprintf("length=%d/invalid=%d", length, position), func(t *testing.T) {
				input := makeASCIIArchsimdInput(length)
				input[position] = 0x80 | byte(position&0x7f)
				if got, want := validateASCIIArchsimd(input), validateASCIIScalar(input); got != want {
					t.Fatalf("validateASCIIArchsimd() = %t, want %t", got, want)
				}
				want := Result{Error: TooLarge, Count: position}
				if got := validateASCIIWithErrorsArchsimd(input); got != want {
					t.Fatalf("validateASCIIWithErrorsArchsimd() = %+v, want %+v", got, want)
				}
			})
		}

		if length > 1 {
			t.Run(fmt.Sprintf("length=%d/multiple", length), func(t *testing.T) {
				input := makeASCIIArchsimdInput(length)
				positions := [...]int{0, length / 2, length - 1}
				for _, position := range positions {
					input[position] = 0xff
				}
				if validateASCIIArchsimd(input) {
					t.Fatal("validateASCIIArchsimd() = true, want false")
				}
				want := Result{Error: TooLarge, Count: positions[0]}
				if got := validateASCIIWithErrorsArchsimd(input); got != want {
					t.Fatalf("validateASCIIWithErrorsArchsimd() = %+v, want %+v", got, want)
				}
			})
		}
	}
}

func TestValidateUTF16AsASCIIArchsimdMatchesScalar(t *testing.T) {
	requireASCIIArchsimdAVX2(t)

	tests := [...]struct {
		name       string
		validRaw   uint16
		invalidRaw uint16
		archsimd   func([]uint16) bool
		scalar     func([]uint16) bool
	}{
		{
			name:       "little-endian-raw",
			validRaw:   0x007f,
			invalidRaw: 0x0080,
			archsimd:   validateUTF16LEAsASCIIArchsimd,
			scalar:     validateUTF16LEAsASCIIScalar,
		},
		{
			name:       "big-endian-raw",
			validRaw:   0x7f00,
			invalidRaw: 0x8000,
			archsimd:   validateUTF16BEAsASCIIArchsimd,
			scalar:     validateUTF16BEAsASCIIScalar,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, length := range utf16ArchsimdTestLengths {
				t.Run(fmt.Sprintf("length=%d/valid", length), func(t *testing.T) {
					input := make([]uint16, length)
					for i := range input {
						input[i] = test.validRaw
					}
					if got, want := test.archsimd(input), test.scalar(input); got != want {
						t.Fatalf("archsimd() = %t, want scalar %t", got, want)
					}
				})

				for position := 0; position < length; position++ {
					t.Run(fmt.Sprintf("length=%d/invalid=%d", length, position), func(t *testing.T) {
						input := make([]uint16, length)
						for i := range input {
							input[i] = test.validRaw
						}
						input[position] = test.invalidRaw
						if got, want := test.archsimd(input), test.scalar(input); got != want {
							t.Fatalf("archsimd() = %t, want scalar %t", got, want)
						}
					})
				}
			}
		})
	}
}

func makeASCIIArchsimdInput(length int) []byte {
	input := make([]byte, length)
	for i := range input {
		input[i] = byte((i*29 + 7) & 0x7f)
	}
	return input
}

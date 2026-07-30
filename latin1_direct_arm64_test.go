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

// Portions Copyright 2021 The simdutf Authors.
//go:build arm64

package simdutf

// Direct differential coverage for the arm64 Latin-1 NEON translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/implementation.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.

import (
	"bytes"
	"slices"
	"testing"
)

func TestLatin1NEONDirectAgainstScalar(t *testing.T) {
	inputs := [][]byte{
		nil,
		{0},
		bytes.Repeat([]byte{'A'}, 15),
		bytes.Repeat([]byte{'A'}, 16),
		bytes.Repeat([]byte{'A'}, 31),
		bytes.Repeat([]byte{'A'}, 32),
		bytes.Repeat([]byte{'A'}, 63),
		bytes.Repeat([]byte{'A'}, 64),
		append(bytes.Repeat([]byte{'A'}, 63), 0x80),
		append(bytes.Repeat([]byte{0xff}, 64), 0x7f),
	}
	for _, input := range inputs {
		t.Run("length", func(t *testing.T) {
			wantLength := utf8LengthFromLatin1Scalar(input)
			if got := utf8LengthFromLatin1NEON(input); got != wantLength {
				t.Fatalf("UTF-8 length = %d, want %d for %x", got, wantLength, input)
			}

			want8 := make([]byte, wantLength)
			convertLatin1ToUTF8Scalar(input, want8)
			got8 := guardedLatin1Destination[byte](wantLength, 0xa5)
			if got := convertLatin1ToUTF8NEON(input, got8.body); got != len(want8) || !bytes.Equal(got8.body, want8) {
				t.Fatalf("UTF-8 = %x (%d), want %x (%d)", got8.body, got, want8, len(want8))
			}
			got8.require(t)

			want32 := make([]uint32, len(input))
			convertLatin1ToUTF32Scalar(input, want32)
			got32 := guardedLatin1Destination[uint32](len(input), 0xa5a5a5a5)
			if got := convertLatin1ToUTF32NEON(input, got32.body); got != len(input) || !slices.Equal(got32.body, want32) {
				t.Fatalf("UTF-32 = %x (%d), want %x (%d)", got32.body, got, want32, len(input))
			}
			got32.require(t)

			for _, little := range []bool{true, false} {
				want16 := make([]uint16, len(input))
				if little {
					convertLatin1ToUTF16LEScalar(input, want16)
				} else {
					convertLatin1ToUTF16BEScalar(input, want16)
				}
				got16 := guardedLatin1Destination[uint16](len(input), 0xa5a5)
				var got int
				if little {
					got = convertLatin1ToUTF16LENEON(input, got16.body)
				} else {
					got = convertLatin1ToUTF16BENEON(input, got16.body)
				}
				if got != len(input) || !slices.Equal(got16.body, want16) {
					t.Fatalf("UTF-16 little=%t = %x (%d), want %x (%d)", little, got16.body, got, want16, len(input))
				}
				got16.require(t)
			}
		})
	}
}

func TestLatin1NEONDirectPreflightPreservesCanaries(t *testing.T) {
	input := append(bytes.Repeat([]byte{'A'}, 63), 0xff)
	want8 := utf8LengthFromLatin1Scalar(input)
	dst8 := guardedLatin1Destination[byte](want8-1, 0xa5)
	requireLatin1Panic(t, func() { convertLatin1ToUTF8NEON(input, dst8.body) })
	dst8.require(t)

	dst16 := guardedLatin1Destination[uint16](len(input)-1, 0xa5a5)
	requireLatin1Panic(t, func() { convertLatin1ToUTF16LENEON(input, dst16.body) })
	dst16.require(t)
	requireLatin1Panic(t, func() { convertLatin1ToUTF16BENEON(input, dst16.body) })
	dst16.require(t)

	dst32 := guardedLatin1Destination[uint32](len(input)-1, 0xa5a5a5a5)
	requireLatin1Panic(t, func() { convertLatin1ToUTF32NEON(input, dst32.body) })
	dst32.require(t)
}

func FuzzLatin1NEONAgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte{'A'}, 64))
	f.Add(append(bytes.Repeat([]byte{0xff}, 64), 0x80))
	f.Fuzz(func(t *testing.T, input []byte) {
		wantLength := utf8LengthFromLatin1Scalar(input)
		if got := utf8LengthFromLatin1NEON(input); got != wantLength {
			t.Fatalf("UTF-8 length = %d, want %d", got, wantLength)
		}

		want8, got8 := make([]byte, wantLength), make([]byte, wantLength)
		convertLatin1ToUTF8Scalar(input, want8)
		convertLatin1ToUTF8NEON(input, got8)
		if !bytes.Equal(got8, want8) {
			t.Fatalf("UTF-8 = %x, want %x", got8, want8)
		}

		want16, got16 := make([]uint16, len(input)), make([]uint16, len(input))
		convertLatin1ToUTF16LEScalar(input, want16)
		convertLatin1ToUTF16LENEON(input, got16)
		if !slices.Equal(got16, want16) {
			t.Fatalf("UTF-16LE = %x, want %x", got16, want16)
		}
		convertLatin1ToUTF16BEScalar(input, want16)
		convertLatin1ToUTF16BENEON(input, got16)
		if !slices.Equal(got16, want16) {
			t.Fatalf("UTF-16BE = %x, want %x", got16, want16)
		}

		want32, got32 := make([]uint32, len(input)), make([]uint32, len(input))
		convertLatin1ToUTF32Scalar(input, want32)
		convertLatin1ToUTF32NEON(input, got32)
		if !slices.Equal(got32, want32) {
			t.Fatalf("UTF-32 = %x, want %x", got32, want32)
		}
	})
}

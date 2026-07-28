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
	"slices"
	"strconv"
	"testing"
)

// Hand-authored Go-only direct differential coverage for the separate
// Westmere and Haswell count_code_points_bytemask ports pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/utf8.h:21-68.

func TestCountUTF8AMD64ScalarParity(t *testing.T) {
	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 4031, 4032, 4033, 4095, 4096, 4097, 8063, 8064, 8065, 8191, 8192, 8193, 16128, 65536}
	for _, length := range lengths {
		for alignment := 0; alignment < 32; alignment++ {
			t.Run("length="+strconv.Itoa(length)+"/alignment="+strconv.Itoa(alignment), func(t *testing.T) {
				storage := make([]byte, alignment+length+32)
				input := storage[alignment : alignment+length]
				for i := range input {
					input[i] = byte(i*131 + length*17 + alignment)
				}
				checkCountUTF8AMD64(t, input)
			})
		}
	}
}

func TestCountUTF8AMD64AllByteClassesAndSignedPredicate(t *testing.T) {
	classes := make([]byte, 256)
	for value := range classes {
		classes[value] = byte(value)
		want := 0
		if int8(byte(value)) > -65 {
			want = 1
		}
		if got := countUTF8Scalar([]byte{byte(value)}); got != want {
			t.Fatalf("byte %#02x scalar predicate = %d, signed > -65 = %d", value, got, want)
		}
		checkCountUTF8AMD64(t, bytes.Repeat([]byte{byte(value)}, 129))
	}
	checkCountUTF8AMD64(t, classes)
	checkCountUTF8AMD64(t, append(slices.Clone(classes), classes...))
	checkCountUTF8AMD64(t, bytes.Repeat([]byte{0x80, 0xbf, 0x00, 0x7f, 0xc0, 0xff}, 1400))
}

func TestCountUTF8AMD64RawBlockContracts(t *testing.T) {
	for _, length := range []int{0, 1, 63, 64, 65, 127, 128, 129, 4031, 4032, 4033, 4095, 4096, 4097, 8063, 8064, 8065, 8191, 8192, 8193, 16128, 65536} {
		input := make([]byte, length)
		for i := range input {
			input[i] = byte(i*29 + length)
		}
		if got, want := countUTF8BlocksWestmere(input), countUTF8Scalar(input[:length&^63]); got != want {
			t.Errorf("Westmere raw length %d = %d, want %d", length, got, want)
		}
		if hasCountUTF8AVX2() {
			if got, want := countUTF8BlocksHaswell(input), countUTF8Scalar(input[:length&^127]); got != want {
				t.Errorf("Haswell raw length %d = %d, want %d", length, got, want)
			}
		}
	}
}

func TestCountUTF8AMD64AccumulatorFlushBoundaries(t *testing.T) {
	lengths := []int{4031, 4032, 4033, 4095, 4096, 4097, 8063, 8064, 8065, 8191, 8192, 8193, 3*8064 + 1, 1 << 20}
	for _, value := range []byte{0x00, 0x80} {
		for _, length := range lengths {
			t.Run("byte="+strconv.Itoa(int(value))+"/length="+strconv.Itoa(length), func(t *testing.T) {
				input := bytes.Repeat([]byte{value}, length)
				checkCountUTF8AMD64(t, input)
				if got, want := countUTF8BlocksWestmere(input), countUTF8Scalar(input[:length&^63]); got != want {
					t.Errorf("Westmere raw = %d, want %d", got, want)
				}
				if hasCountUTF8AVX2() {
					if got, want := countUTF8BlocksHaswell(input), countUTF8Scalar(input[:length&^127]); got != want {
						t.Errorf("Haswell raw = %d, want %d", got, want)
					}
				}
			})
		}
	}
}

func TestCountUTF8AMD64CanariesAndImmutability(t *testing.T) {
	for _, length := range []int{0, 63, 64, 65, 127, 128, 129, 4031, 4032, 4033, 4095, 4096, 4097, 8063, 8064, 8065, 8191, 8192, 8193} {
		guard := newGuardedSlice(37, length, 41, byte(0xa5))
		for i := range guard.body {
			guard.body[i] = byte(i*73 + length)
		}
		before := slices.Clone(guard.storage)
		checkCountUTF8AMD64(t, guard.body)
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatalf("length %d input or canary modified", length)
		}
	}
}

func checkCountUTF8AMD64(t *testing.T, input []byte) {
	t.Helper()
	want := countUTF8Scalar(input)
	if got := countUTF8Westmere(input); got != want {
		t.Errorf("countUTF8Westmere = %d, scalar = %d for %d bytes", got, want, len(input))
	}
	if hasCountUTF8AVX2() {
		if got := countUTF8Haswell(input); got != want {
			t.Errorf("countUTF8Haswell = %d, scalar = %d for %d bytes", got, want, len(input))
		}
	}
}

func hasCountUTF8AVX2() bool {
	return detectHostFeatures()&cpuAVX2 == cpuAVX2
}

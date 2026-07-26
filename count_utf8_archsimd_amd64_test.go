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
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Hand-authored Go-only direct scalar-differential coverage for the archsimd
// Haswell count_code_points_bytemask adaptation pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee): src/generic/utf8.h:21-68 and
// src/haswell/implementation.cpp:1115-1119.

func TestCountUTF8ArchsimdScalarParity(t *testing.T) {
	requireCountUTF8ArchsimdAVX2(t)
	lengths := []int{0, 1, 31, 32, 33, 63, 64, 65, 95, 96, 97, 127, 128, 129, 255, 256, 257, 8063, 8064, 8065, 8191, 8192, 8193}
	for _, length := range lengths {
		for alignment := 0; alignment < 32; alignment++ {
			t.Run("length="+strconv.Itoa(length)+"/alignment="+strconv.Itoa(alignment), func(t *testing.T) {
				storage := make([]byte, alignment+length+32)
				input := storage[alignment : alignment+length]
				for i := range input {
					input[i] = byte(i*131 + length*17 + alignment)
				}
				checkCountUTF8Archsimd(t, input)
			})
		}
	}
}

func TestCountUTF8ArchsimdAllByteClasses(t *testing.T) {
	requireCountUTF8ArchsimdAVX2(t)
	classes := make([]byte, 256)
	for value := range classes {
		classes[value] = byte(value)
		checkCountUTF8Archsimd(t, bytes.Repeat([]byte{byte(value)}, 129))
	}
	checkCountUTF8Archsimd(t, classes)
	checkCountUTF8Archsimd(t, append(slices.Clone(classes), classes...))
	checkCountUTF8Archsimd(t, bytes.Repeat([]byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xff}, 1400))
}

func TestCountUTF8ArchsimdAccumulatorFlushBoundaries(t *testing.T) {
	requireCountUTF8ArchsimdAVX2(t)
	for _, value := range []byte{0x00, 0x80} {
		for _, length := range []int{8063, 8064, 8065, 8191, 8192, 8193, 3*8064 + 1, 1 << 20} {
			t.Run("byte="+strconv.Itoa(int(value))+"/length="+strconv.Itoa(length), func(t *testing.T) {
				checkCountUTF8Archsimd(t, bytes.Repeat([]byte{value}, length))
			})
		}
	}
}

func TestCountUTF8ArchsimdCanariesAndImmutability(t *testing.T) {
	requireCountUTF8ArchsimdAVX2(t)
	for _, length := range []int{0, 1, 127, 128, 129, 8063, 8064, 8065, 8191, 8192, 8193} {
		guard := newGuardedSlice(37, length, 41, byte(0xa5))
		for i := range guard.body {
			guard.body[i] = byte(i*73 + length)
		}
		before := slices.Clone(guard.storage)
		checkCountUTF8Archsimd(t, guard.body)
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatalf("length %d input or canary modified", length)
		}
	}
}

func TestCountUTF8ArchsimdShortInputScalarCutoffSourceContract(t *testing.T) {
	// The pinned generic driver enters the four-vector bytemask loop only for a
	// complete 128-byte Haswell block. Lock the wrapper control flow so shorter
	// inputs return through the scalar oracle before vector state is initialized.
	source, err := os.ReadFile("count_utf8_archsimd_amd64.go")
	if err != nil {
		t.Fatal(err)
	}
	want := `func countUTF8Archsimd(input []byte) int {
	if len(input) < 128 {
		return countUTF8Scalar(input)
	}`
	if count := strings.Count(string(source), want); count != 1 {
		t.Fatalf("exact short-input scalar cutoff contract occurs %d times, want 1\n%s", count, want)
	}
}

func checkCountUTF8Archsimd(t *testing.T, input []byte) {
	t.Helper()
	if got, want := countUTF8Archsimd(input), countUTF8Scalar(input); got != want {
		t.Errorf("countUTF8Archsimd = %d, scalar = %d for %d bytes", got, want, len(input))
	}
}

func requireCountUTF8ArchsimdAVX2(t *testing.T) {
	t.Helper()
	selection := detectSelectionInput()
	if selection.features&cpuAVX2 != cpuAVX2 || !selection.archsimdAVX2 {
		t.Skip("archsimd CountUTF8 requires repository and archsimd AVX2 gates")
	}
}

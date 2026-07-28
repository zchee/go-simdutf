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

// Hand-authored Go-only direct scalar-differential coverage for the count port
// pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/utf8.h:8-17, src/arm64/implementation.cpp:1113-1117, and
// src/simdutf/arm64/simd.h:446-555.

func TestCountUTF8NEONScalarParity(t *testing.T) {
	checkCountUTF8NEON(t, nil)
	checkCountUTF8NEON(t, []byte{})

	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 61, 62, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129, 256, 1024, 4097, 65536}
	for _, length := range lengths {
		for alignment := 0; alignment < 16; alignment++ {
			t.Run("length="+strconv.Itoa(length)+"/alignment="+strconv.Itoa(alignment), func(t *testing.T) {
				storage := make([]byte, alignment+length+16)
				input := storage[alignment : alignment+length]
				for i := range input {
					input[i] = byte((i*131 + length*17 + alignment) & 0xff)
				}
				checkCountUTF8NEON(t, input)
			})
		}
	}
}

func TestCountUTF8NEONAllByteClasses(t *testing.T) {
	classes := make([]byte, 256)
	for i := range classes {
		classes[i] = byte(i)
	}
	for _, input := range [][]byte{
		classes,
		append(slices.Clone(classes), classes...),
		bytes.Repeat([]byte{0x80, 0xbf, 0x00, 0x7f, 0xc0, 0xff}, 257),
	} {
		checkCountUTF8NEON(t, input)
	}
	for value := 0; value <= 0xff; value++ {
		checkCountUTF8NEON(t, bytes.Repeat([]byte{byte(value)}, 129))
	}
}

// TestCountUTF8NEONByteClassLoweringMatchesSignedGT locks the exact lowering
// used because Go 1.26's Plan 9 arm64 assembler has VCMEQ but no signed integer
// VCMGT mnemonic: (byte >> 6) == 2 is the complement of int8(byte) > -65, so
// subtracting its population yields upstream's non-continuation count.
func TestCountUTF8NEONByteClassLoweringMatchesSignedGT(t *testing.T) {
	for value := 0; value <= 0xff; value++ {
		continuationByNEONLowering := byte(value)>>6 == 2
		continuationByPinnedComparison := !(int8(byte(value)) > -65)
		if continuationByNEONLowering != continuationByPinnedComparison {
			t.Fatalf("byte %#02x lowering continuation = %t, pinned comparison = %t", value, continuationByNEONLowering, continuationByPinnedComparison)
		}
	}
}

func TestCountUTF8BlocksNEONCompleteBlockContract(t *testing.T) {
	for _, length := range []int{0, 1, 63, 64, 65, 127, 128, 129, 4097, 65536} {
		input := make([]byte, length)
		for i := range input {
			input[i] = byte(i*29 + length)
		}
		complete := length &^ 63
		if got, want := countUTF8BlocksNEON(input), countUTF8Scalar(input[:complete]); got != want {
			t.Errorf("length %d block count = %d, want %d", length, got, want)
		}
	}
}

func TestCountUTF8NEONCanariesAndImmutability(t *testing.T) {
	for _, length := range []int{0, 63, 64, 65, 127, 128, 129, 4097} {
		guard := newGuardedSlice(17, length, 19, byte(0xa5))
		for i := range guard.body {
			guard.body[i] = byte(i*73 + length)
		}
		before := slices.Clone(guard.storage)
		checkCountUTF8NEON(t, guard.body)
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatalf("length %d input or canary modified", length)
		}
	}
}

func checkCountUTF8NEON(t *testing.T, input []byte) {
	t.Helper()
	if got, want := countUTF8NEON(input), countUTF8Scalar(input); got != want {
		t.Errorf("countUTF8NEON = %d, scalar = %d for %d bytes", got, want, len(input))
	}
}

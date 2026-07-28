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

// Hand-authored Go-only direct scalar-differential coverage for the arm64
// UTF-8 length ports pinned to
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811): src/generic/utf8.h:8-17,72-86,
// src/arm64/implementation.cpp:1121-1124,1178-1181,1292-1295, and
// src/simdutf/arm64/simd.h:420-529.

func TestUTF8LengthNEONScalarParity(t *testing.T) {
	for _, input := range [][]byte{
		nil,
		{},
		[]byte("plain ASCII"),
		{'a', 0xc2, 0xa2, 0xe2, 0x82, 0xac, 0xf0, 0x90, 0x8d, 0x88},
		{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xf4, 0xf8, 0xff},
	} {
		checkUTF8LengthNEON(t, input)
	}

	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 257, 1024, 4097, 65536}
	for _, length := range lengths {
		for alignment := 0; alignment < 32; alignment++ {
			t.Run("length="+strconv.Itoa(length)+"/alignment="+strconv.Itoa(alignment), func(t *testing.T) {
				guard := newGuardedSlice(alignment, length, 33, byte(0xa5))
				for i := range guard.body {
					guard.body[i] = byte(i*131 + length*17 + alignment)
				}
				before := slices.Clone(guard.storage)
				checkUTF8LengthNEON(t, guard.body)
				guard.requireCanariesIntact(t)
				if !slices.Equal(guard.storage, before) {
					t.Fatal("UTF-8 length NEON input or canary modified")
				}
			})
		}
	}
}

func TestUTF8LengthNEONAllByteValues(t *testing.T) {
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	for _, input := range [][]byte{
		all,
		append(slices.Clone(all), all...),
		bytes.Repeat([]byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xff}, 257),
	} {
		checkUTF8LengthNEON(t, input)
	}
	for value := 0; value <= 0xff; value++ {
		checkUTF8LengthNEON(t, bytes.Repeat([]byte{byte(value)}, 257))
	}
}

// TestUTF16LengthFromUTF8NEONByteClassLowering locks the exact all-byte
// equivalence used by the Plan 9 NEON kernel: continuation bytes are
// (byte>>6)==2, and four-byte leads are (byte>>4)==15.
func TestUTF16LengthFromUTF8NEONByteClassLowering(t *testing.T) {
	for value := 0; value <= 0xff; value++ {
		input := byte(value)
		nonContinuation := input>>6 != 2
		pinnedNonContinuation := int8(input) > -65
		if nonContinuation != pinnedNonContinuation {
			t.Errorf("byte %#02x non-continuation lowering = %t, pinned = %t", value, nonContinuation, pinnedNonContinuation)
		}
		fourByteLead := input>>4 == 15
		pinnedFourByteLead := input >= 0xf0
		if fourByteLead != pinnedFourByteLead {
			t.Errorf("byte %#02x four-byte lowering = %t, pinned = %t", value, fourByteLead, pinnedFourByteLead)
		}
		got := 0
		if nonContinuation {
			got++
		}
		if fourByteLead {
			got++
		}
		if want := utf16LengthFromUTF8Scalar([]byte{input}); got != want {
			t.Errorf("byte %#02x lowered contribution = %d, scalar = %d", value, got, want)
		}
	}
}

func TestUTF16LengthFromUTF8BlocksNEONCompleteBlockContract(t *testing.T) {
	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 257, 4097, 65536}
	for _, length := range lengths {
		for alignment := 0; alignment < 32; alignment++ {
			storage := make([]byte, alignment+length+32)
			input := storage[alignment : alignment+length]
			for i := range input {
				input[i] = byte(i*29 + length + alignment)
			}
			complete := length &^ 63
			if got, want := utf16LengthFromUTF8BlocksNEON(input), utf16LengthFromUTF8Scalar(input[:complete]); got != want {
				t.Errorf("length %d alignment %d block length = %d, scalar = %d", length, alignment, got, want)
			}
		}
	}
}

func checkUTF8LengthNEON(t *testing.T, input []byte) {
	t.Helper()
	if got, want := latin1LengthFromUTF8NEON(input), latin1LengthFromUTF8Scalar(input); got != want {
		t.Errorf("latin1LengthFromUTF8NEON = %d, scalar = %d for %d bytes", got, want, len(input))
	}
	if got, want := utf16LengthFromUTF8NEON(input), utf16LengthFromUTF8Scalar(input); got != want {
		t.Errorf("utf16LengthFromUTF8NEON = %d, scalar = %d for %d bytes", got, want, len(input))
	}
	if got, want := utf32LengthFromUTF8NEON(input), utf32LengthFromUTF8Scalar(input); got != want {
		t.Errorf("utf32LengthFromUTF8NEON = %d, scalar = %d for %d bytes", got, want, len(input))
	}
}

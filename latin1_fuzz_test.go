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

// Scalar-differential fuzzing adapted from the Latin-1 conversion coverage in
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:fuzz/conversion.cpp,
// fuzz/roundtrip.cpp, and fuzz/safe_conversion.cpp. Bounds and canary checks
// are Go-specific.

func FuzzLatin1ScalarInvariants(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{},
		{0x00},
		{0x7f},
		{0x80},
		{0xff},
		{0x00, 0x7f, 0x80, 0xff},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		wantLength := 0
		for _, value := range input {
			wantLength++
			if value >= 0x80 {
				wantLength++
			}
		}
		if got := utf8LengthFromLatin1Scalar(input); got != wantLength {
			t.Fatalf("scalar UTF-8 length = %d, want %d", got, wantLength)
		}
		if got := UTF8LengthFromLatin1(input); got != wantLength {
			t.Fatalf("public UTF-8 length = %d, want %d", got, wantLength)
		}

		utf8 := guardedLatin1Destination[byte](wantLength, 0xa5)
		if got := convertLatin1ToUTF8Scalar(input, utf8.body); got != wantLength {
			t.Fatalf("scalar UTF-8 wrote %d, want %d", got, wantLength)
		}
		utf8.require(t)
		publicUTF8 := guardedLatin1Destination[byte](wantLength, 0xa5)
		if got := ConvertLatin1ToUTF8(input, publicUTF8.body); got != wantLength {
			t.Fatalf("public UTF-8 wrote %d, want %d", got, wantLength)
		}
		publicUTF8.require(t)
		if !slices.Equal(publicUTF8.body, utf8.body) {
			t.Fatalf("public UTF-8 = %x, scalar = %x", publicUTF8.body, utf8.body)
		}

		for capacity := 0; capacity <= wantLength; capacity++ {
			dst := guardedLatin1Destination[byte](capacity, 0xa5)
			written := ConvertLatin1ToUTF8Safe(input, dst.body)
			if written > capacity || !slices.Equal(dst.body[:written], utf8.body[:written]) {
				t.Fatalf("safe capacity %d wrote %d: %x, full %x", capacity, written, dst.body[:written], utf8.body)
			}
			if written < wantLength && written > 0 && inputPrefixUTF8EndsMidSequence(utf8.body[:written]) {
				t.Fatalf("safe capacity %d ended inside UTF-8 sequence", capacity)
			}
			for _, value := range dst.body[written:] {
				if value != 0xa5 {
					t.Fatal("safe conversion modified unwritten output")
				}
			}
			dst.require(t)
		}

		utf32 := guardedLatin1Destination[uint32](len(input), 0xa5a5a5a5)
		if got := ConvertLatin1ToUTF32(input, utf32.body); got != len(input) {
			t.Fatalf("UTF-32 wrote %d, want %d", got, len(input))
		}
		utf32.require(t)
		for index, value := range input {
			if utf32.body[index] != uint32(value) {
				t.Fatalf("UTF-32[%d] = %x, want %x", index, utf32.body[index], value)
			}
		}
	})
}

func inputPrefixUTF8EndsMidSequence(output []byte) bool {
	if len(output) == 0 {
		return false
	}
	return output[len(output)-1]&0xc0 == 0xc0
}

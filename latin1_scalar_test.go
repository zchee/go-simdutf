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
	"math/bits"
	"slices"
	"testing"
)

// Test vectors adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// tests/convert_latin1_to_utf8_tests.cpp, tests/convert_latin1_to_utf16le_tests.cpp,
// tests/convert_latin1_to_utf16be_tests.cpp, and tests/convert_latin1_to_utf32_tests.cpp.
// Canary and short-destination checks are Go-specific slice-contract coverage.

func TestLatin1ConversionsTable(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		utf8  []byte
		utf16 []uint16
	}{
		{"nil", nil, nil, nil},
		{"empty", []byte{}, []byte{}, []uint16{}},
		{"boundaries", []byte{0x00, 0x7f, 0x80, 0xff}, []byte{0x00, 0x7f, 0xc2, 0x80, 0xc3, 0xbf}, []uint16{0x0000, 0x007f, 0x0080, 0x00ff}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := UTF8LengthFromLatin1(test.input); got != len(test.utf8) {
				t.Fatalf("UTF8LengthFromLatin1() = %d, want %d", got, len(test.utf8))
			}
			for _, length := range []int{0, 1, 127, 128, len(test.input)} {
				if got := UTF16LengthFromLatin1(length); got != length {
					t.Fatalf("UTF16LengthFromLatin1(%d) = %d", length, got)
				}
				if got := UTF32LengthFromLatin1(length); got != length {
					t.Fatalf("UTF32LengthFromLatin1(%d) = %d", length, got)
				}
			}

			utf8 := guardedLatin1Destination[byte](len(test.utf8), 0xa5)
			if got := ConvertLatin1ToUTF8(test.input, utf8.body); got != len(test.utf8) {
				t.Fatalf("ConvertLatin1ToUTF8() = %d, want %d", got, len(test.utf8))
			}
			utf8.require(t)
			if !slices.Equal(utf8.body, test.utf8) {
				t.Fatalf("UTF-8 = %x, want %x", utf8.body, test.utf8)
			}

			utf32 := guardedLatin1Destination[uint32](len(test.input), 0xa5a5a5a5)
			if got := ConvertLatin1ToUTF32(test.input, utf32.body); got != len(test.input) {
				t.Fatalf("ConvertLatin1ToUTF32() = %d, want %d", got, len(test.input))
			}
			utf32.require(t)
			for index, value := range test.input {
				if utf32.body[index] != uint32(value) {
					t.Fatalf("UTF-32[%d] = %x, want %x", index, utf32.body[index], value)
				}
			}

			for _, little := range []bool{true, false} {
				dst := guardedLatin1Destination[uint16](len(test.input), 0xa55a)
				var got int
				if little {
					got = ConvertLatin1ToUTF16LE(test.input, dst.body)
				} else {
					got = ConvertLatin1ToUTF16BE(test.input, dst.body)
				}
				if got != len(test.input) {
					t.Fatalf("little=%t: wrote %d, want %d", little, got, len(test.input))
				}
				dst.require(t)
				want := rawLatin1UTF16(test.utf16, little)
				if !slices.Equal(dst.body, want) {
					t.Fatalf("little=%t: UTF-16 = %x, want %x", little, dst.body, want)
				}
			}
		})
	}
}

func TestLatin1NativeUTF16Equivalence(t *testing.T) {
	input := []byte{0x00, 0x7f, 0x80, 0xff}
	native := make([]uint16, len(input))
	if got := ConvertLatin1ToUTF16(input, native); got != len(input) {
		t.Fatalf("ConvertLatin1ToUTF16() = %d", got)
	}
	explicit := make([]uint16, len(input))
	if nativeLittleEndian() {
		ConvertLatin1ToUTF16LE(input, explicit)
	} else {
		ConvertLatin1ToUTF16BE(input, explicit)
	}
	if !slices.Equal(native, explicit) {
		t.Fatalf("native = %x, explicit = %x", native, explicit)
	}
}

func TestLatin1ConversionsShortDestinationPanicsBeforeWrite(t *testing.T) {
	input := []byte{0x7f, 0x80, 0xff}
	tests := []struct {
		name string
		run  func()
		dst  []byte
	}{
		{"utf8", func() { ConvertLatin1ToUTF8(input, shortLatin1Bytes) }, shortLatin1Bytes},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := slices.Clone(test.dst)
			requireLatin1Panic(t, test.run)
			if !slices.Equal(test.dst, before) {
				t.Fatal("short destination was modified")
			}
		})
	}
	for _, convert := range []func([]byte, []uint16) int{ConvertLatin1ToUTF16LE, ConvertLatin1ToUTF16BE, ConvertLatin1ToUTF16} {
		dst := []uint16{0xa55a, 0xa55a}
		before := slices.Clone(dst)
		requireLatin1Panic(t, func() { convert(input, dst) })
		if !slices.Equal(dst, before) {
			t.Fatal("short UTF-16 destination was modified")
		}
	}
	dst32 := []uint32{0xa5a5a5a5, 0xa5a5a5a5}
	before32 := slices.Clone(dst32)
	requireLatin1Panic(t, func() { ConvertLatin1ToUTF32(input, dst32) })
	if !slices.Equal(dst32, before32) {
		t.Fatal("short UTF-32 destination was modified")
	}
}

func TestConvertLatin1ToUTF8Safe(t *testing.T) {
	input := []byte{0x7f, 0x80, 0xff}
	for _, capacity := range []int{0, 1, 2, 3, 4, 5} {
		dst := guardedLatin1Destination[byte](capacity, 0xa5)
		written := ConvertLatin1ToUTF8Safe(input, dst.body)
		want := []byte{0x7f, 0xc2, 0x80, 0xc3, 0xbf}[:written]
		if !slices.Equal(dst.body[:written], want) {
			t.Fatalf("capacity %d: output = %x, want %x", capacity, dst.body[:written], want)
		}
		for _, value := range dst.body[written:] {
			if value != 0xa5 {
				t.Fatalf("capacity %d: unwritten suffix modified: %x", capacity, dst.body)
			}
		}
		dst.require(t)
	}
}

var shortLatin1Bytes = []byte{0xa5, 0xa5, 0xa5, 0xa5}

type latin1Destination[T comparable] struct {
	storage  []T
	body     []T
	sentinel T
}

func guardedLatin1Destination[T comparable](size int, sentinel T) latin1Destination[T] {
	storage := make([]T, size+2)
	storage[0], storage[len(storage)-1] = sentinel, sentinel
	for index := range storage[1 : len(storage)-1] {
		storage[index+1] = sentinel
	}
	return latin1Destination[T]{storage: storage, body: storage[1 : len(storage)-1], sentinel: sentinel}
}

func (destination latin1Destination[T]) require(t *testing.T) {
	t.Helper()
	if destination.storage[0] != destination.sentinel || destination.storage[len(destination.storage)-1] != destination.sentinel {
		t.Fatal("destination canary was modified")
	}
}

func rawLatin1UTF16(semantic []uint16, little bool) []uint16 {
	raw := slices.Clone(semantic)
	if little != nativeLittleEndian() {
		for index := range raw {
			raw[index] = bits.ReverseBytes16(raw[index])
		}
	}
	return raw
}

func requireLatin1Panic(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("operation did not panic")
		}
	}()
	operation()
}

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

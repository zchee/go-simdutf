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
	"testing"
)

func requireLatin1ArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdLatin1AgainstScalar(t *testing.T) {
	requireLatin1ArchsimdAVX2(t)

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "nil"},
		{name: "one-ascii", input: []byte{0x7f}},
		{name: "one-high", input: []byte{0x80}},
		{name: "mixed-short", input: []byte{0x00, 0x7f, 0x80, 0xff, 'A'}},
	}
	for _, length := range [...]int{7, 8, 15, 16, 31, 32, 33, 63, 64, 65, 127, 128, 129} {
		tests = append(tests, struct {
			name  string
			input []byte
		}{
			name:  fmt.Sprintf("mixed-length-%d", length),
			input: latin1ArchsimdInput(length),
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkLatin1Archsimd(t, test.input)
		})
	}
}

func FuzzLatin1ArchsimdAgainstScalar(f *testing.F) {
	for _, input := range [][]byte{
		nil,
		{0x00},
		{0x7f, 0x80, 0xff},
		latin1ArchsimdInput(31),
		latin1ArchsimdInput(32),
		latin1ArchsimdInput(33),
		latin1ArchsimdInput(129),
	} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		requireLatin1ArchsimdAVX2(t)
		checkLatin1Archsimd(t, input)
	})
}

func checkLatin1Archsimd(t *testing.T, input []byte) {
	t.Helper()

	want8 := make([]byte, utf8LengthFromLatin1Scalar(input))
	convertLatin1ToUTF8Scalar(input, want8)
	got8 := bytes.Repeat([]byte{0xa5}, len(want8)+16)
	if got := convertLatin1ToUTF8Archsimd(input, got8); got != len(want8) || !bytes.Equal(got8[:got], want8) || !allBytes(got8[got:], 0xa5) {
		t.Fatal("UTF-8 mismatch or canary overwrite")
	}
	if got, want := utf8LengthFromLatin1Archsimd(input), len(want8); got != want {
		t.Fatalf("UTF-8 length = %d, want %d", got, want)
	}

	checkLatin1ArchsimdUTF16(t, input, false)
	checkLatin1ArchsimdUTF16(t, input, true)

	want32 := make([]uint32, len(input))
	convertLatin1ToUTF32Scalar(input, want32)
	got32 := make([]uint32, len(input)+8)
	fillU32(got32[len(input):], 0xa5a5a5a5)
	if got := convertLatin1ToUTF32Archsimd(input, got32); got != len(input) || !equalU32(got32[:got], want32) || !allU32(got32[got:], 0xa5a5a5a5) {
		t.Fatal("UTF-32 mismatch or canary overwrite")
	}

	checkLatin1ArchsimdPreflight(t, input)
}

func checkLatin1ArchsimdUTF16(t *testing.T, input []byte, bigEndian bool) {
	t.Helper()

	want := make([]uint16, len(input))
	got := make([]uint16, len(input)+8)
	fillU16(got[len(input):], 0xa5a5)
	var converted int
	if bigEndian {
		convertLatin1ToUTF16BEScalar(input, want)
		converted = convertLatin1ToUTF16BEArchsimd(input, got)
	} else {
		convertLatin1ToUTF16LEScalar(input, want)
		converted = convertLatin1ToUTF16LEArchsimd(input, got)
	}
	if converted != len(input) || !equalU16(got[:converted], want) || !allU16(got[converted:], 0xa5a5) {
		t.Fatal("UTF-16 mismatch or canary overwrite")
	}
}

func checkLatin1ArchsimdPreflight(t *testing.T, input []byte) {
	t.Helper()
	if len(input) == 0 {
		return
	}

	required8 := utf8LengthFromLatin1Scalar(input)
	dst8 := bytes.Repeat([]byte{0xa5}, required8-1)
	requireLatin1ArchsimdPanic(t, func() { convertLatin1ToUTF8Archsimd(input, dst8) })
	if !allBytes(dst8, 0xa5) {
		t.Fatal("UTF-8 short destination was modified")
	}

	dst16 := make([]uint16, len(input)-1)
	fillU16(dst16, 0xa5a5)
	requireLatin1ArchsimdPanic(t, func() { convertLatin1ToUTF16LEArchsimd(input, dst16) })
	if !allU16(dst16, 0xa5a5) {
		t.Fatal("UTF-16LE short destination was modified")
	}
	requireLatin1ArchsimdPanic(t, func() { convertLatin1ToUTF16BEArchsimd(input, dst16) })
	if !allU16(dst16, 0xa5a5) {
		t.Fatal("UTF-16BE short destination was modified")
	}

	dst32 := make([]uint32, len(input)-1)
	fillU32(dst32, 0xa5a5a5a5)
	requireLatin1ArchsimdPanic(t, func() { convertLatin1ToUTF32Archsimd(input, dst32) })
	if !allU32(dst32, 0xa5a5a5a5) {
		t.Fatal("UTF-32 short destination was modified")
	}
}

func requireLatin1ArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

func latin1ArchsimdInput(length int) []byte {
	input := make([]byte, length)
	for i := range input {
		switch i % 5 {
		case 0:
			input[i] = 0x00
		case 1:
			input[i] = 0x7f
		case 2:
			input[i] = 0x80
		case 3:
			input[i] = 0xff
		default:
			input[i] = byte(i)
		}
	}
	return input
}

func allBytes(values []byte, want byte) bool {
	for _, value := range values {
		if value != want {
			return false
		}
	}
	return true
}

func fillU16(values []uint16, value uint16) {
	for i := range values {
		values[i] = value
	}
}

func allU16(values []uint16, want uint16) bool {
	for _, value := range values {
		if value != want {
			return false
		}
	}
	return true
}

func fillU32(values []uint32, value uint32) {
	for i := range values {
		values[i] = value
	}
}

func allU32(values []uint32, want uint32) bool {
	for _, value := range values {
		if value != want {
			return false
		}
	}
	return true
}

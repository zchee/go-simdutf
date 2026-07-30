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

// Direct differential coverage for the arm64 UTF-16→Latin-1 NEON translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf16_to_latin1.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// NEON providers are invoked directly (no detectARM64Features).

import (
	"bytes"
	"testing"
)

func TestUTF16Latin1NEONDirectAgainstScalar(t *testing.T) {
	natives := [][]uint16{
		nil,
		{},
		{0},
		{'a', 'b', 'c'},
		{0x00, 0x7f, 0xff},
		repeatUTF16('A', 7),
		repeatUTF16('A', 8),
		repeatUTF16('A', 9),
		repeatUTF16('A', 15),
		repeatUTF16('A', 16),
		repeatUTF16('A', 17),
		append(repeatUTF16('A', 7), 0xff),
		append(repeatUTF16(0xff, 8), 0x7f),
		append(repeatUTF16('A', 8), 0x100),
		append(repeatUTF16('A', 16), 0x100, 'b'),
		{'a', 0x100, 'b'},
		{0x100},
	}
	for _, native := range natives {
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			checkUTF16Latin1DirectNEON(t, input, little)
		}
	}
}

func TestUTF16Latin1NEONDirectPreflightPreservesCanaries(t *testing.T) {
	native := append(repeatUTF16('A', 15), 0xff)
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		dst := guardedLatin1Destination[byte](len(input)-1, 0xa5)
		if little {
			requireLatin1Panic(t, func() { convertUTF16LEToLatin1NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16LEToLatin1WithErrorsNEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertValidUTF16LEToLatin1NEON(input, dst.body) })
			dst.require(t)
		} else {
			requireLatin1Panic(t, func() { convertUTF16BEToLatin1NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16BEToLatin1WithErrorsNEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertValidUTF16BEToLatin1NEON(input, dst.body) })
			dst.require(t)
		}
	}
}

func FuzzUTF16Latin1NEONAgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(utf16NativeBytes(repeatUTF16('A', 8)))
	f.Add(utf16NativeBytes(repeatUTF16(0xff, 16)))
	f.Add(utf16NativeBytes([]uint16{'a', 0x100, 'b'}))
	f.Add(utf16NativeBytes(append(repeatUTF16('A', 16), 0x100)))
	f.Fuzz(func(t *testing.T, raw []byte) {
		native := utf16NativeFromBytes(raw)
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			checkUTF16Latin1DirectNEON(t, input, little)
		}
	})
}

func utf16NativeBytes(words []uint16) []byte {
	out := make([]byte, len(words)*2)
	for i, word := range words {
		out[2*i] = byte(word)
		out[2*i+1] = byte(word >> 8)
	}
	return out
}

func utf16NativeFromBytes(raw []byte) []uint16 {
	n := len(raw) / 2
	out := make([]uint16, n)
	for i := 0; i < n; i++ {
		out[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
	}
	return out
}

func checkUTF16Latin1DirectNEON(t *testing.T, input []uint16, little bool) {
	t.Helper()

	want := bytes.Repeat([]byte{0xa5}, len(input))
	got := guardedLatin1Destination[byte](len(input), 0xa5)
	var (
		wantN, gotN int
		wantE, gotE Result
		wantV, gotV int
	)
	if little {
		wantN = convertUTF16LEToLatin1Scalar(input, want)
		gotN = convertUTF16LEToLatin1NEON(input, got.body)
	} else {
		wantN = convertUTF16BEToLatin1Scalar(input, want)
		gotN = convertUTF16BEToLatin1NEON(input, got.body)
	}
	if gotN != wantN || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t convert = %d/%x, want %d/%x", little, gotN, got.body, wantN, want)
	}
	got.require(t)

	want = bytes.Repeat([]byte{0xa5}, len(input))
	got = guardedLatin1Destination[byte](len(input), 0xa5)
	if little {
		wantE = convertUTF16LEToLatin1WithErrorsScalar(input, want)
		gotE = convertUTF16LEToLatin1WithErrorsNEON(input, got.body)
	} else {
		wantE = convertUTF16BEToLatin1WithErrorsScalar(input, want)
		gotE = convertUTF16BEToLatin1WithErrorsNEON(input, got.body)
	}
	if gotE != wantE || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t with_errors = %#v/%x, want %#v/%x", little, gotE, got.body, wantE, want)
	}
	got.require(t)

	if wantE.Error != Success {
		return
	}
	want = bytes.Repeat([]byte{0xa5}, len(input))
	got = guardedLatin1Destination[byte](len(input), 0xa5)
	if little {
		wantV = convertValidUTF16LEToLatin1Scalar(input, want)
		gotV = convertValidUTF16LEToLatin1NEON(input, got.body)
	} else {
		wantV = convertValidUTF16BEToLatin1Scalar(input, want)
		gotV = convertValidUTF16BEToLatin1NEON(input, got.body)
	}
	if gotV != wantV || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t valid = %d/%x, want %d/%x", little, gotV, got.body, wantV, want)
	}
	got.require(t)
}

func repeatUTF16(value uint16, n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		out[i] = value
	}
	return out
}

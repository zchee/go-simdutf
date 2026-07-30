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
	"testing"
	"unicode/utf8"
)

func TestUTF8ConvertNEONDirectAgainstScalar(t *testing.T) {
	inputs := []struct {
		name  string
		input []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"short-ascii", bytes.Repeat([]byte{'A'}, 15)},
		{"ascii-16", bytes.Repeat([]byte{'A'}, 16)},
		{"ascii-32", bytes.Repeat([]byte{'A'}, 32)},
		{"long-ascii-64", bytes.Repeat([]byte{'A'}, 64)},
		{"long-ascii-65", bytes.Repeat([]byte{'A'}, 65)},
		{"ascii-then-latin1", append(bytes.Repeat([]byte{'A'}, 64), []byte("caf\u00e9")...)},
		{"mixed-emoji", append(bytes.Repeat([]byte{'A'}, 64), []byte("A\U0001F600B")...)},
		{"mixed-arabic", append(bytes.Repeat([]byte{'A'}, 64), []byte("\u0645\u0631\u062d\u0628\u0627")...)},
		{"emoji", []byte("A\U0001F600B")},
		{"arabic", []byte("\u0645\u0631\u062d\u0628\u0627")},
		{"latin1", []byte("caf\u00e9")},
	}
	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			checkUTF8ConvertDirectNEON(t, tc.input)
		})
	}
}

func TestUTF8ConvertNEONDirectWithErrorsAgainstScalar(t *testing.T) {
	invalids := []struct {
		name  string
		input []byte
	}{
		{"too_short", []byte{0xc2}},
		{"overlong", []byte{0xc0, 0xaf}},
		{"surrogate", []byte{0xed, 0xa0, 0x80}},
		{"header", []byte{0xff}},
		{"too_long", []byte{0x80}},
		{"ascii-then-too_short", append(bytes.Repeat([]byte{'A'}, 64), 0xc2)},
		{"ascii-then-header", append(bytes.Repeat([]byte{'A'}, 64), 0xff)},
		{"ascii-then-surrogate", append(bytes.Repeat([]byte{'A'}, 64), 0xed, 0xa0, 0x80)},
	}
	for _, tc := range invalids {
		t.Run(tc.name, func(t *testing.T) {
			checkUTF8ConvertErrorsNEON(t, tc.input)
		})
	}
}

func TestUTF8ConvertNEONDirectPreflightPreservesCanaries(t *testing.T) {
	input := bytes.Repeat([]byte{'A'}, 65)

	dst8 := guardedLatin1Destination[byte](latin1LengthFromUTF8Scalar(input)-1, 0xa5)
	requireLatin1Panic(t, func() { convertUTF8ToLatin1NEON(input, dst8.body) })
	dst8.require(t)
	requireLatin1Panic(t, func() { convertUTF8ToLatin1WithErrorsNEON(input, dst8.body) })
	dst8.require(t)
	requireLatin1Panic(t, func() { convertValidUTF8ToLatin1NEON(input, dst8.body) })
	dst8.require(t)

	dst16 := guardedLatin1Destination[uint16](utf16LengthFromUTF8Scalar(input)-1, 0xa5a5)
	requireLatin1Panic(t, func() { convertUTF8ToUTF16LENEON(input, dst16.body) })
	dst16.require(t)
	requireLatin1Panic(t, func() { convertUTF8ToUTF16BENEON(input, dst16.body) })
	dst16.require(t)
	requireLatin1Panic(t, func() { convertUTF8ToUTF16LEWithErrorsNEON(input, dst16.body) })
	dst16.require(t)
	requireLatin1Panic(t, func() { convertUTF8ToUTF16BEWithErrorsNEON(input, dst16.body) })
	dst16.require(t)
	requireLatin1Panic(t, func() { convertValidUTF8ToUTF16LENEON(input, dst16.body) })
	dst16.require(t)
	requireLatin1Panic(t, func() { convertValidUTF8ToUTF16BENEON(input, dst16.body) })
	dst16.require(t)

	dst32 := guardedLatin1Destination[uint32](utf32LengthFromUTF8Scalar(input)-1, 0xa5a5a5a5)
	requireLatin1Panic(t, func() { convertUTF8ToUTF32NEON(input, dst32.body) })
	dst32.require(t)
	requireLatin1Panic(t, func() { convertUTF8ToUTF32WithErrorsNEON(input, dst32.body) })
	dst32.require(t)
	requireLatin1Panic(t, func() { convertValidUTF8ToUTF32NEON(input, dst32.body) })
	dst32.require(t)
}

func FuzzUTF8ConvertNEONAgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte{'A'}, 64))
	f.Add([]byte("caf\u00e9"))
	f.Add([]byte("A\U0001F600B"))
	f.Add([]byte("\u0645\u0631\u062d\u0628\u0627"))
	f.Add([]byte{0xc2})
	f.Add([]byte{0xff})
	f.Add(append(bytes.Repeat([]byte{'A'}, 64), 0xc2))
	f.Fuzz(func(t *testing.T, input []byte) {
		checkUTF8ConvertDirectNEON(t, input)
	})
}

func checkUTF8ConvertDirectNEON(t *testing.T, input []byte) {
	t.Helper()

	wantLatin1Len := latin1LengthFromUTF8Scalar(input)
	want8 := bytes.Repeat([]byte{0xa5}, wantLatin1Len)
	got8 := guardedLatin1Destination[byte](wantLatin1Len, 0xa5)
	wantN := convertUTF8ToLatin1Scalar(input, want8)
	if got := convertUTF8ToLatin1NEON(input, got8.body); got != wantN || !bytes.Equal(got8.body, want8) {
		t.Fatalf("Latin1 = %d/%x, want %d/%x", got, got8.body, wantN, want8)
	}
	got8.require(t)

	want8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len)
	got8 = guardedLatin1Destination[byte](wantLatin1Len, 0xa5)
	wantE := convertUTF8ToLatin1WithErrorsScalar(input, want8)
	if got := convertUTF8ToLatin1WithErrorsNEON(input, got8.body); got != wantE || !bytes.Equal(got8.body, want8) {
		t.Fatalf("Latin1 WithErrors = %#v, want %#v", got, wantE)
	}
	got8.require(t)

	if utf8.Valid(input) {
		want8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len)
		got8 = guardedLatin1Destination[byte](wantLatin1Len, 0xa5)
		wantV := convertValidUTF8ToLatin1Scalar(input, want8)
		gotV := convertValidUTF8ToLatin1NEON(input, got8.body)
		if gotV != wantV || !bytes.Equal(got8.body, want8) {
			t.Fatalf("Valid Latin1 = %d/%x, want %d/%x", gotV, got8.body, wantV, want8)
		}
		got8.require(t)
	}

	want16Len := utf16LengthFromUTF8Scalar(input)
	want16 := make([]uint16, want16Len)
	for i := range want16 {
		want16[i] = 0xa5a5
	}
	got16 := guardedLatin1Destination[uint16](want16Len, 0xa5a5)
	wantN16 := convertUTF8ToUTF16LEScalar(input, want16)
	if got := convertUTF8ToUTF16LENEON(input, got16.body); got != wantN16 || !equalU16(got16.body, want16) {
		t.Fatalf("UTF-16LE = %d/%x, want %d/%x", got, got16.body, wantN16, want16)
	}
	got16.require(t)

	for i := range want16 {
		want16[i] = 0xa5a5
	}
	got16 = guardedLatin1Destination[uint16](want16Len, 0xa5a5)
	wantE16 := convertUTF8ToUTF16LEWithErrorsScalar(input, want16)
	if got := convertUTF8ToUTF16LEWithErrorsNEON(input, got16.body); got != wantE16 || !equalU16(got16.body, want16) {
		t.Fatalf("UTF-16LE WithErrors = %#v, want %#v", got, wantE16)
	}
	got16.require(t)

	if utf8.Valid(input) {
		for i := range want16 {
			want16[i] = 0xa5a5
		}
		got16 = guardedLatin1Destination[uint16](want16Len, 0xa5a5)
		wantV16 := convertValidUTF8ToUTF16LEScalar(input, want16)
		if got := convertValidUTF8ToUTF16LENEON(input, got16.body); got != wantV16 || !equalU16(got16.body, want16) {
			t.Fatalf("Valid UTF-16LE = %d, want %d", got, wantV16)
		}
		got16.require(t)
	}

	for i := range want16 {
		want16[i] = 0xa5a5
	}
	got16 = guardedLatin1Destination[uint16](want16Len, 0xa5a5)
	wantN16 = convertUTF8ToUTF16BEScalar(input, want16)
	if got := convertUTF8ToUTF16BENEON(input, got16.body); got != wantN16 || !equalU16(got16.body, want16) {
		t.Fatalf("UTF-16BE = %d/%x, want %d/%x", got, got16.body, wantN16, want16)
	}
	got16.require(t)

	for i := range want16 {
		want16[i] = 0xa5a5
	}
	got16 = guardedLatin1Destination[uint16](want16Len, 0xa5a5)
	wantE16 = convertUTF8ToUTF16BEWithErrorsScalar(input, want16)
	if got := convertUTF8ToUTF16BEWithErrorsNEON(input, got16.body); got != wantE16 || !equalU16(got16.body, want16) {
		t.Fatalf("UTF-16BE WithErrors = %#v, want %#v", got, wantE16)
	}
	got16.require(t)

	if utf8.Valid(input) {
		for i := range want16 {
			want16[i] = 0xa5a5
		}
		got16 = guardedLatin1Destination[uint16](want16Len, 0xa5a5)
		wantV16 := convertValidUTF8ToUTF16BEScalar(input, want16)
		if got := convertValidUTF8ToUTF16BENEON(input, got16.body); got != wantV16 || !equalU16(got16.body, want16) {
			t.Fatalf("Valid UTF-16BE = %d, want %d", got, wantV16)
		}
		got16.require(t)
	}

	want32Len := utf32LengthFromUTF8Scalar(input)
	want32 := make([]uint32, want32Len)
	for i := range want32 {
		want32[i] = 0xa5a5a5a5
	}
	got32 := guardedLatin1Destination[uint32](want32Len, 0xa5a5a5a5)
	wantN32 := convertUTF8ToUTF32Scalar(input, want32)
	if got := convertUTF8ToUTF32NEON(input, got32.body); got != wantN32 || !equalU32(got32.body, want32) {
		t.Fatalf("UTF-32 = %d/%x, want %d/%x", got, got32.body, wantN32, want32)
	}
	got32.require(t)

	for i := range want32 {
		want32[i] = 0xa5a5a5a5
	}
	got32 = guardedLatin1Destination[uint32](want32Len, 0xa5a5a5a5)
	wantE32 := convertUTF8ToUTF32WithErrorsScalar(input, want32)
	if got := convertUTF8ToUTF32WithErrorsNEON(input, got32.body); got != wantE32 || !equalU32(got32.body, want32) {
		t.Fatalf("UTF-32 WithErrors = %#v, want %#v", got, wantE32)
	}
	got32.require(t)

	if utf8.Valid(input) {
		for i := range want32 {
			want32[i] = 0xa5a5a5a5
		}
		got32 = guardedLatin1Destination[uint32](want32Len, 0xa5a5a5a5)
		wantV32 := convertValidUTF8ToUTF32Scalar(input, want32)
		if got := convertValidUTF8ToUTF32NEON(input, got32.body); got != wantV32 || !equalU32(got32.body, want32) {
			t.Fatalf("Valid UTF-32 = %d, want %d", got, wantV32)
		}
		got32.require(t)
	}
}

func checkUTF8ConvertErrorsNEON(t *testing.T, input []byte) {
	t.Helper()

	dst8 := make([]byte, latin1LengthFromUTF8Scalar(input)+8)
	want := convertUTF8ToLatin1WithErrorsScalar(input, dst8)
	if got := convertUTF8ToLatin1WithErrorsNEON(input, dst8); got != want {
		t.Fatalf("Latin1 WithErrors = %#v, want %#v", got, want)
	}

	dst16 := make([]uint16, utf16LengthFromUTF8Scalar(input)+8)
	want = convertUTF8ToUTF16LEWithErrorsScalar(input, dst16)
	if got := convertUTF8ToUTF16LEWithErrorsNEON(input, dst16); got != want {
		t.Fatalf("UTF-16LE WithErrors = %#v, want %#v", got, want)
	}
	want = convertUTF8ToUTF16BEWithErrorsScalar(input, dst16)
	if got := convertUTF8ToUTF16BEWithErrorsNEON(input, dst16); got != want {
		t.Fatalf("UTF-16BE WithErrors = %#v, want %#v", got, want)
	}

	dst32 := make([]uint32, utf32LengthFromUTF8Scalar(input)+8)
	want = convertUTF8ToUTF32WithErrorsScalar(input, dst32)
	if got := convertUTF8ToUTF32WithErrorsNEON(input, dst32); got != want {
		t.Fatalf("UTF-32 WithErrors = %#v, want %#v", got, want)
	}
}

// equalU16 reports whether a and b contain the same UTF-16 code units.
// Duplicated because the amd64 latin1 helpers are build-tagged out on arm64.
func equalU16(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// equalU32 reports whether a and b contain the same UTF-32 code units.
// Duplicated because the amd64 latin1 helpers are build-tagged out on arm64.
func equalU32(a, b []uint32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

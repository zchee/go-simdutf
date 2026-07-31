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

// Direct differential coverage for the arm64 UTF-32 convert/length NEON
// translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf32_to_*.cpp
// and utf8/utf16_length_from_utf32 in src/arm64/implementation.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// NEON providers are invoked directly (no detectARM64Features).

import (
	"bytes"
	"slices"
	"testing"
)

func TestUTF32ConvertNEONDirectAgainstScalar(t *testing.T) {
	inputs := [][]uint32{
		nil,
		{},
		{0},
		{'a', 'b', 'c'},
		{0x00, 0x7f, 0xff},
		{0x00e9, 0x20ac},
		{0x1f600},
		{'A', 0x20ac, 0x1f600},
		repeatUTF32('A', 3),
		repeatUTF32('A', 4),
		repeatUTF32('A', 7),
		repeatUTF32('A', 8),
		repeatUTF32('A', 9),
		repeatUTF32('A', 15),
		repeatUTF32('A', 16),
		repeatUTF32('A', 17),
		append(repeatUTF32('A', 7), 0xff),
		append(repeatUTF32('A', 8), 0x100),
		append(repeatUTF32('A', 8), 0x20ac),
		append(repeatUTF32('A', 8), 0x1f600),
		append(repeatUTF32('A', 15), 0xd800),
		append(repeatUTF32('A', 16), 0xd800),
		append(repeatUTF32('A', 16), 0x110000),
		{'a', 0xd800, 'b'},
		{'a', 0x110000, 'b'},
		{0xd800},
		{0x110000},
		{0xffff, 0x10000, 0x10ffff},
	}
	for _, input := range inputs {
		checkUTF32ConvertDirectNEON(t, input)
	}
}

func TestUTF32ConvertNEONDirectPreflightPreservesCanaries(t *testing.T) {
	input := append(repeatUTF32('A', 15), 0xff)
	dstL1 := guardedLatin1Destination[byte](len(input)-1, 0xa5)
	requireUTF32ShortDstPanic(t, func() { convertUTF32ToLatin1NEON(input, dstL1.body) })
	dstL1.require(t)
	requireUTF32ShortDstPanic(t, func() { convertUTF32ToLatin1WithErrorsNEON(input, dstL1.body) })
	dstL1.require(t)
	requireUTF32ShortDstPanic(t, func() { convertValidUTF32ToLatin1NEON(input, dstL1.body) })
	dstL1.require(t)

	need8 := utf8LengthFromUTF32Scalar(input)
	dst8 := guardedLatin1Destination[byte](need8-1, 0xa5)
	requireUTF32ShortDstPanic(t, func() { convertUTF32ToUTF8NEON(input, dst8.body) })
	dst8.require(t)
	requireUTF32ShortDstPanic(t, func() { convertUTF32ToUTF8WithErrorsNEON(input, dst8.body) })
	dst8.require(t)
	requireUTF32ShortDstPanic(t, func() { convertValidUTF32ToUTF8NEON(input, dst8.body) })
	dst8.require(t)

	need16 := utf16LengthFromUTF32Scalar(input)
	dst16 := guardedLatin1Destination[uint16](need16-1, 0xa5a5)
	requireUTF32ShortDstPanic(t, func() { convertUTF32ToUTF16LENEON(input, dst16.body) })
	dst16.require(t)
	requireUTF32ShortDstPanic(t, func() { convertUTF32ToUTF16BENEON(input, dst16.body) })
	dst16.require(t)
	requireUTF32ShortDstPanic(t, func() { convertUTF32ToUTF16LEWithErrorsNEON(input, dst16.body) })
	dst16.require(t)
	requireUTF32ShortDstPanic(t, func() { convertUTF32ToUTF16BEWithErrorsNEON(input, dst16.body) })
	dst16.require(t)
	requireUTF32ShortDstPanic(t, func() { convertValidUTF32ToUTF16LENEON(input, dst16.body) })
	dst16.require(t)
	requireUTF32ShortDstPanic(t, func() { convertValidUTF32ToUTF16BENEON(input, dst16.body) })
	dst16.require(t)
}

func FuzzUTF32ConvertNEONAgainstScalar(f *testing.F) {
	f.Add(utf32NativeBytes(nil))
	f.Add(utf32NativeBytes(repeatUTF32('A', 8)))
	f.Add(utf32NativeBytes(repeatUTF32(0xff, 16)))
	f.Add(utf32NativeBytes([]uint32{'A', 0x20ac, 0x1f600}))
	f.Add(utf32NativeBytes(append(repeatUTF32('A', 16), 0xd800)))
	f.Add(utf32NativeBytes(append(repeatUTF32('A', 16), 0x110000)))
	f.Fuzz(func(t *testing.T, raw []byte) {
		checkUTF32ConvertDirectNEON(t, utf32NativeFromBytes(raw))
	})
}

func checkUTF32ConvertDirectNEON(t *testing.T, input []uint32) {
	t.Helper()

	if got, want := utf8LengthFromUTF32NEON(input), utf8LengthFromUTF32Scalar(input); got != want {
		t.Fatalf("utf8_length = %d, want %d for %x", got, want, input)
	}
	if got, want := utf16LengthFromUTF32NEON(input), utf16LengthFromUTF32Scalar(input); got != want {
		t.Fatalf("utf16_length = %d, want %d for %x", got, want, input)
	}

	wantL1 := bytes.Repeat([]byte{0xa5}, len(input))
	gotL1 := guardedLatin1Destination[byte](len(input), 0xa5)
	wantN := convertUTF32ToLatin1Scalar(input, wantL1)
	gotN := convertUTF32ToLatin1NEON(input, gotL1.body)
	if gotN != wantN || !bytes.Equal(gotL1.body, wantL1) {
		t.Fatalf("latin1 convert = %d/%x, want %d/%x", gotN, gotL1.body, wantN, wantL1)
	}
	gotL1.require(t)

	wantL1 = bytes.Repeat([]byte{0xa5}, len(input))
	gotL1 = guardedLatin1Destination[byte](len(input), 0xa5)
	wantE := convertUTF32ToLatin1WithErrorsScalar(input, wantL1)
	gotE := convertUTF32ToLatin1WithErrorsNEON(input, gotL1.body)
	if gotE != wantE || !bytes.Equal(gotL1.body, wantL1) {
		t.Fatalf("latin1 with_errors = %#v/%x, want %#v/%x", gotE, gotL1.body, wantE, wantL1)
	}
	gotL1.require(t)
	if wantE.Error == Success {
		wantL1 = bytes.Repeat([]byte{0xa5}, len(input))
		gotL1 = guardedLatin1Destination[byte](len(input), 0xa5)
		wantV := convertValidUTF32ToLatin1Scalar(input, wantL1)
		gotV := convertValidUTF32ToLatin1NEON(input, gotL1.body)
		if gotV != wantV || !bytes.Equal(gotL1.body, wantL1) {
			t.Fatalf("latin1 valid = %d/%x, want %d/%x", gotV, gotL1.body, wantV, wantL1)
		}
		gotL1.require(t)
	}

	need8 := utf8LengthFromUTF32Scalar(input)
	want8 := bytes.Repeat([]byte{0xa5}, need8)
	got8 := guardedLatin1Destination[byte](need8, 0xa5)
	wantN = convertUTF32ToUTF8Scalar(input, want8)
	gotN = convertUTF32ToUTF8NEON(input, got8.body)
	if gotN != wantN || !bytes.Equal(got8.body, want8) {
		t.Fatalf("utf8 convert = %d/%x, want %d/%x", gotN, got8.body, wantN, want8)
	}
	got8.require(t)

	want8 = bytes.Repeat([]byte{0xa5}, need8)
	got8 = guardedLatin1Destination[byte](need8, 0xa5)
	wantE = convertUTF32ToUTF8WithErrorsScalar(input, want8)
	gotE = convertUTF32ToUTF8WithErrorsNEON(input, got8.body)
	if gotE != wantE || !bytes.Equal(got8.body, want8) {
		t.Fatalf("utf8 with_errors = %#v/%x, want %#v/%x", gotE, got8.body, wantE, want8)
	}
	got8.require(t)
	if wantE.Error == Success {
		want8 = bytes.Repeat([]byte{0xa5}, need8)
		got8 = guardedLatin1Destination[byte](need8, 0xa5)
		wantV := convertValidUTF32ToUTF8Scalar(input, want8)
		gotV := convertValidUTF32ToUTF8NEON(input, got8.body)
		if gotV != wantV || !bytes.Equal(got8.body, want8) {
			t.Fatalf("utf8 valid = %d/%x, want %d/%x", gotV, got8.body, wantV, want8)
		}
		got8.require(t)
	}

	need16 := utf16LengthFromUTF32Scalar(input)
	for _, little := range []bool{true, false} {
		want16 := make([]uint16, need16)
		for i := range want16 {
			want16[i] = 0xa5a5
		}
		got16 := guardedLatin1Destination[uint16](need16, 0xa5a5)
		if little {
			wantN = convertUTF32ToUTF16LEScalar(input, want16)
			gotN = convertUTF32ToUTF16LENEON(input, got16.body)
		} else {
			wantN = convertUTF32ToUTF16BEScalar(input, want16)
			gotN = convertUTF32ToUTF16BENEON(input, got16.body)
		}
		if gotN != wantN || !slices.Equal(got16.body, want16) {
			t.Fatalf("utf16 little=%t convert = %d/%x, want %d/%x", little, gotN, got16.body, wantN, want16)
		}
		got16.require(t)

		want16 = make([]uint16, need16)
		for i := range want16 {
			want16[i] = 0xa5a5
		}
		got16 = guardedLatin1Destination[uint16](need16, 0xa5a5)
		if little {
			wantE = convertUTF32ToUTF16LEWithErrorsScalar(input, want16)
			gotE = convertUTF32ToUTF16LEWithErrorsNEON(input, got16.body)
		} else {
			wantE = convertUTF32ToUTF16BEWithErrorsScalar(input, want16)
			gotE = convertUTF32ToUTF16BEWithErrorsNEON(input, got16.body)
		}
		if gotE != wantE || !slices.Equal(got16.body, want16) {
			t.Fatalf("utf16 little=%t with_errors = %#v/%x, want %#v/%x", little, gotE, got16.body, wantE, want16)
		}
		got16.require(t)
		if wantE.Error != Success {
			continue
		}
		want16 = make([]uint16, need16)
		for i := range want16 {
			want16[i] = 0xa5a5
		}
		got16 = guardedLatin1Destination[uint16](need16, 0xa5a5)
		var wantV, gotV int
		if little {
			wantV = convertValidUTF32ToUTF16LEScalar(input, want16)
			gotV = convertValidUTF32ToUTF16LENEON(input, got16.body)
		} else {
			wantV = convertValidUTF32ToUTF16BEScalar(input, want16)
			gotV = convertValidUTF32ToUTF16BENEON(input, got16.body)
		}
		if gotV != wantV || !slices.Equal(got16.body, want16) {
			t.Fatalf("utf16 little=%t valid = %d/%x, want %d/%x", little, gotV, got16.body, wantV, want16)
		}
		got16.require(t)
	}
}

func requireUTF32ShortDstPanic(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		v := recover()
		if v == nil {
			t.Fatal("operation did not panic")
		}
		msg, ok := v.(string)
		if !ok || msg != "simdutf: destination is too short" {
			t.Fatalf("panic = %#v, want %q", v, "simdutf: destination is too short")
		}
	}()
	operation()
}

func repeatUTF32(value uint32, n int) []uint32 {
	out := make([]uint32, n)
	for i := range out {
		out[i] = value
	}
	return out
}

func utf32NativeBytes(words []uint32) []byte {
	out := make([]byte, len(words)*4)
	for i, word := range words {
		out[4*i] = byte(word)
		out[4*i+1] = byte(word >> 8)
		out[4*i+2] = byte(word >> 16)
		out[4*i+3] = byte(word >> 24)
	}
	return out
}

func utf32NativeFromBytes(raw []byte) []uint32 {
	n := len(raw) / 4
	out := make([]uint32, n)
	for i := range n {
		out[i] = uint32(raw[4*i]) | uint32(raw[4*i+1])<<8 | uint32(raw[4*i+2])<<16 | uint32(raw[4*i+3])<<24
	}
	return out
}

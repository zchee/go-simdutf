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

//go:build amd64

package simdutf

// Direct differential coverage for the amd64 UTF-32→Latin-1/UTF-8/UTF-16
// Westmere/Haswell translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf32_to_*.cpp
// and src/haswell/avx2_convert_utf32_to_*.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// Providers are invoked directly; feature skips only gate unavailable ISA.

import (
	"bytes"
	"testing"
)

func requireUTF32ConvertAMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skipf("missing required CPU features %#x", feature)
	}
}

func TestDirectAMD64UTF32ConvertAgainstScalar(t *testing.T) {
	inputs := [][]uint32{
		nil,
		{},
		{0},
		{'a', 'b', 'c'},
		{0x00, 0x7f, 0x80, 0xff},
		{0x100},
		{0x7ff, 0x800, 0xffff},
		{'a', 0x100, 'b'},
		{0x20ac},
		{0xd800},
		{0xdc00},
		{0xdfff},
		{0x10000},
		{0x1f600},
		{0x10ffff},
		{0x110000},
		{'A', 0x20ac, 0x1f600},
		{'A', 0xd800},
		{'A', 0x110000},
		repeatU32Convert('A', 3),
		repeatU32Convert('A', 4),
		repeatU32Convert('A', 5),
		repeatU32Convert('A', 7),
		repeatU32Convert('A', 8),
		repeatU32Convert('A', 9),
		repeatU32Convert('A', 15),
		repeatU32Convert('A', 16),
		repeatU32Convert('A', 17),
		append(repeatU32Convert('A', 3), 0xff),
		append(repeatU32Convert(0xff, 4), 0x7f),
		append(repeatU32Convert('A', 4), 0x100),
		append(repeatU32Convert('A', 4), 0x20ac),
		append(repeatU32Convert('A', 4), 0x1f600),
		append(repeatU32Convert('A', 4), 0xd800),
		append(repeatU32Convert('A', 8), 0x100, 'b'),
		append(repeatU32Convert('A', 8), 0x20ac),
		append(repeatU32Convert('A', 8), 0x1f600, 'z'),
		append(repeatU32Convert('A', 8), 0xd800),
		append(repeatU32Convert('A', 8), 0x110000),
		repeatU32Convert(0x00ff, 33),
		repeatU32Convert(0x0041, 65),
		repeatU32Convert(0x20ac, 17),
		repeatU32Convert(0x1f600, 9),
	}

	variants := []struct {
		name     string
		feature  cpuFeatures
		latin1   func([]uint32, []byte) int
		latin1E  func([]uint32, []byte) Result
		latin1V  func([]uint32, []byte) int
		utf8     func([]uint32, []byte) int
		utf8E    func([]uint32, []byte) Result
		utf8V    func([]uint32, []byte) int
		utf16LE  func([]uint32, []uint16) int
		utf16BE  func([]uint32, []uint16) int
		utf16LEE func([]uint32, []uint16) Result
		utf16BEE func([]uint32, []uint16) Result
		utf16LEV func([]uint32, []uint16) int
		utf16BEV func([]uint32, []uint16) int
		utf8Len  func([]uint32) int
		utf16Len func([]uint32) int
	}{
		{
			name:     "westmere",
			feature:  cpuSSSE3,
			latin1:   convertUTF32ToLatin1Westmere,
			latin1E:  convertUTF32ToLatin1WithErrorsWestmere,
			latin1V:  convertValidUTF32ToLatin1Westmere,
			utf8:     convertUTF32ToUTF8Westmere,
			utf8E:    convertUTF32ToUTF8WithErrorsWestmere,
			utf8V:    convertValidUTF32ToUTF8Westmere,
			utf16LE:  convertUTF32ToUTF16LEWestmere,
			utf16BE:  convertUTF32ToUTF16BEWestmere,
			utf16LEE: convertUTF32ToUTF16LEWithErrorsWestmere,
			utf16BEE: convertUTF32ToUTF16BEWithErrorsWestmere,
			utf16LEV: convertValidUTF32ToUTF16LEWestmere,
			utf16BEV: convertValidUTF32ToUTF16BEWestmere,
			utf8Len:  utf8LengthFromUTF32Westmere,
			utf16Len: utf16LengthFromUTF32Westmere,
		},
		{
			name:     "haswell",
			feature:  cpuAVX2,
			latin1:   convertUTF32ToLatin1Haswell,
			latin1E:  convertUTF32ToLatin1WithErrorsHaswell,
			latin1V:  convertValidUTF32ToLatin1Haswell,
			utf8:     convertUTF32ToUTF8Haswell,
			utf8E:    convertUTF32ToUTF8WithErrorsHaswell,
			utf8V:    convertValidUTF32ToUTF8Haswell,
			utf16LE:  convertUTF32ToUTF16LEHaswell,
			utf16BE:  convertUTF32ToUTF16BEHaswell,
			utf16LEE: convertUTF32ToUTF16LEWithErrorsHaswell,
			utf16BEE: convertUTF32ToUTF16BEWithErrorsHaswell,
			utf16LEV: convertValidUTF32ToUTF16LEHaswell,
			utf16BEV: convertValidUTF32ToUTF16BEHaswell,
			utf8Len:  utf8LengthFromUTF32Haswell,
			utf16Len: utf16LengthFromUTF32Haswell,
		},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF32ConvertAMD64Variant(t, v.feature)
			for _, input := range inputs {
				checkUTF32ConvertDirect(t, input, v.latin1, v.latin1E, v.latin1V, v.utf8, v.utf8E, v.utf8V, v.utf16LE, v.utf16BE, v.utf16LEE, v.utf16BEE, v.utf16LEV, v.utf16BEV, v.utf8Len, v.utf16Len)
			}
		})
	}
}

func TestDirectAMD64UTF32ConvertPreflightPreservesDestination(t *testing.T) {
	input := []uint32{'A', 0x20ac, 0x1f600, 'z', 'A', 'B', 'C', 'D'}
	variants := []struct {
		name     string
		feature  cpuFeatures
		latin1   func([]uint32, []byte) int
		latin1E  func([]uint32, []byte) Result
		latin1V  func([]uint32, []byte) int
		utf8     func([]uint32, []byte) int
		utf8E    func([]uint32, []byte) Result
		utf8V    func([]uint32, []byte) int
		utf16LE  func([]uint32, []uint16) int
		utf16BE  func([]uint32, []uint16) int
		utf16LEE func([]uint32, []uint16) Result
		utf16BEE func([]uint32, []uint16) Result
		utf16LEV func([]uint32, []uint16) int
		utf16BEV func([]uint32, []uint16) int
	}{
		{
			name: "westmere", feature: cpuSSSE3,
			latin1: convertUTF32ToLatin1Westmere, latin1E: convertUTF32ToLatin1WithErrorsWestmere, latin1V: convertValidUTF32ToLatin1Westmere,
			utf8: convertUTF32ToUTF8Westmere, utf8E: convertUTF32ToUTF8WithErrorsWestmere, utf8V: convertValidUTF32ToUTF8Westmere,
			utf16LE: convertUTF32ToUTF16LEWestmere, utf16BE: convertUTF32ToUTF16BEWestmere,
			utf16LEE: convertUTF32ToUTF16LEWithErrorsWestmere, utf16BEE: convertUTF32ToUTF16BEWithErrorsWestmere,
			utf16LEV: convertValidUTF32ToUTF16LEWestmere, utf16BEV: convertValidUTF32ToUTF16BEWestmere,
		},
		{
			name: "haswell", feature: cpuAVX2,
			latin1: convertUTF32ToLatin1Haswell, latin1E: convertUTF32ToLatin1WithErrorsHaswell, latin1V: convertValidUTF32ToLatin1Haswell,
			utf8: convertUTF32ToUTF8Haswell, utf8E: convertUTF32ToUTF8WithErrorsHaswell, utf8V: convertValidUTF32ToUTF8Haswell,
			utf16LE: convertUTF32ToUTF16LEHaswell, utf16BE: convertUTF32ToUTF16BEHaswell,
			utf16LEE: convertUTF32ToUTF16LEWithErrorsHaswell, utf16BEE: convertUTF32ToUTF16BEWithErrorsHaswell,
			utf16LEV: convertValidUTF32ToUTF16LEHaswell, utf16BEV: convertValidUTF32ToUTF16BEHaswell,
		},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF32ConvertAMD64Variant(t, v.feature)

			latin1Need := latin1LengthFromUTF32Scalar(len(input))
			if latin1Need == 0 {
				t.Fatal("preflight fixture produced empty latin1 length")
			}
			dst := guardedLatin1Destination[byte](latin1Need-1, 0xa5)
			requireUTF32ConvertAMD64Panic(t, func() { v.latin1(input, dst.body) })
			dst.require(t)
			requireUTF32ConvertAMD64Panic(t, func() { v.latin1E(input, dst.body) })
			dst.require(t)
			requireUTF32ConvertAMD64Panic(t, func() { v.latin1V(input, dst.body) })
			dst.require(t)

			utf8Need := utf8LengthFromUTF32Scalar(input)
			if utf8Need == 0 {
				t.Fatal("preflight fixture produced empty utf8 length")
			}
			dst = guardedLatin1Destination[byte](utf8Need-1, 0xa5)
			requireUTF32ConvertAMD64Panic(t, func() { v.utf8(input, dst.body) })
			dst.require(t)
			requireUTF32ConvertAMD64Panic(t, func() { v.utf8E(input, dst.body) })
			dst.require(t)
			requireUTF32ConvertAMD64Panic(t, func() { v.utf8V(input, dst.body) })
			dst.require(t)

			utf16Need := utf16LengthFromUTF32Scalar(input)
			if utf16Need == 0 {
				t.Fatal("preflight fixture produced empty utf16 length")
			}
			dst16 := guardedLatin1Destination[uint16](utf16Need-1, 0xa5a5)
			requireUTF32ConvertAMD64Panic(t, func() { v.utf16LE(input, dst16.body) })
			dst16.require(t)
			requireUTF32ConvertAMD64Panic(t, func() { v.utf16BE(input, dst16.body) })
			dst16.require(t)
			requireUTF32ConvertAMD64Panic(t, func() { v.utf16LEE(input, dst16.body) })
			dst16.require(t)
			requireUTF32ConvertAMD64Panic(t, func() { v.utf16BEE(input, dst16.body) })
			dst16.require(t)
			requireUTF32ConvertAMD64Panic(t, func() { v.utf16LEV(input, dst16.body) })
			dst16.require(t)
			requireUTF32ConvertAMD64Panic(t, func() { v.utf16BEV(input, dst16.body) })
			dst16.require(t)
		})
	}
}

func requireUTF32ConvertAMD64Panic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		if r != "simdutf: destination is too short" {
			t.Fatalf("panic = %v, want %q", r, "simdutf: destination is too short")
		}
	}()
	fn()
}

func checkUTF32ConvertDirect(
	t *testing.T,
	input []uint32,
	latin1 func([]uint32, []byte) int,
	latin1E func([]uint32, []byte) Result,
	latin1V func([]uint32, []byte) int,
	utf8 func([]uint32, []byte) int,
	utf8E func([]uint32, []byte) Result,
	utf8V func([]uint32, []byte) int,
	utf16LE func([]uint32, []uint16) int,
	utf16BE func([]uint32, []uint16) int,
	utf16LEE func([]uint32, []uint16) Result,
	utf16BEE func([]uint32, []uint16) Result,
	utf16LEV func([]uint32, []uint16) int,
	utf16BEV func([]uint32, []uint16) int,
	utf8Len func([]uint32) int,
	utf16Len func([]uint32) int,
) {
	t.Helper()

	if got, want := utf8Len(input), utf8LengthFromUTF32Scalar(input); got != want {
		t.Fatalf("utf8Length = %d, want %d", got, want)
	}
	if got, want := utf16Len(input), utf16LengthFromUTF32Scalar(input); got != want {
		t.Fatalf("utf16Length = %d, want %d", got, want)
	}

	{
		want := bytes.Repeat([]byte{0xa5}, len(input))
		got := guardedLatin1Destination[byte](len(input), 0xa5)
		wantN := convertUTF32ToLatin1Scalar(input, want)
		gotN := latin1(input, got.body)
		if gotN != wantN || !bytes.Equal(got.body, want) {
			t.Fatalf("latin1 convert = %d/%x, want %d/%x", gotN, got.body, wantN, want)
		}
		got.require(t)

		want = bytes.Repeat([]byte{0xa5}, len(input))
		got = guardedLatin1Destination[byte](len(input), 0xa5)
		wantE := convertUTF32ToLatin1WithErrorsScalar(input, want)
		gotE := latin1E(input, got.body)
		if gotE != wantE || !bytes.Equal(got.body, want) {
			t.Fatalf("latin1 with_errors = %#v/%x, want %#v/%x", gotE, got.body, wantE, want)
		}
		got.require(t)

		want = bytes.Repeat([]byte{0xa5}, len(input))
		got = guardedLatin1Destination[byte](len(input), 0xa5)
		wantV := convertValidUTF32ToLatin1Scalar(input, want)
		gotV := latin1V(input, got.body)
		if gotV != wantV || !bytes.Equal(got.body, want) {
			t.Fatalf("latin1 valid = %d/%x, want %d/%x", gotV, got.body, wantV, want)
		}
		got.require(t)
	}

	{
		need := utf8LengthFromUTF32Scalar(input)
		want := bytes.Repeat([]byte{0xa5}, need)
		got := guardedLatin1Destination[byte](need, 0xa5)
		wantN := convertUTF32ToUTF8Scalar(input, want)
		gotN := utf8(input, got.body)
		if gotN != wantN || !bytes.Equal(got.body, want) {
			t.Fatalf("utf8 convert = %d/%x, want %d/%x input=%v", gotN, got.body, wantN, want, input)
		}
		got.require(t)

		want = bytes.Repeat([]byte{0xa5}, need)
		got = guardedLatin1Destination[byte](need, 0xa5)
		wantE := convertUTF32ToUTF8WithErrorsScalar(input, want)
		gotE := utf8E(input, got.body)
		if gotE != wantE || !bytes.Equal(got.body, want) {
			t.Fatalf("utf8 with_errors = %#v/%x, want %#v/%x input=%v", gotE, got.body, wantE, want, input)
		}
		got.require(t)

		// Valid assumes representable UTF-32; skip known invalid fixtures.
		if !utf32ConvertHasInvalid(input) {
			want = bytes.Repeat([]byte{0xa5}, need)
			got = guardedLatin1Destination[byte](need, 0xa5)
			wantV := convertValidUTF32ToUTF8Scalar(input, want)
			gotV := utf8V(input, got.body)
			if gotV != wantV || !bytes.Equal(got.body, want) {
				t.Fatalf("utf8 valid = %d/%x, want %d/%x input=%v", gotV, got.body, wantV, want, input)
			}
			got.require(t)
		}
	}

	{
		need := utf16LengthFromUTF32Scalar(input)
		want := repeatU16Convert(0xa5a5, need)
		got := guardedLatin1Destination[uint16](need, 0xa5a5)
		wantN := convertUTF32ToUTF16LEScalar(input, want)
		gotN := utf16LE(input, got.body)
		if gotN != wantN || !uint16SliceEqual(got.body, want) {
			t.Fatalf("utf16le convert = %d/%x, want %d/%x input=%v", gotN, got.body, wantN, want, input)
		}
		got.require(t)

		want = repeatU16Convert(0xa5a5, need)
		got = guardedLatin1Destination[uint16](need, 0xa5a5)
		wantN = convertUTF32ToUTF16BEScalar(input, want)
		gotN = utf16BE(input, got.body)
		if gotN != wantN || !uint16SliceEqual(got.body, want) {
			t.Fatalf("utf16be convert = %d/%x, want %d/%x input=%v", gotN, got.body, wantN, want, input)
		}
		got.require(t)

		want = repeatU16Convert(0xa5a5, need)
		got = guardedLatin1Destination[uint16](need, 0xa5a5)
		wantE := convertUTF32ToUTF16LEWithErrorsScalar(input, want)
		gotE := utf16LEE(input, got.body)
		if gotE != wantE || !uint16SliceEqual(got.body, want) {
			t.Fatalf("utf16le with_errors = %#v/%x, want %#v/%x input=%v", gotE, got.body, wantE, want, input)
		}
		got.require(t)

		want = repeatU16Convert(0xa5a5, need)
		got = guardedLatin1Destination[uint16](need, 0xa5a5)
		wantE = convertUTF32ToUTF16BEWithErrorsScalar(input, want)
		gotE = utf16BEE(input, got.body)
		if gotE != wantE || !uint16SliceEqual(got.body, want) {
			t.Fatalf("utf16be with_errors = %#v/%x, want %#v/%x input=%v", gotE, got.body, wantE, want, input)
		}
		got.require(t)

		if !utf32ConvertHasInvalid(input) {
			want = repeatU16Convert(0xa5a5, need)
			got = guardedLatin1Destination[uint16](need, 0xa5a5)
			wantV := convertValidUTF32ToUTF16LEScalar(input, want)
			gotV := utf16LEV(input, got.body)
			if gotV != wantV || !uint16SliceEqual(got.body, want) {
				t.Fatalf("utf16le valid = %d/%x, want %d/%x input=%v", gotV, got.body, wantV, want, input)
			}
			got.require(t)

			want = repeatU16Convert(0xa5a5, need)
			got = guardedLatin1Destination[uint16](need, 0xa5a5)
			wantV = convertValidUTF32ToUTF16BEScalar(input, want)
			gotV = utf16BEV(input, got.body)
			if gotV != wantV || !uint16SliceEqual(got.body, want) {
				t.Fatalf("utf16be valid = %d/%x, want %d/%x input=%v", gotV, got.body, wantV, want, input)
			}
			got.require(t)
		}
	}
}

func utf32ConvertHasInvalid(input []uint32) bool {
	for _, word := range input {
		if word > 0x10ffff || (word >= 0xd800 && word <= 0xdfff) {
			return true
		}
	}
	return false
}

func repeatU32Convert(value uint32, n int) []uint32 {
	out := make([]uint32, n)
	for i := range out {
		out[i] = value
	}
	return out
}

func repeatU16Convert(value uint16, n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		out[i] = value
	}
	return out
}

func uint16SliceEqual(a, b []uint16) bool {
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

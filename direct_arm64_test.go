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
	"math/bits"
	"slices"
	"testing"
	"unicode/utf8"
)

// Package simdutf provides a Go port of https://github.com/simdutf/simdutf.
//
// Unicode routines (UTF8, UTF16, UTF32) and Base64: billions of characters per second using SSE2, AVX2, NEON, AVX-512, RISC-V Vector Extension, LoongArch64, POWER.

func TestBase64NEONEncodeMatchesScalar(t *testing.T) {
	inputs := [][]byte{
		[]byte("Hello, simdutf Base64!"),            // 22 bytes - no NEON blocks
		make([]byte, 48),                            // exactly one block
		make([]byte, 96),                            // two blocks
		bytes.Repeat([]byte("0123456789abcdef"), 8), // 128
	}
	for i := range inputs[1] {
		inputs[1][i] = byte(i)
	}
	for i := range inputs[2] {
		inputs[2][i] = byte(i)
	}
	for _, opt := range []Base64Options{Base64Default, Base64URL} {
		for _, in := range inputs {
			dstN := make([]byte, base64LengthFromBinaryScalar(len(in), opt))
			dstS := make([]byte, len(dstN))
			nN := binaryToBase64NEON(in, dstN, opt)
			nS := binaryToBase64Scalar(in, dstS, opt)
			if nN != nS || !bytes.Equal(dstN[:nN], dstS[:nS]) {
				t.Fatalf("opt=%v len=%d neon=%q scalar=%q", opt, len(in), dstN[:min(nN, 96)], dstS[:min(nS, 96)])
			}
		}
	}
}

func TestBase64NEONLengthMatchesScalar(t *testing.T) {
	inputs := [][]byte{
		[]byte("AQID"),
		bytes.Repeat([]byte("A"), 128),
		append(bytes.Repeat([]byte("A"), 100), '=', '=', '\n'),
		{},
	}
	for _, in := range inputs {
		if got, want := binaryLengthFromBase64NEON(in), binaryLengthFromBase64Scalar(in); got != want {
			t.Fatalf("len neon=%d scalar=%d input=%q", got, want, in)
		}
	}
}

func TestBase64NEONDecodeMatchesScalar(t *testing.T) {
	raw := bytes.Repeat([]byte("Hello NEON Base64 decode path!!"), 6)
	enc := make([]byte, base64LengthFromBinaryScalar(len(raw), Base64Default))
	n := binaryToBase64Scalar(raw, enc, Base64Default)
	enc = enc[:n]
	dstN := make([]byte, maximalBinaryLengthFromBase64Scalar(enc))
	dstS := make([]byte, len(dstN))
	rN := base64ToBinaryDetailsNEON(enc, dstN, Base64Default, Loose)
	rS := base64ToBinaryDetailsScalar(enc, dstS, Base64Default, Loose)
	if rN != rS || !bytes.Equal(dstN[:rN.OutputCount], dstS[:rS.OutputCount]) {
		t.Fatalf("neon=%+v scalar=%+v", rN, rS)
	}
}

func TestBase64NEONWithLinesMatchesScalar(t *testing.T) {
	in := bytes.Repeat([]byte("abcdef"), 40)
	for _, line := range []int{76, 64, 16, 4} {
		dstN := make([]byte, base64LengthFromBinaryWithLinesScalar(len(in), Base64Default, line))
		dstS := make([]byte, len(dstN))
		nN := binaryToBase64WithLinesNEON(in, dstN, line, Base64Default)
		nS := binaryToBase64WithLinesScalar(in, dstS, line, Base64Default)
		if nN != nS || !bytes.Equal(dstN[:nN], dstS[:nS]) {
			t.Fatalf("line=%d neon(%d) scalar(%d)", line, nN, nS)
		}
	}
}

func TestBase64NEONBlocksDirect(t *testing.T) {
	for _, n := range []int{48, 96, 144} {
		in := make([]byte, n)
		for i := range in {
			in[i] = byte(i * 3)
		}
		dst := make([]byte, n/3*4)
		binaryToBase64BlocksDefaultNEON(in, dst)
		want := make([]byte, len(dst))
		tailEncodeBase64(want, in, Base64Default, false, 0)
		if !bytes.Equal(dst, want) {
			t.Fatalf("n=%d mismatch\n got %q\nwant %q", n, dst, want)
		}
	}
}

func TestBase64NEONLengthUTF16MatchesScalar(t *testing.T) {
	in := make([]uint16, 128)
	for i := range in {
		in[i] = 'A'
	}
	in[120] = '='
	in[121] = '='
	if got, want := binaryLengthFromBase64UTF16NEON(in), binaryLengthFromBase64UTF16Scalar(in); got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestBase64NEONDecodeUTF16MatchesScalar(t *testing.T) {
	raw := bytes.Repeat([]byte("abcdef"), 30)
	enc := make([]byte, base64LengthFromBinaryScalar(len(raw), Base64Default))
	n := binaryToBase64Scalar(raw, enc, Base64Default)
	u16 := make([]uint16, n)
	for i := range n {
		u16[i] = uint16(enc[i])
	}
	dstN := make([]byte, maximalBinaryLengthFromBase64UTF16Scalar(u16))
	dstS := make([]byte, len(dstN))
	rN := base64ToBinaryDetailsUTF16NEON(u16, dstN, Base64Default, Loose)
	rS := base64ToBinaryDetailsUTF16Scalar(u16, dstS, Base64Default, Loose)
	if rN != rS || !bytes.Equal(dstN[:rN.OutputCount], dstS[:rS.OutputCount]) {
		t.Fatalf("neon=%+v scalar=%+v", rN, rS)
	}
}

func detectEncodingsDirectCases() []struct {
	name  string
	input []byte
} {
	return []struct {
		name  string
		input []byte
	}{
		{name: "nil", input: nil},
		{name: "empty", input: []byte{}},
		{name: "utf8-only-odd", input: []byte("hello")},
		{name: "utf8-bom", input: []byte{0xef, 0xbb, 0xbf}},
		{name: "utf16le-bom", input: []byte{0xff, 0xfe}},
		{name: "utf16be-bom", input: []byte{0xfe, 0xff}},
		{name: "utf32le-bom", input: []byte{0xff, 0xfe, 0x00, 0x00}},
		{name: "utf32be-bom", input: []byte{0x00, 0x00, 0xfe, 0xff}},
		{name: "utf16le-even-ascii", input: []byte{'A', 0, 'B', 0}},
		{name: "utf32le-ascii-unit", input: []byte{'A', 0, 0, 0}},
		{name: "ascii-even", input: []byte("hi")},
		{name: "short-odd-invalid", input: []byte{0xff}},
		{name: "invalid-mix-issue516", input: []byte{0x20, 0xd8, 0x00, 0x00}},
		{name: "utf16le-only-surrogate-pair", input: []byte{0x00, 0xd8, 0x00, 0xdc}},
		{name: "issue519", input: []byte{
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 223, 164, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
		}},
	}
}

func TestDirectARM64DetectEncodingsAgainstScalar(t *testing.T) {
	for _, tc := range detectEncodingsDirectCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := detectEncodingsNEON(tc.input)
			want := detectEncodingsScalar(tc.input)
			if got != want {
				t.Fatalf("detectEncodingsNEON(%q) = %d, want scalar %d", tc.input, got, want)
			}
		})
	}
}

// Direct differential coverage for the arm64 NEON translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/arm64/arm_find.cpp.
// Cases stay darwin/arm64-friendly: empty, short, aligned/unaligned lengths,
// first/middle/last hits, misses, and NUL/high-unit needles.

func TestDirectARM64FindNEONAgainstScalar(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		value byte
	}{
		{name: "nil", input: nil, value: 'a'},
		{name: "empty", input: []byte{}, value: 'a'},
		{name: "hit-first", input: []byte("abc"), value: 'a'},
		{name: "hit-middle", input: []byte("bac"), value: 'a'},
		{name: "hit-last", input: []byte("bca"), value: 'a'},
		{name: "miss", input: []byte("bcd"), value: 'a'},
		{name: "nul-hit", input: []byte{'A', 'B', 0, 'C'}, value: 0},
		{name: "short", input: []byte{7}, value: 7},
		{name: "len15", input: append(bytes.Repeat([]byte{'x'}, 14), 'a'), value: 'a'},
		{name: "len16-first", input: append([]byte{'a'}, bytes.Repeat([]byte{'x'}, 15)...), value: 'a'},
		{name: "len16-last", input: append(bytes.Repeat([]byte{'x'}, 15), 'a'), value: 'a'},
		{name: "len16-miss", input: bytes.Repeat([]byte{'x'}, 16), value: 'a'},
		{name: "len63", input: append(bytes.Repeat([]byte{'x'}, 62), 'a'), value: 'a'},
		{name: "len64-first", input: append([]byte{'a'}, bytes.Repeat([]byte{'x'}, 63)...), value: 'a'},
		{name: "len64-mid", input: append(append(bytes.Repeat([]byte{'x'}, 40), 'a'), bytes.Repeat([]byte{'x'}, 23)...), value: 'a'},
		{name: "len64-last", input: append(bytes.Repeat([]byte{'x'}, 63), 'a'), value: 'a'},
		{name: "len64-miss", input: bytes.Repeat([]byte{'x'}, 64), value: 'a'},
		{name: "len65", input: append(bytes.Repeat([]byte{'x'}, 64), 'a'), value: 'a'},
		{name: "len127", input: append(bytes.Repeat([]byte{'x'}, 126), 'a'), value: 'a'},
		{name: "len128-chunk2", input: append(bytes.Repeat([]byte{'x'}, 80), append([]byte{'a'}, bytes.Repeat([]byte{'x'}, 47)...)...), value: 'a'},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := findScalar(tc.input, tc.value)
			if got := findNEON(tc.input, tc.value); got != want {
				t.Fatalf("findNEON(%q, %q) = %d, want %d", tc.input, tc.value, got, want)
			}
		})
	}
}

func TestDirectARM64FindUTF16NEONAgainstScalar(t *testing.T) {
	repeat := func(unit uint16, n int) []uint16 {
		out := make([]uint16, n)
		for i := range out {
			out[i] = unit
		}
		return out
	}
	cases := []struct {
		name  string
		input []uint16
		value uint16
	}{
		{name: "nil", input: nil, value: 'a'},
		{name: "empty", input: []uint16{}, value: 'a'},
		{name: "hit-first", input: []uint16{'a', 'b', 'c'}, value: 'a'},
		{name: "hit-middle", input: []uint16{'b', 'a', 'c'}, value: 'a'},
		{name: "hit-last", input: []uint16{'b', 'c', 'a'}, value: 'a'},
		{name: "miss", input: []uint16{'b', 'c', 'd'}, value: 'a'},
		{name: "nul-hit", input: []uint16{'A', 0, 'C'}, value: 0},
		{name: "high-unit", input: []uint16{0x20, 0xd800, 0x20}, value: 0xd800},
		{name: "short", input: []uint16{7}, value: 7},
		{name: "len7", input: append(repeat('x', 6), 'a'), value: 'a'},
		{name: "len8-first", input: append([]uint16{'a'}, repeat('x', 7)...), value: 'a'},
		{name: "len8-last", input: append(repeat('x', 7), 'a'), value: 'a'},
		{name: "len8-miss", input: repeat('x', 8), value: 'a'},
		{name: "len31", input: append(repeat('x', 30), 'a'), value: 'a'},
		{name: "len32-first", input: append([]uint16{'a'}, repeat('x', 31)...), value: 'a'},
		{name: "len32-mid", input: append(append(repeat('x', 20), 'a'), repeat('x', 11)...), value: 'a'},
		{name: "len32-last", input: append(repeat('x', 31), 'a'), value: 'a'},
		{name: "len32-miss", input: repeat('x', 32), value: 'a'},
		{name: "len33", input: append(repeat('x', 32), 'a'), value: 'a'},
		{name: "len64-chunk2", input: append(repeat('x', 40), append([]uint16{'a'}, repeat('x', 23)...)...), value: 'a'},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := findUTF16Scalar(tc.input, tc.value)
			if got := findUTF16NEON(tc.input, tc.value); got != want {
				t.Fatalf("findUTF16NEON(%v, %#x) = %d, want %d", tc.input, tc.value, got, want)
			}
		})
	}
}

func TestDirectARM64FindNEONUnalignedSubslice(t *testing.T) {
	buf := bytes.Repeat([]byte{'x'}, 96)
	buf[17] = 'a'
	buf[50] = 'a'
	for off := 1; off < 16; off++ {
		input := buf[off:80]
		want := findScalar(input, 'a')
		if got := findNEON(input, 'a'); got != want {
			t.Fatalf("off=%d findNEON = %d, want %d", off, got, want)
		}
		if got := findNEON(input, 'z'); got != findScalar(input, 'z') {
			t.Fatalf("off=%d miss mismatch", off)
		}
	}
	u16 := make([]uint16, 96)
	for i := range u16 {
		u16[i] = 'x'
	}
	u16[17] = 'a'
	u16[50] = 'a'
	for off := 1; off < 8; off++ {
		input := u16[off:80]
		want := findUTF16Scalar(input, 'a')
		if got := findUTF16NEON(input, 'a'); got != want {
			t.Fatalf("u16 off=%d findUTF16NEON = %d, want %d", off, got, want)
		}
	}
}

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the arm64 Latin-1 NEON translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/implementation.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.

func TestLatin1NEONDirectAgainstScalar(t *testing.T) {
	inputs := [][]byte{
		nil,
		{0},
		bytes.Repeat([]byte{'A'}, 15),
		bytes.Repeat([]byte{'A'}, 16),
		bytes.Repeat([]byte{'A'}, 31),
		bytes.Repeat([]byte{'A'}, 32),
		bytes.Repeat([]byte{'A'}, 63),
		bytes.Repeat([]byte{'A'}, 64),
		append(bytes.Repeat([]byte{'A'}, 63), 0x80),
		append(bytes.Repeat([]byte{0xff}, 64), 0x7f),
	}
	for _, input := range inputs {
		t.Run("length", func(t *testing.T) {
			wantLength := utf8LengthFromLatin1Scalar(input)
			if got := utf8LengthFromLatin1NEON(input); got != wantLength {
				t.Fatalf("UTF-8 length = %d, want %d for %x", got, wantLength, input)
			}

			want8 := make([]byte, wantLength)
			convertLatin1ToUTF8Scalar(input, want8)
			got8 := guardedLatin1Destination[byte](wantLength, 0xa5)
			if got := convertLatin1ToUTF8NEON(input, got8.body); got != len(want8) || !bytes.Equal(got8.body, want8) {
				t.Fatalf("UTF-8 = %x (%d), want %x (%d)", got8.body, got, want8, len(want8))
			}
			got8.require(t)

			want32 := make([]uint32, len(input))
			convertLatin1ToUTF32Scalar(input, want32)
			got32 := guardedLatin1Destination[uint32](len(input), 0xa5a5a5a5)
			if got := convertLatin1ToUTF32NEON(input, got32.body); got != len(input) || !slices.Equal(got32.body, want32) {
				t.Fatalf("UTF-32 = %x (%d), want %x (%d)", got32.body, got, want32, len(input))
			}
			got32.require(t)

			for _, little := range []bool{true, false} {
				want16 := make([]uint16, len(input))
				if little {
					convertLatin1ToUTF16LEScalar(input, want16)
				} else {
					convertLatin1ToUTF16BEScalar(input, want16)
				}
				got16 := guardedLatin1Destination[uint16](len(input), 0xa5a5)
				var got int
				if little {
					got = convertLatin1ToUTF16LENEON(input, got16.body)
				} else {
					got = convertLatin1ToUTF16BENEON(input, got16.body)
				}
				if got != len(input) || !slices.Equal(got16.body, want16) {
					t.Fatalf("UTF-16 little=%t = %x (%d), want %x (%d)", little, got16.body, got, want16, len(input))
				}
				got16.require(t)
			}
		})
	}
}

func TestLatin1NEONDirectPreflightPreservesCanaries(t *testing.T) {
	input := append(bytes.Repeat([]byte{'A'}, 63), 0xff)
	want8 := utf8LengthFromLatin1Scalar(input)
	dst8 := guardedLatin1Destination[byte](want8-1, 0xa5)
	requireLatin1Panic(t, func() { convertLatin1ToUTF8NEON(input, dst8.body) })
	dst8.require(t)

	dst16 := guardedLatin1Destination[uint16](len(input)-1, 0xa5a5)
	requireLatin1Panic(t, func() { convertLatin1ToUTF16LENEON(input, dst16.body) })
	dst16.require(t)
	requireLatin1Panic(t, func() { convertLatin1ToUTF16BENEON(input, dst16.body) })
	dst16.require(t)

	dst32 := guardedLatin1Destination[uint32](len(input)-1, 0xa5a5a5a5)
	requireLatin1Panic(t, func() { convertLatin1ToUTF32NEON(input, dst32.body) })
	dst32.require(t)
}

func FuzzLatin1NEONAgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte{'A'}, 64))
	f.Add(append(bytes.Repeat([]byte{0xff}, 64), 0x80))
	f.Fuzz(func(t *testing.T, input []byte) {
		wantLength := utf8LengthFromLatin1Scalar(input)
		if got := utf8LengthFromLatin1NEON(input); got != wantLength {
			t.Fatalf("UTF-8 length = %d, want %d", got, wantLength)
		}

		want8, got8 := make([]byte, wantLength), make([]byte, wantLength)
		convertLatin1ToUTF8Scalar(input, want8)
		convertLatin1ToUTF8NEON(input, got8)
		if !bytes.Equal(got8, want8) {
			t.Fatalf("UTF-8 = %x, want %x", got8, want8)
		}

		want16, got16 := make([]uint16, len(input)), make([]uint16, len(input))
		convertLatin1ToUTF16LEScalar(input, want16)
		convertLatin1ToUTF16LENEON(input, got16)
		if !slices.Equal(got16, want16) {
			t.Fatalf("UTF-16LE = %x, want %x", got16, want16)
		}
		convertLatin1ToUTF16BEScalar(input, want16)
		convertLatin1ToUTF16BENEON(input, got16)
		if !slices.Equal(got16, want16) {
			t.Fatalf("UTF-16BE = %x, want %x", got16, want16)
		}

		want32, got32 := make([]uint32, len(input)), make([]uint32, len(input))
		convertLatin1ToUTF32Scalar(input, want32)
		convertLatin1ToUTF32NEON(input, got32)
		if !slices.Equal(got32, want32) {
			t.Fatalf("UTF-32 = %x, want %x", got32, want32)
		}
	})
}

func TestDirectARM64UTF16HelpersAgainstScalar(t *testing.T) {
	cases := [][]uint16{
		{},
		{0x61, 0x62},
		{0x00ff, 0xff00, 0xd800, 0xdc00, 0xd83d, 0xde00},
		make([]uint16, 32),
		make([]uint16, 33),
		make([]uint16, 64),
	}
	for i := range cases[3] {
		cases[3][i] = uint16(0x4e00 + i)
	}
	for i := range cases[4] {
		cases[4][i] = uint16(i * 7)
	}
	for i := range cases[5] {
		if i%4 == 0 {
			cases[5][i] = 0xd800
		} else if i%4 == 1 {
			cases[5][i] = 0xdc00
		} else {
			cases[5][i] = uint16(i)
		}
	}
	for _, input := range cases {
		want := make([]uint16, len(input))
		got := make([]uint16, len(input))
		changeEndiannessUTF16Scalar(input, want)
		changeEndiannessUTF16NEON(input, got)
		if len(got) != len(want) {
			t.Fatalf("changeEndianness NEON length mismatch")
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("changeEndianness NEON mismatch input=%v want=%v got=%v", input, want, got)
			}
		}
		if countUTF16LENEON(input) != countUTF16LEScalar(input) {
			t.Fatalf("countLE NEON mismatch input=%v", input)
		}
		if countUTF16BENEON(input) != countUTF16BEScalar(input) {
			t.Fatalf("countBE NEON mismatch input=%v", input)
		}
		if utf32LengthFromUTF16LENEON(input) != utf32LengthFromUTF16LEScalar(input) {
			t.Fatalf("utf32LengthLE NEON mismatch")
		}
		if utf32LengthFromUTF16BENEON(input) != utf32LengthFromUTF16BEScalar(input) {
			t.Fatalf("utf32LengthBE NEON mismatch")
		}
	}
}

func TestDirectARM64UTF16HelpersPreflightPreservesDestination(t *testing.T) {
	input := []uint16{1, 2, 3, 4}
	dst := []uint16{9, 9}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic")
			}
		}()
		changeEndiannessUTF16NEON(input, dst)
	}()
	if dst[0] != 9 || dst[1] != 9 {
		t.Fatal("NEON preflight mutated short destination")
	}
}

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the arm64 UTF-16→Latin-1 NEON translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf16_to_latin1.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// NEON providers are invoked directly (no detectARM64Features).

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
	for i := range n {
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

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the arm64 UTF-16→UTF-32 NEON translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf16_to_utf32.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// NEON providers are invoked directly (no detectARM64Features).

func TestUTF16UTF32NEONDirectAgainstScalar(t *testing.T) {
	natives := [][]uint16{
		nil,
		{},
		{0},
		{'a', 'b', 'c'},
		{0x00, 0x7f, 0xff, 0x100, 0x7ff, 0x800, 0xffff},
		{0x20ac, 0xd83d, 0xde00},
		repeatUTF16('A', 7),
		repeatUTF16('A', 8),
		repeatUTF16('A', 9),
		repeatUTF16('A', 15),
		repeatUTF16('A', 16),
		repeatUTF16('A', 17),
		append(repeatUTF16('A', 7), 0x20ac),
		append(repeatUTF16('A', 8), 0xd83d, 0xde00),
		append(repeatUTF16('A', 15), 0xd83d, 0xde00, 'Z'),
		append(repeatUTF16('A', 16), 0xd800),
		append(repeatUTF16('A', 16), 0xdc00),
		append(repeatUTF16('A', 16), 0xd800, 0x0041),
		{'a', 0xd83d, 0xde00, 'b'},
		{0xd800},
		{0xdc00},
		{0xd800, 0x0041},
		{0xd83d, 0xde00},
		{0xd800, 0xdc00, 0xdbff, 0xdfff},
	}
	for _, native := range natives {
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			checkUTF16UTF32DirectNEON(t, input, little)
		}
	}
}

func TestUTF16UTF32NEONDirectPreflightPreservesCanaries(t *testing.T) {
	native := append(repeatUTF16('A', 15), 0x20ac)
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		need := utf32LengthFromUTF16Scalar(input, little == nativeLittleEndian())
		if need == 0 {
			continue
		}
		dst := guardedLatin1Destination[uint32](need-1, 0xa5a5a5a5)
		if little {
			requireLatin1Panic(t, func() { convertUTF16LEToUTF32NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16LEToUTF32WithErrorsNEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertValidUTF16LEToUTF32NEON(input, dst.body) })
			dst.require(t)
		} else {
			requireLatin1Panic(t, func() { convertUTF16BEToUTF32NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16BEToUTF32WithErrorsNEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertValidUTF16BEToUTF32NEON(input, dst.body) })
			dst.require(t)
		}
	}
}

func FuzzUTF16UTF32NEONAgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(utf16NativeBytes(repeatUTF16('A', 8)))
	f.Add(utf16NativeBytes(repeatUTF16(0x20ac, 16)))
	f.Add(utf16NativeBytes([]uint16{'a', 0xd83d, 0xde00, 'b'}))
	f.Add(utf16NativeBytes(append(repeatUTF16('A', 16), 0xd800)))
	f.Add(utf16NativeBytes([]uint16{0xd800, 0xdc00, 0xdbff, 0xdfff}))
	f.Fuzz(func(t *testing.T, raw []byte) {
		native := utf16NativeFromBytes(raw)
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			checkUTF16UTF32DirectNEON(t, input, little)
		}
	})
}

func checkUTF16UTF32DirectNEON(t *testing.T, input []uint16, little bool) {
	t.Helper()

	storageIsNative := little == nativeLittleEndian()
	wantLen := utf32LengthFromUTF16Scalar(input, storageIsNative)

	want := make([]uint32, wantLen)
	for i := range want {
		want[i] = 0xa5a5a5a5
	}
	got := guardedLatin1Destination[uint32](wantLen, 0xa5a5a5a5)
	var (
		wantN, gotN int
		wantE, gotE Result
		wantV, gotV int
	)
	if little {
		wantN = convertUTF16LEToUTF32Scalar(input, want)
		gotN = convertUTF16LEToUTF32NEON(input, got.body)
	} else {
		wantN = convertUTF16BEToUTF32Scalar(input, want)
		gotN = convertUTF16BEToUTF32NEON(input, got.body)
	}
	if gotN != wantN || !slices.Equal(got.body, want) {
		t.Fatalf("little=%t convert = %d/%x, want %d/%x", little, gotN, got.body, wantN, want)
	}
	got.require(t)

	want = make([]uint32, wantLen)
	for i := range want {
		want[i] = 0xa5a5a5a5
	}
	got = guardedLatin1Destination[uint32](wantLen, 0xa5a5a5a5)
	if little {
		wantE = convertUTF16LEToUTF32WithErrorsScalar(input, want)
		gotE = convertUTF16LEToUTF32WithErrorsNEON(input, got.body)
	} else {
		wantE = convertUTF16BEToUTF32WithErrorsScalar(input, want)
		gotE = convertUTF16BEToUTF32WithErrorsNEON(input, got.body)
	}
	if gotE != wantE || !slices.Equal(got.body, want) {
		t.Fatalf("little=%t with_errors = %#v/%x, want %#v/%x", little, gotE, got.body, wantE, want)
	}
	got.require(t)

	if wantE.Error != Success {
		return
	}
	want = make([]uint32, wantLen)
	for i := range want {
		want[i] = 0xa5a5a5a5
	}
	got = guardedLatin1Destination[uint32](wantLen, 0xa5a5a5a5)
	if little {
		wantV = convertValidUTF16LEToUTF32Scalar(input, want)
		gotV = convertValidUTF16LEToUTF32NEON(input, got.body)
	} else {
		wantV = convertValidUTF16BEToUTF32Scalar(input, want)
		gotV = convertValidUTF16BEToUTF32NEON(input, got.body)
	}
	if gotV != wantV || !slices.Equal(got.body, want) {
		t.Fatalf("little=%t valid = %d/%x, want %d/%x", little, gotV, got.body, wantV, want)
	}
	got.require(t)
}

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the arm64 UTF-16→UTF-8 NEON translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf16_to_utf8.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// NEON providers are invoked directly (no detectARM64Features).

func TestUTF16UTF8NEONDirectAgainstScalar(t *testing.T) {
	natives := [][]uint16{
		nil,
		{},
		{0},
		{'a', 'b', 'c'},
		{0x00, 0x7f, 0xff, 0x100, 0x7ff, 0x800, 0xffff},
		{0x20ac, 0xd83d, 0xde00},
		repeatUTF16('A', 7),
		repeatUTF16('A', 8),
		repeatUTF16('A', 9),
		repeatUTF16('A', 15),
		repeatUTF16('A', 16),
		repeatUTF16('A', 17),
		append(repeatUTF16('A', 7), 0x20ac),
		append(repeatUTF16('A', 8), 0xd83d, 0xde00),
		append(repeatUTF16('A', 15), 0xd83d, 0xde00, 'Z'),
		append(repeatUTF16('A', 16), 0xd800),
		append(repeatUTF16('A', 16), 0xdc00),
		append(repeatUTF16('A', 16), 0xd800, 0x0041),
		{'a', 0xd83d, 0xde00, 'b'},
		{0xd800},
		{0xdc00},
		{0xd800, 0x0041},
		{0xd83d, 0xde00},
		{0xd800, 0xdc00, 0xdbff, 0xdfff},
	}
	for _, native := range natives {
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			checkUTF16UTF8DirectNEON(t, input, little)
		}
	}
}

func TestUTF16UTF8NEONDirectPreflightPreservesCanaries(t *testing.T) {
	native := append(repeatUTF16('A', 15), 0x20ac)
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		storageIsNative := little == nativeLittleEndian()
		need := utf8LengthFromUTF16Scalar(input, storageIsNative)
		needRepl := utf8LengthFromUTF16WithReplacementScalar(input, storageIsNative)
		if need == 0 || needRepl == 0 {
			continue
		}
		dst := guardedLatin1Destination[byte](need-1, 0xa5)
		dstRepl := guardedLatin1Destination[byte](needRepl-1, 0xa5)
		if little {
			requireLatin1Panic(t, func() { convertUTF16LEToUTF8NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16LEToUTF8WithErrorsNEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertValidUTF16LEToUTF8NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16LEToUTF8WithReplacementNEON(input, dstRepl.body) })
			dstRepl.require(t)
		} else {
			requireLatin1Panic(t, func() { convertUTF16BEToUTF8NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16BEToUTF8WithErrorsNEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertValidUTF16BEToUTF8NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16BEToUTF8WithReplacementNEON(input, dstRepl.body) })
			dstRepl.require(t)
		}
	}
}

func FuzzUTF16UTF8NEONAgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(utf16NativeBytes(repeatUTF16('A', 8)))
	f.Add(utf16NativeBytes(repeatUTF16(0x20ac, 16)))
	f.Add(utf16NativeBytes([]uint16{'a', 0xd83d, 0xde00, 'b'}))
	f.Add(utf16NativeBytes(append(repeatUTF16('A', 16), 0xd800)))
	f.Add(utf16NativeBytes([]uint16{0xd800, 0xdc00, 0xdbff, 0xdfff}))
	f.Fuzz(func(t *testing.T, raw []byte) {
		native := utf16NativeFromBytes(raw)
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			checkUTF16UTF8DirectNEON(t, input, little)
		}
	})
}

func checkUTF16UTF8DirectNEON(t *testing.T, input []uint16, little bool) {
	t.Helper()

	storageIsNative := little == nativeLittleEndian()
	wantLen := utf8LengthFromUTF16Scalar(input, storageIsNative)
	wantLenRepl := utf8LengthFromUTF16WithReplacementScalar(input, storageIsNative)

	var (
		wantN, gotN       int
		wantE, gotE       Result
		wantV, gotV       int
		wantR, gotR       int
		wantLenN, gotLenN int
		wantLR, gotLR     Result
	)

	if little {
		wantLenN = utf8LengthFromUTF16LEScalar(input)
		gotLenN = utf8LengthFromUTF16LENEON(input)
		wantLR = utf8LengthFromUTF16LEWithReplacementScalar(input)
		gotLR = utf8LengthFromUTF16LEWithReplacementNEON(input)
	} else {
		wantLenN = utf8LengthFromUTF16BEScalar(input)
		gotLenN = utf8LengthFromUTF16BENEON(input)
		wantLR = utf8LengthFromUTF16BEWithReplacementScalar(input)
		gotLR = utf8LengthFromUTF16BEWithReplacementNEON(input)
	}
	if gotLenN != wantLenN {
		t.Fatalf("little=%t utf8_length = %d, want %d", little, gotLenN, wantLenN)
	}
	if gotLR != wantLR {
		t.Fatalf("little=%t utf8_length_with_replacement = %#v, want %#v", little, gotLR, wantLR)
	}

	want := bytes.Repeat([]byte{0xa5}, wantLen)
	got := guardedLatin1Destination[byte](wantLen, 0xa5)
	if little {
		wantN = convertUTF16LEToUTF8Scalar(input, want)
		gotN = convertUTF16LEToUTF8NEON(input, got.body)
	} else {
		wantN = convertUTF16BEToUTF8Scalar(input, want)
		gotN = convertUTF16BEToUTF8NEON(input, got.body)
	}
	if gotN != wantN || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t convert = %d/%x, want %d/%x", little, gotN, got.body, wantN, want)
	}
	got.require(t)

	want = bytes.Repeat([]byte{0xa5}, wantLen)
	got = guardedLatin1Destination[byte](wantLen, 0xa5)
	if little {
		wantE = convertUTF16LEToUTF8WithErrorsScalar(input, want)
		gotE = convertUTF16LEToUTF8WithErrorsNEON(input, got.body)
	} else {
		wantE = convertUTF16BEToUTF8WithErrorsScalar(input, want)
		gotE = convertUTF16BEToUTF8WithErrorsNEON(input, got.body)
	}
	if gotE != wantE || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t with_errors = %#v/%x, want %#v/%x", little, gotE, got.body, wantE, want)
	}
	got.require(t)

	want = bytes.Repeat([]byte{0xa5}, wantLenRepl)
	got = guardedLatin1Destination[byte](wantLenRepl, 0xa5)
	if little {
		wantR = convertUTF16LEToUTF8WithReplacementScalar(input, want)
		gotR = convertUTF16LEToUTF8WithReplacementNEON(input, got.body)
	} else {
		wantR = convertUTF16BEToUTF8WithReplacementScalar(input, want)
		gotR = convertUTF16BEToUTF8WithReplacementNEON(input, got.body)
	}
	if gotR != wantR || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t with_replacement = %d/%x, want %d/%x", little, gotR, got.body, wantR, want)
	}
	got.require(t)

	if wantE.Error != Success {
		return
	}
	want = bytes.Repeat([]byte{0xa5}, wantLen)
	got = guardedLatin1Destination[byte](wantLen, 0xa5)
	if little {
		wantV = convertValidUTF16LEToUTF8Scalar(input, want)
		gotV = convertValidUTF16LEToUTF8NEON(input, got.body)
	} else {
		wantV = convertValidUTF16BEToUTF8Scalar(input, want)
		gotV = convertValidUTF16BEToUTF8NEON(input, got.body)
	}
	if gotV != wantV || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t valid = %d/%x, want %d/%x", little, gotV, got.body, wantV, want)
	}
	got.require(t)
}

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the arm64 UTF-32 convert/length NEON
// translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf32_to_*.cpp
// and utf8/utf16_length_from_utf32 in src/arm64/implementation.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// NEON providers are invoked directly (no detectARM64Features).

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

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the arm64 translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b.

func rawUTF16NEONTest(words []uint16, little bool) []uint16 {
	out := append([]uint16(nil), words...)
	if !little {
		for i := range out {
			out[i] = bits.ReverseBytes16(out[i])
		}
	}
	return out
}

func TestUTF16NEONDirectAgainstScalar(t *testing.T) {
	semantic := []uint16{0x0061, 0xd800, 0xdc00, 0x0062, 0xdc00, 0x0063, 0xdbff, 0x0064, 0xd800}
	for _, little := range []bool{true, false} {
		for _, n := range []int{0, 1, 15, 16, 17, 31, 32, 33} {
			for _, bad := range []int{-1, 0, n / 2, n - 1} {
				if bad >= n {
					continue
				}
				words := make([]uint16, n)
				for i := range words {
					words[i] = 0x0061
				}
				if bad >= 0 {
					words[bad] = semantic[(bad+4)%len(semantic)]
				}
				input := rawUTF16NEONTest(words, little)
				var gotBool, wantBool bool
				var gotResult, wantResult Result
				if little {
					gotBool, wantBool = validateUTF16LENEON(input), validateUTF16LEScalar(input)
					gotResult, wantResult = validateUTF16LEWithErrorsNEON(input), validateUTF16LEWithErrorsScalar(input)
				} else {
					gotBool, wantBool = validateUTF16BENEON(input), validateUTF16BEScalar(input)
					gotResult, wantResult = validateUTF16BEWithErrorsNEON(input), validateUTF16BEWithErrorsScalar(input)
				}
				if gotBool != wantBool || gotResult != wantResult {
					t.Fatalf("little=%t n=%d bad=%d: bool=%t/%t result=%+v/%+v", little, n, bad, gotBool, wantBool, gotResult, wantResult)
				}
			}
		}
	}
}

func TestToWellFormedUTF16NEONDirect(t *testing.T) {
	semantic := []uint16{0x0061, 0xd800, 0xdc00, 0xdc00, 0xdbff, 0x0062, 0xd800}
	for _, little := range []bool{true, false} {
		input := rawUTF16NEONTest(semantic, little)
		want := make([]uint16, len(input))
		if little {
			toWellFormedUTF16LEScalar(input, want)
		} else {
			toWellFormedUTF16BEScalar(input, want)
		}
		buf := make([]uint16, len(input)+2)
		buf[0], buf[len(buf)-1] = 0xa55a, 0x5aa5
		if little {
			toWellFormedUTF16LENEON(input, buf[1:len(buf)-1])
		} else {
			toWellFormedUTF16BENEON(input, buf[1:len(buf)-1])
		}
		if buf[0] != 0xa55a || buf[len(buf)-1] != 0x5aa5 || !slices.Equal(buf[1:len(buf)-1], want) {
			t.Fatalf("little=%t: canary or output differs: %x want %x", little, buf, want)
		}
		inPlace := append([]uint16(nil), input...)
		if little {
			toWellFormedUTF16LENEON(inPlace, inPlace)
		} else {
			toWellFormedUTF16BENEON(inPlace, inPlace)
		}
		if !slices.Equal(inPlace, want) {
			t.Fatalf("little=%t: in-place output=%x want=%x", little, inPlace, want)
		}
		short := []uint16{0xa55a, 0x5aa5}
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("short destination did not panic")
				}
			}()
			if little {
				toWellFormedUTF16LENEON(input, short)
			} else {
				toWellFormedUTF16BENEON(input, short)
			}
		}()
		if !slices.Equal(short, []uint16{0xa55a, 0x5aa5}) {
			t.Fatalf("little=%t: short destination was modified: %x", little, short)
		}
	}
}

func TestUTF32NEONDirectAgainstScalar(t *testing.T) {
	for _, n := range []int{0, 1, 3, 4, 5, 7, 8, 9} {
		for _, invalid := range []uint32{0, 0xd800, 0xdfff, 0x110000} {
			input := make([]uint32, n)
			for i := range input {
				input[i] = 0x10ffff
			}
			if n > 0 && invalid != 0 {
				input[n/2] = invalid
			}
			if got, want := validateUTF32NEON(input), validateUTF32Scalar(input); got != want {
				t.Fatalf("n=%d invalid=%x: bool=%t want=%t", n, invalid, got, want)
			}
			if got, want := validateUTF32WithErrorsNEON(input), validateUTF32WithErrorsScalar(input); got != want {
				t.Fatalf("n=%d invalid=%x: result=%+v want=%+v", n, invalid, got, want)
			}
		}
	}
}

func FuzzUTFValidationNEONAgainstScalar(f *testing.F) {
	for _, seed := range [][]byte{nil, {0, 0}, {0, 0xd8, 0, 0xdc}, {0, 0, 0x11, 0}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		semantic := fuzzUTF16Words(data)
		for _, little := range []bool{true, false} {
			input := rawUTF16Scalar(semantic, little)
			var gotResult, wantResult Result
			gotDst, wantDst := make([]uint16, len(input)), make([]uint16, len(input))
			if little {
				gotResult = validateUTF16LEWithErrorsNEON(input)
				wantResult = validateUTF16LEWithErrorsScalar(input)
				toWellFormedUTF16LENEON(input, gotDst)
				toWellFormedUTF16LEScalar(input, wantDst)
			} else {
				gotResult = validateUTF16BEWithErrorsNEON(input)
				wantResult = validateUTF16BEWithErrorsScalar(input)
				toWellFormedUTF16BENEON(input, gotDst)
				toWellFormedUTF16BEScalar(input, wantDst)
			}
			if gotResult != wantResult || !slices.Equal(gotDst, wantDst) {
				t.Fatalf("UTF-16 little=%t: result=%+v/%+v output=%x/%x", little, gotResult, wantResult, gotDst, wantDst)
			}
		}

		input32 := make([]uint32, len(data)/4)
		for i := range input32 {
			input32[i] = uint32(data[4*i]) | uint32(data[4*i+1])<<8 |
				uint32(data[4*i+2])<<16 | uint32(data[4*i+3])<<24
		}
		if got, want := validateUTF32WithErrorsNEON(input32), validateUTF32WithErrorsScalar(input32); got != want {
			t.Fatalf("UTF-32 result=%+v want=%+v input=%x", got, want, input32)
		}
	})
}

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

//go:build amd64

package simdutf

import (
	"bytes"
	"math/bits"
	"reflect"
	"testing"
	"unicode/utf8"
)

func TestBase64AMD64EncodeMatchesScalar(t *testing.T) {
	inputs := [][]byte{
		[]byte("Hello, simdutf Base64!"),
		make([]byte, 48),
		make([]byte, 96),
		bytes.Repeat([]byte("0123456789abcdef"), 8),
	}
	for i := range inputs[1] {
		inputs[1][i] = byte(i)
	}
	for i := range inputs[2] {
		inputs[2][i] = byte(i)
	}
	for _, opt := range []Base64Options{Base64Default, Base64URL} {
		for _, in := range inputs {
			dstW := make([]byte, base64LengthFromBinaryScalar(len(in), opt))
			dstH := make([]byte, len(dstW))
			dstS := make([]byte, len(dstW))
			nW := binaryToBase64Westmere(in, dstW, opt)
			nH := binaryToBase64Haswell(in, dstH, opt)
			nS := binaryToBase64Scalar(in, dstS, opt)
			if nW != nS || !bytes.Equal(dstW[:nW], dstS[:nS]) {
				t.Fatalf("westmere opt=%v len=%d got %q want %q", opt, len(in), dstW[:min(nW, 96)], dstS[:min(nS, 96)])
			}
			if nH != nS || !bytes.Equal(dstH[:nH], dstS[:nS]) {
				t.Fatalf("haswell opt=%v len=%d got %q want %q", opt, len(in), dstH[:min(nH, 96)], dstS[:min(nS, 96)])
			}
		}
	}
}

func TestBase64AMD64LengthMatchesScalar(t *testing.T) {
	inputs := [][]byte{
		[]byte("AQID"),
		bytes.Repeat([]byte("A"), 128),
		append(bytes.Repeat([]byte("A"), 100), '=', '=', '\n'),
		{},
	}
	for _, in := range inputs {
		want := binaryLengthFromBase64Scalar(in)
		if got := binaryLengthFromBase64Westmere(in); got != want {
			t.Fatalf("westmere len=%d scalar=%d input=%q", got, want, in)
		}
		if got := binaryLengthFromBase64Haswell(in); got != want {
			t.Fatalf("haswell len=%d scalar=%d input=%q", got, want, in)
		}
	}
}

func TestBase64AMD64DecodeMatchesScalar(t *testing.T) {
	raw := bytes.Repeat([]byte("Hello AMD64 Base64 decode path!!"), 6)
	for _, opt := range []Base64Options{Base64Default, Base64URL} {
		enc := make([]byte, base64LengthFromBinaryScalar(len(raw), opt))
		n := binaryToBase64Scalar(raw, enc, opt)
		enc = enc[:n]
		dstW := make([]byte, maximalBinaryLengthFromBase64Scalar(enc))
		dstH := make([]byte, len(dstW))
		dstS := make([]byte, len(dstW))
		rW := base64ToBinaryDetailsWestmere(enc, dstW, opt, Loose)
		rH := base64ToBinaryDetailsHaswell(enc, dstH, opt, Loose)
		rS := base64ToBinaryDetailsScalar(enc, dstS, opt, Loose)
		if rW != rS || !bytes.Equal(dstW[:rW.OutputCount], dstS[:rS.OutputCount]) {
			t.Fatalf("westmere opt=%v got=%+v want=%+v", opt, rW, rS)
		}
		if rH != rS || !bytes.Equal(dstH[:rH.OutputCount], dstS[:rS.OutputCount]) {
			t.Fatalf("haswell opt=%v got=%+v want=%+v", opt, rH, rS)
		}
	}
}

func TestBase64AMD64DecodeBlocksDirect(t *testing.T) {
	raw := bytes.Repeat([]byte("0123456789abcdef"), 12) // 192 bytes -> 256 base64 chars
	enc := make([]byte, base64LengthFromBinaryScalar(len(raw), Base64Default))
	n := binaryToBase64Scalar(raw, enc, Base64Default)
	enc = enc[:n]
	if len(enc) < 64 || len(enc)%64 != 0 {
		t.Fatalf("expected multiple of 64, got %d", len(enc))
	}
	toBase64 := base64ToValueTable(Base64Default)
	buf := make([]byte, len(enc))
	for i, c := range enc {
		buf[i] = toBase64[c]
	}
	dstW := make([]byte, len(enc)/4*3)
	dstH := make([]byte, len(dstW))
	base64DecodeBlocksWestmere(buf, dstW)
	base64DecodeBlocksHaswell(buf, dstH)
	if !bytes.Equal(dstW, raw) {
		t.Fatalf("westmere blocks mismatch: got %q want %q", dstW[:64], raw[:64])
	}
	if !bytes.Equal(dstH, raw) {
		t.Fatalf("haswell blocks mismatch: got %q want %q", dstH[:64], raw[:64])
	}
}

func TestBase64AMD64DecodeUTF16MatchesScalar(t *testing.T) {
	raw := bytes.Repeat([]byte("abcdef"), 30)
	enc := make([]byte, base64LengthFromBinaryScalar(len(raw), Base64Default))
	n := binaryToBase64Scalar(raw, enc, Base64Default)
	u16 := make([]uint16, n)
	for i := 0; i < n; i++ {
		u16[i] = uint16(enc[i])
	}
	dstW := make([]byte, maximalBinaryLengthFromBase64UTF16Scalar(u16))
	dstH := make([]byte, len(dstW))
	dstS := make([]byte, len(dstW))
	rW := base64ToBinaryDetailsUTF16Westmere(u16, dstW, Base64Default, Loose)
	rH := base64ToBinaryDetailsUTF16Haswell(u16, dstH, Base64Default, Loose)
	rS := base64ToBinaryDetailsUTF16Scalar(u16, dstS, Base64Default, Loose)
	if rW != rS || !bytes.Equal(dstW[:rW.OutputCount], dstS[:rS.OutputCount]) {
		t.Fatalf("westmere utf16 got=%+v want=%+v", rW, rS)
	}
	if rH != rS || !bytes.Equal(dstH[:rH.OutputCount], dstS[:rS.OutputCount]) {
		t.Fatalf("haswell utf16 got=%+v want=%+v", rH, rS)
	}
}

func TestBase64AMD64DecodeFallsBackOnWhitespace(t *testing.T) {
	// Contiguous path rejects ignorable bytes; whitespace must still decode via scalar residual.
	enc := []byte("SGVsbG8sIHdvcmxkIQ==")
	withWS := []byte("SGVs bG8s\nIHdv cmxk IQ==")
	dstW := make([]byte, maximalBinaryLengthFromBase64Scalar(withWS))
	dstS := make([]byte, len(dstW))
	rW := base64ToBinaryDetailsWestmere(withWS, dstW, Base64Default, Loose)
	rS := base64ToBinaryDetailsScalar(withWS, dstS, Base64Default, Loose)
	if rW != rS || !bytes.Equal(dstW[:rW.OutputCount], dstS[:rS.OutputCount]) {
		t.Fatalf("whitespace westmere=%+v scalar=%+v", rW, rS)
	}
	// Sanity: compact form also matches.
	dstC := make([]byte, maximalBinaryLengthFromBase64Scalar(enc))
	rC := base64ToBinaryDetailsWestmere(enc, dstC, Base64Default, Loose)
	if rC.Error != Success || rC.OutputCount != rS.OutputCount {
		t.Fatalf("compact decode failed: %+v", rC)
	}
}

func TestBase64AMD64WithLinesMatchesScalar(t *testing.T) {
	in := bytes.Repeat([]byte("abcdef"), 40)
	for _, line := range []int{76, 64, 16, 4} {
		dstW := make([]byte, base64LengthFromBinaryWithLinesScalar(len(in), Base64Default, line))
		dstH := make([]byte, len(dstW))
		dstS := make([]byte, len(dstW))
		nW := binaryToBase64WithLinesWestmere(in, dstW, line, Base64Default)
		nH := binaryToBase64WithLinesHaswell(in, dstH, line, Base64Default)
		nS := binaryToBase64WithLinesScalar(in, dstS, line, Base64Default)
		if nW != nS || !bytes.Equal(dstW[:nW], dstS[:nS]) {
			t.Fatalf("westmere line=%d got(%d) want(%d)", line, nW, nS)
		}
		if nH != nS || !bytes.Equal(dstH[:nH], dstS[:nS]) {
			t.Fatalf("haswell line=%d got(%d) want(%d)", line, nH, nS)
		}
	}
}

func requireDetectAMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skip("required amd64 SIMD feature is unavailable")
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

func TestDirectAMD64DetectEncodingsAgainstScalar(t *testing.T) {
	variants := []struct {
		name    string
		feature cpuFeatures
		fn      func([]byte) Encoding
	}{
		{name: "westmere", feature: cpuSSSE3, fn: detectEncodingsWestmere},
		{name: "haswell", feature: cpuAVX2, fn: detectEncodingsHaswell},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireDetectAMD64Variant(t, v.feature)
			for _, tc := range detectEncodingsDirectCases() {
				t.Run(tc.name, func(t *testing.T) {
					got := v.fn(tc.input)
					want := detectEncodingsScalar(tc.input)
					if got != want {
						t.Fatalf("%s(%q) = %d, want scalar %d", v.name, tc.input, got, want)
					}
				})
			}
		})
	}
}

// Direct differential coverage for the Westmere/Haswell translations of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/generic/find.h.
// Cases cover empty/nil, first-byte hits, misses, unaligned subslices, and long
// buffers against findScalar/findUTF16Scalar.

func requireFindAMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skipf("missing required CPU features %#x", feature)
	}
}

func TestDirectAMD64FindAgainstScalar(t *testing.T) {
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
		{name: "long-miss", input: bytes.Repeat([]byte{'x'}, 257), value: 'a'},
		{name: "long-late", input: append(bytes.Repeat([]byte{'x'}, 200), 'a'), value: 'a'},
	}
	variants := []struct {
		name    string
		feature cpuFeatures
		fn      func([]byte, byte) int
	}{
		{name: "westmere", feature: cpuSSSE3, fn: findWestmere},
		{name: "haswell", feature: cpuAVX2, fn: findHaswell},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireFindAMD64Variant(t, v.feature)
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					want := findScalar(tc.input, tc.value)
					if got := v.fn(tc.input, tc.value); got != want {
						t.Fatalf("%s(%q, %q) = %d, want %d", v.name, tc.input, tc.value, got, want)
					}
				})
			}
		})
	}
}

func TestDirectAMD64FindUTF16AgainstScalar(t *testing.T) {
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
		{name: "long-miss", input: repeat('x', 257), value: 'a'},
		{name: "long-late", input: append(repeat('x', 200), 'a'), value: 'a'},
	}
	variants := []struct {
		name    string
		feature cpuFeatures
		fn      func([]uint16, uint16) int
	}{
		{name: "westmere", feature: cpuSSSE3, fn: findUTF16Westmere},
		{name: "haswell", feature: cpuAVX2, fn: findUTF16Haswell},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireFindAMD64Variant(t, v.feature)
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					want := findUTF16Scalar(tc.input, tc.value)
					if got := v.fn(tc.input, tc.value); got != want {
						t.Fatalf("%s(%v, %#x) = %d, want %d", v.name, tc.input, tc.value, got, want)
					}
				})
			}
		})
	}
}

func TestDirectAMD64FindUnalignedSubslice(t *testing.T) {
	buf := bytes.Repeat([]byte{'x'}, 96)
	buf[17] = 'a'
	buf[50] = 'a'
	u16 := make([]uint16, 96)
	for i := range u16 {
		u16[i] = 'x'
	}
	u16[17] = 'a'
	u16[50] = 'a'

	variants := []struct {
		name    string
		feature cpuFeatures
		find    func([]byte, byte) int
		find16  func([]uint16, uint16) int
	}{
		{name: "westmere", feature: cpuSSSE3, find: findWestmere, find16: findUTF16Westmere},
		{name: "haswell", feature: cpuAVX2, find: findHaswell, find16: findUTF16Haswell},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireFindAMD64Variant(t, v.feature)
			for off := 1; off < 16; off++ {
				input := buf[off:80]
				want := findScalar(input, 'a')
				if got := v.find(input, 'a'); got != want {
					t.Fatalf("off=%d find = %d, want %d", off, got, want)
				}
				if got := v.find(input, 'z'); got != findScalar(input, 'z') {
					t.Fatalf("off=%d miss mismatch", off)
				}
			}
			for off := 1; off < 8; off++ {
				input := u16[off:80]
				want := findUTF16Scalar(input, 'a')
				if got := v.find16(input, 'a'); got != want {
					t.Fatalf("u16 off=%d findUTF16 = %d, want %d", off, got, want)
				}
			}
		})
	}
}

func requireLatin1AMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skip("required amd64 SIMD feature is unavailable")
	}
}

func TestDirectAMD64Latin1AgainstScalar(t *testing.T) {
	input := bytes.Repeat([]byte{0x00, 0x7f, 0x80, 0xff, 'A'}, 41)
	variants := []struct {
		name    string
		feature cpuFeatures
		utf8    func([]byte, []byte) int
		utf16le func([]byte, []uint16) int
		utf16be func([]byte, []uint16) int
		utf32   func([]byte, []uint32) int
		length  func([]byte) int
	}{
		{"westmere", cpuSSSE3, convertLatin1ToUTF8Westmere, convertLatin1ToUTF16LEWestmere, convertLatin1ToUTF16BEWestmere, convertLatin1ToUTF32Westmere, utf8LengthFromLatin1Westmere},
		{"haswell", cpuAVX2, convertLatin1ToUTF8Haswell, convertLatin1ToUTF16LEHaswell, convertLatin1ToUTF16BEHaswell, convertLatin1ToUTF32Haswell, utf8LengthFromLatin1Haswell},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireLatin1AMD64Variant(t, v.feature)
			want8 := make([]byte, utf8LengthFromLatin1Scalar(input))
			convertLatin1ToUTF8Scalar(input, want8)
			got8 := bytes.Repeat([]byte{0xa5}, len(want8)+16)
			n := v.utf8(input, got8)
			if n != len(want8) || !bytes.Equal(got8[:n], want8) || !bytes.Equal(got8[n:], bytes.Repeat([]byte{0xa5}, 16)) {
				t.Fatal("UTF-8 mismatch or canary overwrite")
			}
			want16 := make([]uint16, len(input))
			convertLatin1ToUTF16LEScalar(input, want16)
			got16 := make([]uint16, len(input)+8)
			for i := len(input); i < len(got16); i++ {
				got16[i] = 0xa5a5
			}
			if n := v.utf16le(input, got16); n != len(input) || !equalU16(got16[:n], want16) || got16[len(input)] != 0xa5a5 {
				t.Fatal("UTF-16LE mismatch or canary overwrite")
			}
			convertLatin1ToUTF16BEScalar(input, want16)
			if n := v.utf16be(input, got16); n != len(input) || !equalU16(got16[:n], want16) || got16[len(input)] != 0xa5a5 {
				t.Fatal("UTF-16BE mismatch or canary overwrite")
			}
			want32 := make([]uint32, len(input))
			convertLatin1ToUTF32Scalar(input, want32)
			got32 := make([]uint32, len(input)+8)
			for i := len(input); i < len(got32); i++ {
				got32[i] = 0xa5a5a5a5
			}
			if n := v.utf32(input, got32); n != len(input) || !equalU32(got32[:n], want32) || got32[len(input)] != 0xa5a5a5a5 {
				t.Fatal("UTF-32 mismatch or canary overwrite")
			}
			if got, want := v.length(input), len(want8); got != want {
				t.Fatalf("UTF-8 length = %d, want %d", got, want)
			}
		})
	}
}

func TestDirectAMD64Latin1PreflightPreservesDestination(t *testing.T) {
	input := bytes.Repeat([]byte{0xff}, 65)
	variants := []struct {
		name    string
		feature cpuFeatures
		utf8    func([]byte, []byte) int
		utf16le func([]byte, []uint16) int
		utf16be func([]byte, []uint16) int
		utf32   func([]byte, []uint32) int
	}{
		{"westmere", cpuSSSE3, convertLatin1ToUTF8Westmere, convertLatin1ToUTF16LEWestmere, convertLatin1ToUTF16BEWestmere, convertLatin1ToUTF32Westmere},
		{"haswell", cpuAVX2, convertLatin1ToUTF8Haswell, convertLatin1ToUTF16LEHaswell, convertLatin1ToUTF16BEHaswell, convertLatin1ToUTF32Haswell},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireLatin1AMD64Variant(t, v.feature)
			dst8 := bytes.Repeat([]byte{0xa5}, 2*len(input)-1)
			requireLatin1AMD64Panic(t, func() { v.utf8(input, dst8) })
			if !bytes.Equal(dst8, bytes.Repeat([]byte{0xa5}, len(dst8))) {
				t.Fatal("UTF-8 destination changed before short-destination panic")
			}
			for name, convert := range map[string]func([]byte, []uint16) int{"UTF-16LE": v.utf16le, "UTF-16BE": v.utf16be} {
				dst := make([]uint16, len(input)-1)
				for i := range dst {
					dst[i] = 0xa5a5
				}
				requireLatin1AMD64Panic(t, func() { convert(input, dst) })
				for _, value := range dst {
					if value != 0xa5a5 {
						t.Fatalf("%s destination changed before short-destination panic", name)
					}
				}
			}
			dst32 := make([]uint32, len(input)-1)
			for i := range dst32 {
				dst32[i] = 0xa5a5a5a5
			}
			requireLatin1AMD64Panic(t, func() { v.utf32(input, dst32) })
			for _, value := range dst32 {
				if value != 0xa5a5a5a5 {
					t.Fatal("UTF-32 destination changed before short-destination panic")
				}
			}
		})
	}
}

func requireLatin1AMD64Panic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func FuzzLatin1AMD64AgainstScalar(f *testing.F) {
	f.Add([]byte{0, 0x7f, 0x80, 0xff})
	f.Fuzz(func(t *testing.T, input []byte) {
		if detectAMD64Features()&cpuSSSE3 == cpuSSSE3 {
			checkLatin1Direct(t, input, convertLatin1ToUTF8Westmere, convertLatin1ToUTF16LEWestmere, convertLatin1ToUTF16BEWestmere, convertLatin1ToUTF32Westmere, utf8LengthFromLatin1Westmere)
		}
		if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
			checkLatin1Direct(t, input, convertLatin1ToUTF8Haswell, convertLatin1ToUTF16LEHaswell, convertLatin1ToUTF16BEHaswell, convertLatin1ToUTF32Haswell, utf8LengthFromLatin1Haswell)
		}
	})
}

func checkLatin1Direct(t *testing.T, input []byte, to8 func([]byte, []byte) int, to16le, to16be func([]byte, []uint16) int, to32 func([]byte, []uint32) int, length func([]byte) int) {
	want8 := make([]byte, utf8LengthFromLatin1Scalar(input))
	convertLatin1ToUTF8Scalar(input, want8)
	got8 := make([]byte, len(want8))
	if to8(input, got8) != len(want8) || !bytes.Equal(got8, want8) || length(input) != len(want8) {
		t.Fatal("UTF-8 differential mismatch")
	}
	want16, got16 := make([]uint16, len(input)), make([]uint16, len(input))
	convertLatin1ToUTF16LEScalar(input, want16)
	to16le(input, got16)
	if !equalU16(got16, want16) {
		t.Fatal("UTF-16LE differential mismatch")
	}
	convertLatin1ToUTF16BEScalar(input, want16)
	to16be(input, got16)
	if !equalU16(got16, want16) {
		t.Fatal("UTF-16BE differential mismatch")
	}
	want32, got32 := make([]uint32, len(input)), make([]uint32, len(input))
	convertLatin1ToUTF32Scalar(input, want32)
	to32(input, got32)
	if !equalU32(got32, want32) {
		t.Fatal("UTF-32 differential mismatch")
	}
}

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

func requireUTF16HelpersAMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skipf("missing required CPU features %#x", feature)
	}
}

func TestDirectAMD64UTF16HelpersAgainstScalar(t *testing.T) {
	cases := [][]uint16{
		{},
		{0},
		{0x61, 0x62, 0x63},
		{0x00ff, 0xff00, 0x1234, 0xd800, 0xdc00, 0xd83d, 0xde00},
		{0xd800}, // lone high
		{0xdc00}, // lone low
		make([]uint16, 31),
		make([]uint16, 32),
		make([]uint16, 33),
		make([]uint16, 64),
		make([]uint16, 65),
	}
	for i := range cases[6] {
		cases[6][i] = uint16(i * 3)
		cases[7][i%32] = uint16(0x100 + i)
	}
	for i := range cases[8] {
		cases[8][i] = uint16(0xd800 + (i % 0x400))
	}
	for i := range cases[9] {
		if i%5 == 0 {
			cases[9][i] = 0xd83d
		} else if i%5 == 1 {
			cases[9][i] = 0xde00
		} else {
			cases[9][i] = uint16(0x4e00 + i)
		}
	}
	for i := range cases[10] {
		cases[10][i] = uint16(i)
	}

	requireUTF16HelpersAMD64Variant(t, cpuSSSE3)
	for _, input := range cases {
		checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Westmere, countUTF16LEWestmere, countUTF16BEWestmere, utf32LengthFromUTF16LEWestmere, utf32LengthFromUTF16BEWestmere)
	}
	if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
		for _, input := range cases {
			checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Haswell, countUTF16LEHaswell, countUTF16BEHaswell, utf32LengthFromUTF16LEHaswell, utf32LengthFromUTF16BEHaswell)
		}
	}
}

func TestDirectAMD64UTF16HelpersPreflightPreservesDestination(t *testing.T) {
	requireUTF16HelpersAMD64Variant(t, cpuSSSE3)
	input := []uint16{1, 2, 3, 4}
	dst := []uint16{9, 9}
	requireUTF16HelpersPanic(t, func() { changeEndiannessUTF16Westmere(input, dst) })
	if dst[0] != 9 || dst[1] != 9 {
		t.Fatal("westmere preflight mutated short destination")
	}
	if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
		dst = []uint16{9, 9}
		requireUTF16HelpersPanic(t, func() { changeEndiannessUTF16Haswell(input, dst) })
		if dst[0] != 9 || dst[1] != 9 {
			t.Fatal("haswell preflight mutated short destination")
		}
	}
}

func requireUTF16HelpersPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func checkUTF16HelpersDirect(
	t *testing.T,
	input []uint16,
	change func([]uint16, []uint16),
	countLE, countBE, lenLE, lenBE func([]uint16) int,
) {
	t.Helper()
	want := make([]uint16, len(input))
	got := make([]uint16, len(input))
	changeEndiannessUTF16Scalar(input, want)
	change(input, got)
	if !equalU16(got, want) {
		t.Fatalf("changeEndianness mismatch input=%v want=%v got=%v", input, want, got)
	}
	if countLE(input) != countUTF16LEScalar(input) {
		t.Fatalf("countLE mismatch input=%v", input)
	}
	if countBE(input) != countUTF16BEScalar(input) {
		t.Fatalf("countBE mismatch input=%v", input)
	}
	if lenLE(input) != utf32LengthFromUTF16LEScalar(input) {
		t.Fatalf("utf32LengthLE mismatch input=%v", input)
	}
	if lenBE(input) != utf32LengthFromUTF16BEScalar(input) {
		t.Fatalf("utf32LengthBE mismatch input=%v", input)
	}
}

func FuzzUTF16HelpersAMD64AgainstScalar(f *testing.F) {
	f.Add([]byte{0, 0x7f, 0xd8, 0x00, 0xdc, 0x00, 0xff, 0xff})
	f.Fuzz(func(t *testing.T, raw []byte) {
		input := make([]uint16, len(raw)/2)
		for i := range input {
			input[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		if detectAMD64Features()&cpuSSSE3 == cpuSSSE3 {
			checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Westmere, countUTF16LEWestmere, countUTF16BEWestmere, utf32LengthFromUTF16LEWestmere, utf32LengthFromUTF16BEWestmere)
		}
		if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
			checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Haswell, countUTF16LEHaswell, countUTF16BEHaswell, utf32LengthFromUTF16LEHaswell, utf32LengthFromUTF16BEHaswell)
		}
	})
}

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the amd64 UTF-16→Latin-1 Westmere/Haswell
// translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_latin1.cpp
// and src/haswell/avx2_convert_utf16_to_latin1.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// Providers are invoked directly; feature skips only gate unavailable ISA.

func requireUTF16Latin1AMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skipf("missing required CPU features %#x", feature)
	}
}

func TestDirectAMD64UTF16Latin1AgainstScalar(t *testing.T) {
	natives := [][]uint16{
		nil,
		{},
		{0},
		{'a', 'b', 'c'},
		{0x00, 0x7f, 0x80, 0xff},
		{0x100},
		{'a', 0x100, 'b'},
		{0xd800},
		{0xdc00},
		{0xd83d, 0xde00},
		repeatU16Latin1('A', 7),
		repeatU16Latin1('A', 8),
		repeatU16Latin1('A', 9),
		repeatU16Latin1('A', 15),
		repeatU16Latin1('A', 16),
		repeatU16Latin1('A', 17),
		append(repeatU16Latin1('A', 7), 0xff),
		append(repeatU16Latin1(0xff, 8), 0x7f),
		append(repeatU16Latin1('A', 8), 0x100),
		append(repeatU16Latin1('A', 16), 0x100, 'b'),
		append(repeatU16Latin1('A', 16), 0x20ac),
		repeatU16Latin1(0x00ff, 65),
		repeatU16Latin1(0x0041, 129),
	}

	variants := []struct {
		name    string
		feature cpuFeatures
		le      func([]uint16, []byte) int
		be      func([]uint16, []byte) int
		leErr   func([]uint16, []byte) Result
		beErr   func([]uint16, []byte) Result
		leValid func([]uint16, []byte) int
		beValid func([]uint16, []byte) int
	}{
		{
			name:    "westmere",
			feature: cpuSSSE3,
			le:      convertUTF16LEToLatin1Westmere,
			be:      convertUTF16BEToLatin1Westmere,
			leErr:   convertUTF16LEToLatin1WithErrorsWestmere,
			beErr:   convertUTF16BEToLatin1WithErrorsWestmere,
			leValid: convertValidUTF16LEToLatin1Westmere,
			beValid: convertValidUTF16BEToLatin1Westmere,
		},
		{
			name:    "haswell",
			feature: cpuAVX2,
			le:      convertUTF16LEToLatin1Haswell,
			be:      convertUTF16BEToLatin1Haswell,
			leErr:   convertUTF16LEToLatin1WithErrorsHaswell,
			beErr:   convertUTF16BEToLatin1WithErrorsHaswell,
			leValid: convertValidUTF16LEToLatin1Haswell,
			beValid: convertValidUTF16BEToLatin1Haswell,
		},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF16Latin1AMD64Variant(t, v.feature)
			for _, native := range natives {
				for _, little := range []bool{true, false} {
					input := rawUTF16Words(native, little)
					if little {
						checkUTF16Latin1Direct(t, input, true, v.le, v.leErr, v.leValid)
					} else {
						checkUTF16Latin1Direct(t, input, false, v.be, v.beErr, v.beValid)
					}
				}
			}
		})
	}
}

func TestDirectAMD64UTF16Latin1PreflightPreservesDestination(t *testing.T) {
	native := append(repeatU16Latin1('A', 15), 0xff)
	variants := []struct {
		name    string
		feature cpuFeatures
		le      func([]uint16, []byte) int
		be      func([]uint16, []byte) int
		leErr   func([]uint16, []byte) Result
		beErr   func([]uint16, []byte) Result
		leValid func([]uint16, []byte) int
		beValid func([]uint16, []byte) int
	}{
		{
			name: "westmere", feature: cpuSSSE3,
			le: convertUTF16LEToLatin1Westmere, be: convertUTF16BEToLatin1Westmere,
			leErr: convertUTF16LEToLatin1WithErrorsWestmere, beErr: convertUTF16BEToLatin1WithErrorsWestmere,
			leValid: convertValidUTF16LEToLatin1Westmere, beValid: convertValidUTF16BEToLatin1Westmere,
		},
		{
			name: "haswell", feature: cpuAVX2,
			le: convertUTF16LEToLatin1Haswell, be: convertUTF16BEToLatin1Haswell,
			leErr: convertUTF16LEToLatin1WithErrorsHaswell, beErr: convertUTF16BEToLatin1WithErrorsHaswell,
			leValid: convertValidUTF16LEToLatin1Haswell, beValid: convertValidUTF16BEToLatin1Haswell,
		},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF16Latin1AMD64Variant(t, v.feature)
			for _, little := range []bool{true, false} {
				input := rawUTF16Words(native, little)
				dst := guardedLatin1Destination[byte](len(input)-1, 0xa5)
				convert, convertErr, convertValid := v.le, v.leErr, v.leValid
				if !little {
					convert, convertErr, convertValid = v.be, v.beErr, v.beValid
				}
				requireUTF16Latin1AMD64Panic(t, func() { convert(input, dst.body) })
				dst.require(t)
				requireUTF16Latin1AMD64Panic(t, func() { convertErr(input, dst.body) })
				dst.require(t)
				requireUTF16Latin1AMD64Panic(t, func() { convertValid(input, dst.body) })
				dst.require(t)
			}
		})
	}
}

func requireUTF16Latin1AMD64Panic(t *testing.T, fn func()) {
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

func FuzzUTF16Latin1AMD64AgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(utf16Latin1AMD64NativeBytes(repeatU16Latin1('A', 8)))
	f.Add(utf16Latin1AMD64NativeBytes(repeatU16Latin1(0xff, 16)))
	f.Add(utf16Latin1AMD64NativeBytes([]uint16{'a', 0x100, 'b'}))
	f.Add(utf16Latin1AMD64NativeBytes(append(repeatU16Latin1('A', 16), 0x100)))
	f.Add([]byte{0x00, 0x00, 0x7f, 0x00, 0xff, 0x00, 0x00, 0x01})
	f.Add([]byte{0x41, 0x00, 0x00, 0xd8})
	f.Fuzz(func(t *testing.T, raw []byte) {
		native := utf16Latin1AMD64NativeFromBytes(raw)
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			if detectAMD64Features()&cpuSSSE3 == cpuSSSE3 {
				if little {
					checkUTF16Latin1Direct(t, input, true, convertUTF16LEToLatin1Westmere, convertUTF16LEToLatin1WithErrorsWestmere, convertValidUTF16LEToLatin1Westmere)
				} else {
					checkUTF16Latin1Direct(t, input, false, convertUTF16BEToLatin1Westmere, convertUTF16BEToLatin1WithErrorsWestmere, convertValidUTF16BEToLatin1Westmere)
				}
			}
			if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
				if little {
					checkUTF16Latin1Direct(t, input, true, convertUTF16LEToLatin1Haswell, convertUTF16LEToLatin1WithErrorsHaswell, convertValidUTF16LEToLatin1Haswell)
				} else {
					checkUTF16Latin1Direct(t, input, false, convertUTF16BEToLatin1Haswell, convertUTF16BEToLatin1WithErrorsHaswell, convertValidUTF16BEToLatin1Haswell)
				}
			}
		}
	})
}

func utf16Latin1AMD64NativeBytes(words []uint16) []byte {
	out := make([]byte, len(words)*2)
	for i, word := range words {
		out[2*i] = byte(word)
		out[2*i+1] = byte(word >> 8)
	}
	return out
}

func utf16Latin1AMD64NativeFromBytes(raw []byte) []uint16 {
	n := len(raw) / 2
	out := make([]uint16, n)
	for i := 0; i < n; i++ {
		out[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
	}
	return out
}

func checkUTF16Latin1Direct(t *testing.T, input []uint16, little bool, convert func([]uint16, []byte) int, convertErr func([]uint16, []byte) Result, convertValid func([]uint16, []byte) int) {
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
		gotN = convert(input, got.body)
	} else {
		wantN = convertUTF16BEToLatin1Scalar(input, want)
		gotN = convert(input, got.body)
	}
	if gotN != wantN || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t convert = %d/%x, want %d/%x", little, gotN, got.body, wantN, want)
	}
	got.require(t)

	want = bytes.Repeat([]byte{0xa5}, len(input))
	got = guardedLatin1Destination[byte](len(input), 0xa5)
	if little {
		wantE = convertUTF16LEToLatin1WithErrorsScalar(input, want)
		gotE = convertErr(input, got.body)
	} else {
		wantE = convertUTF16BEToLatin1WithErrorsScalar(input, want)
		gotE = convertErr(input, got.body)
	}
	if gotE != wantE || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t with_errors = %#v/%x, want %#v/%x", little, gotE, got.body, wantE, want)
	}
	got.require(t)

	// Valid assumes latin1-only input and packs every low byte (unlike convert's 0-on-error).
	want = bytes.Repeat([]byte{0xa5}, len(input))
	got = guardedLatin1Destination[byte](len(input), 0xa5)
	if little {
		wantV = convertValidUTF16LEToLatin1Scalar(input, want)
		gotV = convertValid(input, got.body)
	} else {
		wantV = convertValidUTF16BEToLatin1Scalar(input, want)
		gotV = convertValid(input, got.body)
	}
	if gotV != wantV || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t valid = %d/%x, want %d/%x", little, gotV, got.body, wantV, want)
	}
	got.require(t)
}

func repeatU16Latin1(value uint16, n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		out[i] = value
	}
	return out
}

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the amd64 UTF-16→UTF-32 Westmere/Haswell
// translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_utf32.cpp
// and src/haswell/avx2_convert_utf16_to_utf32.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// Providers are invoked directly; feature skips only gate unavailable ISA.

func requireUTF16UTF32AMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skipf("missing required CPU features %#x", feature)
	}
}

func TestDirectAMD64UTF16UTF32AgainstScalar(t *testing.T) {
	natives := [][]uint16{
		nil,
		{},
		{0},
		{'a', 'b', 'c'},
		{0x00, 0x7f, 0x80, 0xff},
		{0x100},
		{0x7ff, 0x800, 0xffff},
		{'a', 0x100, 'b'},
		{0xd800},
		{0xdc00},
		{0xd83d, 0xde00},
		{0xd800, 0xdc00},
		{0xdbff, 0xdfff},
		{0xd800, 'a'},
		{'a', 0xdc00},
		repeatU16UTF32('A', 7),
		repeatU16UTF32('A', 8),
		repeatU16UTF32('A', 9),
		repeatU16UTF32('A', 15),
		repeatU16UTF32('A', 16),
		repeatU16UTF32('A', 17),
		append(repeatU16UTF32('A', 7), 0xff),
		append(repeatU16UTF32(0xff, 8), 0x7f),
		append(repeatU16UTF32('A', 8), 0x100),
		append(repeatU16UTF32('A', 8), 0xd83d, 0xde00),
		append(repeatU16UTF32('A', 16), 0x100, 'b'),
		append(repeatU16UTF32('A', 16), 0x20ac),
		append(repeatU16UTF32('A', 16), 0xd83d, 0xde00, 'z'),
		append(repeatU16UTF32('A', 15), 0xd800),
		append(repeatU16UTF32('A', 15), 0xd83d, 0xde00),
		repeatU16UTF32(0x00ff, 65),
		repeatU16UTF32(0x0041, 129),
		repeatU16UTF32(0x20ac, 33),
	}

	variants := []struct {
		name    string
		feature cpuFeatures
		le      func([]uint16, []uint32) int
		be      func([]uint16, []uint32) int
		leErr   func([]uint16, []uint32) Result
		beErr   func([]uint16, []uint32) Result
		leValid func([]uint16, []uint32) int
		beValid func([]uint16, []uint32) int
	}{
		{
			name:    "westmere",
			feature: cpuSSSE3,
			le:      convertUTF16LEToUTF32Westmere,
			be:      convertUTF16BEToUTF32Westmere,
			leErr:   convertUTF16LEToUTF32WithErrorsWestmere,
			beErr:   convertUTF16BEToUTF32WithErrorsWestmere,
			leValid: convertValidUTF16LEToUTF32Westmere,
			beValid: convertValidUTF16BEToUTF32Westmere,
		},
		{
			name:    "haswell",
			feature: cpuAVX2,
			le:      convertUTF16LEToUTF32Haswell,
			be:      convertUTF16BEToUTF32Haswell,
			leErr:   convertUTF16LEToUTF32WithErrorsHaswell,
			beErr:   convertUTF16BEToUTF32WithErrorsHaswell,
			leValid: convertValidUTF16LEToUTF32Haswell,
			beValid: convertValidUTF16BEToUTF32Haswell,
		},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF16UTF32AMD64Variant(t, v.feature)
			for _, native := range natives {
				for _, little := range []bool{true, false} {
					input := rawUTF16Words(native, little)
					if little {
						checkUTF16UTF32Direct(t, input, true, v.le, v.leErr, v.leValid)
					} else {
						checkUTF16UTF32Direct(t, input, false, v.be, v.beErr, v.beValid)
					}
				}
			}
		})
	}
}

func TestDirectAMD64UTF16UTF32PreflightPreservesDestination(t *testing.T) {
	native := append(repeatU16UTF32('A', 15), 0xff, 0xd83d, 0xde00)
	variants := []struct {
		name    string
		feature cpuFeatures
		le      func([]uint16, []uint32) int
		be      func([]uint16, []uint32) int
		leErr   func([]uint16, []uint32) Result
		beErr   func([]uint16, []uint32) Result
		leValid func([]uint16, []uint32) int
		beValid func([]uint16, []uint32) int
	}{
		{
			name: "westmere", feature: cpuSSSE3,
			le: convertUTF16LEToUTF32Westmere, be: convertUTF16BEToUTF32Westmere,
			leErr: convertUTF16LEToUTF32WithErrorsWestmere, beErr: convertUTF16BEToUTF32WithErrorsWestmere,
			leValid: convertValidUTF16LEToUTF32Westmere, beValid: convertValidUTF16BEToUTF32Westmere,
		},
		{
			name: "haswell", feature: cpuAVX2,
			le: convertUTF16LEToUTF32Haswell, be: convertUTF16BEToUTF32Haswell,
			leErr: convertUTF16LEToUTF32WithErrorsHaswell, beErr: convertUTF16BEToUTF32WithErrorsHaswell,
			leValid: convertValidUTF16LEToUTF32Haswell, beValid: convertValidUTF16BEToUTF32Haswell,
		},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF16UTF32AMD64Variant(t, v.feature)
			for _, little := range []bool{true, false} {
				input := rawUTF16Words(native, little)
				need := utf32LengthFromUTF16LEScalar(input)
				if !little {
					need = utf32LengthFromUTF16BEScalar(input)
				}
				if need == 0 {
					t.Fatal("preflight fixture produced empty required length")
				}
				dst := guardedLatin1Destination[uint32](need-1, 0xa5a5a5a5)
				convert, convertErr, convertValid := v.le, v.leErr, v.leValid
				if !little {
					convert, convertErr, convertValid = v.be, v.beErr, v.beValid
				}
				requireUTF16UTF32AMD64Panic(t, func() { convert(input, dst.body) })
				dst.require(t)
				requireUTF16UTF32AMD64Panic(t, func() { convertErr(input, dst.body) })
				dst.require(t)
				requireUTF16UTF32AMD64Panic(t, func() { convertValid(input, dst.body) })
				dst.require(t)
			}
		})
	}
}

func requireUTF16UTF32AMD64Panic(t *testing.T, fn func()) {
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

func FuzzUTF16UTF32AMD64AgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(utf16UTF32AMD64NativeBytes(repeatU16UTF32('A', 8)))
	f.Add(utf16UTF32AMD64NativeBytes(repeatU16UTF32(0xff, 16)))
	f.Add(utf16UTF32AMD64NativeBytes([]uint16{'a', 0x100, 'b'}))
	f.Add(utf16UTF32AMD64NativeBytes(append(repeatU16UTF32('A', 16), 0x100)))
	f.Add(utf16UTF32AMD64NativeBytes(append(repeatU16UTF32('A', 8), 0xd83d, 0xde00)))
	f.Add(utf16UTF32AMD64NativeBytes([]uint16{0xd800}))
	f.Add(utf16UTF32AMD64NativeBytes([]uint16{0xdc00}))
	f.Add([]byte{0x00, 0x00, 0x7f, 0x00, 0xff, 0x00, 0x00, 0x01})
	f.Add([]byte{0x41, 0x00, 0x00, 0xd8})
	f.Fuzz(func(t *testing.T, raw []byte) {
		native := utf16UTF32AMD64NativeFromBytes(raw)
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			if detectAMD64Features()&cpuSSSE3 == cpuSSSE3 {
				if little {
					checkUTF16UTF32Direct(t, input, true, convertUTF16LEToUTF32Westmere, convertUTF16LEToUTF32WithErrorsWestmere, convertValidUTF16LEToUTF32Westmere)
				} else {
					checkUTF16UTF32Direct(t, input, false, convertUTF16BEToUTF32Westmere, convertUTF16BEToUTF32WithErrorsWestmere, convertValidUTF16BEToUTF32Westmere)
				}
			}
			if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
				if little {
					checkUTF16UTF32Direct(t, input, true, convertUTF16LEToUTF32Haswell, convertUTF16LEToUTF32WithErrorsHaswell, convertValidUTF16LEToUTF32Haswell)
				} else {
					checkUTF16UTF32Direct(t, input, false, convertUTF16BEToUTF32Haswell, convertUTF16BEToUTF32WithErrorsHaswell, convertValidUTF16BEToUTF32Haswell)
				}
			}
		}
	})
}

func utf16UTF32AMD64NativeBytes(words []uint16) []byte {
	out := make([]byte, len(words)*2)
	for i, word := range words {
		out[2*i] = byte(word)
		out[2*i+1] = byte(word >> 8)
	}
	return out
}

func utf16UTF32AMD64NativeFromBytes(raw []byte) []uint16 {
	n := len(raw) / 2
	out := make([]uint16, n)
	for i := 0; i < n; i++ {
		out[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
	}
	return out
}

func checkUTF16UTF32Direct(t *testing.T, input []uint16, little bool, convert func([]uint16, []uint32) int, convertErr func([]uint16, []uint32) Result, convertValid func([]uint16, []uint32) int) {
	t.Helper()

	need := utf32LengthFromUTF16LEScalar(input)
	if !little {
		need = utf32LengthFromUTF16BEScalar(input)
	}
	want := repeatU32UTF32(0xa5a5a5a5, need)
	got := guardedLatin1Destination[uint32](need, 0xa5a5a5a5)
	var (
		wantN, gotN int
		wantE, gotE Result
		wantV, gotV int
	)
	if little {
		wantN = convertUTF16LEToUTF32Scalar(input, want)
		gotN = convert(input, got.body)
	} else {
		wantN = convertUTF16BEToUTF32Scalar(input, want)
		gotN = convert(input, got.body)
	}
	if gotN != wantN || !equalU32(got.body, want) {
		t.Fatalf("little=%t convert = %d/%x, want %d/%x", little, gotN, got.body, wantN, want)
	}
	got.require(t)

	want = repeatU32UTF32(0xa5a5a5a5, need)
	got = guardedLatin1Destination[uint32](need, 0xa5a5a5a5)
	if little {
		wantE = convertUTF16LEToUTF32WithErrorsScalar(input, want)
		gotE = convertErr(input, got.body)
	} else {
		wantE = convertUTF16BEToUTF32WithErrorsScalar(input, want)
		gotE = convertErr(input, got.body)
	}
	if gotE != wantE || !equalU32(got.body, want) {
		t.Fatalf("little=%t with_errors = %#v/%x, want %#v/%x", little, gotE, got.body, wantE, want)
	}
	got.require(t)

	// convert_valid_utf16_to_utf32 assumes well-formed input. Skip it when the
	// with_errors oracle already rejected the sequence (lone/mismatched surrogates
	// can make utf32_length_from_utf16 undersize the destination for valid).
	if wantE.Error != Success {
		return
	}
	want = repeatU32UTF32(0xa5a5a5a5, need)
	got = guardedLatin1Destination[uint32](need, 0xa5a5a5a5)
	if little {
		wantV = convertValidUTF16LEToUTF32Scalar(input, want)
		gotV = convertValid(input, got.body)
	} else {
		wantV = convertValidUTF16BEToUTF32Scalar(input, want)
		gotV = convertValid(input, got.body)
	}
	if gotV != wantV || !equalU32(got.body, want) {
		t.Fatalf("little=%t valid = %d/%x, want %d/%x", little, gotV, got.body, wantV, want)
	}
	got.require(t)
}

func repeatU16UTF32(value uint16, n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		out[i] = value
	}
	return out
}

func repeatU32UTF32(value uint32, n int) []uint32 {
	out := make([]uint32, n)
	for i := range out {
		out[i] = value
	}
	return out
}

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the amd64 UTF-16→UTF-8 Westmere/Haswell
// translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_utf8.cpp
// and src/haswell/avx2_convert_utf16_to_utf8.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// Providers are invoked directly; feature skips only gate unavailable ISA.

func requireUTF16UTF8AMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skipf("missing required CPU features %#x", feature)
	}
}

func TestDirectAMD64UTF16UTF8AgainstScalar(t *testing.T) {
	natives := [][]uint16{
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
		{0xd83d, 0xde00},
		{0xd800, 0xdc00},
		{0xdbff, 0xdfff},
		{0xd800, 'a'},
		{'a', 0xdc00},
		{'A', 0x20ac, 0xd83d, 0xde00},
		repeatU16UTF8('A', 7),
		repeatU16UTF8('A', 8),
		repeatU16UTF8('A', 9),
		repeatU16UTF8('A', 15),
		repeatU16UTF8('A', 16),
		repeatU16UTF8('A', 17),
		append(repeatU16UTF8('A', 7), 0xff),
		append(repeatU16UTF8(0xff, 8), 0x7f),
		append(repeatU16UTF8('A', 8), 0x100),
		append(repeatU16UTF8('A', 8), 0xd83d, 0xde00),
		append(repeatU16UTF8('A', 16), 0x100, 'b'),
		append(repeatU16UTF8('A', 16), 0x20ac),
		append(repeatU16UTF8('A', 16), 0xd83d, 0xde00, 'z'),
		append(repeatU16UTF8('A', 15), 0xd800),
		append(repeatU16UTF8('A', 15), 0xd83d, 0xde00),
		repeatU16UTF8(0x00ff, 65),
		repeatU16UTF8(0x0041, 129),
		repeatU16UTF8(0x20ac, 33),
	}

	variants := []struct {
		name      string
		feature   cpuFeatures
		le        func([]uint16, []byte) int
		be        func([]uint16, []byte) int
		leErr     func([]uint16, []byte) Result
		beErr     func([]uint16, []byte) Result
		leReplace func([]uint16, []byte) int
		beReplace func([]uint16, []byte) int
		leValid   func([]uint16, []byte) int
		beValid   func([]uint16, []byte) int
		leLen     func([]uint16) int
		beLen     func([]uint16) int
		leLenR    func([]uint16) Result
		beLenR    func([]uint16) Result
	}{
		{
			name:      "westmere",
			feature:   cpuSSSE3,
			le:        convertUTF16LEToUTF8Westmere,
			be:        convertUTF16BEToUTF8Westmere,
			leErr:     convertUTF16LEToUTF8WithErrorsWestmere,
			beErr:     convertUTF16BEToUTF8WithErrorsWestmere,
			leReplace: convertUTF16LEToUTF8WithReplacementWestmere,
			beReplace: convertUTF16BEToUTF8WithReplacementWestmere,
			leValid:   convertValidUTF16LEToUTF8Westmere,
			beValid:   convertValidUTF16BEToUTF8Westmere,
			leLen:     utf8LengthFromUTF16LEWestmere,
			beLen:     utf8LengthFromUTF16BEWestmere,
			leLenR:    utf8LengthFromUTF16LEWithReplacementWestmere,
			beLenR:    utf8LengthFromUTF16BEWithReplacementWestmere,
		},
		{
			name:      "haswell",
			feature:   cpuAVX2,
			le:        convertUTF16LEToUTF8Haswell,
			be:        convertUTF16BEToUTF8Haswell,
			leErr:     convertUTF16LEToUTF8WithErrorsHaswell,
			beErr:     convertUTF16BEToUTF8WithErrorsHaswell,
			leReplace: convertUTF16LEToUTF8WithReplacementHaswell,
			beReplace: convertUTF16BEToUTF8WithReplacementHaswell,
			leValid:   convertValidUTF16LEToUTF8Haswell,
			beValid:   convertValidUTF16BEToUTF8Haswell,
			leLen:     utf8LengthFromUTF16LEHaswell,
			beLen:     utf8LengthFromUTF16BEHaswell,
			leLenR:    utf8LengthFromUTF16LEWithReplacementHaswell,
			beLenR:    utf8LengthFromUTF16BEWithReplacementHaswell,
		},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF16UTF8AMD64Variant(t, v.feature)
			for _, native := range natives {
				for _, little := range []bool{true, false} {
					input := rawUTF16Words(native, little)
					if little {
						checkUTF16UTF8Direct(t, input, true, v.le, v.leErr, v.leReplace, v.leValid, v.leLen, v.leLenR)
					} else {
						checkUTF16UTF8Direct(t, input, false, v.be, v.beErr, v.beReplace, v.beValid, v.beLen, v.beLenR)
					}
				}
			}
		})
	}
}

func TestDirectAMD64UTF16UTF8PreflightPreservesDestination(t *testing.T) {
	native := append(repeatU16UTF8('A', 15), 0xff, 0xd83d, 0xde00)
	variants := []struct {
		name      string
		feature   cpuFeatures
		le        func([]uint16, []byte) int
		be        func([]uint16, []byte) int
		leErr     func([]uint16, []byte) Result
		beErr     func([]uint16, []byte) Result
		leReplace func([]uint16, []byte) int
		beReplace func([]uint16, []byte) int
		leValid   func([]uint16, []byte) int
		beValid   func([]uint16, []byte) int
	}{
		{
			name: "westmere", feature: cpuSSSE3,
			le: convertUTF16LEToUTF8Westmere, be: convertUTF16BEToUTF8Westmere,
			leErr: convertUTF16LEToUTF8WithErrorsWestmere, beErr: convertUTF16BEToUTF8WithErrorsWestmere,
			leReplace: convertUTF16LEToUTF8WithReplacementWestmere, beReplace: convertUTF16BEToUTF8WithReplacementWestmere,
			leValid: convertValidUTF16LEToUTF8Westmere, beValid: convertValidUTF16BEToUTF8Westmere,
		},
		{
			name: "haswell", feature: cpuAVX2,
			le: convertUTF16LEToUTF8Haswell, be: convertUTF16BEToUTF8Haswell,
			leErr: convertUTF16LEToUTF8WithErrorsHaswell, beErr: convertUTF16BEToUTF8WithErrorsHaswell,
			leReplace: convertUTF16LEToUTF8WithReplacementHaswell, beReplace: convertUTF16BEToUTF8WithReplacementHaswell,
			leValid: convertValidUTF16LEToUTF8Haswell, beValid: convertValidUTF16BEToUTF8Haswell,
		},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF16UTF8AMD64Variant(t, v.feature)
			for _, little := range []bool{true, false} {
				input := rawUTF16Words(native, little)
				need := utf8LengthFromUTF16LEScalar(input)
				needReplace := utf8LengthFromUTF16LEWithReplacementScalar(input)
				if !little {
					need = utf8LengthFromUTF16BEScalar(input)
					needReplace = utf8LengthFromUTF16BEWithReplacementScalar(input)
				}
				if need == 0 || needReplace.Count == 0 {
					t.Fatal("preflight fixture produced empty required length")
				}
				dst := guardedLatin1Destination[byte](need-1, 0xa5)
				replaceDst := guardedLatin1Destination[byte](needReplace.Count-1, 0xa5)
				convert, convertErr, convertReplace, convertValid := v.le, v.leErr, v.leReplace, v.leValid
				if !little {
					convert, convertErr, convertReplace, convertValid = v.be, v.beErr, v.beReplace, v.beValid
				}
				requireUTF16UTF8AMD64Panic(t, func() { convert(input, dst.body) })
				dst.require(t)
				requireUTF16UTF8AMD64Panic(t, func() { convertErr(input, dst.body) })
				dst.require(t)
				requireUTF16UTF8AMD64Panic(t, func() { convertValid(input, dst.body) })
				dst.require(t)
				requireUTF16UTF8AMD64Panic(t, func() { convertReplace(input, replaceDst.body) })
				replaceDst.require(t)
			}
		})
	}
}

func requireUTF16UTF8AMD64Panic(t *testing.T, fn func()) {
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

func FuzzUTF16UTF8AMD64AgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(utf16UTF8AMD64NativeBytes(repeatU16UTF8('A', 8)))
	f.Add(utf16UTF8AMD64NativeBytes(repeatU16UTF8(0xff, 16)))
	f.Add(utf16UTF8AMD64NativeBytes([]uint16{'a', 0x100, 'b'}))
	f.Add(utf16UTF8AMD64NativeBytes(append(repeatU16UTF8('A', 16), 0x100)))
	f.Add(utf16UTF8AMD64NativeBytes(append(repeatU16UTF8('A', 8), 0xd83d, 0xde00)))
	f.Add(utf16UTF8AMD64NativeBytes([]uint16{0xd800}))
	f.Add(utf16UTF8AMD64NativeBytes([]uint16{0xdc00}))
	f.Add(utf16UTF8AMD64NativeBytes([]uint16{'A', 0x20ac, 0xd83d, 0xde00}))
	f.Add([]byte{0x00, 0x00, 0x7f, 0x00, 0xff, 0x00, 0x00, 0x01})
	f.Add([]byte{0x41, 0x00, 0x00, 0xd8})
	f.Fuzz(func(t *testing.T, raw []byte) {
		native := utf16UTF8AMD64NativeFromBytes(raw)
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			if detectAMD64Features()&cpuSSSE3 == cpuSSSE3 {
				if little {
					checkUTF16UTF8Direct(t, input, true,
						convertUTF16LEToUTF8Westmere, convertUTF16LEToUTF8WithErrorsWestmere,
						convertUTF16LEToUTF8WithReplacementWestmere, convertValidUTF16LEToUTF8Westmere,
						utf8LengthFromUTF16LEWestmere, utf8LengthFromUTF16LEWithReplacementWestmere)
				} else {
					checkUTF16UTF8Direct(t, input, false,
						convertUTF16BEToUTF8Westmere, convertUTF16BEToUTF8WithErrorsWestmere,
						convertUTF16BEToUTF8WithReplacementWestmere, convertValidUTF16BEToUTF8Westmere,
						utf8LengthFromUTF16BEWestmere, utf8LengthFromUTF16BEWithReplacementWestmere)
				}
			}
			if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
				if little {
					checkUTF16UTF8Direct(t, input, true,
						convertUTF16LEToUTF8Haswell, convertUTF16LEToUTF8WithErrorsHaswell,
						convertUTF16LEToUTF8WithReplacementHaswell, convertValidUTF16LEToUTF8Haswell,
						utf8LengthFromUTF16LEHaswell, utf8LengthFromUTF16LEWithReplacementHaswell)
				} else {
					checkUTF16UTF8Direct(t, input, false,
						convertUTF16BEToUTF8Haswell, convertUTF16BEToUTF8WithErrorsHaswell,
						convertUTF16BEToUTF8WithReplacementHaswell, convertValidUTF16BEToUTF8Haswell,
						utf8LengthFromUTF16BEHaswell, utf8LengthFromUTF16BEWithReplacementHaswell)
				}
			}
		}
	})
}

func utf16UTF8AMD64NativeBytes(words []uint16) []byte {
	out := make([]byte, len(words)*2)
	for i, word := range words {
		out[2*i] = byte(word)
		out[2*i+1] = byte(word >> 8)
	}
	return out
}

func utf16UTF8AMD64NativeFromBytes(raw []byte) []uint16 {
	n := len(raw) / 2
	out := make([]uint16, n)
	for i := 0; i < n; i++ {
		out[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
	}
	return out
}

func checkUTF16UTF8Direct(t *testing.T, input []uint16, little bool,
	convert func([]uint16, []byte) int,
	convertErr func([]uint16, []byte) Result,
	convertReplace func([]uint16, []byte) int,
	convertValid func([]uint16, []byte) int,
	length func([]uint16) int,
	lengthReplace func([]uint16) Result,
) {
	t.Helper()

	need := utf8LengthFromUTF16LEScalar(input)
	needReplace := utf8LengthFromUTF16LEWithReplacementScalar(input)
	wantLen := utf8LengthFromUTF16LEScalar(input)
	wantLenR := utf8LengthFromUTF16LEWithReplacementScalar(input)
	if !little {
		need = utf8LengthFromUTF16BEScalar(input)
		needReplace = utf8LengthFromUTF16BEWithReplacementScalar(input)
		wantLen = utf8LengthFromUTF16BEScalar(input)
		wantLenR = utf8LengthFromUTF16BEWithReplacementScalar(input)
	}
	if got := length(input); got != wantLen {
		t.Fatalf("little=%t length = %d, want %d", little, got, wantLen)
	}
	if got := lengthReplace(input); got != wantLenR {
		t.Fatalf("little=%t length_replace = %#v, want %#v", little, got, wantLenR)
	}

	want := bytes.Repeat([]byte{0xa5}, need)
	got := guardedLatin1Destination[byte](need, 0xa5)
	var (
		wantN, gotN int
		wantE, gotE Result
		wantR, gotR int
		wantV, gotV int
	)
	if little {
		wantN = convertUTF16LEToUTF8Scalar(input, want)
		gotN = convert(input, got.body)
	} else {
		wantN = convertUTF16BEToUTF8Scalar(input, want)
		gotN = convert(input, got.body)
	}
	if gotN != wantN || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t convert = %d/%x, want %d/%x", little, gotN, got.body, wantN, want)
	}
	got.require(t)

	want = bytes.Repeat([]byte{0xa5}, need)
	got = guardedLatin1Destination[byte](need, 0xa5)
	if little {
		wantE = convertUTF16LEToUTF8WithErrorsScalar(input, want)
		gotE = convertErr(input, got.body)
	} else {
		wantE = convertUTF16BEToUTF8WithErrorsScalar(input, want)
		gotE = convertErr(input, got.body)
	}
	if gotE != wantE || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t with_errors = %#v/%x, want %#v/%x", little, gotE, got.body, wantE, want)
	}
	got.require(t)

	wantReplace := bytes.Repeat([]byte{0xa5}, needReplace.Count)
	gotReplace := guardedLatin1Destination[byte](needReplace.Count, 0xa5)
	if little {
		wantR = convertUTF16LEToUTF8WithReplacementScalar(input, wantReplace)
		gotR = convertReplace(input, gotReplace.body)
	} else {
		wantR = convertUTF16BEToUTF8WithReplacementScalar(input, wantReplace)
		gotR = convertReplace(input, gotReplace.body)
	}
	if gotR != wantR || !bytes.Equal(gotReplace.body, wantReplace) {
		t.Fatalf("little=%t with_replacement = %d/%x, want %d/%x", little, gotR, gotReplace.body, wantR, wantReplace)
	}
	gotReplace.require(t)

	// convert_valid_utf16_to_utf8 assumes well-formed input. Skip it when the
	// with_errors oracle already rejected the sequence.
	if wantE.Error != Success {
		return
	}
	want = bytes.Repeat([]byte{0xa5}, need)
	got = guardedLatin1Destination[byte](need, 0xa5)
	if little {
		wantV = convertValidUTF16LEToUTF8Scalar(input, want)
		gotV = convertValid(input, got.body)
	} else {
		wantV = convertValidUTF16BEToUTF8Scalar(input, want)
		gotV = convertValid(input, got.body)
	}
	if gotV != wantV || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t valid = %d/%x, want %d/%x", little, gotV, got.body, wantV, want)
	}
	got.require(t)
}

func repeatU16UTF8(value uint16, n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		out[i] = value
	}
	return out
}

// Portions Copyright 2021 The simdutf Authors.

// Direct differential coverage for the amd64 UTF-32→Latin-1/UTF-8/UTF-16
// Westmere/Haswell translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf32_to_*.cpp
// and src/haswell/avx2_convert_utf32_to_*.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// Providers are invoked directly; feature skips only gate unavailable ISA.

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

func requireUTF8ConvertAMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skip("required amd64 SIMD feature is unavailable")
	}
}

func TestDirectAMD64UTF8ConvertAgainstScalar(t *testing.T) {
	variants := []struct {
		name     string
		feature  cpuFeatures
		latin1   func([]byte, []byte) int
		latin1E  func([]byte, []byte) Result
		latin1V  func([]byte, []byte) int
		utf16le  func([]byte, []uint16) int
		utf16be  func([]byte, []uint16) int
		utf16leE func([]byte, []uint16) Result
		utf16beE func([]byte, []uint16) Result
		utf16leV func([]byte, []uint16) int
		utf16beV func([]byte, []uint16) int
		utf32    func([]byte, []uint32) int
		utf32E   func([]byte, []uint32) Result
		utf32V   func([]byte, []uint32) int
	}{
		{
			name: "westmere", feature: cpuSSSE3,
			latin1: convertUTF8ToLatin1Westmere, latin1E: convertUTF8ToLatin1WithErrorsWestmere, latin1V: convertValidUTF8ToLatin1Westmere,
			utf16le: convertUTF8ToUTF16LEWestmere, utf16be: convertUTF8ToUTF16BEWestmere,
			utf16leE: convertUTF8ToUTF16LEWithErrorsWestmere, utf16beE: convertUTF8ToUTF16BEWithErrorsWestmere,
			utf16leV: convertValidUTF8ToUTF16LEWestmere, utf16beV: convertValidUTF8ToUTF16BEWestmere,
			utf32: convertUTF8ToUTF32Westmere, utf32E: convertUTF8ToUTF32WithErrorsWestmere, utf32V: convertValidUTF8ToUTF32Westmere,
		},
		{
			name: "haswell", feature: cpuAVX2,
			latin1: convertUTF8ToLatin1Haswell, latin1E: convertUTF8ToLatin1WithErrorsHaswell, latin1V: convertValidUTF8ToLatin1Haswell,
			utf16le: convertUTF8ToUTF16LEHaswell, utf16be: convertUTF8ToUTF16BEHaswell,
			utf16leE: convertUTF8ToUTF16LEWithErrorsHaswell, utf16beE: convertUTF8ToUTF16BEWithErrorsHaswell,
			utf16leV: convertValidUTF8ToUTF16LEHaswell, utf16beV: convertValidUTF8ToUTF16BEHaswell,
			utf32: convertUTF8ToUTF32Haswell, utf32E: convertUTF8ToUTF32WithErrorsHaswell, utf32V: convertValidUTF8ToUTF32Haswell,
		},
	}

	inputs := []struct {
		name  string
		input []byte
	}{
		{"long-ascii-64", bytes.Repeat([]byte{'A'}, 64)},
		{"long-ascii-65", bytes.Repeat([]byte{'A'}, 65)},
		{"long-ascii-128", bytes.Repeat([]byte{'A'}, 128)},
		{"ascii-then-latin1", append(bytes.Repeat([]byte{'A'}, 64), []byte("caf\u00e9")...)},
		{"mixed-emoji", append(bytes.Repeat([]byte{'A'}, 64), []byte("A\U0001F600B")...)},
		{"mixed-arabic", append(bytes.Repeat([]byte{'A'}, 64), []byte("\u0645\u0631\u062d\u0628\u0627")...)},
		{"emoji", []byte("A\U0001F600B")},
		{"arabic", []byte("\u0645\u0631\u062d\u0628\u0627")},
		{"latin1", []byte("caf\u00e9")},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF8ConvertAMD64Variant(t, v.feature)
			for _, tc := range inputs {
				t.Run(tc.name, func(t *testing.T) {
					checkUTF8ConvertDirectAMD64(t, tc.input, v.latin1, v.latin1E, v.latin1V, v.utf16le, v.utf16be, v.utf16leE, v.utf16beE, v.utf16leV, v.utf16beV, v.utf32, v.utf32E, v.utf32V)
				})
			}
		})
	}
}

func TestDirectAMD64UTF8ConvertWithErrorsAgainstScalar(t *testing.T) {
	variants := []struct {
		name     string
		feature  cpuFeatures
		latin1E  func([]byte, []byte) Result
		utf16leE func([]byte, []uint16) Result
		utf16beE func([]byte, []uint16) Result
		utf32E   func([]byte, []uint32) Result
	}{
		{"westmere", cpuSSSE3, convertUTF8ToLatin1WithErrorsWestmere, convertUTF8ToUTF16LEWithErrorsWestmere, convertUTF8ToUTF16BEWithErrorsWestmere, convertUTF8ToUTF32WithErrorsWestmere},
		{"haswell", cpuAVX2, convertUTF8ToLatin1WithErrorsHaswell, convertUTF8ToUTF16LEWithErrorsHaswell, convertUTF8ToUTF16BEWithErrorsHaswell, convertUTF8ToUTF32WithErrorsHaswell},
	}
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
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF8ConvertAMD64Variant(t, v.feature)
			for _, tc := range invalids {
				t.Run(tc.name, func(t *testing.T) {
					checkUTF8ConvertErrorsAMD64(t, tc.input, v.latin1E, v.utf16leE, v.utf16beE, v.utf32E)
				})
			}
		})
	}
}

func TestDirectAMD64UTF8ConvertPreflightPreservesDestination(t *testing.T) {
	input := bytes.Repeat([]byte{'A'}, 65)
	variants := []struct {
		name     string
		feature  cpuFeatures
		latin1   func([]byte, []byte) int
		latin1E  func([]byte, []byte) Result
		latin1V  func([]byte, []byte) int
		utf16le  func([]byte, []uint16) int
		utf16be  func([]byte, []uint16) int
		utf16leE func([]byte, []uint16) Result
		utf16beE func([]byte, []uint16) Result
		utf16leV func([]byte, []uint16) int
		utf16beV func([]byte, []uint16) int
		utf32    func([]byte, []uint32) int
		utf32E   func([]byte, []uint32) Result
		utf32V   func([]byte, []uint32) int
	}{
		{
			name: "westmere", feature: cpuSSSE3,
			latin1: convertUTF8ToLatin1Westmere, latin1E: convertUTF8ToLatin1WithErrorsWestmere, latin1V: convertValidUTF8ToLatin1Westmere,
			utf16le: convertUTF8ToUTF16LEWestmere, utf16be: convertUTF8ToUTF16BEWestmere,
			utf16leE: convertUTF8ToUTF16LEWithErrorsWestmere, utf16beE: convertUTF8ToUTF16BEWithErrorsWestmere,
			utf16leV: convertValidUTF8ToUTF16LEWestmere, utf16beV: convertValidUTF8ToUTF16BEWestmere,
			utf32: convertUTF8ToUTF32Westmere, utf32E: convertUTF8ToUTF32WithErrorsWestmere, utf32V: convertValidUTF8ToUTF32Westmere,
		},
		{
			name: "haswell", feature: cpuAVX2,
			latin1: convertUTF8ToLatin1Haswell, latin1E: convertUTF8ToLatin1WithErrorsHaswell, latin1V: convertValidUTF8ToLatin1Haswell,
			utf16le: convertUTF8ToUTF16LEHaswell, utf16be: convertUTF8ToUTF16BEHaswell,
			utf16leE: convertUTF8ToUTF16LEWithErrorsHaswell, utf16beE: convertUTF8ToUTF16BEWithErrorsHaswell,
			utf16leV: convertValidUTF8ToUTF16LEHaswell, utf16beV: convertValidUTF8ToUTF16BEHaswell,
			utf32: convertUTF8ToUTF32Haswell, utf32E: convertUTF8ToUTF32WithErrorsHaswell, utf32V: convertValidUTF8ToUTF32Haswell,
		},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireUTF8ConvertAMD64Variant(t, v.feature)

			dst8 := bytes.Repeat([]byte{0xa5}, latin1LengthFromUTF8Scalar(input)-1)
			for name, convert := range map[string]func(){
				"Latin1":       func() { v.latin1(input, dst8) },
				"Latin1Errors": func() { v.latin1E(input, dst8) },
				"ValidLatin1":  func() { v.latin1V(input, dst8) },
			} {
				requireUTF8ConvertAMD64Panic(t, convert)
				if !bytes.Equal(dst8, bytes.Repeat([]byte{0xa5}, len(dst8))) {
					t.Fatalf("%s destination changed before short-destination panic", name)
				}
			}

			for name, convert := range map[string]func([]byte, []uint16) int{
				"UTF-16LE": v.utf16le, "UTF-16BE": v.utf16be, "ValidUTF-16LE": v.utf16leV, "ValidUTF-16BE": v.utf16beV,
			} {
				dst := make([]uint16, utf16LengthFromUTF8Scalar(input)-1)
				for i := range dst {
					dst[i] = 0xa5a5
				}
				requireUTF8ConvertAMD64Panic(t, func() { convert(input, dst) })
				for _, value := range dst {
					if value != 0xa5a5 {
						t.Fatalf("%s destination changed before short-destination panic", name)
					}
				}
			}
			for name, convert := range map[string]func([]byte, []uint16) Result{
				"UTF-16LEErrors": v.utf16leE, "UTF-16BEErrors": v.utf16beE,
			} {
				dst := make([]uint16, utf16LengthFromUTF8Scalar(input)-1)
				for i := range dst {
					dst[i] = 0xa5a5
				}
				requireUTF8ConvertAMD64Panic(t, func() { convert(input, dst) })
				for _, value := range dst {
					if value != 0xa5a5 {
						t.Fatalf("%s destination changed before short-destination panic", name)
					}
				}
			}

			dst32 := make([]uint32, utf32LengthFromUTF8Scalar(input)-1)
			for i := range dst32 {
				dst32[i] = 0xa5a5a5a5
			}
			for name, convert := range map[string]func(){
				"UTF-32":       func() { v.utf32(input, dst32) },
				"UTF-32Errors": func() { v.utf32E(input, dst32) },
				"ValidUTF-32":  func() { v.utf32V(input, dst32) },
			} {
				requireUTF8ConvertAMD64Panic(t, convert)
				for _, value := range dst32 {
					if value != 0xa5a5a5a5 {
						t.Fatalf("%s destination changed before short-destination panic", name)
					}
				}
			}
		})
	}
}

func requireUTF8ConvertAMD64Panic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func FuzzUTF8ConvertAMD64AgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte{'A'}, 64))
	f.Add([]byte("caf\u00e9"))
	f.Add([]byte("A\U0001F600B"))
	f.Add([]byte("\u0645\u0631\u062d\u0628\u0627"))
	f.Add([]byte{0xc2})
	f.Add([]byte{0xff})
	f.Add(append(bytes.Repeat([]byte{'A'}, 64), 0xc2))
	f.Fuzz(func(t *testing.T, input []byte) {
		if detectAMD64Features()&cpuSSSE3 == cpuSSSE3 {
			checkUTF8ConvertDirectAMD64(
				t, input,
				convertUTF8ToLatin1Westmere, convertUTF8ToLatin1WithErrorsWestmere, convertValidUTF8ToLatin1Westmere,
				convertUTF8ToUTF16LEWestmere, convertUTF8ToUTF16BEWestmere,
				convertUTF8ToUTF16LEWithErrorsWestmere, convertUTF8ToUTF16BEWithErrorsWestmere,
				convertValidUTF8ToUTF16LEWestmere, convertValidUTF8ToUTF16BEWestmere,
				convertUTF8ToUTF32Westmere, convertUTF8ToUTF32WithErrorsWestmere, convertValidUTF8ToUTF32Westmere,
			)
		}
		if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
			checkUTF8ConvertDirectAMD64(
				t, input,
				convertUTF8ToLatin1Haswell, convertUTF8ToLatin1WithErrorsHaswell, convertValidUTF8ToLatin1Haswell,
				convertUTF8ToUTF16LEHaswell, convertUTF8ToUTF16BEHaswell,
				convertUTF8ToUTF16LEWithErrorsHaswell, convertUTF8ToUTF16BEWithErrorsHaswell,
				convertValidUTF8ToUTF16LEHaswell, convertValidUTF8ToUTF16BEHaswell,
				convertUTF8ToUTF32Haswell, convertUTF8ToUTF32WithErrorsHaswell, convertValidUTF8ToUTF32Haswell,
			)
		}
	})
}

func checkUTF8ConvertDirectAMD64(
	t *testing.T,
	input []byte,
	latin1 func([]byte, []byte) int,
	latin1E func([]byte, []byte) Result,
	latin1V func([]byte, []byte) int,
	utf16le, utf16be func([]byte, []uint16) int,
	utf16leE, utf16beE func([]byte, []uint16) Result,
	utf16leV, utf16beV func([]byte, []uint16) int,
	utf32 func([]byte, []uint32) int,
	utf32E func([]byte, []uint32) Result,
	utf32V func([]byte, []uint32) int,
) {
	t.Helper()

	wantLatin1Len := latin1LengthFromUTF8Scalar(input)
	want8 := bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
	got8 := bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
	wantN := convertUTF8ToLatin1Scalar(input, want8[:wantLatin1Len])
	if got := latin1(input, got8[:wantLatin1Len]); got != wantN || !bytes.Equal(got8, want8) {
		t.Fatal("Latin1 mismatch or canary overwrite")
	}

	want8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
	got8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
	wantE := convertUTF8ToLatin1WithErrorsScalar(input, want8[:wantLatin1Len])
	if got := latin1E(input, got8[:wantLatin1Len]); got != wantE || !bytes.Equal(got8, want8) {
		t.Fatalf("Latin1 WithErrors = %#v, want %#v (or canary overwrite)", got, wantE)
	}

	if utf8.Valid(input) {
		want8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
		got8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
		wantV := convertValidUTF8ToLatin1Scalar(input, want8[:wantLatin1Len])
		gotV := latin1V(input, got8[:wantLatin1Len])
		if gotV != wantV || !bytes.Equal(got8, want8) {
			t.Fatalf("Valid Latin1 = %d, want %d (or payload/canary mismatch)", gotV, wantV)
		}
	}

	want16Len := utf16LengthFromUTF8Scalar(input)
	want16 := make([]uint16, want16Len)
	got16 := make([]uint16, want16Len+8)
	for i := range want16 {
		want16[i] = 0xa5a5
	}
	for i := want16Len; i < len(got16); i++ {
		got16[i] = 0xa5a5
	}
	// Body of got16 starts zeroed; match that for fair failure comparison.
	for i := range want16 {
		want16[i] = 0
	}
	wantN16 := convertUTF8ToUTF16LEScalar(input, want16)
	if got := utf16le(input, got16); got != wantN16 || !equalU16(got16[:want16Len], want16) || got16[want16Len] != 0xa5a5 {
		t.Fatal("UTF-16LE mismatch or canary overwrite")
	}
	wantE16 := convertUTF8ToUTF16LEWithErrorsScalar(input, want16)
	if got := utf16leE(input, got16); got != wantE16 || got16[want16Len] != 0xa5a5 {
		t.Fatalf("UTF-16LE WithErrors = %#v, want %#v", got, wantE16)
	}
	if utf8.Valid(input) {
		wantV16 := convertValidUTF8ToUTF16LEScalar(input, want16)
		if got := utf16leV(input, got16); got != wantV16 || !equalU16(got16[:want16Len], want16) || got16[want16Len] != 0xa5a5 {
			t.Fatalf("Valid UTF-16LE = %d, want %d", got, wantV16)
		}
	}

	for i := range want16 {
		want16[i] = 0
	}
	for i := range got16 {
		got16[i] = 0
	}
	for i := want16Len; i < len(got16); i++ {
		got16[i] = 0xa5a5
	}
	wantN16 = convertUTF8ToUTF16BEScalar(input, want16)
	if got := utf16be(input, got16); got != wantN16 || !equalU16(got16[:want16Len], want16) || got16[want16Len] != 0xa5a5 {
		t.Fatal("UTF-16BE mismatch or canary overwrite")
	}
	wantE16 = convertUTF8ToUTF16BEWithErrorsScalar(input, want16)
	if got := utf16beE(input, got16); got != wantE16 || got16[want16Len] != 0xa5a5 {
		t.Fatalf("UTF-16BE WithErrors = %#v, want %#v", got, wantE16)
	}
	if utf8.Valid(input) {
		wantV16 := convertValidUTF8ToUTF16BEScalar(input, want16)
		if got := utf16beV(input, got16); got != wantV16 || !equalU16(got16[:want16Len], want16) || got16[want16Len] != 0xa5a5 {
			t.Fatalf("Valid UTF-16BE = %d, want %d", got, wantV16)
		}
	}

	want32Len := utf32LengthFromUTF8Scalar(input)
	want32 := make([]uint32, want32Len)
	got32 := make([]uint32, want32Len+8)
	for i := want32Len; i < len(got32); i++ {
		got32[i] = 0xa5a5a5a5
	}
	wantN32 := convertUTF8ToUTF32Scalar(input, want32)
	if got := utf32(input, got32); got != wantN32 || !equalU32(got32[:want32Len], want32) || got32[want32Len] != 0xa5a5a5a5 {
		t.Fatal("UTF-32 mismatch or canary overwrite")
	}
	wantE32 := convertUTF8ToUTF32WithErrorsScalar(input, want32)
	if got := utf32E(input, got32); got != wantE32 || got32[want32Len] != 0xa5a5a5a5 {
		t.Fatalf("UTF-32 WithErrors = %#v, want %#v", got, wantE32)
	}
	if utf8.Valid(input) {
		wantV32 := convertValidUTF8ToUTF32Scalar(input, want32)
		if got := utf32V(input, got32); got != wantV32 || !equalU32(got32[:want32Len], want32) || got32[want32Len] != 0xa5a5a5a5 {
			t.Fatalf("Valid UTF-32 = %d, want %d", got, wantV32)
		}
	}
}

func checkUTF8ConvertErrorsAMD64(
	t *testing.T,
	input []byte,
	latin1E func([]byte, []byte) Result,
	utf16leE, utf16beE func([]byte, []uint16) Result,
	utf32E func([]byte, []uint32) Result,
) {
	t.Helper()

	dst8 := make([]byte, latin1LengthFromUTF8Scalar(input)+8)
	want := convertUTF8ToLatin1WithErrorsScalar(input, dst8)
	if got := latin1E(input, dst8); got != want {
		t.Fatalf("Latin1 WithErrors = %#v, want %#v", got, want)
	}

	dst16 := make([]uint16, utf16LengthFromUTF8Scalar(input)+8)
	want = convertUTF8ToUTF16LEWithErrorsScalar(input, dst16)
	if got := utf16leE(input, dst16); got != want {
		t.Fatalf("UTF-16LE WithErrors = %#v, want %#v", got, want)
	}
	want = convertUTF8ToUTF16BEWithErrorsScalar(input, dst16)
	if got := utf16beE(input, dst16); got != want {
		t.Fatalf("UTF-16BE WithErrors = %#v, want %#v", got, want)
	}

	dst32 := make([]uint32, utf32LengthFromUTF8Scalar(input)+8)
	want = convertUTF8ToUTF32WithErrorsScalar(input, dst32)
	if got := utf32E(input, dst32); got != want {
		t.Fatalf("UTF-32 WithErrors = %#v, want %#v", got, want)
	}
}

func TestDirectAMD64UTFValidationAgainstScalar(t *testing.T) {
	utf16Cases := [][]uint16{
		nil,
		{0x41},
		{0xd800},
		{0xdc00},
		{0xd800, 0xdc00},
		{0x41, 0xd800, 0x42},
		{0x41, 0xdc00, 0x42},
		{0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0xd800, 0xdc00, 0x48},
	}
	providers := []struct {
		name string
		le   func([]uint16) bool
		be   func([]uint16) bool
		leE  func([]uint16) Result
		beE  func([]uint16) Result
	}{
		{"westmere", validateUTF16LEWestmere, validateUTF16BEWestmere, validateUTF16LEWithErrorsWestmere, validateUTF16BEWithErrorsWestmere},
		{"haswell", validateUTF16LEHaswell, validateUTF16BEHaswell, validateUTF16LEWithErrorsHaswell, validateUTF16BEWithErrorsHaswell},
	}
	for _, p := range providers {
		for _, in := range utf16Cases {
			be := make([]uint16, len(in))
			for i := range in {
				be[i] = bits.ReverseBytes16(in[i])
			}
			if got, want := p.le(in), validateUTF16LEScalar(in); got != want {
				t.Fatalf("%s LE bool %#v: got %v want %v", p.name, in, got, want)
			}
			if got, want := p.be(be), validateUTF16BEScalar(be); got != want {
				t.Fatalf("%s BE bool %#v: got %v want %v", p.name, in, got, want)
			}
			if got, want := p.leE(in), validateUTF16LEWithErrorsScalar(in); got != want {
				t.Fatalf("%s LE result %#v: got %#v want %#v", p.name, in, got, want)
			}
			if got, want := p.beE(be), validateUTF16BEWithErrorsScalar(be); got != want {
				t.Fatalf("%s BE result %#v: got %#v want %#v", p.name, in, got, want)
			}
		}
	}
	utf32Cases := [][]uint32{nil, {0}, {0x7f, 0x80}, {0x10ffff}, {0x110000}, {0xd800}, {0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x110000}}
	for _, p := range []struct {
		name   string
		ok     func([]uint32) bool
		errors func([]uint32) Result
	}{
		{"westmere", validateUTF32Westmere, validateUTF32WithErrorsWestmere},
		{"haswell", validateUTF32Haswell, validateUTF32WithErrorsHaswell},
	} {
		for _, in := range utf32Cases {
			if got, want := p.ok(in), validateUTF32Scalar(in); got != want {
				t.Fatalf("%s UTF-32 bool %#v: got %v want %v", p.name, in, got, want)
			}
			if got, want := p.errors(in), validateUTF32WithErrorsScalar(in); got != want {
				t.Fatalf("%s UTF-32 result %#v: got %#v want %#v", p.name, in, got, want)
			}
		}
	}
}

func TestDirectAMD64ToWellFormedAgainstScalar(t *testing.T) {
	input := []uint16{0x41, 0xd800, 0xdc00, 0x42, 0xd800, 0x43, 0xdc00, 0x44, 0xd800}
	providers := []struct {
		name string
		big  bool
		fn   func([]uint16, []uint16)
	}{
		{"westmere-le", false, toWellFormedUTF16LEWestmere},
		{"haswell-le", false, toWellFormedUTF16LEHaswell},
		{"westmere-be", true, toWellFormedUTF16BEWestmere},
		{"haswell-be", true, toWellFormedUTF16BEHaswell},
	}
	for _, p := range providers {
		in := append([]uint16(nil), input...)
		if p.big {
			for i := range in {
				in[i] = bits.ReverseBytes16(in[i])
			}
		}
		want := make([]uint16, len(in))
		toWellFormedUTF16Scalar(in, want, !p.big)
		backing := make([]uint16, len(in)+2)
		backing[0], backing[len(backing)-1] = 0xaaaa, 0xbbbb
		got := backing[1 : len(backing)-1]
		p.fn(in, got)
		if !reflect.DeepEqual(got, want) || backing[0] != 0xaaaa || backing[len(backing)-1] != 0xbbbb {
			t.Fatalf("%s output/canary: got %#v backing %#v want %#v", p.name, got, backing, want)
		}
		inPlace := append([]uint16(nil), in...)
		p.fn(inPlace, inPlace)
		if !reflect.DeepEqual(inPlace, want) {
			t.Fatalf("%s in-place: got %#v want %#v", p.name, inPlace, want)
		}
		short := []uint16{0xaaaa, 0xbbbb}
		didPanic := false
		func() {
			defer func() { didPanic = recover() != nil }()
			p.fn(in, short)
		}()
		if !didPanic || short[0] != 0xaaaa || short[1] != 0xbbbb {
			t.Fatalf("%s short destination did not panic before storing: %#v", p.name, short)
		}
	}
}

func FuzzUTFValidationAMD64AgainstScalar(f *testing.F) {
	for _, seed := range [][]byte{nil, {0, 0}, {0, 0xd8, 0, 0xdc}, {0, 0, 0x11, 0}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		semantic := fuzzUTF16Words(data)
		input32 := make([]uint32, len(data)/4)
		for i := range input32 {
			input32[i] = uint32(data[4*i]) | uint32(data[4*i+1])<<8 |
				uint32(data[4*i+2])<<16 | uint32(data[4*i+3])<<24
		}
		providers := []struct {
			name     string
			le       func([]uint16) Result
			be       func([]uint16) Result
			repairLE func([]uint16, []uint16)
			repairBE func([]uint16, []uint16)
			utf32    func([]uint32) Result
		}{
			{"westmere", validateUTF16LEWithErrorsWestmere, validateUTF16BEWithErrorsWestmere, toWellFormedUTF16LEWestmere, toWellFormedUTF16BEWestmere, validateUTF32WithErrorsWestmere},
			{"haswell", validateUTF16LEWithErrorsHaswell, validateUTF16BEWithErrorsHaswell, toWellFormedUTF16LEHaswell, toWellFormedUTF16BEHaswell, validateUTF32WithErrorsHaswell},
		}
		for _, provider := range providers {
			for _, little := range []bool{true, false} {
				input := rawUTF16Scalar(semantic, little)
				gotDst, wantDst := make([]uint16, len(input)), make([]uint16, len(input))
				var gotResult, wantResult Result
				if little {
					gotResult, wantResult = provider.le(input), validateUTF16LEWithErrorsScalar(input)
					provider.repairLE(input, gotDst)
					toWellFormedUTF16LEScalar(input, wantDst)
				} else {
					gotResult, wantResult = provider.be(input), validateUTF16BEWithErrorsScalar(input)
					provider.repairBE(input, gotDst)
					toWellFormedUTF16BEScalar(input, wantDst)
				}
				if gotResult != wantResult || !reflect.DeepEqual(gotDst, wantDst) {
					t.Fatalf("%s UTF-16 little=%t: result=%+v/%+v output=%x/%x", provider.name, little, gotResult, wantResult, gotDst, wantDst)
				}
			}
			if got, want := provider.utf32(input32), validateUTF32WithErrorsScalar(input32); got != want {
				t.Fatalf("%s UTF-32 result=%+v want=%+v input=%x", provider.name, got, want, input32)
			}
		}
	})
}

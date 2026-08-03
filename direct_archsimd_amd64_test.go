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
	"encoding/binary"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func requireBase64ArchsimdAVX2(t *testing.T) {
	t.Helper()
	// Direct archsimd kernels need the Go simd/archsimd AVX2 runtime gate.
	// On some hosts (e.g. Rosetta) that is set even when detectAMD64Features
	// omits cpuAVX2; public selection remains scalar-first either way.
	if !archsimdAVX2Available() {
		t.Skip("direct archsimd AVX2 implementation is unsupported")
	}
}

func TestBase64ArchsimdLengthMatchesScalar(t *testing.T) {
	requireBase64ArchsimdAVX2(t)
	inputs := [][]byte{
		nil,
		[]byte("A"),
		[]byte("AAAA"),
		[]byte("AAAA AA=="),
		bytes.Repeat([]byte("A"), 63),
		bytes.Repeat([]byte("A"), 64),
		bytes.Repeat([]byte("A"), 65),
		append(bytes.Repeat([]byte("A"), 60), '=', '='),
		[]byte("AAAA\nBBBB\tCCCC "),
	}
	for _, in := range inputs {
		if got, want := binaryLengthFromBase64Archsimd(in), binaryLengthFromBase64Scalar(in); got != want {
			t.Fatalf("len archsimd=%d scalar=%d input=%q", got, want, in)
		}
	}
}

func TestBase64ArchsimdLengthUTF16MatchesScalar(t *testing.T) {
	requireBase64ArchsimdAVX2(t)
	in := make([]uint16, 64)
	for i := range in {
		in[i] = 'A'
	}
	in[60] = ' '
	in[61] = '='
	in[62] = '='
	if got, want := binaryLengthFromBase64UTF16Archsimd(in), binaryLengthFromBase64UTF16Scalar(in); got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestBase64ArchsimdEncodeMatchesScalar(t *testing.T) {
	requireBase64ArchsimdAVX2(t)
	for _, opt := range []Base64Options{Base64Default, Base64URL, Base64URL | Base64URLWithPadding} {
		for _, n := range []int{0, 1, 2, 3, 12, 23, 24, 25, 27, 28, 47, 48, 96, 97} {
			in := bytes.Repeat([]byte{byte(n), 0x7f, 0x80}, (n+2)/3)[:n]
			dstA := make([]byte, base64LengthFromBinaryScalar(len(in), opt))
			dstS := make([]byte, len(dstA))
			nA := binaryToBase64Archsimd(in, dstA, opt)
			nS := binaryToBase64Scalar(in, dstS, opt)
			if nA != nS || !bytes.Equal(dstA[:nA], dstS[:nS]) {
				t.Fatalf("opt=%v len=%d archsimd=%q scalar=%q", opt, n, dstA[:min(nA, 96)], dstS[:min(nS, 96)])
			}
		}
	}
}

func TestBase64ArchsimdWithLinesMatchesScalar(t *testing.T) {
	requireBase64ArchsimdAVX2(t)
	in := bytes.Repeat([]byte("0123456789abcdef"), 8)
	for _, line := range []int{4, 16, 32, 76} {
		dstA := make([]byte, base64LengthFromBinaryWithLinesScalar(len(in), Base64Default, line))
		dstS := make([]byte, len(dstA))
		nA := binaryToBase64WithLinesArchsimd(in, dstA, line, Base64Default)
		nS := binaryToBase64WithLinesScalar(in, dstS, line, Base64Default)
		if nA != nS || !bytes.Equal(dstA[:nA], dstS[:nS]) {
			t.Fatalf("line=%d archsimd(%d) scalar(%d)", line, nA, nS)
		}
	}
}

func TestBase64ArchsimdDecodeMatchesScalar(t *testing.T) {
	requireBase64ArchsimdAVX2(t)
	raw := bytes.Repeat([]byte("Hello Archsimd Base64 decode path!!"), 6)
	for _, opt := range []Base64Options{Base64Default, Base64URL} {
		enc := make([]byte, base64LengthFromBinaryScalar(len(raw), opt))
		n := binaryToBase64Scalar(raw, enc, opt)
		enc = enc[:n]
		dstA := make([]byte, maximalBinaryLengthFromBase64Scalar(enc))
		dstS := make([]byte, len(dstA))
		rA := base64ToBinaryDetailsArchsimd(enc, dstA, opt, Loose)
		rS := base64ToBinaryDetailsScalar(enc, dstS, opt, Loose)
		if rA != rS || !bytes.Equal(dstA[:rA.OutputCount], dstS[:rS.OutputCount]) {
			t.Fatalf("opt=%v archsimd=%+v scalar=%+v", opt, rA, rS)
		}
		// Encode→decode roundtrip through archsimd providers.
		encA := make([]byte, base64LengthFromBinaryScalar(len(raw), opt))
		nA := binaryToBase64Archsimd(raw, encA, opt)
		dstR := make([]byte, maximalBinaryLengthFromBase64Scalar(encA[:nA]))
		rR := base64ToBinaryArchsimd(encA[:nA], dstR, opt, Loose)
		if rR.Error != Success || !bytes.Equal(dstR[:rR.Count], raw) {
			t.Fatalf("roundtrip opt=%v err=%v got=%q", opt, rR.Error, dstR[:min(rR.Count, 64)])
		}
	}
}

func TestBase64ArchsimdDecodeUTF16MatchesScalar(t *testing.T) {
	requireBase64ArchsimdAVX2(t)
	raw := bytes.Repeat([]byte("abcdef"), 30)
	enc := make([]byte, base64LengthFromBinaryScalar(len(raw), Base64Default))
	n := binaryToBase64Scalar(raw, enc, Base64Default)
	u16 := make([]uint16, n)
	for i := 0; i < n; i++ {
		u16[i] = uint16(enc[i])
	}
	dstA := make([]byte, maximalBinaryLengthFromBase64UTF16Scalar(u16))
	dstS := make([]byte, len(dstA))
	rA := base64ToBinaryDetailsUTF16Archsimd(u16, dstA, Base64Default, Loose)
	rS := base64ToBinaryDetailsUTF16Scalar(u16, dstS, Base64Default, Loose)
	if rA != rS || !bytes.Equal(dstA[:rA.OutputCount], dstS[:rS.OutputCount]) {
		t.Fatalf("archsimd=%+v scalar=%+v", rA, rS)
	}
}

func TestBase64ArchsimdDecodeFallsBackOnWhitespace(t *testing.T) {
	requireBase64ArchsimdAVX2(t)
	// Contiguous path rejects ignorable bytes; whitespace must still decode via scalar residual.
	enc := []byte("SGVsbG8sIHdvcmxkIQ==")
	withWS := []byte("SGVs bG8s\nIHdv cmxk IQ==")
	dstA := make([]byte, maximalBinaryLengthFromBase64Scalar(withWS))
	dstS := make([]byte, len(dstA))
	rA := base64ToBinaryDetailsArchsimd(withWS, dstA, Base64Default, Loose)
	rS := base64ToBinaryDetailsScalar(withWS, dstS, Base64Default, Loose)
	if rA != rS || !bytes.Equal(dstA[:rA.OutputCount], dstS[:rS.OutputCount]) {
		t.Fatalf("whitespace archsimd=%+v scalar=%+v", rA, rS)
	}
	dstC := make([]byte, maximalBinaryLengthFromBase64Scalar(enc))
	rC := base64ToBinaryDetailsArchsimd(enc, dstC, Base64Default, Loose)
	if rC.Error != Success || rC.OutputCount != rS.OutputCount {
		t.Fatalf("compact decode failed: %+v", rC)
	}
}

func TestBase64ArchsimdDecodeBlocksDirect(t *testing.T) {
	requireBase64ArchsimdAVX2(t)
	raw := bytes.Repeat([]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}, 16) // 96 bytes
	enc := make([]byte, base64LengthFromBinaryScalar(len(raw), Base64Default))
	n := binaryToBase64Scalar(raw, enc, Base64Default)
	enc = enc[:n]
	if len(enc) < 64 {
		t.Fatalf("need >=64 encoded bytes, got %d", len(enc))
	}
	toBase64 := base64ToValueTable(Base64Default)
	buf := make([]byte, len(enc))
	for i, c := range enc {
		buf[i] = toBase64[c]
	}
	blocks := len(buf) &^ 63
	dstA := make([]byte, blocks/4*3)
	dstH := make([]byte, len(dstA))
	base64DecodeBlocksArchsimd(buf[:blocks], dstA)
	base64DecodeBlocksHaswell(buf[:blocks], dstH)
	if !bytes.Equal(dstA, dstH) {
		t.Fatalf("archsimd blocks mismatch haswell\n got %q\nwant %q", dstA[:min(len(dstA), 48)], dstH[:min(len(dstH), 48)])
	}
	if !bytes.Equal(dstA, raw[:len(dstA)]) {
		t.Fatalf("archsimd blocks mismatch raw\n got %q\nwant %q", dstA[:min(len(dstA), 48)], raw[:min(len(dstA), 48)])
	}
}

func requireDetectArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdDetectEncodingsAgainstScalar(t *testing.T) {
	requireDetectArchsimdAVX2(t)
	cases := []struct {
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
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectEncodingsArchsimd(tc.input)
			want := detectEncodingsScalar(tc.input)
			if got != want {
				t.Fatalf("detectEncodingsArchsimd(%q) = %d, want scalar %d", tc.input, got, want)
			}
		})
	}
}

func TestArchsimdDetectEncodingsHelperExportsProvider(t *testing.T) {
	fn := archsimdDetectEncodings()
	if fn == nil {
		t.Fatal("archsimdDetectEncodings() = nil, want detectEncodingsArchsimd")
	}
	if !sameFunction(fn, detectEncodingsArchsimd) {
		t.Fatalf("archsimdDetectEncodings() = %p, want detectEncodingsArchsimd %p", fn, detectEncodingsArchsimd)
	}
}

func requireFindArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdFindAgainstScalar(t *testing.T) {
	requireFindArchsimdAVX2(t)

	byteCases := []struct {
		name  string
		input []byte
		value byte
	}{
		{name: "nil", value: 'a'},
		{name: "empty", input: []byte{}, value: 'a'},
		{name: "hit-first", input: []byte("abc"), value: 'a'},
		{name: "hit-middle", input: []byte("bac"), value: 'a'},
		{name: "hit-last", input: []byte("bca"), value: 'a'},
		{name: "miss", input: []byte("bcd"), value: 'a'},
		{name: "nul-hit", input: []byte{'A', 'B', 0, 'C'}, value: 0},
		{name: "short", input: []byte{7}, value: 7},
	}
	for _, length := range [...]int{31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 257} {
		input := make([]byte, length)
		for i := range input {
			input[i] = byte(i%251 + 1)
		}
		byteCases = append(
			byteCases,
			struct {
				name  string
				input []byte
				value byte
			}{name: fmt.Sprintf("miss-length-%d", length), input: input, value: 0},
			struct {
				name  string
				input []byte
				value byte
			}{name: fmt.Sprintf("hit-end-length-%d", length), input: append(append([]byte{}, input[:length-1]...), 0), value: 0},
			struct {
				name  string
				input []byte
				value byte
			}{name: fmt.Sprintf("hit-mid-length-%d", length), input: func() []byte {
				out := append([]byte{}, input...)
				out[length/2] = 0
				return out
			}(), value: 0},
		)
	}
	for _, tc := range byteCases {
		t.Run("byte/"+tc.name, func(t *testing.T) {
			got := findArchsimd(tc.input, tc.value)
			want := findScalar(tc.input, tc.value)
			if got != want {
				t.Fatalf("findArchsimd = %d, scalar = %d", got, want)
			}
		})
	}

	utf16Cases := []struct {
		name  string
		input []uint16
		value uint16
	}{
		{name: "nil", value: 'a'},
		{name: "empty", input: []uint16{}, value: 'a'},
		{name: "hit-first", input: []uint16{'a', 'b', 'c'}, value: 'a'},
		{name: "hit-middle", input: []uint16{'b', 'a', 'c'}, value: 'a'},
		{name: "hit-last", input: []uint16{'b', 'c', 'a'}, value: 'a'},
		{name: "miss", input: []uint16{'b', 'c', 'd'}, value: 'a'},
		{name: "nul-hit", input: []uint16{'A', 0, 'C'}, value: 0},
		{name: "high-unit", input: []uint16{0x20, 0xd800, 0x20}, value: 0xd800},
		{name: "short", input: []uint16{7}, value: 7},
	}
	for _, length := range [...]int{15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129} {
		input := make([]uint16, length)
		for i := range input {
			input[i] = uint16(i%1009 + 1)
		}
		utf16Cases = append(
			utf16Cases,
			struct {
				name  string
				input []uint16
				value uint16
			}{name: fmt.Sprintf("miss-length-%d", length), input: input, value: 0},
			struct {
				name  string
				input []uint16
				value uint16
			}{name: fmt.Sprintf("hit-end-length-%d", length), input: append(append([]uint16{}, input[:length-1]...), 0), value: 0},
			struct {
				name  string
				input []uint16
				value uint16
			}{name: fmt.Sprintf("hit-mid-length-%d", length), input: func() []uint16 {
				out := append([]uint16{}, input...)
				out[length/2] = 0
				return out
			}(), value: 0},
		)
	}
	for _, tc := range utf16Cases {
		t.Run("utf16/"+tc.name, func(t *testing.T) {
			got := findUTF16Archsimd(tc.input, tc.value)
			want := findUTF16Scalar(tc.input, tc.value)
			if got != want {
				t.Fatalf("findUTF16Archsimd = %d, scalar = %d", got, want)
			}
		})
	}
}

func TestMakeImplementationAMD64FindArchsimdForceable(t *testing.T) {
	requireFindArchsimdAVX2(t)

	// Scalar-first coverage lives in dispatch_amd64_test.go; this checks force selection.
	t.Setenv("SIMDUTF_FORCE_PROVIDER", "archsimd")
	forced := makeImplementation(selectionInput{features: cpuAVX2, archsimdAVX2: true})
	if !sameFunction(forced.find, findArchsimd) {
		t.Fatalf("forced find selected %p, want archsimd %p", forced.find, findArchsimd)
	}
	if !sameFunction(forced.findUTF16, findUTF16Archsimd) {
		t.Fatalf("forced findUTF16 selected %p, want archsimd %p", forced.findUTF16, findUTF16Archsimd)
	}
}

func FuzzFindArchsimdAgainstScalar(f *testing.F) {
	for _, input := range [][]byte{
		nil,
		{},
		{0},
		{1, 2, 3},
		make([]byte, 64),
		make([]byte, 65),
		make([]byte, 128),
	} {
		f.Add(input, byte(0))
		f.Add(input, byte(1))
	}
	f.Fuzz(func(t *testing.T, input []byte, value byte) {
		requireFindArchsimdAVX2(t)
		if got, want := findArchsimd(input, value), findScalar(input, value); got != want {
			t.Fatalf("findArchsimd = %d, scalar = %d (len=%d value=%d)", got, want, len(input), value)
		}
	})
}

func FuzzFindUTF16ArchsimdAgainstScalar(f *testing.F) {
	for _, input := range [][]uint16{
		nil,
		{},
		{0},
		{1, 2, 3},
		make([]uint16, 32),
		make([]uint16, 33),
		make([]uint16, 64),
	} {
		// Seed via bytes: encode little-endian pairs roughly by adding length+value.
		raw := make([]byte, len(input)*2+1)
		raw[0] = 0
		for i, v := range input {
			raw[1+2*i] = byte(v)
			raw[1+2*i+1] = byte(v >> 8)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireFindArchsimdAVX2(t)
		if len(raw) == 0 {
			if got, want := findUTF16Archsimd(nil, 0), findUTF16Scalar(nil, 0); got != want {
				t.Fatalf("findUTF16Archsimd = %d, scalar = %d", got, want)
			}
			return
		}
		value := uint16(raw[0]) | uint16(raw[0])<<8
		units := make([]uint16, (len(raw)-1)/2)
		for i := range units {
			units[i] = uint16(raw[1+2*i]) | uint16(raw[1+2*i+1])<<8
		}
		if got, want := findUTF16Archsimd(units, value), findUTF16Scalar(units, value); got != want {
			t.Fatalf("findUTF16Archsimd = %d, scalar = %d (len=%d value=%#x)", got, want, len(units), value)
		}
	})
}

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

func TestDirectArchsimdUTF16HelpersAgainstScalar(t *testing.T) {
	if detectAMD64Features()&cpuAVX2 != cpuAVX2 {
		t.Skip("AVX2 unavailable")
	}
	cases := [][]uint16{
		{},
		{0x20, 0xd800, 0xdc00, 0xffff},
		make([]uint16, 16),
		make([]uint16, 17),
		make([]uint16, 48),
	}
	for i := range cases[2] {
		cases[2][i] = uint16(0x1000 + i)
	}
	for i := range cases[3] {
		cases[3][i] = uint16(i * 11)
	}
	for i := range cases[4] {
		cases[4][i] = uint16(0xdc00 - 8 + i)
	}
	for _, input := range cases {
		checkUTF16HelpersDirect(t, input, changeEndiannessUTF16Archsimd, countUTF16LEArchsimd, countUTF16BEArchsimd, utf32LengthFromUTF16LEArchsimd, utf32LengthFromUTF16BEArchsimd)
	}
}

func requireUTF16Latin1ArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdUTF16ToLatin1AgainstScalar(t *testing.T) {
	requireUTF16Latin1ArchsimdAVX2(t)

	tests := []struct {
		name   string
		native []uint16
	}{
		{name: "nil"},
		{name: "one-ascii", native: []uint16{0x7f}},
		{name: "one-high", native: []uint16{0xff}},
		{name: "mixed-short", native: []uint16{0x00, 0x7f, 0x80, 0xff, 'A'}},
		{name: "too-large-short", native: []uint16{'a', 0x100, 'b'}},
	}
	for _, length := range [...]int{7, 8, 15, 16, 17, 31, 32, 33, 63, 64, 65} {
		tests = append(
			tests,
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("latin1-length-%d", length), native: utf16Latin1ArchsimdInput(length, false)},
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("too-large-length-%d", length), native: utf16Latin1ArchsimdInput(length, true)},
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkUTF16Latin1Archsimd(t, test.native)
		})
	}
}

func FuzzUTF16Latin1ArchsimdAgainstScalar(f *testing.F) {
	for _, native := range [][]uint16{
		nil,
		{0x00},
		{0x7f, 0x80, 0xff},
		{0x100},
		utf16Latin1ArchsimdInput(15, false),
		utf16Latin1ArchsimdInput(16, false),
		utf16Latin1ArchsimdInput(17, true),
		utf16Latin1ArchsimdInput(33, false),
	} {
		raw := make([]byte, len(native)*2)
		for i, word := range native {
			raw[2*i] = byte(word)
			raw[2*i+1] = byte(word >> 8)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireUTF16Latin1ArchsimdAVX2(t)
		if len(raw)&1 != 0 {
			raw = raw[:len(raw)&^1]
		}
		native := make([]uint16, len(raw)/2)
		for i := range native {
			native[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		checkUTF16Latin1Archsimd(t, native)
	})
}

func checkUTF16Latin1Archsimd(t *testing.T, native []uint16) {
	t.Helper()
	checkUTF16Latin1ArchsimdEndian(t, native, true)
	checkUTF16Latin1ArchsimdEndian(t, native, false)
	checkUTF16Latin1ArchsimdPreflight(t, native)
}

func checkUTF16Latin1ArchsimdEndian(t *testing.T, native []uint16, little bool) {
	t.Helper()

	input := rawUTF16Words(native, little)

	want := make([]byte, len(input))
	got := bytes.Repeat([]byte{0xa5}, len(input)+16)
	wantErrBuf := make([]byte, len(input))
	gotErrBuf := bytes.Repeat([]byte{0xa5}, len(input)+16)

	var wantN, gotN, wantValid, gotValid int
	var wantErr, gotErr Result
	if little {
		wantN = convertUTF16LEToLatin1Scalar(input, want)
		gotN = convertUTF16LEToLatin1Archsimd(input, got)
		wantErr = convertUTF16LEToLatin1WithErrorsScalar(input, wantErrBuf)
		gotErr = convertUTF16LEToLatin1WithErrorsArchsimd(input, gotErrBuf)
	} else {
		wantN = convertUTF16BEToLatin1Scalar(input, want)
		gotN = convertUTF16BEToLatin1Archsimd(input, got)
		wantErr = convertUTF16BEToLatin1WithErrorsScalar(input, wantErrBuf)
		gotErr = convertUTF16BEToLatin1WithErrorsArchsimd(input, gotErrBuf)
	}

	if gotN != wantN {
		t.Fatalf("little=%v convert = %d, want %d", little, gotN, wantN)
	}
	if wantN > 0 {
		if !bytes.Equal(got[:wantN], want[:wantN]) || !allBytes(got[wantN:], 0xa5) {
			t.Fatalf("little=%v convert output mismatch or canary overwrite", little)
		}
	} else if !allBytes(got[len(input):], 0xa5) {
		t.Fatalf("little=%v convert canary overwrite on failure", little)
	}

	if gotErr != wantErr {
		t.Fatalf("little=%v with_errors = %+v, want %+v", little, gotErr, wantErr)
	}
	if wantErr.Error == Success {
		if !bytes.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allBytes(gotErrBuf[wantErr.Count:], 0xa5) {
			t.Fatalf("little=%v with_errors output mismatch or canary overwrite", little)
		}
		wantValidBuf := make([]byte, len(input))
		gotValidBuf := bytes.Repeat([]byte{0xa5}, len(input)+16)
		if little {
			wantValid = convertValidUTF16LEToLatin1Scalar(input, wantValidBuf)
			gotValid = convertValidUTF16LEToLatin1Archsimd(input, gotValidBuf)
		} else {
			wantValid = convertValidUTF16BEToLatin1Scalar(input, wantValidBuf)
			gotValid = convertValidUTF16BEToLatin1Archsimd(input, gotValidBuf)
		}
		if gotValid != wantValid || !bytes.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allBytes(gotValidBuf[gotValid:], 0xa5) {
			t.Fatalf("little=%v valid mismatch or canary overwrite", little)
		}
	}
}

func checkUTF16Latin1ArchsimdPreflight(t *testing.T, native []uint16) {
	t.Helper()
	if len(native) == 0 {
		return
	}
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		dst := bytes.Repeat([]byte{0xa5}, len(input)-1)
		if little {
			requireUTF16Latin1ArchsimdPanic(t, func() { convertUTF16LEToLatin1Archsimd(input, dst) })
			requireUTF16Latin1ArchsimdPanic(t, func() { convertUTF16LEToLatin1WithErrorsArchsimd(input, dst) })
			requireUTF16Latin1ArchsimdPanic(t, func() { convertValidUTF16LEToLatin1Archsimd(input, dst) })
		} else {
			requireUTF16Latin1ArchsimdPanic(t, func() { convertUTF16BEToLatin1Archsimd(input, dst) })
			requireUTF16Latin1ArchsimdPanic(t, func() { convertUTF16BEToLatin1WithErrorsArchsimd(input, dst) })
			requireUTF16Latin1ArchsimdPanic(t, func() { convertValidUTF16BEToLatin1Archsimd(input, dst) })
		}
		if !allBytes(dst, 0xa5) {
			t.Fatalf("little=%v short destination was modified", little)
		}
	}
}

func requireUTF16Latin1ArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

func utf16Latin1ArchsimdInput(length int, injectTooLarge bool) []uint16 {
	input := make([]uint16, length)
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
			input[i] = uint16(i & 0xff)
		}
	}
	if injectTooLarge && length > 0 {
		input[length/2] = 0x100 + uint16(length%0xff)
	}
	return input
}

func requireUTF16UTF32ArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdUTF16ToUTF32AgainstScalar(t *testing.T) {
	requireUTF16UTF32ArchsimdAVX2(t)

	tests := []struct {
		name   string
		native []uint16
	}{
		{name: "nil"},
		{name: "one-ascii", native: []uint16{0x7f}},
		{name: "one-bmp", native: []uint16{0x20ac}},
		{name: "mixed-short", native: []uint16{0x00, 0x7f, 0x80, 0xff, 'A', 0xffff}},
		{name: "paired-surrogate", native: []uint16{0xd83d, 0xde00}},
		{name: "unpaired-high", native: []uint16{'a', 0xd800, 'b'}},
		{name: "unpaired-low", native: []uint16{'a', 0xdc00, 'b'}},
		{name: "truncated-high", native: []uint16{'a', 0xd800}},
	}
	for _, length := range [...]int{7, 8, 15, 16, 17, 31, 32, 33, 63, 64, 65} {
		tests = append(
			tests,
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("bmp-length-%d", length), native: utf16UTF32ArchsimdInput(length, false)},
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("surrogate-length-%d", length), native: utf16UTF32ArchsimdInput(length, true)},
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkUTF16UTF32Archsimd(t, test.native)
		})
	}
}

func FuzzUTF16UTF32ArchsimdAgainstScalar(f *testing.F) {
	for _, native := range [][]uint16{
		nil,
		{0x00},
		{0x7f, 0x80, 0xff, 0x20ac},
		{0xd83d, 0xde00},
		{0xd800},
		{0xdc00},
		utf16UTF32ArchsimdInput(15, false),
		utf16UTF32ArchsimdInput(16, false),
		utf16UTF32ArchsimdInput(17, true),
		utf16UTF32ArchsimdInput(33, false),
	} {
		raw := make([]byte, len(native)*2)
		for i, word := range native {
			raw[2*i] = byte(word)
			raw[2*i+1] = byte(word >> 8)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireUTF16UTF32ArchsimdAVX2(t)
		if len(raw)&1 != 0 {
			raw = raw[:len(raw)&^1]
		}
		native := make([]uint16, len(raw)/2)
		for i := range native {
			native[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		checkUTF16UTF32Archsimd(t, native)
	})
}

func checkUTF16UTF32Archsimd(t *testing.T, native []uint16) {
	t.Helper()
	checkUTF16UTF32ArchsimdEndian(t, native, true)
	checkUTF16UTF32ArchsimdEndian(t, native, false)
	checkUTF16UTF32ArchsimdPreflight(t, native)
}

func checkUTF16UTF32ArchsimdEndian(t *testing.T, native []uint16, little bool) {
	t.Helper()

	input := rawUTF16Words(native, little)
	need := utf32LengthFromUTF16LEScalar(input)
	if !little {
		need = utf32LengthFromUTF16BEScalar(input)
	}

	want := make([]uint32, need)
	got := make([]uint32, need+16)
	fillU32(got, 0xa5a5a5a5)
	wantErrBuf := make([]uint32, need)
	gotErrBuf := make([]uint32, need+16)
	fillU32(gotErrBuf, 0xa5a5a5a5)

	var wantN, gotN, wantValid, gotValid int
	var wantErr, gotErr Result
	if little {
		wantN = convertUTF16LEToUTF32Scalar(input, want)
		gotN = convertUTF16LEToUTF32Archsimd(input, got)
		wantErr = convertUTF16LEToUTF32WithErrorsScalar(input, wantErrBuf)
		gotErr = convertUTF16LEToUTF32WithErrorsArchsimd(input, gotErrBuf)
	} else {
		wantN = convertUTF16BEToUTF32Scalar(input, want)
		gotN = convertUTF16BEToUTF32Archsimd(input, got)
		wantErr = convertUTF16BEToUTF32WithErrorsScalar(input, wantErrBuf)
		gotErr = convertUTF16BEToUTF32WithErrorsArchsimd(input, gotErrBuf)
	}

	if gotN != wantN {
		t.Fatalf("little=%v convert = %d, want %d", little, gotN, wantN)
	}
	if wantN > 0 {
		if !slices.Equal(got[:wantN], want[:wantN]) || !allU32(got[wantN:], 0xa5a5a5a5) {
			t.Fatalf("little=%v convert output mismatch or canary overwrite", little)
		}
	} else if !allU32(got[need:], 0xa5a5a5a5) {
		t.Fatalf("little=%v convert canary overwrite on failure", little)
	}

	if gotErr != wantErr {
		t.Fatalf("little=%v with_errors = %+v, want %+v", little, gotErr, wantErr)
	}
	if wantErr.Error == Success {
		if !slices.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allU32(gotErrBuf[wantErr.Count:], 0xa5a5a5a5) {
			t.Fatalf("little=%v with_errors output mismatch or canary overwrite", little)
		}
		wantValidBuf := make([]uint32, need)
		gotValidBuf := make([]uint32, need+16)
		fillU32(gotValidBuf, 0xa5a5a5a5)
		if little {
			wantValid = convertValidUTF16LEToUTF32Scalar(input, wantValidBuf)
			gotValid = convertValidUTF16LEToUTF32Archsimd(input, gotValidBuf)
		} else {
			wantValid = convertValidUTF16BEToUTF32Scalar(input, wantValidBuf)
			gotValid = convertValidUTF16BEToUTF32Archsimd(input, gotValidBuf)
		}
		if gotValid != wantValid || !slices.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allU32(gotValidBuf[gotValid:], 0xa5a5a5a5) {
			t.Fatalf("little=%v valid mismatch or canary overwrite", little)
		}
	}
}

func checkUTF16UTF32ArchsimdPreflight(t *testing.T, native []uint16) {
	t.Helper()
	if len(native) == 0 {
		return
	}
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		need := utf32LengthFromUTF16LEScalar(input)
		if !little {
			need = utf32LengthFromUTF16BEScalar(input)
		}
		if need == 0 {
			continue
		}
		dst := make([]uint32, need-1)
		fillU32(dst, 0xa5a5a5a5)
		if little {
			requireUTF16UTF32ArchsimdPanic(t, func() { convertUTF16LEToUTF32Archsimd(input, dst) })
			requireUTF16UTF32ArchsimdPanic(t, func() { convertUTF16LEToUTF32WithErrorsArchsimd(input, dst) })
			requireUTF16UTF32ArchsimdPanic(t, func() { convertValidUTF16LEToUTF32Archsimd(input, dst) })
		} else {
			requireUTF16UTF32ArchsimdPanic(t, func() { convertUTF16BEToUTF32Archsimd(input, dst) })
			requireUTF16UTF32ArchsimdPanic(t, func() { convertUTF16BEToUTF32WithErrorsArchsimd(input, dst) })
			requireUTF16UTF32ArchsimdPanic(t, func() { convertValidUTF16BEToUTF32Archsimd(input, dst) })
		}
		if !allU32(dst, 0xa5a5a5a5) {
			t.Fatalf("little=%v short destination was modified", little)
		}
	}
}

func requireUTF16UTF32ArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

func utf16UTF32ArchsimdInput(length int, injectSurrogate bool) []uint16 {
	input := make([]uint16, length)
	for i := range input {
		switch i % 6 {
		case 0:
			input[i] = 0x00
		case 1:
			input[i] = 0x7f
		case 2:
			input[i] = 0x80
		case 3:
			input[i] = 0xff
		case 4:
			input[i] = 0x20ac
		default:
			input[i] = uint16(0x0100 + (i & 0xff))
		}
	}
	if injectSurrogate && length >= 2 {
		pos := length / 2
		if pos+1 >= length {
			pos = length - 2
		}
		input[pos] = 0xd83d
		input[pos+1] = 0xde00
	}
	return input
}

func requireUTF16UTF8ArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdUTF16ToUTF8AgainstScalar(t *testing.T) {
	requireUTF16UTF8ArchsimdAVX2(t)

	tests := []struct {
		name   string
		native []uint16
	}{
		{name: "nil"},
		{name: "one-ascii", native: []uint16{0x7f}},
		{name: "one-bmp", native: []uint16{0x20ac}},
		{name: "mixed-short", native: []uint16{0x00, 0x7f, 0x80, 0xff, 'A', 0xffff}},
		{name: "paired-surrogate", native: []uint16{0xd83d, 0xde00}},
		{name: "unpaired-high", native: []uint16{'a', 0xd800, 'b'}},
		{name: "unpaired-low", native: []uint16{'a', 0xdc00, 'b'}},
		{name: "truncated-high", native: []uint16{'a', 0xd800}},
	}
	for _, length := range [...]int{7, 8, 15, 16, 17, 31, 32, 33, 63, 64, 65} {
		tests = append(
			tests,
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("ascii-length-%d", length), native: utf16UTF8ArchsimdInput(length, false)},
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("surrogate-length-%d", length), native: utf16UTF8ArchsimdInput(length, true)},
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkUTF16UTF8Archsimd(t, test.native)
		})
	}
}

func FuzzUTF16UTF8ArchsimdAgainstScalar(f *testing.F) {
	for _, native := range [][]uint16{
		nil,
		{0x00},
		{0x7f, 0x80, 0xff, 0x20ac},
		{0xd83d, 0xde00},
		{0xd800},
		{0xdc00},
		utf16UTF8ArchsimdInput(15, false),
		utf16UTF8ArchsimdInput(16, false),
		utf16UTF8ArchsimdInput(17, true),
		utf16UTF8ArchsimdInput(33, false),
	} {
		raw := make([]byte, len(native)*2)
		for i, word := range native {
			raw[2*i] = byte(word)
			raw[2*i+1] = byte(word >> 8)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireUTF16UTF8ArchsimdAVX2(t)
		if len(raw)&1 != 0 {
			raw = raw[:len(raw)&^1]
		}
		native := make([]uint16, len(raw)/2)
		for i := range native {
			native[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		checkUTF16UTF8Archsimd(t, native)
	})
}

func checkUTF16UTF8Archsimd(t *testing.T, native []uint16) {
	t.Helper()
	checkUTF16UTF8ArchsimdEndian(t, native, true)
	checkUTF16UTF8ArchsimdEndian(t, native, false)
	checkUTF16UTF8ArchsimdPreflight(t, native)
}

func checkUTF16UTF8ArchsimdEndian(t *testing.T, native []uint16, little bool) {
	t.Helper()

	input := rawUTF16Words(native, little)
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
		gotLenN = utf8LengthFromUTF16LEArchsimd(input)
		wantLR = utf8LengthFromUTF16LEWithReplacementScalar(input)
		gotLR = utf8LengthFromUTF16LEWithReplacementArchsimd(input)
	} else {
		wantLenN = utf8LengthFromUTF16BEScalar(input)
		gotLenN = utf8LengthFromUTF16BEArchsimd(input)
		wantLR = utf8LengthFromUTF16BEWithReplacementScalar(input)
		gotLR = utf8LengthFromUTF16BEWithReplacementArchsimd(input)
	}
	if gotLenN != wantLenN {
		t.Fatalf("little=%v utf8_length = %d, want %d", little, gotLenN, wantLenN)
	}
	if gotLR != wantLR {
		t.Fatalf("little=%v utf8_length_with_replacement = %+v, want %+v", little, gotLR, wantLR)
	}

	want := bytes.Repeat([]byte{0xa5}, wantLen)
	got := bytes.Repeat([]byte{0xa5}, wantLen+16)
	if little {
		wantN = convertUTF16LEToUTF8Scalar(input, want)
		gotN = convertUTF16LEToUTF8Archsimd(input, got)
	} else {
		wantN = convertUTF16BEToUTF8Scalar(input, want)
		gotN = convertUTF16BEToUTF8Archsimd(input, got)
	}
	if gotN != wantN {
		t.Fatalf("little=%v convert = %d, want %d", little, gotN, wantN)
	}
	if wantN > 0 {
		if !bytes.Equal(got[:wantN], want[:wantN]) || !allBytes(got[wantN:], 0xa5) {
			t.Fatalf("little=%v convert output mismatch or canary overwrite", little)
		}
	} else if !allBytes(got[wantLen:], 0xa5) {
		t.Fatalf("little=%v convert canary overwrite on failure", little)
	}

	wantErrBuf := make([]byte, wantLen)
	gotErrBuf := bytes.Repeat([]byte{0xa5}, wantLen+16)
	if little {
		wantE = convertUTF16LEToUTF8WithErrorsScalar(input, wantErrBuf)
		gotE = convertUTF16LEToUTF8WithErrorsArchsimd(input, gotErrBuf)
	} else {
		wantE = convertUTF16BEToUTF8WithErrorsScalar(input, wantErrBuf)
		gotE = convertUTF16BEToUTF8WithErrorsArchsimd(input, gotErrBuf)
	}
	if gotE != wantE {
		t.Fatalf("little=%v with_errors = %+v, want %+v", little, gotE, wantE)
	}
	if wantE.Error == Success {
		if !bytes.Equal(gotErrBuf[:wantE.Count], wantErrBuf[:wantE.Count]) || !allBytes(gotErrBuf[wantE.Count:], 0xa5) {
			t.Fatalf("little=%v with_errors output mismatch or canary overwrite", little)
		}
	} else if !allBytes(gotErrBuf[wantLen:], 0xa5) {
		t.Fatalf("little=%v with_errors canary overwrite on failure", little)
	}

	wantRepl := make([]byte, wantLenRepl)
	gotRepl := bytes.Repeat([]byte{0xa5}, wantLenRepl+16)
	if little {
		wantR = convertUTF16LEToUTF8WithReplacementScalar(input, wantRepl)
		gotR = convertUTF16LEToUTF8WithReplacementArchsimd(input, gotRepl)
	} else {
		wantR = convertUTF16BEToUTF8WithReplacementScalar(input, wantRepl)
		gotR = convertUTF16BEToUTF8WithReplacementArchsimd(input, gotRepl)
	}
	if gotR != wantR || !bytes.Equal(gotRepl[:gotR], wantRepl[:wantR]) || !allBytes(gotRepl[gotR:], 0xa5) {
		t.Fatalf("little=%v with_replacement mismatch or canary overwrite", little)
	}

	if wantE.Error != Success {
		return
	}
	wantValidBuf := make([]byte, wantLen)
	gotValidBuf := bytes.Repeat([]byte{0xa5}, wantLen+16)
	if little {
		wantV = convertValidUTF16LEToUTF8Scalar(input, wantValidBuf)
		gotV = convertValidUTF16LEToUTF8Archsimd(input, gotValidBuf)
	} else {
		wantV = convertValidUTF16BEToUTF8Scalar(input, wantValidBuf)
		gotV = convertValidUTF16BEToUTF8Archsimd(input, gotValidBuf)
	}
	if gotV != wantV || !bytes.Equal(gotValidBuf[:gotV], wantValidBuf[:wantV]) || !allBytes(gotValidBuf[gotV:], 0xa5) {
		t.Fatalf("little=%v valid mismatch or canary overwrite", little)
	}
}

func checkUTF16UTF8ArchsimdPreflight(t *testing.T, native []uint16) {
	t.Helper()
	if len(native) == 0 {
		return
	}
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		storageIsNative := little == nativeLittleEndian()
		need := utf8LengthFromUTF16Scalar(input, storageIsNative)
		needRepl := utf8LengthFromUTF16WithReplacementScalar(input, storageIsNative)
		if need == 0 || needRepl == 0 {
			continue
		}
		dst := bytes.Repeat([]byte{0xa5}, need-1)
		dstRepl := bytes.Repeat([]byte{0xa5}, needRepl-1)
		if little {
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16LEToUTF8Archsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16LEToUTF8WithErrorsArchsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertValidUTF16LEToUTF8Archsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16LEToUTF8WithReplacementArchsimd(input, dstRepl) })
		} else {
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16BEToUTF8Archsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16BEToUTF8WithErrorsArchsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertValidUTF16BEToUTF8Archsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16BEToUTF8WithReplacementArchsimd(input, dstRepl) })
		}
		if !allBytes(dst, 0xa5) {
			t.Fatalf("little=%v short destination was modified", little)
		}
		if !allBytes(dstRepl, 0xa5) {
			t.Fatalf("little=%v short replacement destination was modified", little)
		}
	}
}

func requireUTF16UTF8ArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

func utf16UTF8ArchsimdInput(length int, injectSurrogate bool) []uint16 {
	input := make([]uint16, length)
	for i := range input {
		switch i % 6 {
		case 0:
			input[i] = 0x00
		case 1:
			input[i] = 0x7f
		case 2:
			input[i] = 0x80
		case 3:
			input[i] = 0xff
		case 4:
			input[i] = 0x20ac
		default:
			input[i] = uint16(0x0100 + (i & 0xff))
		}
	}
	if injectSurrogate && length >= 2 {
		pos := length / 2
		if pos+1 >= length {
			pos = length - 2
		}
		input[pos] = 0xd83d
		input[pos+1] = 0xde00
	}
	return input
}

func requireUTF32ConvertArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdUTF32ConvertAgainstScalar(t *testing.T) {
	requireUTF32ConvertArchsimdAVX2(t)

	tests := []struct {
		name  string
		input []uint32
	}{
		{name: "nil"},
		{name: "one-ascii", input: []uint32{0x7f}},
		{name: "one-latin1", input: []uint32{0xff}},
		{name: "mixed-short", input: []uint32{0x00, 0x7f, 0x80, 0xff, 'A'}},
		{name: "too-large-latin1", input: []uint32{'a', 0x100, 'b'}},
		{name: "bmp", input: []uint32{0x07ff, 0x0800, 0xffff}},
		{name: "surrogate", input: []uint32{'a', 0xd800, 'b'}},
		{name: "supplementary", input: []uint32{0x10000, 0x10ffff}},
		{name: "too-large", input: []uint32{'a', 0x110000, 'b'}},
	}
	for _, length := range [...]int{7, 8, 9, 15, 16, 17, 31, 32, 33} {
		tests = append(
			tests,
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("latin1-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindLatin1)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("ascii-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindASCII)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("bmp-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindBMP)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("mixed-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindMixed)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("surrogate-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindSurrogate)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("too-large-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindTooLarge)},
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkUTF32ConvertArchsimd(t, test.input)
		})
	}
}

func FuzzUTF32ConvertArchsimdAgainstScalar(f *testing.F) {
	for _, input := range [][]uint32{
		nil,
		{0x00},
		{0x7f, 0x80, 0xff},
		{0x100},
		{0xd800},
		{0x10000, 0x10ffff},
		{0x110000},
		utf32ArchsimdInput(15, utf32ArchsimdKindLatin1),
		utf32ArchsimdInput(16, utf32ArchsimdKindASCII),
		utf32ArchsimdInput(17, utf32ArchsimdKindBMP),
		utf32ArchsimdInput(33, utf32ArchsimdKindMixed),
	} {
		raw := make([]byte, len(input)*4)
		for i, word := range input {
			binary.LittleEndian.PutUint32(raw[4*i:], word)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireUTF32ConvertArchsimdAVX2(t)
		if len(raw)&3 != 0 {
			raw = raw[:len(raw)&^3]
		}
		input := make([]uint32, len(raw)/4)
		for i := range input {
			input[i] = binary.LittleEndian.Uint32(raw[4*i:])
		}
		checkUTF32ConvertArchsimd(t, input)
	})
}

func checkUTF32ConvertArchsimd(t *testing.T, input []uint32) {
	t.Helper()
	checkUTF32Latin1Archsimd(t, input)
	checkUTF32UTF8Archsimd(t, input)
	checkUTF32UTF16Archsimd(t, input)
	checkUTF32LengthArchsimd(t, input)
	checkUTF32ConvertArchsimdPreflight(t, input)
}

func checkUTF32Latin1Archsimd(t *testing.T, input []uint32) {
	t.Helper()

	want := make([]byte, len(input))
	got := bytes.Repeat([]byte{0xa5}, len(input)+16)
	wantErrBuf := make([]byte, len(input))
	gotErrBuf := bytes.Repeat([]byte{0xa5}, len(input)+16)

	wantN := convertUTF32ToLatin1Scalar(input, want)
	gotN := convertUTF32ToLatin1Archsimd(input, got)
	wantErr := convertUTF32ToLatin1WithErrorsScalar(input, wantErrBuf)
	gotErr := convertUTF32ToLatin1WithErrorsArchsimd(input, gotErrBuf)

	if gotN != wantN {
		t.Fatalf("latin1 convert = %d, want %d", gotN, wantN)
	}
	if wantN > 0 {
		if !bytes.Equal(got[:wantN], want[:wantN]) || !allBytes(got[wantN:], 0xa5) {
			t.Fatalf("latin1 convert output mismatch or canary overwrite")
		}
	} else if !allBytes(got[len(input):], 0xa5) {
		t.Fatalf("latin1 convert canary overwrite on failure")
	}

	if gotErr != wantErr {
		t.Fatalf("latin1 with_errors = %+v, want %+v", gotErr, wantErr)
	}
	if wantErr.Error == Success {
		if !bytes.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allBytes(gotErrBuf[wantErr.Count:], 0xa5) {
			t.Fatalf("latin1 with_errors output mismatch or canary overwrite")
		}
		wantValidBuf := make([]byte, len(input))
		gotValidBuf := bytes.Repeat([]byte{0xa5}, len(input)+16)
		wantValid := convertValidUTF32ToLatin1Scalar(input, wantValidBuf)
		gotValid := convertValidUTF32ToLatin1Archsimd(input, gotValidBuf)
		if gotValid != wantValid || !bytes.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allBytes(gotValidBuf[gotValid:], 0xa5) {
			t.Fatalf("latin1 valid mismatch or canary overwrite")
		}
	}
}

func checkUTF32UTF8Archsimd(t *testing.T, input []uint32) {
	t.Helper()

	need := utf8LengthFromUTF32Scalar(input)
	want := make([]byte, need)
	got := bytes.Repeat([]byte{0xa5}, need+16)
	wantErrBuf := make([]byte, need)
	gotErrBuf := bytes.Repeat([]byte{0xa5}, need+16)

	wantN := convertUTF32ToUTF8Scalar(input, want)
	gotN := convertUTF32ToUTF8Archsimd(input, got)
	wantErr := convertUTF32ToUTF8WithErrorsScalar(input, wantErrBuf)
	gotErr := convertUTF32ToUTF8WithErrorsArchsimd(input, gotErrBuf)

	if gotN != wantN {
		t.Fatalf("utf8 convert = %d, want %d", gotN, wantN)
	}
	if wantN > 0 {
		if !bytes.Equal(got[:wantN], want[:wantN]) || !allBytes(got[wantN:], 0xa5) {
			t.Fatalf("utf8 convert output mismatch or canary overwrite")
		}
	} else if !allBytes(got[need:], 0xa5) {
		t.Fatalf("utf8 convert canary overwrite on failure")
	}

	if gotErr != wantErr {
		t.Fatalf("utf8 with_errors = %+v, want %+v", gotErr, wantErr)
	}
	if wantErr.Error == Success {
		if !bytes.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allBytes(gotErrBuf[wantErr.Count:], 0xa5) {
			t.Fatalf("utf8 with_errors output mismatch or canary overwrite")
		}
		wantValidBuf := make([]byte, need)
		gotValidBuf := bytes.Repeat([]byte{0xa5}, need+16)
		wantValid := convertValidUTF32ToUTF8Scalar(input, wantValidBuf)
		gotValid := convertValidUTF32ToUTF8Archsimd(input, gotValidBuf)
		if gotValid != wantValid || !bytes.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allBytes(gotValidBuf[gotValid:], 0xa5) {
			t.Fatalf("utf8 valid mismatch or canary overwrite")
		}
	}
}

func checkUTF32UTF16Archsimd(t *testing.T, input []uint32) {
	t.Helper()
	need := utf16LengthFromUTF32Scalar(input)

	for _, little := range []bool{true, false} {
		want := make([]uint16, need)
		got := make([]uint16, need+16)
		fillU16(got, 0xa5a5)
		wantErrBuf := make([]uint16, need)
		gotErrBuf := make([]uint16, need+16)
		fillU16(gotErrBuf, 0xa5a5)

		var wantN, gotN, wantValid, gotValid int
		var wantErr, gotErr Result
		if little {
			wantN = convertUTF32ToUTF16LEScalar(input, want)
			gotN = convertUTF32ToUTF16LEArchsimd(input, got)
			wantErr = convertUTF32ToUTF16LEWithErrorsScalar(input, wantErrBuf)
			gotErr = convertUTF32ToUTF16LEWithErrorsArchsimd(input, gotErrBuf)
		} else {
			wantN = convertUTF32ToUTF16BEScalar(input, want)
			gotN = convertUTF32ToUTF16BEArchsimd(input, got)
			wantErr = convertUTF32ToUTF16BEWithErrorsScalar(input, wantErrBuf)
			gotErr = convertUTF32ToUTF16BEWithErrorsArchsimd(input, gotErrBuf)
		}

		if gotN != wantN {
			t.Fatalf("little=%v utf16 convert = %d, want %d", little, gotN, wantN)
		}
		if wantN > 0 {
			if !slices.Equal(got[:wantN], want[:wantN]) || !allU16(got[wantN:], 0xa5a5) {
				t.Fatalf("little=%v utf16 convert output mismatch or canary overwrite", little)
			}
		} else if !allU16(got[need:], 0xa5a5) {
			t.Fatalf("little=%v utf16 convert canary overwrite on failure", little)
		}

		if gotErr != wantErr {
			t.Fatalf("little=%v utf16 with_errors = %+v, want %+v", little, gotErr, wantErr)
		}
		if wantErr.Error == Success {
			if !slices.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allU16(gotErrBuf[wantErr.Count:], 0xa5a5) {
				t.Fatalf("little=%v utf16 with_errors output mismatch or canary overwrite", little)
			}
			wantValidBuf := make([]uint16, need)
			gotValidBuf := make([]uint16, need+16)
			fillU16(gotValidBuf, 0xa5a5)
			if little {
				wantValid = convertValidUTF32ToUTF16LEScalar(input, wantValidBuf)
				gotValid = convertValidUTF32ToUTF16LEArchsimd(input, gotValidBuf)
			} else {
				wantValid = convertValidUTF32ToUTF16BEScalar(input, wantValidBuf)
				gotValid = convertValidUTF32ToUTF16BEArchsimd(input, gotValidBuf)
			}
			if gotValid != wantValid || !slices.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allU16(gotValidBuf[gotValid:], 0xa5a5) {
				t.Fatalf("little=%v utf16 valid mismatch or canary overwrite", little)
			}
		}
	}
}

func checkUTF32LengthArchsimd(t *testing.T, input []uint32) {
	t.Helper()
	if got, want := utf8LengthFromUTF32Archsimd(input), utf8LengthFromUTF32Scalar(input); got != want {
		t.Fatalf("utf8 length = %d, want %d", got, want)
	}
	if got, want := utf16LengthFromUTF32Archsimd(input), utf16LengthFromUTF32Scalar(input); got != want {
		t.Fatalf("utf16 length = %d, want %d", got, want)
	}
}

func checkUTF32ConvertArchsimdPreflight(t *testing.T, input []uint32) {
	t.Helper()
	if len(input) == 0 {
		return
	}

	dstLatin1 := bytes.Repeat([]byte{0xa5}, len(input)-1)
	requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToLatin1Archsimd(input, dstLatin1) })
	requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToLatin1WithErrorsArchsimd(input, dstLatin1) })
	requireUTF32ConvertArchsimdPanic(t, func() { convertValidUTF32ToLatin1Archsimd(input, dstLatin1) })
	if !allBytes(dstLatin1, 0xa5) {
		t.Fatalf("latin1 short destination was modified")
	}

	need8 := utf8LengthFromUTF32Scalar(input)
	if need8 > 0 {
		dst8 := bytes.Repeat([]byte{0xa5}, need8-1)
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF8Archsimd(input, dst8) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF8WithErrorsArchsimd(input, dst8) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertValidUTF32ToUTF8Archsimd(input, dst8) })
		if !allBytes(dst8, 0xa5) {
			t.Fatalf("utf8 short destination was modified")
		}
	}

	need16 := utf16LengthFromUTF32Scalar(input)
	if need16 > 0 {
		dst16 := make([]uint16, need16-1)
		fillU16(dst16, 0xa5a5)
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF16LEArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF16BEArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF16LEWithErrorsArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF16BEWithErrorsArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertValidUTF32ToUTF16LEArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertValidUTF32ToUTF16BEArchsimd(input, dst16) })
		if !allU16(dst16, 0xa5a5) {
			t.Fatalf("utf16 short destination was modified")
		}
	}
}

func requireUTF32ConvertArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

type utf32ArchsimdKind int

const (
	utf32ArchsimdKindASCII utf32ArchsimdKind = iota
	utf32ArchsimdKindLatin1
	utf32ArchsimdKindBMP
	utf32ArchsimdKindMixed
	utf32ArchsimdKindSurrogate
	utf32ArchsimdKindTooLarge
)

func utf32ArchsimdInput(length int, kind utf32ArchsimdKind) []uint32 {
	input := make([]uint32, length)
	for i := range input {
		switch kind {
		case utf32ArchsimdKindASCII:
			input[i] = uint32(i & 0x7f)
		case utf32ArchsimdKindLatin1:
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
				input[i] = uint32(i & 0xff)
			}
		case utf32ArchsimdKindBMP:
			switch i % 4 {
			case 0:
				input[i] = 0x7f
			case 1:
				input[i] = 0x7ff
			case 2:
				input[i] = 0x800
			default:
				input[i] = 0xffff
			}
		case utf32ArchsimdKindMixed:
			switch i % 6 {
			case 0:
				input[i] = uint32('A' + i%26)
			case 1:
				input[i] = 0xff
			case 2:
				input[i] = 0x800
			case 3:
				input[i] = 0xffff
			case 4:
				input[i] = 0x10000 + uint32(i%0x100)
			default:
				input[i] = 0x1f600
			}
		case utf32ArchsimdKindSurrogate:
			input[i] = uint32(i & 0x7f)
		case utf32ArchsimdKindTooLarge:
			input[i] = uint32(i & 0xff)
		}
	}
	if length == 0 {
		return input
	}
	switch kind {
	case utf32ArchsimdKindSurrogate:
		input[length/2] = 0xd800 + uint32(length%0x400)
	case utf32ArchsimdKindTooLarge:
		input[length/2] = 0x110000 + uint32(length%0xff)
	case utf32ArchsimdKindLatin1:
		// keep pure latin1
	}
	return input
}

func requireUTF8ConvertArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdUTF8ConvertAgainstScalar(t *testing.T) {
	requireUTF8ConvertArchsimdAVX2(t)

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "nil"},
		{name: "empty", input: []byte{}},
		{name: "short-ascii", input: bytes.Repeat([]byte{'A'}, 15)},
		{name: "ascii-32", input: bytes.Repeat([]byte{'A'}, 32)},
		{name: "long-ascii-64", input: bytes.Repeat([]byte{'A'}, 64)},
		{name: "long-ascii-65", input: bytes.Repeat([]byte{'A'}, 65)},
		{name: "ascii-then-latin1", input: append(bytes.Repeat([]byte{'A'}, 64), []byte("caf\u00e9")...)},
		{name: "mixed-emoji", input: append(bytes.Repeat([]byte{'A'}, 32), []byte("A\U0001F600B")...)},
		{name: "mixed-arabic", input: append(bytes.Repeat([]byte{'A'}, 32), []byte("\u0645\u0631\u062d\u0628\u0627")...)},
		{name: "emoji", input: []byte("A\U0001F600B")},
		{name: "arabic", input: []byte("\u0645\u0631\u062d\u0628\u0627")},
		{name: "latin1", input: []byte("caf\u00e9")},
	}
	for _, length := range [...]int{31, 32, 33, 63, 64, 65, 127, 128, 129} {
		tests = append(tests, struct {
			name  string
			input []byte
		}{
			name:  fmt.Sprintf("ascii-length-%d", length),
			input: bytes.Repeat([]byte{'A'}, length),
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkUTF8ConvertArchsimd(t, test.input)
		})
	}
}

func TestDirectArchsimdUTF8ConvertWithErrorsAgainstScalar(t *testing.T) {
	requireUTF8ConvertArchsimdAVX2(t)
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
		{"ascii-then-header", append(bytes.Repeat([]byte{'A'}, 32), 0xff)},
		{"ascii-then-surrogate", append(bytes.Repeat([]byte{'A'}, 32), 0xed, 0xa0, 0x80)},
	}
	for _, tc := range invalids {
		t.Run(tc.name, func(t *testing.T) {
			checkUTF8ConvertErrorsArchsimd(t, tc.input)
		})
	}
}

func FuzzUTF8ConvertArchsimdAgainstScalar(f *testing.F) {
	for _, input := range [][]byte{
		nil,
		{},
		bytes.Repeat([]byte{'A'}, 32),
		bytes.Repeat([]byte{'A'}, 64),
		[]byte("caf\u00e9"),
		[]byte("A\U0001F600B"),
		[]byte("\u0645\u0631\u062d\u0628\u0627"),
		{0xc2},
		{0xff},
		append(bytes.Repeat([]byte{'A'}, 32), 0xc2),
	} {
		f.Add(input)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		requireUTF8ConvertArchsimdAVX2(t)
		checkUTF8ConvertArchsimd(t, input)
	})
}

func checkUTF8ConvertArchsimd(t *testing.T, input []byte) {
	t.Helper()

	wantLatin1Len := latin1LengthFromUTF8Scalar(input)
	want8 := bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
	got8 := bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
	wantN := convertUTF8ToLatin1Scalar(input, want8[:wantLatin1Len])
	if got := convertUTF8ToLatin1Archsimd(input, got8[:wantLatin1Len]); got != wantN || !bytes.Equal(got8, want8) {
		t.Fatal("Latin1 mismatch or canary overwrite")
	}

	want8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
	got8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
	wantE := convertUTF8ToLatin1WithErrorsScalar(input, want8[:wantLatin1Len])
	if got := convertUTF8ToLatin1WithErrorsArchsimd(input, got8[:wantLatin1Len]); got != wantE || !bytes.Equal(got8, want8) {
		t.Fatalf("Latin1 WithErrors = %#v, want %#v (or canary overwrite)", got, wantE)
	}

	if utf8.Valid(input) {
		want8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
		got8 = bytes.Repeat([]byte{0xa5}, wantLatin1Len+16)
		wantV := convertValidUTF8ToLatin1Scalar(input, want8[:wantLatin1Len])
		gotV := convertValidUTF8ToLatin1Archsimd(input, got8[:wantLatin1Len])
		if gotV != wantV || !bytes.Equal(got8, want8) {
			t.Fatalf("Valid Latin1 = %d, want %d (or payload/canary mismatch)", gotV, wantV)
		}
	}

	checkUTF8ConvertArchsimdUTF16(t, input, false)
	checkUTF8ConvertArchsimdUTF16(t, input, true)

	want32Len := utf32LengthFromUTF8Scalar(input)
	want32 := make([]uint32, want32Len)
	got32 := make([]uint32, want32Len+8)
	fillU32(got32[want32Len:], 0xa5a5a5a5)
	wantN32 := convertUTF8ToUTF32Scalar(input, want32)
	if got := convertUTF8ToUTF32Archsimd(input, got32); got != wantN32 || !equalU32(got32[:want32Len], want32) || !allU32(got32[want32Len:], 0xa5a5a5a5) {
		t.Fatal("UTF-32 mismatch or canary overwrite")
	}
	wantE32 := convertUTF8ToUTF32WithErrorsScalar(input, want32)
	if got := convertUTF8ToUTF32WithErrorsArchsimd(input, got32); got != wantE32 || !allU32(got32[want32Len:], 0xa5a5a5a5) {
		t.Fatalf("UTF-32 WithErrors = %#v, want %#v", got, wantE32)
	}
	if utf8.Valid(input) {
		wantV32 := convertValidUTF8ToUTF32Scalar(input, want32)
		if got := convertValidUTF8ToUTF32Archsimd(input, got32); got != wantV32 || !equalU32(got32[:want32Len], want32) || !allU32(got32[want32Len:], 0xa5a5a5a5) {
			t.Fatalf("Valid UTF-32 = %d, want %d", got, wantV32)
		}
	}

	checkUTF8ConvertArchsimdPreflight(t, input)
}

func checkUTF8ConvertArchsimdUTF16(t *testing.T, input []byte, bigEndian bool) {
	t.Helper()

	wantLen := utf16LengthFromUTF8Scalar(input)
	want := make([]uint16, wantLen)
	got := make([]uint16, wantLen+8)
	fillU16(got[wantLen:], 0xa5a5)

	var (
		wantN      int
		wantE      Result
		converted  int
		convertedE Result
	)
	if bigEndian {
		wantN = convertUTF8ToUTF16BEScalar(input, want)
		converted = convertUTF8ToUTF16BEArchsimd(input, got)
		wantE = convertUTF8ToUTF16BEWithErrorsScalar(input, want)
		convertedE = convertUTF8ToUTF16BEWithErrorsArchsimd(input, got)
	} else {
		wantN = convertUTF8ToUTF16LEScalar(input, want)
		converted = convertUTF8ToUTF16LEArchsimd(input, got)
		wantE = convertUTF8ToUTF16LEWithErrorsScalar(input, want)
		convertedE = convertUTF8ToUTF16LEWithErrorsArchsimd(input, got)
	}
	if converted != wantN || !equalU16(got[:wantLen], want) || !allU16(got[wantLen:], 0xa5a5) {
		t.Fatal("UTF-16 mismatch or canary overwrite")
	}
	if convertedE != wantE || !allU16(got[wantLen:], 0xa5a5) {
		t.Fatalf("UTF-16 WithErrors = %#v, want %#v", convertedE, wantE)
	}
	if utf8.Valid(input) {
		var wantV, convertedV int
		if bigEndian {
			wantV = convertValidUTF8ToUTF16BEScalar(input, want)
			convertedV = convertValidUTF8ToUTF16BEArchsimd(input, got)
		} else {
			wantV = convertValidUTF8ToUTF16LEScalar(input, want)
			convertedV = convertValidUTF8ToUTF16LEArchsimd(input, got)
		}
		if convertedV != wantV || !equalU16(got[:wantLen], want) || !allU16(got[wantLen:], 0xa5a5) {
			t.Fatalf("Valid UTF-16 = %d, want %d", convertedV, wantV)
		}
	}
}

func checkUTF8ConvertArchsimdPreflight(t *testing.T, input []byte) {
	t.Helper()
	if len(input) == 0 {
		return
	}

	required8 := latin1LengthFromUTF8Scalar(input)
	dst8 := bytes.Repeat([]byte{0xa5}, required8-1)
	requireUTF8ConvertArchsimdPanic(t, func() { convertUTF8ToLatin1Archsimd(input, dst8) })
	if !allBytes(dst8, 0xa5) {
		t.Fatal("Latin1 short destination was modified")
	}
	requireUTF8ConvertArchsimdPanic(t, func() { convertUTF8ToLatin1WithErrorsArchsimd(input, dst8) })
	if !allBytes(dst8, 0xa5) {
		t.Fatal("Latin1 WithErrors short destination was modified")
	}
	requireUTF8ConvertArchsimdPanic(t, func() { convertValidUTF8ToLatin1Archsimd(input, dst8) })
	if !allBytes(dst8, 0xa5) {
		t.Fatal("Valid Latin1 short destination was modified")
	}

	required16 := utf16LengthFromUTF8Scalar(input)
	dst16 := make([]uint16, required16-1)
	fillU16(dst16, 0xa5a5)
	requireUTF8ConvertArchsimdPanic(t, func() { convertUTF8ToUTF16LEArchsimd(input, dst16) })
	if !allU16(dst16, 0xa5a5) {
		t.Fatal("UTF-16LE short destination was modified")
	}
	requireUTF8ConvertArchsimdPanic(t, func() { convertUTF8ToUTF16BEArchsimd(input, dst16) })
	if !allU16(dst16, 0xa5a5) {
		t.Fatal("UTF-16BE short destination was modified")
	}
	requireUTF8ConvertArchsimdPanic(t, func() { convertUTF8ToUTF16LEWithErrorsArchsimd(input, dst16) })
	if !allU16(dst16, 0xa5a5) {
		t.Fatal("UTF-16LE WithErrors short destination was modified")
	}
	requireUTF8ConvertArchsimdPanic(t, func() { convertUTF8ToUTF16BEWithErrorsArchsimd(input, dst16) })
	if !allU16(dst16, 0xa5a5) {
		t.Fatal("UTF-16BE WithErrors short destination was modified")
	}
	requireUTF8ConvertArchsimdPanic(t, func() { convertValidUTF8ToUTF16LEArchsimd(input, dst16) })
	if !allU16(dst16, 0xa5a5) {
		t.Fatal("Valid UTF-16LE short destination was modified")
	}
	requireUTF8ConvertArchsimdPanic(t, func() { convertValidUTF8ToUTF16BEArchsimd(input, dst16) })
	if !allU16(dst16, 0xa5a5) {
		t.Fatal("Valid UTF-16BE short destination was modified")
	}

	required32 := utf32LengthFromUTF8Scalar(input)
	dst32 := make([]uint32, required32-1)
	fillU32(dst32, 0xa5a5a5a5)
	requireUTF8ConvertArchsimdPanic(t, func() { convertUTF8ToUTF32Archsimd(input, dst32) })
	if !allU32(dst32, 0xa5a5a5a5) {
		t.Fatal("UTF-32 short destination was modified")
	}
	requireUTF8ConvertArchsimdPanic(t, func() { convertUTF8ToUTF32WithErrorsArchsimd(input, dst32) })
	if !allU32(dst32, 0xa5a5a5a5) {
		t.Fatal("UTF-32 WithErrors short destination was modified")
	}
	requireUTF8ConvertArchsimdPanic(t, func() { convertValidUTF8ToUTF32Archsimd(input, dst32) })
	if !allU32(dst32, 0xa5a5a5a5) {
		t.Fatal("Valid UTF-32 short destination was modified")
	}
}

func checkUTF8ConvertErrorsArchsimd(t *testing.T, input []byte) {
	t.Helper()

	dst8 := make([]byte, latin1LengthFromUTF8Scalar(input)+8)
	want := convertUTF8ToLatin1WithErrorsScalar(input, dst8)
	if got := convertUTF8ToLatin1WithErrorsArchsimd(input, dst8); got != want {
		t.Fatalf("Latin1 WithErrors = %#v, want %#v", got, want)
	}

	dst16 := make([]uint16, utf16LengthFromUTF8Scalar(input)+8)
	want = convertUTF8ToUTF16LEWithErrorsScalar(input, dst16)
	if got := convertUTF8ToUTF16LEWithErrorsArchsimd(input, dst16); got != want {
		t.Fatalf("UTF-16LE WithErrors = %#v, want %#v", got, want)
	}
	want = convertUTF8ToUTF16BEWithErrorsScalar(input, dst16)
	if got := convertUTF8ToUTF16BEWithErrorsArchsimd(input, dst16); got != want {
		t.Fatalf("UTF-16BE WithErrors = %#v, want %#v", got, want)
	}

	dst32 := make([]uint32, utf32LengthFromUTF8Scalar(input)+8)
	want = convertUTF8ToUTF32WithErrorsScalar(input, dst32)
	if got := convertUTF8ToUTF32WithErrorsArchsimd(input, dst32); got != want {
		t.Fatalf("UTF-32 WithErrors = %#v, want %#v", got, want)
	}
}

func requireUTF8ConvertArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

func TestToWellFormedUTF16ArchsimdDifferential(t *testing.T) {
	cases := [][]uint16{
		nil, make([]uint16, 15), make([]uint16, 16), make([]uint16, 17),
		append(make([]uint16, 15), 0xd800),
		append(append(make([]uint16, 15), 0xd800), 0xdc00),
		{0xdc00, 0x61, 0xd800, 0x62, 0xd800, 0xdc00, 0xdc00},
	}
	for _, semantic := range cases {
		for _, bigEndian := range []bool{false, true} {
			input := append([]uint16(nil), semantic...)
			if bigEndian {
				for i := range input {
					input[i] = input[i]>>8 | input[i]<<8
				}
			}
			want := make([]uint16, len(input))
			got := make([]uint16, len(input)+2)
			got[len(input)], got[len(input)+1] = 0xa55a, 0x5aa5
			if bigEndian {
				toWellFormedUTF16BEScalar(input, want)
				toWellFormedUTF16BEArchsimd(input, got[:len(input)])
			} else {
				toWellFormedUTF16LEScalar(input, want)
				toWellFormedUTF16LEArchsimd(input, got[:len(input)])
			}
			if !slices.Equal(got[:len(input)], want) || got[len(input)] != 0xa55a || got[len(input)+1] != 0x5aa5 {
				t.Fatalf("input=%#v be=%t got=%#v want=%#v", semantic, bigEndian, got, want)
			}
			inPlace := append([]uint16(nil), input...)
			if bigEndian {
				toWellFormedUTF16BEArchsimd(inPlace, inPlace)
			} else {
				toWellFormedUTF16LEArchsimd(inPlace, inPlace)
			}
			if !slices.Equal(inPlace, want) {
				t.Fatalf("in-place input=%#v be=%t got=%#v want=%#v", semantic, bigEndian, inPlace, want)
			}
		}
	}
}

func TestToWellFormedUTF16ArchsimdShortDestinationDoesNotStore(t *testing.T) {
	for _, repair := range []func([]uint16, []uint16){toWellFormedUTF16LEArchsimd, toWellFormedUTF16BEArchsimd} {
		dst := []uint16{0xa55a}
		panicked := false
		func() {
			defer func() { panicked = recover() != nil }()
			repair([]uint16{0xd800, 0x61}, dst)
		}()
		if !panicked || dst[0] != 0xa55a {
			t.Fatalf("panicked=%t dst=%#v", panicked, dst)
		}
	}
}

func FuzzUTFValidationArchsimdAgainstScalar(f *testing.F) {
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
				gotResult = validateUTF16LEWithErrorsArchsimd(input)
				wantResult = validateUTF16LEWithErrorsScalar(input)
				toWellFormedUTF16LEArchsimd(input, gotDst)
				toWellFormedUTF16LEScalar(input, wantDst)
			} else {
				gotResult = validateUTF16BEWithErrorsArchsimd(input)
				wantResult = validateUTF16BEWithErrorsScalar(input)
				toWellFormedUTF16BEArchsimd(input, gotDst)
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
		if got, want := validateUTF32WithErrorsArchsimd(input32), validateUTF32WithErrorsScalar(input32); got != want {
			t.Fatalf("UTF-32 result=%+v want=%+v input=%x", got, want, input32)
		}
	})
}

// Independently adapted direct differential coverage for the algorithms at
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
// src/generic/ascii_validation.h:6-45 and
// src/haswell/implementation.cpp:278-307. The direct archsimd invocation guard
// follows Go 1.26.5 src/simd/archsimd/cpu_amd64.go:7-61.

var asciiArchsimdTestLengths = [...]int{
	0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129,
}

var utf16ArchsimdTestLengths = [...]int{
	0, 1, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65,
}

func requireASCIIArchsimdAVX2(t *testing.T) {
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

func TestValidateASCIIArchsimdMatchesScalar(t *testing.T) {
	requireASCIIArchsimdAVX2(t)

	for _, length := range asciiArchsimdTestLengths {
		t.Run(fmt.Sprintf("length=%d/valid", length), func(t *testing.T) {
			input := makeASCIIArchsimdInput(length)
			if got, want := validateASCIIArchsimd(input), validateASCIIScalar(input); got != want {
				t.Fatalf("validateASCIIArchsimd() = %t, want %t", got, want)
			}
			if got, want := validateASCIIWithErrorsArchsimd(input), validateASCIIWithErrorsScalar(input); got != want {
				t.Fatalf("validateASCIIWithErrorsArchsimd() = %+v, want %+v", got, want)
			}
		})

		for position := 0; position < length; position++ {
			t.Run(fmt.Sprintf("length=%d/invalid=%d", length, position), func(t *testing.T) {
				input := makeASCIIArchsimdInput(length)
				input[position] = 0x80 | byte(position&0x7f)
				if got, want := validateASCIIArchsimd(input), validateASCIIScalar(input); got != want {
					t.Fatalf("validateASCIIArchsimd() = %t, want %t", got, want)
				}
				want := Result{Error: TooLarge, Count: position}
				if got := validateASCIIWithErrorsArchsimd(input); got != want {
					t.Fatalf("validateASCIIWithErrorsArchsimd() = %+v, want %+v", got, want)
				}
			})
		}

		if length > 1 {
			t.Run(fmt.Sprintf("length=%d/multiple", length), func(t *testing.T) {
				input := makeASCIIArchsimdInput(length)
				positions := [...]int{0, length / 2, length - 1}
				for _, position := range positions {
					input[position] = 0xff
				}
				if validateASCIIArchsimd(input) {
					t.Fatal("validateASCIIArchsimd() = true, want false")
				}
				want := Result{Error: TooLarge, Count: positions[0]}
				if got := validateASCIIWithErrorsArchsimd(input); got != want {
					t.Fatalf("validateASCIIWithErrorsArchsimd() = %+v, want %+v", got, want)
				}
			})
		}
	}
}

func TestValidateUTF16AsASCIIArchsimdMatchesScalar(t *testing.T) {
	requireASCIIArchsimdAVX2(t)

	tests := [...]struct {
		name       string
		validRaw   uint16
		invalidRaw uint16
		archsimd   func([]uint16) bool
		scalar     func([]uint16) bool
	}{
		{
			name:       "little-endian-raw",
			validRaw:   0x007f,
			invalidRaw: 0x0080,
			archsimd:   validateUTF16LEAsASCIIArchsimd,
			scalar:     validateUTF16LEAsASCIIScalar,
		},
		{
			name:       "big-endian-raw",
			validRaw:   0x7f00,
			invalidRaw: 0x8000,
			archsimd:   validateUTF16BEAsASCIIArchsimd,
			scalar:     validateUTF16BEAsASCIIScalar,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, length := range utf16ArchsimdTestLengths {
				t.Run(fmt.Sprintf("length=%d/valid", length), func(t *testing.T) {
					input := make([]uint16, length)
					for i := range input {
						input[i] = test.validRaw
					}
					if got, want := test.archsimd(input), test.scalar(input); got != want {
						t.Fatalf("archsimd() = %t, want scalar %t", got, want)
					}
				})

				for position := 0; position < length; position++ {
					t.Run(fmt.Sprintf("length=%d/invalid=%d", length, position), func(t *testing.T) {
						input := make([]uint16, length)
						for i := range input {
							input[i] = test.validRaw
						}
						input[position] = test.invalidRaw
						if got, want := test.archsimd(input), test.scalar(input); got != want {
							t.Fatalf("archsimd() = %t, want scalar %t", got, want)
						}
					})
				}
			}
		})
	}
}

func makeASCIIArchsimdInput(length int) []byte {
	input := make([]byte, length)
	for i := range input {
		input[i] = byte((i*29 + 7) & 0x7f)
	}
	return input
}

// Hand-authored Go-only benchmark registration for the independent archsimd
// adaptation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f),
// src/generic/ascii_validation.h:6-45. It uses the test-only registry defined
// by ascii_direct_variants_test.go and adds no benchmark procedure or result.

func init() {
	registerASCIIDirectBenchmarkVariants(
		"archsimd",
		variant[func([]byte) bool]{
			value:     validateASCIIArchsimd,
			kind:      implementationArchsimd,
			required:  cpuAVX2,
			available: true,
		},
		variant[func([]byte) Result]{
			value:     validateASCIIWithErrorsArchsimd,
			kind:      implementationArchsimd,
			required:  cpuAVX2,
			available: true,
		},
	)
}

// Hand-authored Go-only direct fuzz registration for the archsimd adaptation
// pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.
// It registers test functions only and adds no product behavior.

func init() {
	registerASCIIFuzzVariant(asciiFuzzVariant{
		name: "archsimd",
		validate: variant[func([]byte) bool]{
			value: validateASCIIArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
		withErrors: variant[func([]byte) Result]{
			value: validateASCIIWithErrorsArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
	})
	registerUTF16ASCIIFuzzVariant(utf16ASCIIFuzzVariant{
		name: "archsimd",
		le: variant[func([]uint16) bool]{
			value: validateUTF16LEAsASCIIArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
		be: variant[func([]uint16) bool]{
			value: validateUTF16BEAsASCIIArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
	})
}

// Hand-authored Go-only direct scalar-differential coverage for the archsimd
// Haswell count_code_points_bytemask adaptation pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/utf8.h:21-68 and
// src/haswell/implementation.cpp:1115-1119.

func TestCountUTF8ArchsimdScalarParity(t *testing.T) {
	requireCountUTF8ArchsimdAVX2(t)
	lengths := []int{0, 1, 31, 32, 33, 63, 64, 65, 95, 96, 97, 127, 128, 129, 255, 256, 257, 8063, 8064, 8065, 8191, 8192, 8193}
	for _, length := range lengths {
		for alignment := 0; alignment < 32; alignment++ {
			t.Run("length="+strconv.Itoa(length)+"/alignment="+strconv.Itoa(alignment), func(t *testing.T) {
				storage := make([]byte, alignment+length+32)
				input := storage[alignment : alignment+length]
				for i := range input {
					input[i] = byte(i*131 + length*17 + alignment)
				}
				checkCountUTF8Archsimd(t, input)
			})
		}
	}
}

func TestCountUTF8ArchsimdAllByteClasses(t *testing.T) {
	requireCountUTF8ArchsimdAVX2(t)
	classes := make([]byte, 256)
	for value := range classes {
		classes[value] = byte(value)
		checkCountUTF8Archsimd(t, bytes.Repeat([]byte{byte(value)}, 129))
	}
	checkCountUTF8Archsimd(t, classes)
	checkCountUTF8Archsimd(t, append(slices.Clone(classes), classes...))
	checkCountUTF8Archsimd(t, bytes.Repeat([]byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xff}, 1400))
}

func TestCountUTF8ArchsimdAccumulatorFlushBoundaries(t *testing.T) {
	requireCountUTF8ArchsimdAVX2(t)
	for _, value := range []byte{0x00, 0x80} {
		for _, length := range []int{8063, 8064, 8065, 8191, 8192, 8193, 3*8064 + 1, 1 << 20} {
			t.Run("byte="+strconv.Itoa(int(value))+"/length="+strconv.Itoa(length), func(t *testing.T) {
				checkCountUTF8Archsimd(t, bytes.Repeat([]byte{value}, length))
			})
		}
	}
}

func TestCountUTF8ArchsimdCanariesAndImmutability(t *testing.T) {
	requireCountUTF8ArchsimdAVX2(t)
	for _, length := range []int{0, 1, 127, 128, 129, 8063, 8064, 8065, 8191, 8192, 8193} {
		guard := newGuardedSlice(37, length, 41, byte(0xa5))
		for i := range guard.body {
			guard.body[i] = byte(i*73 + length)
		}
		before := slices.Clone(guard.storage)
		checkCountUTF8Archsimd(t, guard.body)
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatalf("length %d input or canary modified", length)
		}
	}
}

func TestCountUTF8ArchsimdShortInputScalarCutoffSourceContract(t *testing.T) {
	// The pinned generic driver enters the four-vector bytemask loop only for a
	// complete 128-byte Haswell block. Lock the wrapper control flow so shorter
	// inputs return through the scalar oracle before vector state is initialized.
	source, err := os.ReadFile("count_utf8_archsimd_amd64.go")
	if err != nil {
		t.Fatal(err)
	}
	want := `func countUTF8Archsimd(input []byte) int {
	if len(input) < 128 {
		return countUTF8Scalar(input)
	}`
	if count := strings.Count(string(source), want); count != 1 {
		t.Fatalf("exact short-input scalar cutoff contract occurs %d times, want 1\n%s", count, want)
	}
}

func checkCountUTF8Archsimd(t *testing.T, input []byte) {
	t.Helper()
	if got, want := countUTF8Archsimd(input), countUTF8Scalar(input); got != want {
		t.Errorf("countUTF8Archsimd = %d, scalar = %d for %d bytes", got, want, len(input))
	}
}

func requireCountUTF8ArchsimdAVX2(t *testing.T) {
	t.Helper()
	selection := detectSelectionInput()
	if selection.features&cpuAVX2 != cpuAVX2 || !selection.archsimdAVX2 {
		t.Skip("archsimd CountUTF8 requires repository and archsimd AVX2 gates")
	}
}

// Go-only direct benchmark and differential-fuzz registration for the tagged
// CountUTF8 adaptation pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b. It changes no
// frozen benchmark name, corpus, or setup.
func init() {
	candidate := variant[func([]byte) int]{
		value: countUTF8Archsimd, kind: implementationArchsimd,
		required: cpuAVX2, available: true,
	}
	registerCountUTF8DirectVariant(countUTF8DirectVariant{name: "archsimd", variant: candidate})
	registerCountUTF8FuzzVariant(countUTF8FuzzVariant{name: "archsimd", variant: candidate})
}

// Direct differential coverage for the lookup4 algorithm at
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216 and
// src/generic/utf8_validation/utf8_validator.h:10-80.

func requireUTF8ArchsimdAVX2(t *testing.T) {
	t.Helper()
	candidate := variant[func()]{value: func() {}, kind: implementationArchsimd, required: cpuAVX2, available: true}
	if !candidate.supportedBy(detectSelectionInput()) {
		t.Skip("direct UTF-8 archsimd AVX2 implementation is unsupported")
	}
}

func TestValidateUTF8ArchsimdMatchesScalar(t *testing.T) {
	requireUTF8ArchsimdAVX2(t)

	validSequences := [][]byte{
		{'a'},
		{0xc2, 0x80},
		{0xe0, 0xa0, 0x80},
		{0xed, 0x9f, 0xbf},
		{0xf0, 0x90, 0x80, 0x80},
		{0xf4, 0x8f, 0xbf, 0xbf},
	}
	invalidSequences := [][]byte{
		{0x80},
		{0xff},
		{0xc0, 0x80},
		{0xe0, 0x80, 0x80},
		{0xed, 0xa0, 0x80},
		{0xf0, 0x80, 0x80, 0x80},
		{0xf4, 0x90, 0x80, 0x80},
		{0xf5, 0x80, 0x80, 0x80},
		{0xc2},
		{0xe1, 0x80},
		{0xf0, 0x90, 0x80},
		{0xe1, 0x80, 'x'},
	}

	inputs := [][]byte{nil, {}}
	for _, length := range []int{1, 15, 16, 17, 31, 32, 33, 61, 62, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129, 191, 192, 193} {
		inputs = append(inputs, bytes.Repeat([]byte{'a'}, length))
		for _, invalid := range invalidSequences {
			for _, position := range []int{0, length / 2, length} {
				input := bytes.Repeat([]byte{'a'}, position)
				input = append(input, invalid...)
				input = append(input, bytes.Repeat([]byte{'b'}, length-position)...)
				inputs = append(inputs, input)
			}
		}
	}
	for _, boundary := range []int{16, 32, 48, 63, 64, 65, 80, 96, 127, 128} {
		for _, sequence := range validSequences[1:] {
			for split := 1; split < len(sequence); split++ {
				input := bytes.Repeat([]byte{'a'}, boundary-split)
				input = append(input, sequence...)
				input = append(input, bytes.Repeat([]byte{'b'}, 67)...)
				inputs = append(inputs, input)
			}
		}
	}

	for i, input := range inputs {
		t.Run(fmt.Sprintf("case=%d/length=%d", i, len(input)), func(t *testing.T) {
			backing := make([]byte, len(input)+2)
			backing[0], backing[len(backing)-1] = 0xa5, 0x5a
			copy(backing[1:], input)
			before := slices.Clone(backing)
			guarded := backing[1 : len(backing)-1]
			if got, want := validateUTF8Archsimd(guarded), validateUTF8Scalar(guarded); got != want {
				t.Fatalf("validateUTF8Archsimd() = %t, scalar = %t for %x", got, want, guarded)
			}
			if got, want := validateUTF8WithErrorsArchsimd(guarded), validateUTF8WithErrorsScalar(guarded); got != want {
				t.Fatalf("validateUTF8WithErrorsArchsimd() = %+v, scalar = %+v for %x", got, want, guarded)
			}
			if !slices.Equal(backing, before) {
				t.Fatal("archsimd UTF-8 validation modified input or canaries")
			}
		})
	}
}

func TestValidateUTF8PrefixArchsimdStopsAtFirstFailingBlock(t *testing.T) {
	requireUTF8ArchsimdAVX2(t)
	for _, test := range []struct {
		position int
		want     int
	}{{30, 0}, {94, 64}, {158, 128}} {
		input := bytes.Repeat([]byte{'a'}, 192)
		input[test.position] = 0x80
		if got := validateUTF8PrefixArchsimd(input); got != test.want {
			t.Errorf("error at %d: prefix = %d, want %d", test.position, got, test.want)
		}
	}
}

func TestValidateUTF8ArchsimdLaneBridgeOrientation(t *testing.T) {
	requireUTF8ArchsimdAVX2(t)
	for _, position := range []int{16, 32, 48} {
		input := bytes.Repeat([]byte{'a'}, 64)
		input[position] = 0xff
		if got := validateUTF8PrefixArchsimd(input); got != 0 {
			t.Errorf("invalid byte at lane boundary %d: prefix = %d, want 0", position, got)
		}
		if got, want := validateUTF8WithErrorsArchsimd(input), validateUTF8WithErrorsScalar(input); got != want {
			t.Errorf("invalid byte at lane boundary %d: with errors = %+v, scalar = %+v", position, got, want)
		}
	}
	for _, boundary := range []int{16, 32, 48} {
		input := bytes.Repeat([]byte{'a'}, boundary-2)
		input = append(input, 0xf0, 0x90, 0x80, 0x80)
		input = append(input, bytes.Repeat([]byte{'b'}, 66-len(input))...)
		if got, want := validateUTF8WithErrorsArchsimd(input), validateUTF8WithErrorsScalar(input); got != want {
			t.Errorf("valid sequence across lane boundary %d: with errors = %+v, scalar = %+v", boundary, got, want)
		}
	}
}

// Go-only direct benchmark and scalar-differential fuzz registration for the
// tagged lookup4 adaptation pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b.

func init() {
	validate := variant[func([]byte) bool]{value: validateUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true}
	withErrors := variant[func([]byte) Result]{value: validateUTF8WithErrorsArchsimd, kind: implementationArchsimd, required: cpuAVX2, available: true}
	registerUTF8DirectVariant(utf8DirectVariant{name: "archsimd", validate: validate, withErrors: withErrors})
	registerUTF8FuzzVariant(utf8FuzzVariant{name: "archsimd", validate: validate, withErrors: withErrors})
}

func TestUTF8LengthArchsimdAllByteValues(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	all := make([]byte, 256)
	for value := range all {
		all[value] = byte(value)
		checkUTF8LengthArchsimd(t, bytes.Repeat([]byte{byte(value)}, 129))
	}
	checkUTF8LengthArchsimd(t, all)
	checkUTF8LengthArchsimd(t, append(slices.Clone(all), all...))
	checkUTF8LengthArchsimd(t, bytes.Repeat([]byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xff}, 1050))
}

func TestUTF8LengthArchsimdAlignmentsAndBoundaries(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	lengths := []int{
		0, 1, 31, 32, 33, 63, 64, 65, 127, 128, 129,
		4063, 4064, 4065, 8063, 8064, 8065, 8127, 8128, 8129,
	}
	for _, length := range lengths {
		for alignment := 0; alignment < 32; alignment++ {
			t.Run("length="+strconv.Itoa(length)+"/alignment="+strconv.Itoa(alignment), func(t *testing.T) {
				storage := make([]byte, alignment+length+32)
				input := storage[alignment : alignment+length]
				for i := range input {
					input[i] = byte(i*131 + length*17 + alignment)
				}
				checkUTF8LengthArchsimd(t, input)
			})
		}
	}
}

func TestUTF16LengthFromUTF8ArchsimdAccumulatorFlushes(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	for _, value := range []byte{0x00, 0x80, 0xf0, 0xff} {
		for _, length := range []int{4063, 4064, 4065, 8127, 8128, 8129, 3*4064 + 17, 1 << 20} {
			t.Run("byte="+strconv.Itoa(int(value))+"/length="+strconv.Itoa(length), func(t *testing.T) {
				input := bytes.Repeat([]byte{value}, length)
				if got, want := utf16LengthFromUTF8Archsimd(input), utf16LengthFromUTF8Scalar(input); got != want {
					t.Errorf("utf16 archsimd = %d, scalar = %d", got, want)
				}
			})
		}
	}
}

func TestUTF32LengthFromUTF8ArchsimdBlocksAndTails(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	for _, length := range []int{0, 1, 31, 32, 33, 63, 64, 65, 127, 128, 129, 191, 192, 193, 4095, 4096, 4097} {
		input := make([]byte, length)
		for i := range input {
			input[i] = []byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xff}[i&7]
		}
		if got, want := utf32LengthFromUTF8Archsimd(input), utf32LengthFromUTF8Scalar(input); got != want {
			t.Errorf("length %d: utf32 archsimd = %d, scalar = %d", length, got, want)
		}
	}
}

func TestUTF8LengthArchsimdCanariesAndImmutability(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	for _, length := range []int{0, 1, 31, 32, 33, 63, 64, 65, 127, 128, 129, 4063, 4064, 4065, 8127, 8128, 8129} {
		guard := newGuardedSlice(37, length, 41, byte(0xa5))
		for i := range guard.body {
			guard.body[i] = byte(i*73 + length)
		}
		before := slices.Clone(guard.storage)
		checkUTF8LengthArchsimd(t, guard.body)
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatalf("length %d input or canary modified", length)
		}
	}
}

func TestUTF8LengthArchsimdSourceContracts(t *testing.T) {
	source, err := os.ReadFile("utf8_length_archsimd_amd64.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	contracts := map[string]int{
		"return countUTF8Archsimd(input)":                       1,
		"chunk.Min(fourByteThreshold).Equal(fourByteThreshold)": 1,
		"if iterations == 127":                                  1,
		"local.SumAbsDiff(zero)":                                2,
		"mask := uint64(mask0) | uint64(mask1)<<32":             1,
		"bits.OnesCount64(mask)":                                1,
	}
	for contract, want := range contracts {
		if got := strings.Count(text, contract); got != want {
			t.Errorf("source contract %q occurs %d times, want %d", contract, got, want)
		}
	}
	if strings.Contains(text, ".GreaterEqual(") {
		t.Error("generic unsigned GreaterEqual must not implement the four-byte test")
	}
	if strings.Contains(text, "countUTF8Archsimd(input[offset:])") {
		t.Error("UTF-32 archsimd must not reuse CountUTF8 for its SIMD blocks")
	}
}

func checkUTF8LengthArchsimd(t *testing.T, input []byte) {
	t.Helper()
	checks := []struct {
		name string
		got  int
		want int
	}{
		{name: "latin1", got: latin1LengthFromUTF8Archsimd(input), want: latin1LengthFromUTF8Scalar(input)},
		{name: "utf16", got: utf16LengthFromUTF8Archsimd(input), want: utf16LengthFromUTF8Scalar(input)},
		{name: "utf32", got: utf32LengthFromUTF8Archsimd(input), want: utf32LengthFromUTF8Scalar(input)},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s archsimd = %d, scalar = %d for %d bytes", check.name, check.got, check.want, len(input))
		}
	}
}

func requireUTF8LengthArchsimdFeatures(t *testing.T) {
	t.Helper()
	selection := detectSelectionInput()
	if selection.features&(cpuAVX2|cpuPOPCNT) != cpuAVX2|cpuPOPCNT || !selection.archsimdAVX2 {
		t.Skip("archsimd UTF-8 length tests require repository AVX2/POPCNT and archsimd AVX2 gates")
	}
}

func init() {
	registerUTF8LengthDirectVariant(utf8LengthDirectVariant{
		name:   "archsimd",
		latin1: variant[func([]byte) int]{value: latin1LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
		utf16:  variant[func([]byte) int]{value: utf16LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
		utf32:  variant[func([]byte) int]{value: utf32LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2 | cpuPOPCNT, available: true},
	})
}

func TestUTF8LengthArchsimdDirectRegistration(t *testing.T) {
	candidate := findUTF8LengthArchsimdDirectVariant(t)
	checkUTF8LengthArchsimdRegistration(t, candidate.latin1, candidate.utf16, candidate.utf32)
}

func findUTF8LengthArchsimdDirectVariant(t *testing.T) utf8LengthDirectVariant {
	t.Helper()
	var found *utf8LengthDirectVariant
	for i := range utf8LengthDirectVariants {
		if utf8LengthDirectVariants[i].name != "archsimd" {
			continue
		}
		if found != nil {
			t.Fatal("duplicate archsimd direct registration")
		}
		found = &utf8LengthDirectVariants[i]
	}
	if found == nil {
		t.Fatal("archsimd direct registration not found")
	}
	return *found
}

func checkUTF8LengthArchsimdRegistration(
	t *testing.T,
	latin1, utf16, utf32 variant[func([]byte) int],
) {
	t.Helper()
	checks := []struct {
		name     string
		cell     variant[func([]byte) int]
		want     func([]byte) int
		required cpuFeatures
	}{
		{name: "latin1", cell: latin1, want: latin1LengthFromUTF8Archsimd, required: cpuAVX2},
		{name: "utf16", cell: utf16, want: utf16LengthFromUTF8Archsimd, required: cpuAVX2},
		{name: "utf32", cell: utf32, want: utf32LengthFromUTF8Archsimd, required: cpuAVX2 | cpuPOPCNT},
	}
	for _, check := range checks {
		if !sameFunction(check.cell.value, check.want) || check.cell.kind != implementationArchsimd ||
			check.cell.required != check.required || !check.cell.available {
			t.Errorf("%s metadata/function mismatch: kind %d required %#x available %t",
				check.name, check.cell.kind, check.cell.required, check.cell.available)
		}
		if !check.cell.supportedBy(selectionInput{features: check.required, archsimdAVX2: true}) {
			t.Errorf("%s unsupported with all required gates", check.name)
		}
		if check.cell.supportedBy(selectionInput{features: check.required}) {
			t.Errorf("%s supported without archsimd AVX2 gate", check.name)
		}
		for feature := cpuFeatures(1); feature <= cpuNEON; feature <<= 1 {
			if check.required&feature == 0 {
				continue
			}
			if check.cell.supportedBy(selectionInput{features: check.required &^ feature, archsimdAVX2: true}) {
				t.Errorf("%s supported with feature %#x missing", check.name, feature)
			}
		}
	}
}

func init() {
	registerUTF8LengthFuzzVariant(utf8LengthFuzzVariant{
		name:   "archsimd",
		latin1: variant[func([]byte) int]{value: latin1LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
		utf16:  variant[func([]byte) int]{value: utf16LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
		utf32:  variant[func([]byte) int]{value: utf32LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2 | cpuPOPCNT, available: true},
	})
}

func TestUTF8LengthArchsimdFuzzRegistration(t *testing.T) {
	var found *utf8LengthFuzzVariant
	for i := range utf8LengthFuzzVariants {
		if utf8LengthFuzzVariants[i].name != "archsimd" {
			continue
		}
		if found != nil {
			t.Fatal("duplicate archsimd fuzz registration")
		}
		found = &utf8LengthFuzzVariants[i]
	}
	if found == nil {
		t.Fatal("archsimd fuzz registration not found")
	}
	checkUTF8LengthArchsimdRegistration(t, found.latin1, found.utf16, found.utf32)
}

func TestValidateUTF16ArchsimdDifferential(t *testing.T) {
	cases := [][]uint16{
		nil,
		make([]uint16, 15), make([]uint16, 16), make([]uint16, 17),
		append(make([]uint16, 15), 0xd800),
		append(make([]uint16, 16), 0xdc00),
		append(append(make([]uint16, 15), 0xd800), 0xdc00),
		{0xdc00, 0xd800, 0x61},
		{0xd800, 0x61},
		{0xd800},
	}
	for _, input := range cases {
		for _, bigEndian := range []bool{false, true} {
			raw := append([]uint16(nil), input...)
			if bigEndian {
				for i := range raw {
					raw[i] = raw[i]>>8 | raw[i]<<8
				}
			}
			var got, want Result
			var gotBool, wantBool bool
			if bigEndian {
				got, want = validateUTF16BEWithErrorsArchsimd(raw), validateUTF16BEWithErrorsScalar(raw)
				gotBool, wantBool = validateUTF16BEArchsimd(raw), validateUTF16BEScalar(raw)
			} else {
				got, want = validateUTF16LEWithErrorsArchsimd(raw), validateUTF16LEWithErrorsScalar(raw)
				gotBool, wantBool = validateUTF16LEArchsimd(raw), validateUTF16LEScalar(raw)
			}
			if got != want || gotBool != wantBool {
				t.Fatalf("input=%#v be=%t got=(%v,%t) want=(%v,%t)", input, bigEndian, got, gotBool, want, wantBool)
			}
		}
	}
}

func TestValidateUTF32ArchsimdDifferential(t *testing.T) {
	cases := [][]uint32{
		nil, make([]uint32, 7), make([]uint32, 8), make([]uint32, 9),
		{0x10ffff, 0x61, 0xd7ff},
		{0xd800},
		{0xdfff},
		{0x110000},
		append(make([]uint32, 7), 0x110000), append(make([]uint32, 8), 0xd800),
	}
	for _, input := range cases {
		got, want := validateUTF32WithErrorsArchsimd(input), validateUTF32WithErrorsScalar(input)
		gotBool, wantBool := validateUTF32Archsimd(input), validateUTF32Scalar(input)
		if got != want || gotBool != wantBool {
			t.Fatalf("input=%#v got=(%v,%t) want=(%v,%t)", input, got, gotBool, want, wantBool)
		}
	}
}

// Hand-authored Go-only direct Find and DetectEncodings differential
// fuzz registration for the tagged archsimd adaptations pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/find.h and src/fallback/implementation.cpp:8-32.
func init() {
	registerFindFuzzVariant(findFuzzVariant{
		name: "archsimd",
		variant: variant[func([]byte, byte) int]{
			value: findArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
	})
	registerFindUTF16FuzzVariant(findUTF16FuzzVariant{
		name: "archsimd",
		variant: variant[func([]uint16, uint16) int]{
			value: findUTF16Archsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
	})
	registerDetectEncodingsFuzzVariant(detectEncodingsFuzzVariant{
		name: "archsimd",
		variant: variant[func([]byte) Encoding]{
			value: detectEncodingsArchsimd, kind: implementationArchsimd,
			required: cpuAVX2, available: true,
		},
	})
}

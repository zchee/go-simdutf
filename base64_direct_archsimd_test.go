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
	"testing"
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

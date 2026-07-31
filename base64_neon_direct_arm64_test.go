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

// Package simdutf provides a Go port of https://github.com/simdutf/simdutf.
//
// Unicode routines (UTF8, UTF16, UTF32) and Base64: billions of characters per second using SSE2, AVX2, NEON, AVX-512, RISC-V Vector Extension, LoongArch64, POWER.
//go:build arm64

package simdutf

import (
	"bytes"
	"testing"
)

func TestBase64NEONEncodeMatchesScalar(t *testing.T) {
	inputs := [][]byte{
		[]byte("Hello, simdutf Base64!"), // 22 bytes - no NEON blocks
		make([]byte, 48),                 // exactly one block
		make([]byte, 96),                 // two blocks
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
				t.Fatalf("opt=%v len=%d neon=%q scalar=%q", opt, len(in), dstN[:min(nN,96)], dstS[:min(nS,96)])
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
	for i := 0; i < n; i++ {
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

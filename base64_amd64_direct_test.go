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
	"testing"
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

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

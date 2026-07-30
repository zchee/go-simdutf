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

// Direct differential coverage for the amd64 UTF-16→UTF-8 Westmere/Haswell
// translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_utf8.cpp
// and src/haswell/avx2_convert_utf16_to_utf8.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// Providers are invoked directly; feature skips only gate unavailable ISA.

import (
	"bytes"
	"testing"
)

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

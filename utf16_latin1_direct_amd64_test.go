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

// Direct differential coverage for the amd64 UTF-16→Latin-1 Westmere/Haswell
// translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_latin1.cpp
// and src/haswell/avx2_convert_utf16_to_latin1.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// Providers are invoked directly; feature skips only gate unavailable ISA.

import (
	"bytes"
	"testing"
)

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

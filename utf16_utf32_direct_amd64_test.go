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

// Direct differential coverage for the amd64 UTF-16→UTF-32 Westmere/Haswell
// translation of simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_utf32.cpp
// and src/haswell/avx2_convert_utf16_to_utf32.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// Providers are invoked directly; feature skips only gate unavailable ISA.

import "testing"

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

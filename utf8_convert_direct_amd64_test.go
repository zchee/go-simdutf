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
	"unicode/utf8"
)

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
			checkUTF8ConvertDirectAMD64(t, input,
				convertUTF8ToLatin1Westmere, convertUTF8ToLatin1WithErrorsWestmere, convertValidUTF8ToLatin1Westmere,
				convertUTF8ToUTF16LEWestmere, convertUTF8ToUTF16BEWestmere,
				convertUTF8ToUTF16LEWithErrorsWestmere, convertUTF8ToUTF16BEWithErrorsWestmere,
				convertValidUTF8ToUTF16LEWestmere, convertValidUTF8ToUTF16BEWestmere,
				convertUTF8ToUTF32Westmere, convertUTF8ToUTF32WithErrorsWestmere, convertValidUTF8ToUTF32Westmere,
			)
		}
		if detectAMD64Features()&cpuAVX2 == cpuAVX2 {
			checkUTF8ConvertDirectAMD64(t, input,
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

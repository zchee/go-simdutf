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

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
	"fmt"
	"testing"
	"unicode/utf8"
)

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

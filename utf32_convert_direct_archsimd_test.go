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
	"encoding/binary"
	"fmt"
	"slices"
	"testing"
)

func requireUTF32ConvertArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdUTF32ConvertAgainstScalar(t *testing.T) {
	requireUTF32ConvertArchsimdAVX2(t)

	tests := []struct {
		name  string
		input []uint32
	}{
		{name: "nil"},
		{name: "one-ascii", input: []uint32{0x7f}},
		{name: "one-latin1", input: []uint32{0xff}},
		{name: "mixed-short", input: []uint32{0x00, 0x7f, 0x80, 0xff, 'A'}},
		{name: "too-large-latin1", input: []uint32{'a', 0x100, 'b'}},
		{name: "bmp", input: []uint32{0x07ff, 0x0800, 0xffff}},
		{name: "surrogate", input: []uint32{'a', 0xd800, 'b'}},
		{name: "supplementary", input: []uint32{0x10000, 0x10ffff}},
		{name: "too-large", input: []uint32{'a', 0x110000, 'b'}},
	}
	for _, length := range [...]int{7, 8, 9, 15, 16, 17, 31, 32, 33} {
		tests = append(
			tests,
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("latin1-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindLatin1)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("ascii-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindASCII)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("bmp-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindBMP)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("mixed-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindMixed)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("surrogate-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindSurrogate)},
			struct {
				name  string
				input []uint32
			}{name: fmt.Sprintf("too-large-length-%d", length), input: utf32ArchsimdInput(length, utf32ArchsimdKindTooLarge)},
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkUTF32ConvertArchsimd(t, test.input)
		})
	}
}

func FuzzUTF32ConvertArchsimdAgainstScalar(f *testing.F) {
	for _, input := range [][]uint32{
		nil,
		{0x00},
		{0x7f, 0x80, 0xff},
		{0x100},
		{0xd800},
		{0x10000, 0x10ffff},
		{0x110000},
		utf32ArchsimdInput(15, utf32ArchsimdKindLatin1),
		utf32ArchsimdInput(16, utf32ArchsimdKindASCII),
		utf32ArchsimdInput(17, utf32ArchsimdKindBMP),
		utf32ArchsimdInput(33, utf32ArchsimdKindMixed),
	} {
		raw := make([]byte, len(input)*4)
		for i, word := range input {
			binary.LittleEndian.PutUint32(raw[4*i:], word)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireUTF32ConvertArchsimdAVX2(t)
		if len(raw)&3 != 0 {
			raw = raw[:len(raw)&^3]
		}
		input := make([]uint32, len(raw)/4)
		for i := range input {
			input[i] = binary.LittleEndian.Uint32(raw[4*i:])
		}
		checkUTF32ConvertArchsimd(t, input)
	})
}

func checkUTF32ConvertArchsimd(t *testing.T, input []uint32) {
	t.Helper()
	checkUTF32Latin1Archsimd(t, input)
	checkUTF32UTF8Archsimd(t, input)
	checkUTF32UTF16Archsimd(t, input)
	checkUTF32LengthArchsimd(t, input)
	checkUTF32ConvertArchsimdPreflight(t, input)
}

func checkUTF32Latin1Archsimd(t *testing.T, input []uint32) {
	t.Helper()

	want := make([]byte, len(input))
	got := bytes.Repeat([]byte{0xa5}, len(input)+16)
	wantErrBuf := make([]byte, len(input))
	gotErrBuf := bytes.Repeat([]byte{0xa5}, len(input)+16)

	wantN := convertUTF32ToLatin1Scalar(input, want)
	gotN := convertUTF32ToLatin1Archsimd(input, got)
	wantErr := convertUTF32ToLatin1WithErrorsScalar(input, wantErrBuf)
	gotErr := convertUTF32ToLatin1WithErrorsArchsimd(input, gotErrBuf)

	if gotN != wantN {
		t.Fatalf("latin1 convert = %d, want %d", gotN, wantN)
	}
	if wantN > 0 {
		if !bytes.Equal(got[:wantN], want[:wantN]) || !allBytes(got[wantN:], 0xa5) {
			t.Fatalf("latin1 convert output mismatch or canary overwrite")
		}
	} else if !allBytes(got[len(input):], 0xa5) {
		t.Fatalf("latin1 convert canary overwrite on failure")
	}

	if gotErr != wantErr {
		t.Fatalf("latin1 with_errors = %+v, want %+v", gotErr, wantErr)
	}
	if wantErr.Error == Success {
		if !bytes.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allBytes(gotErrBuf[wantErr.Count:], 0xa5) {
			t.Fatalf("latin1 with_errors output mismatch or canary overwrite")
		}
		wantValidBuf := make([]byte, len(input))
		gotValidBuf := bytes.Repeat([]byte{0xa5}, len(input)+16)
		wantValid := convertValidUTF32ToLatin1Scalar(input, wantValidBuf)
		gotValid := convertValidUTF32ToLatin1Archsimd(input, gotValidBuf)
		if gotValid != wantValid || !bytes.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allBytes(gotValidBuf[gotValid:], 0xa5) {
			t.Fatalf("latin1 valid mismatch or canary overwrite")
		}
	}
}

func checkUTF32UTF8Archsimd(t *testing.T, input []uint32) {
	t.Helper()

	need := utf8LengthFromUTF32Scalar(input)
	want := make([]byte, need)
	got := bytes.Repeat([]byte{0xa5}, need+16)
	wantErrBuf := make([]byte, need)
	gotErrBuf := bytes.Repeat([]byte{0xa5}, need+16)

	wantN := convertUTF32ToUTF8Scalar(input, want)
	gotN := convertUTF32ToUTF8Archsimd(input, got)
	wantErr := convertUTF32ToUTF8WithErrorsScalar(input, wantErrBuf)
	gotErr := convertUTF32ToUTF8WithErrorsArchsimd(input, gotErrBuf)

	if gotN != wantN {
		t.Fatalf("utf8 convert = %d, want %d", gotN, wantN)
	}
	if wantN > 0 {
		if !bytes.Equal(got[:wantN], want[:wantN]) || !allBytes(got[wantN:], 0xa5) {
			t.Fatalf("utf8 convert output mismatch or canary overwrite")
		}
	} else if !allBytes(got[need:], 0xa5) {
		t.Fatalf("utf8 convert canary overwrite on failure")
	}

	if gotErr != wantErr {
		t.Fatalf("utf8 with_errors = %+v, want %+v", gotErr, wantErr)
	}
	if wantErr.Error == Success {
		if !bytes.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allBytes(gotErrBuf[wantErr.Count:], 0xa5) {
			t.Fatalf("utf8 with_errors output mismatch or canary overwrite")
		}
		wantValidBuf := make([]byte, need)
		gotValidBuf := bytes.Repeat([]byte{0xa5}, need+16)
		wantValid := convertValidUTF32ToUTF8Scalar(input, wantValidBuf)
		gotValid := convertValidUTF32ToUTF8Archsimd(input, gotValidBuf)
		if gotValid != wantValid || !bytes.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allBytes(gotValidBuf[gotValid:], 0xa5) {
			t.Fatalf("utf8 valid mismatch or canary overwrite")
		}
	}
}

func checkUTF32UTF16Archsimd(t *testing.T, input []uint32) {
	t.Helper()
	need := utf16LengthFromUTF32Scalar(input)

	for _, little := range []bool{true, false} {
		want := make([]uint16, need)
		got := make([]uint16, need+16)
		fillU16(got, 0xa5a5)
		wantErrBuf := make([]uint16, need)
		gotErrBuf := make([]uint16, need+16)
		fillU16(gotErrBuf, 0xa5a5)

		var wantN, gotN, wantValid, gotValid int
		var wantErr, gotErr Result
		if little {
			wantN = convertUTF32ToUTF16LEScalar(input, want)
			gotN = convertUTF32ToUTF16LEArchsimd(input, got)
			wantErr = convertUTF32ToUTF16LEWithErrorsScalar(input, wantErrBuf)
			gotErr = convertUTF32ToUTF16LEWithErrorsArchsimd(input, gotErrBuf)
		} else {
			wantN = convertUTF32ToUTF16BEScalar(input, want)
			gotN = convertUTF32ToUTF16BEArchsimd(input, got)
			wantErr = convertUTF32ToUTF16BEWithErrorsScalar(input, wantErrBuf)
			gotErr = convertUTF32ToUTF16BEWithErrorsArchsimd(input, gotErrBuf)
		}

		if gotN != wantN {
			t.Fatalf("little=%v utf16 convert = %d, want %d", little, gotN, wantN)
		}
		if wantN > 0 {
			if !slices.Equal(got[:wantN], want[:wantN]) || !allU16(got[wantN:], 0xa5a5) {
				t.Fatalf("little=%v utf16 convert output mismatch or canary overwrite", little)
			}
		} else if !allU16(got[need:], 0xa5a5) {
			t.Fatalf("little=%v utf16 convert canary overwrite on failure", little)
		}

		if gotErr != wantErr {
			t.Fatalf("little=%v utf16 with_errors = %+v, want %+v", little, gotErr, wantErr)
		}
		if wantErr.Error == Success {
			if !slices.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allU16(gotErrBuf[wantErr.Count:], 0xa5a5) {
				t.Fatalf("little=%v utf16 with_errors output mismatch or canary overwrite", little)
			}
			wantValidBuf := make([]uint16, need)
			gotValidBuf := make([]uint16, need+16)
			fillU16(gotValidBuf, 0xa5a5)
			if little {
				wantValid = convertValidUTF32ToUTF16LEScalar(input, wantValidBuf)
				gotValid = convertValidUTF32ToUTF16LEArchsimd(input, gotValidBuf)
			} else {
				wantValid = convertValidUTF32ToUTF16BEScalar(input, wantValidBuf)
				gotValid = convertValidUTF32ToUTF16BEArchsimd(input, gotValidBuf)
			}
			if gotValid != wantValid || !slices.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allU16(gotValidBuf[gotValid:], 0xa5a5) {
				t.Fatalf("little=%v utf16 valid mismatch or canary overwrite", little)
			}
		}
	}
}

func checkUTF32LengthArchsimd(t *testing.T, input []uint32) {
	t.Helper()
	if got, want := utf8LengthFromUTF32Archsimd(input), utf8LengthFromUTF32Scalar(input); got != want {
		t.Fatalf("utf8 length = %d, want %d", got, want)
	}
	if got, want := utf16LengthFromUTF32Archsimd(input), utf16LengthFromUTF32Scalar(input); got != want {
		t.Fatalf("utf16 length = %d, want %d", got, want)
	}
}

func checkUTF32ConvertArchsimdPreflight(t *testing.T, input []uint32) {
	t.Helper()
	if len(input) == 0 {
		return
	}

	dstLatin1 := bytes.Repeat([]byte{0xa5}, len(input)-1)
	requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToLatin1Archsimd(input, dstLatin1) })
	requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToLatin1WithErrorsArchsimd(input, dstLatin1) })
	requireUTF32ConvertArchsimdPanic(t, func() { convertValidUTF32ToLatin1Archsimd(input, dstLatin1) })
	if !allBytes(dstLatin1, 0xa5) {
		t.Fatalf("latin1 short destination was modified")
	}

	need8 := utf8LengthFromUTF32Scalar(input)
	if need8 > 0 {
		dst8 := bytes.Repeat([]byte{0xa5}, need8-1)
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF8Archsimd(input, dst8) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF8WithErrorsArchsimd(input, dst8) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertValidUTF32ToUTF8Archsimd(input, dst8) })
		if !allBytes(dst8, 0xa5) {
			t.Fatalf("utf8 short destination was modified")
		}
	}

	need16 := utf16LengthFromUTF32Scalar(input)
	if need16 > 0 {
		dst16 := make([]uint16, need16-1)
		fillU16(dst16, 0xa5a5)
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF16LEArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF16BEArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF16LEWithErrorsArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertUTF32ToUTF16BEWithErrorsArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertValidUTF32ToUTF16LEArchsimd(input, dst16) })
		requireUTF32ConvertArchsimdPanic(t, func() { convertValidUTF32ToUTF16BEArchsimd(input, dst16) })
		if !allU16(dst16, 0xa5a5) {
			t.Fatalf("utf16 short destination was modified")
		}
	}
}

func requireUTF32ConvertArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

type utf32ArchsimdKind int

const (
	utf32ArchsimdKindASCII utf32ArchsimdKind = iota
	utf32ArchsimdKindLatin1
	utf32ArchsimdKindBMP
	utf32ArchsimdKindMixed
	utf32ArchsimdKindSurrogate
	utf32ArchsimdKindTooLarge
)

func utf32ArchsimdInput(length int, kind utf32ArchsimdKind) []uint32 {
	input := make([]uint32, length)
	for i := range input {
		switch kind {
		case utf32ArchsimdKindASCII:
			input[i] = uint32(i & 0x7f)
		case utf32ArchsimdKindLatin1:
			switch i % 5 {
			case 0:
				input[i] = 0x00
			case 1:
				input[i] = 0x7f
			case 2:
				input[i] = 0x80
			case 3:
				input[i] = 0xff
			default:
				input[i] = uint32(i & 0xff)
			}
		case utf32ArchsimdKindBMP:
			switch i % 4 {
			case 0:
				input[i] = 0x7f
			case 1:
				input[i] = 0x7ff
			case 2:
				input[i] = 0x800
			default:
				input[i] = 0xffff
			}
		case utf32ArchsimdKindMixed:
			switch i % 6 {
			case 0:
				input[i] = uint32('A' + i%26)
			case 1:
				input[i] = 0xff
			case 2:
				input[i] = 0x800
			case 3:
				input[i] = 0xffff
			case 4:
				input[i] = 0x10000 + uint32(i%0x100)
			default:
				input[i] = 0x1f600
			}
		case utf32ArchsimdKindSurrogate:
			input[i] = uint32(i & 0x7f)
		case utf32ArchsimdKindTooLarge:
			input[i] = uint32(i & 0xff)
		}
	}
	if length == 0 {
		return input
	}
	switch kind {
	case utf32ArchsimdKindSurrogate:
		input[length/2] = 0xd800 + uint32(length%0x400)
	case utf32ArchsimdKindTooLarge:
		input[length/2] = 0x110000 + uint32(length%0xff)
	case utf32ArchsimdKindLatin1:
		// keep pure latin1
	}
	return input
}

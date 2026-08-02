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
	"fmt"
	"slices"
	"testing"
)

func requireUTF16UTF32ArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdUTF16ToUTF32AgainstScalar(t *testing.T) {
	requireUTF16UTF32ArchsimdAVX2(t)

	tests := []struct {
		name   string
		native []uint16
	}{
		{name: "nil"},
		{name: "one-ascii", native: []uint16{0x7f}},
		{name: "one-bmp", native: []uint16{0x20ac}},
		{name: "mixed-short", native: []uint16{0x00, 0x7f, 0x80, 0xff, 'A', 0xffff}},
		{name: "paired-surrogate", native: []uint16{0xd83d, 0xde00}},
		{name: "unpaired-high", native: []uint16{'a', 0xd800, 'b'}},
		{name: "unpaired-low", native: []uint16{'a', 0xdc00, 'b'}},
		{name: "truncated-high", native: []uint16{'a', 0xd800}},
	}
	for _, length := range [...]int{7, 8, 15, 16, 17, 31, 32, 33, 63, 64, 65} {
		tests = append(
			tests,
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("bmp-length-%d", length), native: utf16UTF32ArchsimdInput(length, false)},
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("surrogate-length-%d", length), native: utf16UTF32ArchsimdInput(length, true)},
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkUTF16UTF32Archsimd(t, test.native)
		})
	}
}

func FuzzUTF16UTF32ArchsimdAgainstScalar(f *testing.F) {
	for _, native := range [][]uint16{
		nil,
		{0x00},
		{0x7f, 0x80, 0xff, 0x20ac},
		{0xd83d, 0xde00},
		{0xd800},
		{0xdc00},
		utf16UTF32ArchsimdInput(15, false),
		utf16UTF32ArchsimdInput(16, false),
		utf16UTF32ArchsimdInput(17, true),
		utf16UTF32ArchsimdInput(33, false),
	} {
		raw := make([]byte, len(native)*2)
		for i, word := range native {
			raw[2*i] = byte(word)
			raw[2*i+1] = byte(word >> 8)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireUTF16UTF32ArchsimdAVX2(t)
		if len(raw)&1 != 0 {
			raw = raw[:len(raw)&^1]
		}
		native := make([]uint16, len(raw)/2)
		for i := range native {
			native[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		checkUTF16UTF32Archsimd(t, native)
	})
}

func checkUTF16UTF32Archsimd(t *testing.T, native []uint16) {
	t.Helper()
	checkUTF16UTF32ArchsimdEndian(t, native, true)
	checkUTF16UTF32ArchsimdEndian(t, native, false)
	checkUTF16UTF32ArchsimdPreflight(t, native)
}

func checkUTF16UTF32ArchsimdEndian(t *testing.T, native []uint16, little bool) {
	t.Helper()

	input := rawUTF16Words(native, little)
	need := utf32LengthFromUTF16LEScalar(input)
	if !little {
		need = utf32LengthFromUTF16BEScalar(input)
	}

	want := make([]uint32, need)
	got := make([]uint32, need+16)
	fillU32(got, 0xa5a5a5a5)
	wantErrBuf := make([]uint32, need)
	gotErrBuf := make([]uint32, need+16)
	fillU32(gotErrBuf, 0xa5a5a5a5)

	var wantN, gotN, wantValid, gotValid int
	var wantErr, gotErr Result
	if little {
		wantN = convertUTF16LEToUTF32Scalar(input, want)
		gotN = convertUTF16LEToUTF32Archsimd(input, got)
		wantErr = convertUTF16LEToUTF32WithErrorsScalar(input, wantErrBuf)
		gotErr = convertUTF16LEToUTF32WithErrorsArchsimd(input, gotErrBuf)
	} else {
		wantN = convertUTF16BEToUTF32Scalar(input, want)
		gotN = convertUTF16BEToUTF32Archsimd(input, got)
		wantErr = convertUTF16BEToUTF32WithErrorsScalar(input, wantErrBuf)
		gotErr = convertUTF16BEToUTF32WithErrorsArchsimd(input, gotErrBuf)
	}

	if gotN != wantN {
		t.Fatalf("little=%v convert = %d, want %d", little, gotN, wantN)
	}
	if wantN > 0 {
		if !slices.Equal(got[:wantN], want[:wantN]) || !allU32(got[wantN:], 0xa5a5a5a5) {
			t.Fatalf("little=%v convert output mismatch or canary overwrite", little)
		}
	} else if !allU32(got[need:], 0xa5a5a5a5) {
		t.Fatalf("little=%v convert canary overwrite on failure", little)
	}

	if gotErr != wantErr {
		t.Fatalf("little=%v with_errors = %+v, want %+v", little, gotErr, wantErr)
	}
	if wantErr.Error == Success {
		if !slices.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allU32(gotErrBuf[wantErr.Count:], 0xa5a5a5a5) {
			t.Fatalf("little=%v with_errors output mismatch or canary overwrite", little)
		}
		wantValidBuf := make([]uint32, need)
		gotValidBuf := make([]uint32, need+16)
		fillU32(gotValidBuf, 0xa5a5a5a5)
		if little {
			wantValid = convertValidUTF16LEToUTF32Scalar(input, wantValidBuf)
			gotValid = convertValidUTF16LEToUTF32Archsimd(input, gotValidBuf)
		} else {
			wantValid = convertValidUTF16BEToUTF32Scalar(input, wantValidBuf)
			gotValid = convertValidUTF16BEToUTF32Archsimd(input, gotValidBuf)
		}
		if gotValid != wantValid || !slices.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allU32(gotValidBuf[gotValid:], 0xa5a5a5a5) {
			t.Fatalf("little=%v valid mismatch or canary overwrite", little)
		}
	}
}

func checkUTF16UTF32ArchsimdPreflight(t *testing.T, native []uint16) {
	t.Helper()
	if len(native) == 0 {
		return
	}
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		need := utf32LengthFromUTF16LEScalar(input)
		if !little {
			need = utf32LengthFromUTF16BEScalar(input)
		}
		if need == 0 {
			continue
		}
		dst := make([]uint32, need-1)
		fillU32(dst, 0xa5a5a5a5)
		if little {
			requireUTF16UTF32ArchsimdPanic(t, func() { convertUTF16LEToUTF32Archsimd(input, dst) })
			requireUTF16UTF32ArchsimdPanic(t, func() { convertUTF16LEToUTF32WithErrorsArchsimd(input, dst) })
			requireUTF16UTF32ArchsimdPanic(t, func() { convertValidUTF16LEToUTF32Archsimd(input, dst) })
		} else {
			requireUTF16UTF32ArchsimdPanic(t, func() { convertUTF16BEToUTF32Archsimd(input, dst) })
			requireUTF16UTF32ArchsimdPanic(t, func() { convertUTF16BEToUTF32WithErrorsArchsimd(input, dst) })
			requireUTF16UTF32ArchsimdPanic(t, func() { convertValidUTF16BEToUTF32Archsimd(input, dst) })
		}
		if !allU32(dst, 0xa5a5a5a5) {
			t.Fatalf("little=%v short destination was modified", little)
		}
	}
}

func requireUTF16UTF32ArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

func utf16UTF32ArchsimdInput(length int, injectSurrogate bool) []uint16 {
	input := make([]uint16, length)
	for i := range input {
		switch i % 6 {
		case 0:
			input[i] = 0x00
		case 1:
			input[i] = 0x7f
		case 2:
			input[i] = 0x80
		case 3:
			input[i] = 0xff
		case 4:
			input[i] = 0x20ac
		default:
			input[i] = uint16(0x0100 + (i & 0xff))
		}
	}
	if injectSurrogate && length >= 2 {
		pos := length / 2
		if pos+1 >= length {
			pos = length - 2
		}
		input[pos] = 0xd83d
		input[pos+1] = 0xde00
	}
	return input
}

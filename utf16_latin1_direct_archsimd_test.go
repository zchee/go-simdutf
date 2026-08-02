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
)

func requireUTF16Latin1ArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdUTF16ToLatin1AgainstScalar(t *testing.T) {
	requireUTF16Latin1ArchsimdAVX2(t)

	tests := []struct {
		name   string
		native []uint16
	}{
		{name: "nil"},
		{name: "one-ascii", native: []uint16{0x7f}},
		{name: "one-high", native: []uint16{0xff}},
		{name: "mixed-short", native: []uint16{0x00, 0x7f, 0x80, 0xff, 'A'}},
		{name: "too-large-short", native: []uint16{'a', 0x100, 'b'}},
	}
	for _, length := range [...]int{7, 8, 15, 16, 17, 31, 32, 33, 63, 64, 65} {
		tests = append(
			tests,
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("latin1-length-%d", length), native: utf16Latin1ArchsimdInput(length, false)},
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("too-large-length-%d", length), native: utf16Latin1ArchsimdInput(length, true)},
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkUTF16Latin1Archsimd(t, test.native)
		})
	}
}

func FuzzUTF16Latin1ArchsimdAgainstScalar(f *testing.F) {
	for _, native := range [][]uint16{
		nil,
		{0x00},
		{0x7f, 0x80, 0xff},
		{0x100},
		utf16Latin1ArchsimdInput(15, false),
		utf16Latin1ArchsimdInput(16, false),
		utf16Latin1ArchsimdInput(17, true),
		utf16Latin1ArchsimdInput(33, false),
	} {
		raw := make([]byte, len(native)*2)
		for i, word := range native {
			raw[2*i] = byte(word)
			raw[2*i+1] = byte(word >> 8)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireUTF16Latin1ArchsimdAVX2(t)
		if len(raw)&1 != 0 {
			raw = raw[:len(raw)&^1]
		}
		native := make([]uint16, len(raw)/2)
		for i := range native {
			native[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		checkUTF16Latin1Archsimd(t, native)
	})
}

func checkUTF16Latin1Archsimd(t *testing.T, native []uint16) {
	t.Helper()
	checkUTF16Latin1ArchsimdEndian(t, native, true)
	checkUTF16Latin1ArchsimdEndian(t, native, false)
	checkUTF16Latin1ArchsimdPreflight(t, native)
}

func checkUTF16Latin1ArchsimdEndian(t *testing.T, native []uint16, little bool) {
	t.Helper()

	input := rawUTF16Words(native, little)

	want := make([]byte, len(input))
	got := bytes.Repeat([]byte{0xa5}, len(input)+16)
	wantErrBuf := make([]byte, len(input))
	gotErrBuf := bytes.Repeat([]byte{0xa5}, len(input)+16)

	var wantN, gotN, wantValid, gotValid int
	var wantErr, gotErr Result
	if little {
		wantN = convertUTF16LEToLatin1Scalar(input, want)
		gotN = convertUTF16LEToLatin1Archsimd(input, got)
		wantErr = convertUTF16LEToLatin1WithErrorsScalar(input, wantErrBuf)
		gotErr = convertUTF16LEToLatin1WithErrorsArchsimd(input, gotErrBuf)
	} else {
		wantN = convertUTF16BEToLatin1Scalar(input, want)
		gotN = convertUTF16BEToLatin1Archsimd(input, got)
		wantErr = convertUTF16BEToLatin1WithErrorsScalar(input, wantErrBuf)
		gotErr = convertUTF16BEToLatin1WithErrorsArchsimd(input, gotErrBuf)
	}

	if gotN != wantN {
		t.Fatalf("little=%v convert = %d, want %d", little, gotN, wantN)
	}
	if wantN > 0 {
		if !bytes.Equal(got[:wantN], want[:wantN]) || !allBytes(got[wantN:], 0xa5) {
			t.Fatalf("little=%v convert output mismatch or canary overwrite", little)
		}
	} else if !allBytes(got[len(input):], 0xa5) {
		t.Fatalf("little=%v convert canary overwrite on failure", little)
	}

	if gotErr != wantErr {
		t.Fatalf("little=%v with_errors = %+v, want %+v", little, gotErr, wantErr)
	}
	if wantErr.Error == Success {
		if !bytes.Equal(gotErrBuf[:wantErr.Count], wantErrBuf[:wantErr.Count]) || !allBytes(gotErrBuf[wantErr.Count:], 0xa5) {
			t.Fatalf("little=%v with_errors output mismatch or canary overwrite", little)
		}
		wantValidBuf := make([]byte, len(input))
		gotValidBuf := bytes.Repeat([]byte{0xa5}, len(input)+16)
		if little {
			wantValid = convertValidUTF16LEToLatin1Scalar(input, wantValidBuf)
			gotValid = convertValidUTF16LEToLatin1Archsimd(input, gotValidBuf)
		} else {
			wantValid = convertValidUTF16BEToLatin1Scalar(input, wantValidBuf)
			gotValid = convertValidUTF16BEToLatin1Archsimd(input, gotValidBuf)
		}
		if gotValid != wantValid || !bytes.Equal(gotValidBuf[:gotValid], wantValidBuf[:wantValid]) || !allBytes(gotValidBuf[gotValid:], 0xa5) {
			t.Fatalf("little=%v valid mismatch or canary overwrite", little)
		}
	}
}

func checkUTF16Latin1ArchsimdPreflight(t *testing.T, native []uint16) {
	t.Helper()
	if len(native) == 0 {
		return
	}
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		dst := bytes.Repeat([]byte{0xa5}, len(input)-1)
		if little {
			requireUTF16Latin1ArchsimdPanic(t, func() { convertUTF16LEToLatin1Archsimd(input, dst) })
			requireUTF16Latin1ArchsimdPanic(t, func() { convertUTF16LEToLatin1WithErrorsArchsimd(input, dst) })
			requireUTF16Latin1ArchsimdPanic(t, func() { convertValidUTF16LEToLatin1Archsimd(input, dst) })
		} else {
			requireUTF16Latin1ArchsimdPanic(t, func() { convertUTF16BEToLatin1Archsimd(input, dst) })
			requireUTF16Latin1ArchsimdPanic(t, func() { convertUTF16BEToLatin1WithErrorsArchsimd(input, dst) })
			requireUTF16Latin1ArchsimdPanic(t, func() { convertValidUTF16BEToLatin1Archsimd(input, dst) })
		}
		if !allBytes(dst, 0xa5) {
			t.Fatalf("little=%v short destination was modified", little)
		}
	}
}

func requireUTF16Latin1ArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

func utf16Latin1ArchsimdInput(length int, injectTooLarge bool) []uint16 {
	input := make([]uint16, length)
	for i := range input {
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
			input[i] = uint16(i & 0xff)
		}
	}
	if injectTooLarge && length > 0 {
		input[length/2] = 0x100 + uint16(length%0xff)
	}
	return input
}

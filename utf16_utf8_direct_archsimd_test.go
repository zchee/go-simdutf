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

func requireUTF16UTF8ArchsimdAVX2(t *testing.T) {
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

func TestDirectArchsimdUTF16ToUTF8AgainstScalar(t *testing.T) {
	requireUTF16UTF8ArchsimdAVX2(t)

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
		tests = append(tests,
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("ascii-length-%d", length), native: utf16UTF8ArchsimdInput(length, false)},
			struct {
				name   string
				native []uint16
			}{name: fmt.Sprintf("surrogate-length-%d", length), native: utf16UTF8ArchsimdInput(length, true)},
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checkUTF16UTF8Archsimd(t, test.native)
		})
	}
}

func FuzzUTF16UTF8ArchsimdAgainstScalar(f *testing.F) {
	for _, native := range [][]uint16{
		nil,
		{0x00},
		{0x7f, 0x80, 0xff, 0x20ac},
		{0xd83d, 0xde00},
		{0xd800},
		{0xdc00},
		utf16UTF8ArchsimdInput(15, false),
		utf16UTF8ArchsimdInput(16, false),
		utf16UTF8ArchsimdInput(17, true),
		utf16UTF8ArchsimdInput(33, false),
	} {
		raw := make([]byte, len(native)*2)
		for i, word := range native {
			raw[2*i] = byte(word)
			raw[2*i+1] = byte(word >> 8)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireUTF16UTF8ArchsimdAVX2(t)
		if len(raw)&1 != 0 {
			raw = raw[:len(raw)&^1]
		}
		native := make([]uint16, len(raw)/2)
		for i := range native {
			native[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		checkUTF16UTF8Archsimd(t, native)
	})
}

func checkUTF16UTF8Archsimd(t *testing.T, native []uint16) {
	t.Helper()
	checkUTF16UTF8ArchsimdEndian(t, native, true)
	checkUTF16UTF8ArchsimdEndian(t, native, false)
	checkUTF16UTF8ArchsimdPreflight(t, native)
}

func checkUTF16UTF8ArchsimdEndian(t *testing.T, native []uint16, little bool) {
	t.Helper()

	input := rawUTF16Words(native, little)
	storageIsNative := little == nativeLittleEndian()
	wantLen := utf8LengthFromUTF16Scalar(input, storageIsNative)
	wantLenRepl := utf8LengthFromUTF16WithReplacementScalar(input, storageIsNative)

	var (
		wantN, gotN       int
		wantE, gotE       Result
		wantV, gotV       int
		wantR, gotR       int
		wantLenN, gotLenN int
		wantLR, gotLR     Result
	)

	if little {
		wantLenN = utf8LengthFromUTF16LEScalar(input)
		gotLenN = utf8LengthFromUTF16LEArchsimd(input)
		wantLR = utf8LengthFromUTF16LEWithReplacementScalar(input)
		gotLR = utf8LengthFromUTF16LEWithReplacementArchsimd(input)
	} else {
		wantLenN = utf8LengthFromUTF16BEScalar(input)
		gotLenN = utf8LengthFromUTF16BEArchsimd(input)
		wantLR = utf8LengthFromUTF16BEWithReplacementScalar(input)
		gotLR = utf8LengthFromUTF16BEWithReplacementArchsimd(input)
	}
	if gotLenN != wantLenN {
		t.Fatalf("little=%v utf8_length = %d, want %d", little, gotLenN, wantLenN)
	}
	if gotLR != wantLR {
		t.Fatalf("little=%v utf8_length_with_replacement = %+v, want %+v", little, gotLR, wantLR)
	}

	want := bytes.Repeat([]byte{0xa5}, wantLen)
	got := bytes.Repeat([]byte{0xa5}, wantLen+16)
	if little {
		wantN = convertUTF16LEToUTF8Scalar(input, want)
		gotN = convertUTF16LEToUTF8Archsimd(input, got)
	} else {
		wantN = convertUTF16BEToUTF8Scalar(input, want)
		gotN = convertUTF16BEToUTF8Archsimd(input, got)
	}
	if gotN != wantN {
		t.Fatalf("little=%v convert = %d, want %d", little, gotN, wantN)
	}
	if wantN > 0 {
		if !bytes.Equal(got[:wantN], want[:wantN]) || !allBytes(got[wantN:], 0xa5) {
			t.Fatalf("little=%v convert output mismatch or canary overwrite", little)
		}
	} else if !allBytes(got[wantLen:], 0xa5) {
		t.Fatalf("little=%v convert canary overwrite on failure", little)
	}

	wantErrBuf := make([]byte, wantLen)
	gotErrBuf := bytes.Repeat([]byte{0xa5}, wantLen+16)
	if little {
		wantE = convertUTF16LEToUTF8WithErrorsScalar(input, wantErrBuf)
		gotE = convertUTF16LEToUTF8WithErrorsArchsimd(input, gotErrBuf)
	} else {
		wantE = convertUTF16BEToUTF8WithErrorsScalar(input, wantErrBuf)
		gotE = convertUTF16BEToUTF8WithErrorsArchsimd(input, gotErrBuf)
	}
	if gotE != wantE {
		t.Fatalf("little=%v with_errors = %+v, want %+v", little, gotE, wantE)
	}
	if wantE.Error == Success {
		if !bytes.Equal(gotErrBuf[:wantE.Count], wantErrBuf[:wantE.Count]) || !allBytes(gotErrBuf[wantE.Count:], 0xa5) {
			t.Fatalf("little=%v with_errors output mismatch or canary overwrite", little)
		}
	} else if !allBytes(gotErrBuf[wantLen:], 0xa5) {
		t.Fatalf("little=%v with_errors canary overwrite on failure", little)
	}

	wantRepl := make([]byte, wantLenRepl)
	gotRepl := bytes.Repeat([]byte{0xa5}, wantLenRepl+16)
	if little {
		wantR = convertUTF16LEToUTF8WithReplacementScalar(input, wantRepl)
		gotR = convertUTF16LEToUTF8WithReplacementArchsimd(input, gotRepl)
	} else {
		wantR = convertUTF16BEToUTF8WithReplacementScalar(input, wantRepl)
		gotR = convertUTF16BEToUTF8WithReplacementArchsimd(input, gotRepl)
	}
	if gotR != wantR || !bytes.Equal(gotRepl[:gotR], wantRepl[:wantR]) || !allBytes(gotRepl[gotR:], 0xa5) {
		t.Fatalf("little=%v with_replacement mismatch or canary overwrite", little)
	}

	if wantE.Error != Success {
		return
	}
	wantValidBuf := make([]byte, wantLen)
	gotValidBuf := bytes.Repeat([]byte{0xa5}, wantLen+16)
	if little {
		wantV = convertValidUTF16LEToUTF8Scalar(input, wantValidBuf)
		gotV = convertValidUTF16LEToUTF8Archsimd(input, gotValidBuf)
	} else {
		wantV = convertValidUTF16BEToUTF8Scalar(input, wantValidBuf)
		gotV = convertValidUTF16BEToUTF8Archsimd(input, gotValidBuf)
	}
	if gotV != wantV || !bytes.Equal(gotValidBuf[:gotV], wantValidBuf[:wantV]) || !allBytes(gotValidBuf[gotV:], 0xa5) {
		t.Fatalf("little=%v valid mismatch or canary overwrite", little)
	}
}

func checkUTF16UTF8ArchsimdPreflight(t *testing.T, native []uint16) {
	t.Helper()
	if len(native) == 0 {
		return
	}
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		storageIsNative := little == nativeLittleEndian()
		need := utf8LengthFromUTF16Scalar(input, storageIsNative)
		needRepl := utf8LengthFromUTF16WithReplacementScalar(input, storageIsNative)
		if need == 0 || needRepl == 0 {
			continue
		}
		dst := bytes.Repeat([]byte{0xa5}, need-1)
		dstRepl := bytes.Repeat([]byte{0xa5}, needRepl-1)
		if little {
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16LEToUTF8Archsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16LEToUTF8WithErrorsArchsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertValidUTF16LEToUTF8Archsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16LEToUTF8WithReplacementArchsimd(input, dstRepl) })
		} else {
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16BEToUTF8Archsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16BEToUTF8WithErrorsArchsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertValidUTF16BEToUTF8Archsimd(input, dst) })
			requireUTF16UTF8ArchsimdPanic(t, func() { convertUTF16BEToUTF8WithReplacementArchsimd(input, dstRepl) })
		}
		if !allBytes(dst, 0xa5) {
			t.Fatalf("little=%v short destination was modified", little)
		}
		if !allBytes(dstRepl, 0xa5) {
			t.Fatalf("little=%v short replacement destination was modified", little)
		}
	}
}

func requireUTF16UTF8ArchsimdPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("short destination did not panic")
		}
	}()
	f()
}

func utf16UTF8ArchsimdInput(length int, injectSurrogate bool) []uint16 {
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

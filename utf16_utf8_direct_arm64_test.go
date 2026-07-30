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

//go:build arm64

package simdutf

// Direct differential coverage for the arm64 UTF-16→UTF-8 NEON translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/arm64/arm_convert_utf16_to_utf8.cpp.
// Destination canaries and preflight checks are Go slice-contract coverage.
// NEON providers are invoked directly (no detectARM64Features).

import (
	"bytes"
	"testing"
)

func TestUTF16UTF8NEONDirectAgainstScalar(t *testing.T) {
	natives := [][]uint16{
		nil,
		{},
		{0},
		{'a', 'b', 'c'},
		{0x00, 0x7f, 0xff, 0x100, 0x7ff, 0x800, 0xffff},
		{0x20ac, 0xd83d, 0xde00},
		repeatUTF16('A', 7),
		repeatUTF16('A', 8),
		repeatUTF16('A', 9),
		repeatUTF16('A', 15),
		repeatUTF16('A', 16),
		repeatUTF16('A', 17),
		append(repeatUTF16('A', 7), 0x20ac),
		append(repeatUTF16('A', 8), 0xd83d, 0xde00),
		append(repeatUTF16('A', 15), 0xd83d, 0xde00, 'Z'),
		append(repeatUTF16('A', 16), 0xd800),
		append(repeatUTF16('A', 16), 0xdc00),
		append(repeatUTF16('A', 16), 0xd800, 0x0041),
		{'a', 0xd83d, 0xde00, 'b'},
		{0xd800},
		{0xdc00},
		{0xd800, 0x0041},
		{0xd83d, 0xde00},
		{0xd800, 0xdc00, 0xdbff, 0xdfff},
	}
	for _, native := range natives {
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			checkUTF16UTF8DirectNEON(t, input, little)
		}
	}
}

func TestUTF16UTF8NEONDirectPreflightPreservesCanaries(t *testing.T) {
	native := append(repeatUTF16('A', 15), 0x20ac)
	for _, little := range []bool{true, false} {
		input := rawUTF16Words(native, little)
		storageIsNative := little == nativeLittleEndian()
		need := utf8LengthFromUTF16Scalar(input, storageIsNative)
		needRepl := utf8LengthFromUTF16WithReplacementScalar(input, storageIsNative)
		if need == 0 || needRepl == 0 {
			continue
		}
		dst := guardedLatin1Destination[byte](need-1, 0xa5)
		dstRepl := guardedLatin1Destination[byte](needRepl-1, 0xa5)
		if little {
			requireLatin1Panic(t, func() { convertUTF16LEToUTF8NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16LEToUTF8WithErrorsNEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertValidUTF16LEToUTF8NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16LEToUTF8WithReplacementNEON(input, dstRepl.body) })
			dstRepl.require(t)
		} else {
			requireLatin1Panic(t, func() { convertUTF16BEToUTF8NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16BEToUTF8WithErrorsNEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertValidUTF16BEToUTF8NEON(input, dst.body) })
			dst.require(t)
			requireLatin1Panic(t, func() { convertUTF16BEToUTF8WithReplacementNEON(input, dstRepl.body) })
			dstRepl.require(t)
		}
	}
}

func FuzzUTF16UTF8NEONAgainstScalar(f *testing.F) {
	f.Add([]byte{})
	f.Add(utf16NativeBytes(repeatUTF16('A', 8)))
	f.Add(utf16NativeBytes(repeatUTF16(0x20ac, 16)))
	f.Add(utf16NativeBytes([]uint16{'a', 0xd83d, 0xde00, 'b'}))
	f.Add(utf16NativeBytes(append(repeatUTF16('A', 16), 0xd800)))
	f.Add(utf16NativeBytes([]uint16{0xd800, 0xdc00, 0xdbff, 0xdfff}))
	f.Fuzz(func(t *testing.T, raw []byte) {
		native := utf16NativeFromBytes(raw)
		for _, little := range []bool{true, false} {
			input := rawUTF16Words(native, little)
			checkUTF16UTF8DirectNEON(t, input, little)
		}
	})
}

func checkUTF16UTF8DirectNEON(t *testing.T, input []uint16, little bool) {
	t.Helper()

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
		gotLenN = utf8LengthFromUTF16LENEON(input)
		wantLR = utf8LengthFromUTF16LEWithReplacementScalar(input)
		gotLR = utf8LengthFromUTF16LEWithReplacementNEON(input)
	} else {
		wantLenN = utf8LengthFromUTF16BEScalar(input)
		gotLenN = utf8LengthFromUTF16BENEON(input)
		wantLR = utf8LengthFromUTF16BEWithReplacementScalar(input)
		gotLR = utf8LengthFromUTF16BEWithReplacementNEON(input)
	}
	if gotLenN != wantLenN {
		t.Fatalf("little=%t utf8_length = %d, want %d", little, gotLenN, wantLenN)
	}
	if gotLR != wantLR {
		t.Fatalf("little=%t utf8_length_with_replacement = %#v, want %#v", little, gotLR, wantLR)
	}

	want := bytes.Repeat([]byte{0xa5}, wantLen)
	got := guardedLatin1Destination[byte](wantLen, 0xa5)
	if little {
		wantN = convertUTF16LEToUTF8Scalar(input, want)
		gotN = convertUTF16LEToUTF8NEON(input, got.body)
	} else {
		wantN = convertUTF16BEToUTF8Scalar(input, want)
		gotN = convertUTF16BEToUTF8NEON(input, got.body)
	}
	if gotN != wantN || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t convert = %d/%x, want %d/%x", little, gotN, got.body, wantN, want)
	}
	got.require(t)

	want = bytes.Repeat([]byte{0xa5}, wantLen)
	got = guardedLatin1Destination[byte](wantLen, 0xa5)
	if little {
		wantE = convertUTF16LEToUTF8WithErrorsScalar(input, want)
		gotE = convertUTF16LEToUTF8WithErrorsNEON(input, got.body)
	} else {
		wantE = convertUTF16BEToUTF8WithErrorsScalar(input, want)
		gotE = convertUTF16BEToUTF8WithErrorsNEON(input, got.body)
	}
	if gotE != wantE || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t with_errors = %#v/%x, want %#v/%x", little, gotE, got.body, wantE, want)
	}
	got.require(t)

	want = bytes.Repeat([]byte{0xa5}, wantLenRepl)
	got = guardedLatin1Destination[byte](wantLenRepl, 0xa5)
	if little {
		wantR = convertUTF16LEToUTF8WithReplacementScalar(input, want)
		gotR = convertUTF16LEToUTF8WithReplacementNEON(input, got.body)
	} else {
		wantR = convertUTF16BEToUTF8WithReplacementScalar(input, want)
		gotR = convertUTF16BEToUTF8WithReplacementNEON(input, got.body)
	}
	if gotR != wantR || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t with_replacement = %d/%x, want %d/%x", little, gotR, got.body, wantR, want)
	}
	got.require(t)

	if wantE.Error != Success {
		return
	}
	want = bytes.Repeat([]byte{0xa5}, wantLen)
	got = guardedLatin1Destination[byte](wantLen, 0xa5)
	if little {
		wantV = convertValidUTF16LEToUTF8Scalar(input, want)
		gotV = convertValidUTF16LEToUTF8NEON(input, got.body)
	} else {
		wantV = convertValidUTF16BEToUTF8Scalar(input, want)
		gotV = convertValidUTF16BEToUTF8NEON(input, got.body)
	}
	if gotV != wantV || !bytes.Equal(got.body, want) {
		t.Fatalf("little=%t valid = %d/%x, want %d/%x", little, gotV, got.body, wantV, want)
	}
	got.require(t)
}

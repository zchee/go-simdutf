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

package simdutf

import (
	"bytes"
	"testing"
)

// Go-only public/direct-dispatch-versus-scalar differential fuzz scaffold for
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee): fuzz/conversion.cpp,
// fuzz/roundtrip.cpp, fuzz/misc.cpp, and include/simdutf/scalar/utf8.h:258-325.
// The scalar functions are the permanent arbitrary-byte Go oracles.

func FuzzUTF8Lengths(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{},
		[]byte("hello"),
		{'a', 0xc2, 0xa2, 0xe2, 0x82, 0xac, 0xf0, 0x90, 0x8d, 0x88},
		{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xf4, 0xf8, 0xff},
	} {
		f.Add(seed)
	}
	for _, boundary := range []int{16, 32, 64, 128} {
		for _, size := range []int{boundary - 1, boundary, boundary + 1} {
			seed := bytes.Repeat([]byte{'a'}, size)
			seed[len(seed)-1] = 0x80
			// Start four bytes before the first byte past the boundary so the
			// complete boundary+1 seed carries a four-byte sequence across it.
			lead := boundary - 3
			if lead < len(seed) {
				seed[lead] = 0xf0
				for i := lead + 1; i < len(seed) && i < lead+4; i++ {
					seed[i] = 0x80
				}
			}
			f.Add(seed)
		}
	}
	selection := detectSelectionInput()
	f.Fuzz(func(t *testing.T, input []byte) {
		for _, prefix := range [...]int{1, 2, 3, 5, 7, 15, 31, 63} {
			guard := newGuardedSlice(prefix, len(input), 67, byte(0xa5))
			copy(guard.body, input)
			before := bytes.Clone(guard.storage)
			wantLatin1 := latin1LengthFromUTF8Scalar(guard.body)
			wantUTF16 := utf16LengthFromUTF8Scalar(guard.body)
			wantUTF32 := utf32LengthFromUTF8Scalar(guard.body)
			if got := Latin1LengthFromUTF8(guard.body); got != wantLatin1 {
				t.Errorf("prefix %d: Latin1LengthFromUTF8() = %d, scalar = %d", prefix, got, wantLatin1)
			}
			if got := activeImplementation.latin1LengthFromUTF8(guard.body); got != wantLatin1 {
				t.Errorf("prefix %d: direct latin1LengthFromUTF8() = %d, scalar = %d", prefix, got, wantLatin1)
			}
			if got := UTF16LengthFromUTF8(guard.body); got != wantUTF16 {
				t.Errorf("prefix %d: UTF16LengthFromUTF8() = %d, scalar = %d", prefix, got, wantUTF16)
			}
			if got := activeImplementation.utf16LengthFromUTF8(guard.body); got != wantUTF16 {
				t.Errorf("prefix %d: direct utf16LengthFromUTF8() = %d, scalar = %d", prefix, got, wantUTF16)
			}
			if got := UTF32LengthFromUTF8(guard.body); got != wantUTF32 {
				t.Errorf("prefix %d: UTF32LengthFromUTF8() = %d, scalar = %d", prefix, got, wantUTF32)
			}
			if got := activeImplementation.utf32LengthFromUTF8(guard.body); got != wantUTF32 {
				t.Errorf("prefix %d: direct utf32LengthFromUTF8() = %d, scalar = %d", prefix, got, wantUTF32)
			}
			for _, candidate := range utf8LengthFuzzVariants {
				if candidate.latin1.supportedBy(selection) {
					if got := candidate.latin1.value(guard.body); got != wantLatin1 {
						t.Errorf("prefix %d: %s Latin1LengthFromUTF8() = %d, scalar = %d", prefix, candidate.name, got, wantLatin1)
					}
				}
				if candidate.utf16.supportedBy(selection) {
					if got := candidate.utf16.value(guard.body); got != wantUTF16 {
						t.Errorf("prefix %d: %s UTF16LengthFromUTF8() = %d, scalar = %d", prefix, candidate.name, got, wantUTF16)
					}
				}
				if candidate.utf32.supportedBy(selection) {
					if got := candidate.utf32.value(guard.body); got != wantUTF32 {
						t.Errorf("prefix %d: %s UTF32LengthFromUTF8() = %d, scalar = %d", prefix, candidate.name, got, wantUTF32)
					}
				}
			}
			guard.requireCanariesIntact(t)
			if !bytes.Equal(guard.storage, before) {
				t.Errorf("prefix %d: UTF-8 length helper modified input or canaries", prefix)
			}
		}
	})
}

func FuzzTrimPartialUTF8(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{},
		[]byte("abc"),
		{'a', 0xc3},
		{'a', 0xe2, 0x82},
		{'a', 0xf0, 0x90, 0x8d},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		guard := newGuardedSlice(5, len(input), 7, byte(0xa5))
		copy(guard.body, input)
		before := bytes.Clone(guard.storage)
		got := TrimPartialUTF8(guard.body)
		if want := trimPartialUTF8Scalar(guard.body); got != want {
			t.Errorf("TrimPartialUTF8() = %d, scalar = %d", got, want)
		}
		// Pinned fuzz/misc.cpp rejects ret + 3 < N or ret > N. Express the
		// same bounds without overflowing an addition to the returned length.
		if got > len(guard.body) {
			t.Errorf("TrimPartialUTF8() = %d, input length = %d", got, len(guard.body))
		} else if len(guard.body)-got > 3 {
			t.Errorf("TrimPartialUTF8() removed %d bytes, want at most 3", len(guard.body)-got)
		}
		guard.requireCanariesIntact(t)
		if !bytes.Equal(guard.storage, before) {
			t.Fatal("TrimPartialUTF8 modified input or canaries")
		}
	})
}

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
	f.Fuzz(func(t *testing.T, input []byte) {
		before := bytes.Clone(input)
		if got, want := Latin1LengthFromUTF8(input), latin1LengthFromUTF8Scalar(input); got != want {
			t.Errorf("Latin1LengthFromUTF8() = %d, scalar = %d", got, want)
		}
		if got, want := activeImplementation.latin1LengthFromUTF8(input), latin1LengthFromUTF8Scalar(input); got != want {
			t.Errorf("direct latin1LengthFromUTF8() = %d, scalar = %d", got, want)
		}
		if got, want := UTF16LengthFromUTF8(input), utf16LengthFromUTF8Scalar(input); got != want {
			t.Errorf("UTF16LengthFromUTF8() = %d, scalar = %d", got, want)
		}
		if got, want := activeImplementation.utf16LengthFromUTF8(input), utf16LengthFromUTF8Scalar(input); got != want {
			t.Errorf("direct utf16LengthFromUTF8() = %d, scalar = %d", got, want)
		}
		if got, want := UTF32LengthFromUTF8(input), utf32LengthFromUTF8Scalar(input); got != want {
			t.Errorf("UTF32LengthFromUTF8() = %d, scalar = %d", got, want)
		}
		if got, want := activeImplementation.utf32LengthFromUTF8(input), utf32LengthFromUTF8Scalar(input); got != want {
			t.Errorf("direct utf32LengthFromUTF8() = %d, scalar = %d", got, want)
		}
		if !bytes.Equal(input, before) {
			t.Fatal("UTF-8 length helper modified input")
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
		before := bytes.Clone(input)
		if got, want := TrimPartialUTF8(input), trimPartialUTF8Scalar(input); got != want {
			t.Errorf("TrimPartialUTF8() = %d, scalar = %d", got, want)
		}
		if !bytes.Equal(input, before) {
			t.Fatal("TrimPartialUTF8 modified input")
		}
	})
}

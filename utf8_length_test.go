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

// Test vectors translated and adapted from
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// tests/null_safety_tests.cpp:65-73, tests/simdutf_c_tests.cpp:254-265,
// tests/readme_tests.cpp:122-141, and include/simdutf/scalar/utf8.h:258-325.

func TestUTF8LengthsUpstreamCases(t *testing.T) {
	cases := map[string]struct {
		input                []byte
		latin1, utf16, utf32 int
	}{
		"nil":            {input: nil},
		"empty":          {input: []byte{}},
		"hello":          {input: []byte("hello"), latin1: 5, utf16: 5, utf32: 5},
		"one-to-four":    {input: []byte{'a', 0xc2, 0xa2, 0xe2, 0x82, 0xac, 0xf0, 0x90, 0x8d, 0x88}, latin1: 4, utf16: 5, utf32: 4},
		"stream-literal": {input: []byte("école d'été")[:10], latin1: 9, utf16: 9, utf32: 9},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			before := bytes.Clone(test.input)
			if got := Latin1LengthFromUTF8(test.input); got != test.latin1 {
				t.Errorf("Latin1LengthFromUTF8() = %d, want %d", got, test.latin1)
			}
			if got := UTF16LengthFromUTF8(test.input); got != test.utf16 {
				t.Errorf("UTF16LengthFromUTF8() = %d, want %d", got, test.utf16)
			}
			if got := UTF32LengthFromUTF8(test.input); got != test.utf32 {
				t.Errorf("UTF32LengthFromUTF8() = %d, want %d", got, test.utf32)
			}
			if !bytes.Equal(test.input, before) {
				t.Fatal("UTF-8 length helper modified input")
			}
		})
	}
}

func TestUTF8LengthScalarArbitraryByteFormula(t *testing.T) {
	input := []byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xf4, 0xf8, 0xff}
	if got, want := latin1LengthFromUTF8Scalar(input), 8; got != want {
		t.Errorf("latin1LengthFromUTF8Scalar() = %d, want %d", got, want)
	}
	if got, want := utf16LengthFromUTF8Scalar(input), 12; got != want {
		t.Errorf("utf16LengthFromUTF8Scalar() = %d, want %d", got, want)
	}
	if got, want := utf32LengthFromUTF8Scalar(input), 8; got != want {
		t.Errorf("utf32LengthFromUTF8Scalar() = %d, want %d", got, want)
	}
	for value := 0; value <= 0xff; value++ {
		input := []byte{byte(value)}
		codePoints := 1
		if value >= 0x80 && value <= 0xbf {
			codePoints = 0
		}
		utf16 := codePoints
		if value >= 0xf0 {
			utf16++
		}
		if got := Latin1LengthFromUTF8(input); got != codePoints {
			t.Errorf("Latin1LengthFromUTF8({%#02x}) = %d, want %d", value, got, codePoints)
		}
		if got := UTF16LengthFromUTF8(input); got != utf16 {
			t.Errorf("UTF16LengthFromUTF8({%#02x}) = %d, want %d", value, got, utf16)
		}
		if got := UTF32LengthFromUTF8(input); got != codePoints {
			t.Errorf("UTF32LengthFromUTF8({%#02x}) = %d, want %d", value, got, codePoints)
		}
	}
}

func TestTrimPartialUTF8UpstreamTailCases(t *testing.T) {
	cases := map[string]struct {
		input []byte
		want  int
	}{
		"nil":                      {input: nil, want: 0},
		"empty":                    {input: []byte{}, want: 0},
		"ASCII":                    {input: []byte("abc"), want: 3},
		"complete two byte":        {input: []byte{0xc3, 0xa9}, want: 2},
		"partial two byte":         {input: []byte{'a', 0xc3}, want: 1},
		"partial three one byte":   {input: []byte{'a', 0xe2}, want: 1},
		"partial three two bytes":  {input: []byte{'a', 0xe2, 0x82}, want: 1},
		"partial four one byte":    {input: []byte{'a', 0xf0}, want: 1},
		"partial four two bytes":   {input: []byte{'a', 0xf0, 0x90}, want: 1},
		"partial four three bytes": {input: []byte{'a', 0xf0, 0x90, 0x8d}, want: 1},
		"streaming example":        {input: []byte("école d'été")[:10], want: 9},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			before := bytes.Clone(test.input)
			if got := TrimPartialUTF8(test.input); got != test.want {
				t.Errorf("TrimPartialUTF8() = %d, want %d", got, test.want)
			}
			if got := trimPartialUTF8Scalar(test.input); got != test.want {
				t.Errorf("trimPartialUTF8Scalar() = %d, want %d", got, test.want)
			}
			if !bytes.Equal(test.input, before) {
				t.Fatal("TrimPartialUTF8 modified input")
			}
		})
	}
}

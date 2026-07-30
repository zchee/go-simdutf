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

import "testing"

// Contract cases derived from simdutf commit 611becc2a08c27a4edc77d9a45ff74c97130129b,
// tests/detect_encodings_tests.cpp and src/fallback/implementation.cpp:8-32.
// Nil/empty and AutodetectEncoding priority cases are narrow Go wrappers over
// the same DetectEncodings bitset semantics.

func TestDetectEncodings(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  Encoding
	}{
		{name: "nil", input: nil, want: UTF8 | UTF16LE | UTF32LE},
		{name: "empty", input: []byte{}, want: UTF8 | UTF16LE | UTF32LE},
		{name: "ascii", input: []byte("hello"), want: UTF8},
		{name: "ascii-even", input: []byte("hi"), want: UTF8 | UTF16LE},
		{name: "utf8-bom", input: []byte{0xef, 0xbb, 0xbf}, want: UTF8},
		{name: "utf16le-bom", input: []byte{0xff, 0xfe}, want: UTF16LE},
		{name: "utf16be-bom", input: []byte{0xfe, 0xff}, want: UTF16BE},
		{name: "utf32le-bom", input: []byte{0xff, 0xfe, 0x00, 0x00}, want: UTF32LE},
		{name: "utf32be-bom", input: []byte{0x00, 0x00, 0xfe, 0xff}, want: UTF32BE},
		{name: "issue519", input: []byte{
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 223, 164, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
		}, want: UTF8},
		{name: "issue516", input: []byte{0x20, 0xd8, 0x00, 0x00}, want: Unspecified},
		{name: "utf16le-ascii-units", input: []byte{'A', 0, 'B', 0}, want: UTF8 | UTF16LE},
		{name: "utf32le-ascii-unit", input: []byte{'A', 0, 0, 0}, want: UTF8 | UTF16LE | UTF32LE},
		{name: "short-odd", input: []byte{0xff}, want: Unspecified},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectEncodings(tc.input); got != tc.want {
				t.Fatalf("DetectEncodings() = %d, want %d", got, tc.want)
			}
			if got := detectEncodingsScalar(tc.input); got != tc.want {
				t.Fatalf("detectEncodingsScalar() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestAutodetectEncoding(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  Encoding
	}{
		{name: "nil", input: nil, want: UTF8},
		{name: "empty", input: []byte{}, want: UTF8},
		{name: "ascii", input: []byte("hello"), want: UTF8},
		{name: "utf8-bom", input: []byte{0xef, 0xbb, 0xbf}, want: UTF8},
		{name: "utf16le-bom", input: []byte{0xff, 0xfe}, want: UTF16LE},
		{name: "utf16be-bom", input: []byte{0xfe, 0xff}, want: UTF16BE},
		{name: "utf32le-bom", input: []byte{0xff, 0xfe, 0x00, 0x00}, want: UTF32LE},
		{name: "utf32be-bom", input: []byte{0x00, 0x00, 0xfe, 0xff}, want: UTF32BE},
		{name: "issue516", input: []byte{0x20, 0xd8, 0x00, 0x00}, want: Unspecified},
		{name: "utf16le-only-priority", input: []byte{0x00, 0xd8, 0x00, 0xdc}, want: UTF16LE},
		{name: "short-odd", input: []byte{0xff}, want: Unspecified},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := AutodetectEncoding(tc.input); got != tc.want {
				t.Fatalf("AutodetectEncoding() = %d, want %d", got, tc.want)
			}
			if got := autodetectEncodingFromDetected(DetectEncodings(tc.input)); got != tc.want {
				t.Fatalf("autodetectEncodingFromDetected() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDetectEncodingsLiveDispatchIsScalar(t *testing.T) {
	if !sameFunction(activeImplementation.detectEncodings, detectEncodingsScalar) {
		t.Fatalf("live detectEncodings selected %p, want scalar %p", activeImplementation.detectEncodings, detectEncodingsScalar)
	}
}

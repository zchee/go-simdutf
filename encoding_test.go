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

// Contract cases derived from simdutf commit c7bef0ff14a13fd6ea52e3347da2c659383392de,
// include/simdutf/encoding_types.h:15-24 and src/encoding_types.cpp:3-64.
// Narrow Go-only scaffolding covers the underlying type, unknown values,
// truncated inputs, and non-prefix BOMs; these are not upstream test vectors.

import (
	"reflect"
	"testing"
)

func TestEncoding(t *testing.T) {
	if kind := reflect.TypeOf(Unspecified).Kind(); kind != reflect.Uint8 {
		t.Fatalf("Encoding underlying kind = %v, want uint8", kind)
	}

	tests := []struct {
		name     string
		encoding Encoding
		value    uint8
		text     string
	}{
		{"unspecified", Unspecified, 0, "unknown"},
		{"UTF-8", UTF8, 1, "UTF8"},
		{"UTF-16LE", UTF16LE, 2, "UTF16 little-endian"},
		{"UTF-16BE", UTF16BE, 4, "UTF16 big-endian"},
		{"UTF-32LE", UTF32LE, 8, "UTF32 little-endian"},
		{"UTF-32BE", UTF32BE, 16, "UTF32 big-endian"},
		{"Latin-1", Latin1, 32, "error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := uint8(test.encoding); got != test.value {
				t.Errorf("value = %d, want %d", got, test.value)
			}
			if got := EncodingString(test.encoding); got != test.text {
				t.Errorf("EncodingString() = %q, want %q", got, test.text)
			}
		})
	}
}

func TestEncodingStringUnknown(t *testing.T) {
	for _, encoding := range []Encoding{3, 255} {
		if got := EncodingString(encoding); got != "error" {
			t.Errorf("EncodingString(%d) = %q, want error", encoding, got)
		}
	}
}

func TestCheckBOM(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  Encoding
	}{
		{"nil", nil, Unspecified},
		{"empty", []byte{}, Unspecified},
		{"one byte UTF-16LE prefix", []byte{0xff}, Unspecified},
		{"one byte UTF-16BE prefix", []byte{0xfe}, Unspecified},
		{"two byte UTF-8 prefix", []byte{0xef, 0xbb}, Unspecified},
		{"three byte UTF-32BE prefix", []byte{0x00, 0x00, 0xfe}, Unspecified},
		{"UTF-8", []byte{0xef, 0xbb, 0xbf}, UTF8},
		{"UTF-8 with payload", []byte{0xef, 0xbb, 0xbf, 'x'}, UTF8},
		{"UTF-16LE", []byte{0xff, 0xfe}, UTF16LE},
		{"UTF-16LE three byte truncation", []byte{0xff, 0xfe, 0x00}, UTF16LE},
		{"UTF-16LE non-UTF-32 suffix", []byte{0xff, 0xfe, 0x00, 0x01}, UTF16LE},
		{"UTF-16BE", []byte{0xfe, 0xff}, UTF16BE},
		{"UTF-32LE precedence", []byte{0xff, 0xfe, 0x00, 0x00}, UTF32LE},
		{"UTF-32LE with payload", []byte{0xff, 0xfe, 0x00, 0x00, 'x'}, UTF32LE},
		{"UTF-32BE", []byte{0x00, 0x00, 0xfe, 0xff}, UTF32BE},
		{"non-prefix UTF-8", []byte{'x', 0xef, 0xbb, 0xbf}, Unspecified},
		{"non-prefix UTF-16LE", []byte{'x', 0xff, 0xfe}, Unspecified},
		{"no BOM", []byte("plain text"), Unspecified},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CheckBOM(test.input); got != test.want {
				t.Errorf("CheckBOM(% x) = %d, want %d", test.input, got, test.want)
			}
		})
	}
}

func TestBOMByteSize(t *testing.T) {
	tests := []struct {
		name     string
		encoding Encoding
		want     int
	}{
		{"unspecified", Unspecified, 0},
		{"UTF-8", UTF8, 3},
		{"UTF-16LE", UTF16LE, 2},
		{"UTF-16BE", UTF16BE, 2},
		{"UTF-32LE", UTF32LE, 4},
		{"UTF-32BE", UTF32BE, 4},
		{"Latin-1", Latin1, 0},
		{"unknown", Encoding(255), 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := BOMByteSize(test.encoding); got != test.want {
				t.Errorf("BOMByteSize() = %d, want %d", got, test.want)
			}
		})
	}
}

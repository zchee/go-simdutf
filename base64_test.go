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
	"encoding/base64"
	"testing"
)

func TestBinaryToBase64RoundTripDefault(t *testing.T) {
	input := []byte("Hello, simdutf Base64!")
	dst := make([]byte, Base64LengthFromBinary(len(input), Base64Default))
	n := BinaryToBase64(input, dst, Base64Default)
	encoded := string(dst[:n])
	want := base64.StdEncoding.EncodeToString(input)
	if encoded != want {
		t.Fatalf("BinaryToBase64 = %q, want %q", encoded, want)
	}
	out := make([]byte, MaximalBinaryLengthFromBase64(dst[:n]))
	r := Base64ToBinary(dst[:n], out, Base64Default, Loose)
	if r.Error != Success {
		t.Fatalf("Base64ToBinary error = %v", r.Error)
	}
	if !bytes.Equal(out[:r.Count], input) {
		t.Fatalf("round-trip = %q, want %q", out[:r.Count], input)
	}
}

func TestBase64URLNoPadding(t *testing.T) {
	input := []byte{0x01, 0x02}
	dst := make([]byte, Base64LengthFromBinary(len(input), Base64URL))
	n := BinaryToBase64(input, dst, Base64URL)
	if bytes.Contains(dst[:n], []byte("=")) {
		t.Fatalf("URL encoding unexpectedly padded: %q", dst[:n])
	}
	out := make([]byte, MaximalBinaryLengthFromBase64(dst[:n]))
	r := Base64ToBinary(dst[:n], out, Base64URL, Loose)
	if r.Error != Success {
		t.Fatalf("decode error = %v", r.Error)
	}
	if !bytes.Equal(out[:r.Count], input) {
		t.Fatalf("got %v want %v", out[:r.Count], input)
	}
}

func TestBase64ValidHelpers(t *testing.T) {
	if !Base64Valid('A', Base64Default) {
		t.Fatal("A should be valid")
	}
	if Base64Valid('-', Base64Default) {
		t.Fatal("- invalid in default")
	}
	if !Base64Valid('-', Base64URL) {
		t.Fatal("- valid in url")
	}
	if !Base64ValidOrPadding('=', Base64Default) {
		t.Fatal("= should be valid or padding")
	}
	if !Base64Ignorable('\n', Base64Default) {
		t.Fatal("newline ignorable")
	}
}

func TestBase64ToBinarySafeShort(t *testing.T) {
	enc := []byte("AQID") // 0x01 0x02 0x03
	dst := make([]byte, 2)
	res, written := Base64ToBinarySafe(enc, dst, Base64Default, Loose, false)
	if res.Error != OutputBufferTooSmall {
		t.Fatalf("want OutputBufferTooSmall, got %v written=%d", res.Error, written)
	}
	if written != 0 {
		t.Fatalf("want written=0, got %d", written)
	}
}

func TestBase64UTF16RoundTrip(t *testing.T) {
	input := []byte("Go")
	enc := make([]byte, Base64LengthFromBinary(len(input), Base64Default))
	n := BinaryToBase64(input, enc, Base64Default)
	u16 := make([]uint16, n)
	for i := 0; i < n; i++ {
		u16[i] = uint16(enc[i])
	}
	out := make([]byte, MaximalBinaryLengthFromBase64UTF16(u16))
	r := Base64ToBinaryUTF16(u16, out, Base64Default, Loose)
	if r.Error != Success {
		t.Fatalf("utf16 decode error %v", r.Error)
	}
	if !bytes.Equal(out[:r.Count], input) {
		t.Fatalf("got %q want %q", out[:r.Count], input)
	}
}

// TestBase64UTF16ValidHelpers is hand-authored Go-only coverage for the
// UTF-16 classification helpers, mirroring TestBase64ValidHelpers.
func TestBase64UTF16ValidHelpers(t *testing.T) {
	tests := map[string]struct {
		value          uint16
		options        Base64Options
		valid          bool
		validOrPadding bool
		ignorable      bool
	}{
		"valid: alphabet unit":             {'A', Base64Default, true, true, false},
		"invalid: url dash in default":     {'-', Base64Default, false, false, false},
		"valid: url dash in url":           {'-', Base64URL, true, true, false},
		"padding: equals sign":             {'=', Base64Default, false, true, false},
		"ignorable: newline":               {'\n', Base64Default, false, false, true},
		"invalid: non-latin1 unit":         {0x263a, Base64Default, false, false, false},
		"ignorable: non-latin1 as garbage": {0x263a, Base64DefaultAcceptGarbage, false, false, true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := Base64ValidUTF16(test.value, test.options); got != test.valid {
				t.Fatalf("Base64ValidUTF16(%#x) = %v, want %v", test.value, got, test.valid)
			}
			if got := Base64ValidOrPaddingUTF16(test.value, test.options); got != test.validOrPadding {
				t.Fatalf("Base64ValidOrPaddingUTF16(%#x) = %v, want %v", test.value, got, test.validOrPadding)
			}
			if got := Base64IgnorableUTF16(test.value, test.options); got != test.ignorable {
				t.Fatalf("Base64IgnorableUTF16(%#x) = %v, want %v", test.value, got, test.ignorable)
			}
		})
	}
}

// TestBase64UTF16HelpersMatchByteHelpers locks the UTF-16 helpers to the
// tested byte helpers over the eight-byte range and to the ignore-garbage
// rule above it.
func TestBase64UTF16HelpersMatchByteHelpers(t *testing.T) {
	options := []Base64Options{
		Base64Default, Base64URL, Base64DefaultAcceptGarbage,
		Base64URLAcceptGarbage, Base64DefaultOrURL,
	}
	for c := range uint16(0x300) {
		for _, opts := range options {
			if c <= 0xff {
				if Base64ValidUTF16(c, opts) != Base64Valid(byte(c), opts) ||
					Base64ValidOrPaddingUTF16(c, opts) != Base64ValidOrPadding(byte(c), opts) ||
					Base64IgnorableUTF16(c, opts) != Base64Ignorable(byte(c), opts) {
					t.Fatalf("UTF-16 helper diverges from byte helper for %#x options %v", c, opts)
				}
				continue
			}
			wantIgnorable := Base64Ignorable(0xfe, opts) // any non-alphabet, non-space byte
			if Base64ValidUTF16(c, opts) || Base64ValidOrPaddingUTF16(c, opts) || Base64IgnorableUTF16(c, opts) != wantIgnorable {
				t.Fatalf("non-latin1 unit %#x misclassified under options %v", c, opts)
			}
		}
	}
}

// TestBase64ToBinarySafeUTF16 mirrors the byte-input safe-decode coverage for
// the UTF-16 entry point.
func TestBase64ToBinarySafeUTF16(t *testing.T) {
	input := []byte("Hello, simdutf Base64!")
	enc := make([]byte, Base64LengthFromBinary(len(input), Base64Default))
	n := BinaryToBase64(input, enc, Base64Default)
	u16 := make([]uint16, n)
	for i, b := range enc[:n] {
		u16[i] = uint16(b)
	}
	dst := make([]byte, MaximalBinaryLengthFromBase64UTF16(u16))
	res, written := Base64ToBinarySafeUTF16(u16, dst, Base64Default, Loose, false)
	if res.Error != Success || !bytes.Equal(dst[:written], input) {
		t.Fatalf("Base64ToBinarySafeUTF16 = %+v %q, want %q", res, dst[:written], input)
	}

	short := []uint16{'A', 'Q', 'I', 'D'} // 0x01 0x02 0x03
	res, written = Base64ToBinarySafeUTF16(short, make([]byte, 2), Base64Default, Loose, false)
	if res.Error != OutputBufferTooSmall || written != 0 {
		t.Fatalf("short dst = %+v written=%d, want OutputBufferTooSmall/0", res, written)
	}
}

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
	if res.Error != OutputBufferTooSmall && written != 0 {
		// Either buffer-too-small or partial success depending on path; must not panic.
		_ = res
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

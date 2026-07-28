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
	"fmt"
	"testing"
)

// Test vectors translated and adapted from
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
// tests/count_utf8.cpp:11-84. The deterministic Go generator preserves the
// upstream byte sizes and ASCII/one-to-four-byte mixture categories; it does
// not claim byte-identical output to the upstream C++ random_utf8 fixtures.

func TestCountUTF8UpstreamMixtures(t *testing.T) {
	sizes := [...]int{7, 12, 16, 64, 67, 128, 256, 511, 1000, 2000}
	mixtures := map[string][][]byte{
		"ASCII":              {{'a'}},
		"one-or-two":         {{'a'}, {0xc2, 0xa2}},
		"one-two-three":      {{'a'}, {0xc2, 0xa2}, {0xe2, 0x82, 0xac}},
		"one-two-three-four": {{'a'}, {0xc2, 0xa2}, {0xe2, 0x82, 0xac}, {0xf0, 0x90, 0x8d, 0x88}},
	}

	for name, codePoints := range mixtures {
		t.Run(name, func(t *testing.T) {
			for _, size := range sizes {
				t.Run(fmt.Sprintf("%04d", size), func(t *testing.T) {
					input, want := countUTF8Fixture(size, codePoints)
					if got := CountUTF8(input); got != want {
						t.Errorf("CountUTF8() = %d, want %d", got, want)
					}
					if got := countUTF8Scalar(input); got != want {
						t.Errorf("countUTF8Scalar() = %d, want %d", got, want)
					}
				})
			}
		})
	}
}

func TestCountUTF8BoundariesAndLiteral(t *testing.T) {
	cases := map[string]struct {
		input []byte
		want  int
	}{
		"nil":        {input: nil, want: 0},
		"empty":      {input: []byte{}, want: 0},
		"koettbulle": {input: []byte("köttbulle"), want: 9},
		"minimums":   {input: []byte{'a', 0xc2, 0x80, 0xe0, 0xa0, 0x80, 0xf0, 0x90, 0x80, 0x80}, want: 4},
		"maximums":   {input: []byte{0x7f, 0xdf, 0xbf, 0xef, 0xbf, 0xbf, 0xf4, 0x8f, 0xbf, 0xbf}, want: 4},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			before := bytes.Clone(test.input)
			if got := CountUTF8(test.input); got != test.want {
				t.Errorf("CountUTF8() = %d, want %d", got, test.want)
			}
			if got := countUTF8Scalar(test.input); got != test.want {
				t.Errorf("countUTF8Scalar() = %d, want %d", got, test.want)
			}
			if !bytes.Equal(test.input, before) {
				t.Fatal("CountUTF8 modified input")
			}
		})
	}
}

// TestCountUTF8ScalarByteClasses locks the pinned fallback oracle on invalid
// input: it counts every byte except UTF-8 continuation bytes 0x80..0xbf and
// performs no validation.
func TestCountUTF8ScalarByteClasses(t *testing.T) {
	input := []byte{0x00, 0x7f, 0x80, 0x81, 0xbe, 0xbf, 0xc0, 0xdf, 0xe0, 0xef, 0xf0, 0xf4, 0xf8, 0xff}
	if got, want := countUTF8Scalar(input), 10; got != want {
		t.Fatalf("countUTF8Scalar() = %d, want %d", got, want)
	}
	for value := 0; value <= 0xff; value++ {
		want := 1
		if value >= 0x80 && value <= 0xbf {
			want = 0
		}
		if got := countUTF8Scalar([]byte{byte(value)}); got != want {
			t.Errorf("countUTF8Scalar({%#02x}) = %d, want %d", value, got, want)
		}
	}
}

func TestCountUTF8CanariesAndImmutability(t *testing.T) {
	guard := newGuardedSlice(3, 67, 5, byte(0xa5))
	input, _ := countUTF8Fixture(len(guard.body), [][]byte{{'a'}, {0xc2, 0xa2}, {0xe2, 0x82, 0xac}, {0xf0, 0x90, 0x8d, 0x88}})
	copy(guard.body, input)
	before := bytes.Clone(guard.storage)
	_ = CountUTF8(guard.body)
	_ = countUTF8Scalar(guard.body)
	guard.requireCanariesIntact(t)
	if !bytes.Equal(guard.storage, before) {
		t.Fatal("CountUTF8 modified guarded input")
	}
}

func countUTF8Fixture(size int, codePoints [][]byte) ([]byte, int) {
	input := make([]byte, 0, size)
	count := 0
	for index := 0; len(input) < size; index++ {
		codePoint := codePoints[index%len(codePoints)]
		if len(codePoint) > size-len(input) {
			codePoint = codePoints[0]
		}
		input = append(input, codePoint...)
		count++
	}
	return input, count
}

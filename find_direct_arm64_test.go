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

//go:build arm64

package simdutf

import (
	"bytes"
	"testing"
)

// Direct differential coverage for the arm64 NEON translation of
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/arm64/arm_find.cpp.
// Cases stay darwin/arm64-friendly: empty, short, aligned/unaligned lengths,
// first/middle/last hits, misses, and NUL/high-unit needles.

func TestDirectARM64FindNEONAgainstScalar(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		value byte
	}{
		{name: "nil", input: nil, value: 'a'},
		{name: "empty", input: []byte{}, value: 'a'},
		{name: "hit-first", input: []byte("abc"), value: 'a'},
		{name: "hit-middle", input: []byte("bac"), value: 'a'},
		{name: "hit-last", input: []byte("bca"), value: 'a'},
		{name: "miss", input: []byte("bcd"), value: 'a'},
		{name: "nul-hit", input: []byte{'A', 'B', 0, 'C'}, value: 0},
		{name: "short", input: []byte{7}, value: 7},
		{name: "len15", input: append(bytes.Repeat([]byte{'x'}, 14), 'a'), value: 'a'},
		{name: "len16-first", input: append([]byte{'a'}, bytes.Repeat([]byte{'x'}, 15)...), value: 'a'},
		{name: "len16-last", input: append(bytes.Repeat([]byte{'x'}, 15), 'a'), value: 'a'},
		{name: "len16-miss", input: bytes.Repeat([]byte{'x'}, 16), value: 'a'},
		{name: "len63", input: append(bytes.Repeat([]byte{'x'}, 62), 'a'), value: 'a'},
		{name: "len64-first", input: append([]byte{'a'}, bytes.Repeat([]byte{'x'}, 63)...), value: 'a'},
		{name: "len64-mid", input: append(append(bytes.Repeat([]byte{'x'}, 40), 'a'), bytes.Repeat([]byte{'x'}, 23)...), value: 'a'},
		{name: "len64-last", input: append(bytes.Repeat([]byte{'x'}, 63), 'a'), value: 'a'},
		{name: "len64-miss", input: bytes.Repeat([]byte{'x'}, 64), value: 'a'},
		{name: "len65", input: append(bytes.Repeat([]byte{'x'}, 64), 'a'), value: 'a'},
		{name: "len127", input: append(bytes.Repeat([]byte{'x'}, 126), 'a'), value: 'a'},
		{name: "len128-chunk2", input: append(bytes.Repeat([]byte{'x'}, 80), append([]byte{'a'}, bytes.Repeat([]byte{'x'}, 47)...)...), value: 'a'},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := findScalar(tc.input, tc.value)
			if got := findNEON(tc.input, tc.value); got != want {
				t.Fatalf("findNEON(%q, %q) = %d, want %d", tc.input, tc.value, got, want)
			}
		})
	}
}

func TestDirectARM64FindUTF16NEONAgainstScalar(t *testing.T) {
	repeat := func(unit uint16, n int) []uint16 {
		out := make([]uint16, n)
		for i := range out {
			out[i] = unit
		}
		return out
	}
	cases := []struct {
		name  string
		input []uint16
		value uint16
	}{
		{name: "nil", input: nil, value: 'a'},
		{name: "empty", input: []uint16{}, value: 'a'},
		{name: "hit-first", input: []uint16{'a', 'b', 'c'}, value: 'a'},
		{name: "hit-middle", input: []uint16{'b', 'a', 'c'}, value: 'a'},
		{name: "hit-last", input: []uint16{'b', 'c', 'a'}, value: 'a'},
		{name: "miss", input: []uint16{'b', 'c', 'd'}, value: 'a'},
		{name: "nul-hit", input: []uint16{'A', 0, 'C'}, value: 0},
		{name: "high-unit", input: []uint16{0x20, 0xd800, 0x20}, value: 0xd800},
		{name: "short", input: []uint16{7}, value: 7},
		{name: "len7", input: append(repeat('x', 6), 'a'), value: 'a'},
		{name: "len8-first", input: append([]uint16{'a'}, repeat('x', 7)...), value: 'a'},
		{name: "len8-last", input: append(repeat('x', 7), 'a'), value: 'a'},
		{name: "len8-miss", input: repeat('x', 8), value: 'a'},
		{name: "len31", input: append(repeat('x', 30), 'a'), value: 'a'},
		{name: "len32-first", input: append([]uint16{'a'}, repeat('x', 31)...), value: 'a'},
		{name: "len32-mid", input: append(append(repeat('x', 20), 'a'), repeat('x', 11)...), value: 'a'},
		{name: "len32-last", input: append(repeat('x', 31), 'a'), value: 'a'},
		{name: "len32-miss", input: repeat('x', 32), value: 'a'},
		{name: "len33", input: append(repeat('x', 32), 'a'), value: 'a'},
		{name: "len64-chunk2", input: append(repeat('x', 40), append([]uint16{'a'}, repeat('x', 23)...)...), value: 'a'},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := findUTF16Scalar(tc.input, tc.value)
			if got := findUTF16NEON(tc.input, tc.value); got != want {
				t.Fatalf("findUTF16NEON(%v, %#x) = %d, want %d", tc.input, tc.value, got, want)
			}
		})
	}
}

func TestDirectARM64FindNEONUnalignedSubslice(t *testing.T) {
	buf := bytes.Repeat([]byte{'x'}, 96)
	buf[17] = 'a'
	buf[50] = 'a'
	for off := 1; off < 16; off++ {
		input := buf[off:80]
		want := findScalar(input, 'a')
		if got := findNEON(input, 'a'); got != want {
			t.Fatalf("off=%d findNEON = %d, want %d", off, got, want)
		}
		if got := findNEON(input, 'z'); got != findScalar(input, 'z') {
			t.Fatalf("off=%d miss mismatch", off)
		}
	}
	u16 := make([]uint16, 96)
	for i := range u16 {
		u16[i] = 'x'
	}
	u16[17] = 'a'
	u16[50] = 'a'
	for off := 1; off < 8; off++ {
		input := u16[off:80]
		want := findUTF16Scalar(input, 'a')
		if got := findUTF16NEON(input, 'a'); got != want {
			t.Fatalf("u16 off=%d findUTF16NEON = %d, want %d", off, got, want)
		}
	}
}

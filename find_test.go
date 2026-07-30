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
// tests/find_tests.cpp and src/fallback/implementation.cpp:575-593.
// Nil/empty, first-byte hit, and miss->len cases are narrow Go API coverage.

func TestFind(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		value byte
		want  int
	}{
		{name: "nil", input: nil, value: 'a', want: 0},
		{name: "empty", input: []byte{}, value: 'a', want: 0},
		{name: "hit-first", input: []byte("abc"), value: 'a', want: 0},
		{name: "hit-middle", input: []byte("bac"), value: 'a', want: 1},
		{name: "hit-last", input: []byte("bca"), value: 'a', want: 2},
		{name: "miss", input: []byte("bcd"), value: 'a', want: 3},
		{name: "nul-hit", input: []byte{'A', 'B', 0, 'C'}, value: 0, want: 2},
		{name: "nul-miss", input: []byte("ABC"), value: 0, want: 3},
		{name: "short", input: []byte{7}, value: 7, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Find(tc.input, tc.value); got != tc.want {
				t.Fatalf("Find() = %d, want %d", got, tc.want)
			}
			if got := findScalar(tc.input, tc.value); got != tc.want {
				t.Fatalf("findScalar() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFindUTF16(t *testing.T) {
	tests := []struct {
		name  string
		input []uint16
		value uint16
		want  int
	}{
		{name: "nil", input: nil, value: 'a', want: 0},
		{name: "empty", input: []uint16{}, value: 'a', want: 0},
		{name: "hit-first", input: []uint16{'a', 'b', 'c'}, value: 'a', want: 0},
		{name: "hit-middle", input: []uint16{'b', 'a', 'c'}, value: 'a', want: 1},
		{name: "hit-last", input: []uint16{'b', 'c', 'a'}, value: 'a', want: 2},
		{name: "miss", input: []uint16{'b', 'c', 'd'}, value: 'a', want: 3},
		{name: "nul-hit", input: []uint16{'A', 0, 'C'}, value: 0, want: 1},
		{name: "high-unit", input: []uint16{0x20, 0xd800, 0x20}, value: 0xd800, want: 1},
		{name: "short", input: []uint16{7}, value: 7, want: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FindUTF16(tc.input, tc.value); got != tc.want {
				t.Fatalf("FindUTF16() = %d, want %d", got, tc.want)
			}
			if got := findUTF16Scalar(tc.input, tc.value); got != tc.want {
				t.Fatalf("findUTF16Scalar() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFindLiveDispatchIsScalar(t *testing.T) {
	if !sameFunction(activeImplementation.find, findScalar) {
		t.Fatalf("live find selected %p, want scalar %p", activeImplementation.find, findScalar)
	}
	if !sameFunction(activeImplementation.findUTF16, findUTF16Scalar) {
		t.Fatalf("live findUTF16 selected %p, want scalar %p", activeImplementation.findUTF16, findUTF16Scalar)
	}
}

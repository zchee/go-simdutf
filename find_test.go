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
	"slices"
	"testing"
)

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

func TestFindLiveDispatchMatchesQualification(t *testing.T) {
	// Find may be promoted by qualification on linux-amd64; FindUTF16 stays scalar-first.
	want := makeImplementation(detectSelectionInput())
	if !sameFunction(activeImplementation.find, want.find) {
		t.Fatalf("live find selected %p, want qualification selection %p", activeImplementation.find, want.find)
	}
	if !sameFunction(activeImplementation.findUTF16, findUTF16Scalar) {
		t.Fatalf("live findUTF16 selected %p, want scalar %p", activeImplementation.findUTF16, findUTF16Scalar)
	}
}

// Hand-authored Go-only direct Find differential fuzz registry scaffolding for
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b: fuzz/find.cpp,
// src/generic/find.h and src/arm64/arm_find.cpp. It defines test metadata only
// and adds no product behavior.

type findFuzzVariant struct {
	name string
	variant[func([]byte, byte) int]
}

var findFuzzVariants []findFuzzVariant

func registerFindFuzzVariant(candidate findFuzzVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct Find fuzz variant")
	}
	for _, registered := range findFuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct Find fuzz variant " + candidate.name)
		}
	}
	findFuzzVariants = append(findFuzzVariants, candidate)
}

func TestRegisterFindFuzzVariant(t *testing.T) {
	saved := findFuzzVariants
	defer func() { findFuzzVariants = saved }()
	registerFindFuzzVariant(findFuzzVariant{
		name: "test-scalar",
		variant: variant[func([]byte, byte) int]{
			value:     findScalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := findFuzzVariants[len(findFuzzVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, findScalar) {
		t.Fatalf("registered fuzz variant = %q %p, want test-scalar %p", got.name, got.value, findScalar)
	}
}

type findUTF16FuzzVariant struct {
	name string
	variant[func([]uint16, uint16) int]
}

var findUTF16FuzzVariants []findUTF16FuzzVariant

func registerFindUTF16FuzzVariant(candidate findUTF16FuzzVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct FindUTF16 fuzz variant")
	}
	for _, registered := range findUTF16FuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct FindUTF16 fuzz variant " + candidate.name)
		}
	}
	findUTF16FuzzVariants = append(findUTF16FuzzVariants, candidate)
}

func TestRegisterFindUTF16FuzzVariant(t *testing.T) {
	saved := findUTF16FuzzVariants
	defer func() { findUTF16FuzzVariants = saved }()
	registerFindUTF16FuzzVariant(findUTF16FuzzVariant{
		name: "test-scalar",
		variant: variant[func([]uint16, uint16) int]{
			value:     findUTF16Scalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := findUTF16FuzzVariants[len(findUTF16FuzzVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, findUTF16Scalar) {
		t.Fatalf("registered fuzz variant = %q %p, want test-scalar %p", got.name, got.value, findUTF16Scalar)
	}
}

// Go-only public/direct-versus-scalar differential fuzz scaffold for the find
// port pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// fuzz/find.cpp, src/generic/find.h and src/arm64/arm_find.cpp. The scalar
// function is the explicit oracle for the public entry point and every
// registered direct accelerated implementation; the exact match index, not
// merely found/absent, is compared.

func FuzzFind(f *testing.F) {
	for _, seed := range []struct {
		data   []byte
		needle byte
	}{
		{data: nil, needle: 'a'},
		{data: []byte{}, needle: 'a'},
		{data: []byte("abc"), needle: 'a'},
		{data: []byte("bac"), needle: 'a'},
		{data: []byte("bca"), needle: 'a'},
		{data: []byte("bcd"), needle: 'a'},
		{data: []byte{'A', 'B', 0, 'C'}, needle: 0},
		{data: bytes.Repeat([]byte{'x'}, 63), needle: 'a'},
		{data: append(bytes.Repeat([]byte{'x'}, 62), 'a'), needle: 'a'},
		{data: append([]byte{'a'}, bytes.Repeat([]byte{'x'}, 63)...), needle: 'a'},
		{data: append(bytes.Repeat([]byte{'x'}, 63), 'a'), needle: 'a'},
		{data: bytes.Repeat([]byte{'x'}, 64), needle: 'a'},
		{data: append(bytes.Repeat([]byte{'x'}, 64), 'a'), needle: 'a'},
		{data: append(bytes.Repeat([]byte{'x'}, 128), 'a'), needle: 'a'},
		{data: bytes.Repeat([]byte{0xa5}, 129), needle: 0xa5},
		{data: bytes.Repeat([]byte{'x'}, 257), needle: 'a'},
	} {
		f.Add(seed.data, seed.needle)
	}
	f.Fuzz(func(t *testing.T, data []byte, needle byte) {
		guard := newGuardedSlice(2, len(data), 3, byte(0xa5))
		copy(guard.body, data)
		before := bytes.Clone(guard.storage)
		want := findScalar(guard.body, needle)
		if got := Find(guard.body, needle); got != want {
			t.Errorf("Find() = %d, scalar = %d", got, want)
		}
		selection := detectSelectionInput()
		for _, candidate := range findFuzzVariants {
			if !candidate.supportedBy(selection) {
				continue
			}
			if got := candidate.value(guard.body, needle); got != want {
				t.Errorf("%s Find() = %d, scalar = %d", candidate.name, got, want)
			}
		}
		guard.requireCanariesIntact(t)
		if !bytes.Equal(guard.storage, before) {
			t.Fatal("Find modified input or canaries")
		}
	})
}

// Go-only public/direct-versus-scalar differential fuzz scaffold for the
// findUTF16 port pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b: fuzz/find.cpp,
// src/generic/find.h and src/arm64/arm_find.cpp. Go's fuzz engine cannot take
// []uint16, so the haystack arrives as little-endian byte pairs with any odd
// trailing byte dropped, and the needle is assembled from two independent
// fuzzer bytes so the full 16-bit code-unit space stays reachable.

func FuzzFindUTF16(f *testing.F) {
	encode := func(units ...uint16) []byte {
		raw := make([]byte, len(units)*2)
		for i, unit := range units {
			raw[2*i] = byte(unit)
			raw[2*i+1] = byte(unit >> 8)
		}
		return raw
	}
	repeat := func(unit uint16, n int) []uint16 {
		out := make([]uint16, n)
		for i := range out {
			out[i] = unit
		}
		return out
	}
	for _, seed := range []struct {
		raw      []byte
		needleLo byte
		needleHi byte
	}{
		{raw: nil, needleLo: 'a'},
		{raw: []byte{}, needleLo: 'a'},
		{raw: encode('a', 'b', 'c'), needleLo: 'a'},
		{raw: encode('b', 'a', 'c'), needleLo: 'a'},
		{raw: encode('b', 'c', 'a'), needleLo: 'a'},
		{raw: encode('b', 'c', 'd'), needleLo: 'a'},
		{raw: encode('A', 0, 'C'), needleLo: 0},
		{raw: encode(0x20, 0xd800, 0x20), needleLo: 0x00, needleHi: 0xd8},
		{raw: encode(0x1234, 0x5678, 0xfffd), needleLo: 0x78, needleHi: 0x56},
		{raw: encode(0x1234, 0x5678, 0xfffd), needleLo: 0xfd, needleHi: 0xff},
		{raw: append(encode(0x20, 0xd800), 0x41), needleLo: 0x00, needleHi: 0xd8},
		{raw: encode(repeat('x', 31)...), needleLo: 'a'},
		{raw: encode(append(repeat('x', 31), 'a')...), needleLo: 'a'},
		{raw: encode(append([]uint16{'a'}, repeat('x', 31)...)...), needleLo: 'a'},
		{raw: encode(repeat('x', 32)...), needleLo: 'a'},
		{raw: encode(append(repeat('x', 32), 0x0100)...), needleLo: 0x00, needleHi: 0x01},
		{raw: encode(append(repeat(0xd800, 64), 0xdc00)...), needleLo: 0x00, needleHi: 0xdc},
		{raw: encode(repeat('x', 257)...), needleLo: 'a'},
	} {
		f.Add(seed.raw, seed.needleLo, seed.needleHi)
	}
	f.Fuzz(func(t *testing.T, raw []byte, needleLo, needleHi byte) {
		needle := uint16(needleLo) | uint16(needleHi)<<8
		n := len(raw) / 2
		guard := newGuardedSlice(2, n, 3, uint16(0xa55a))
		for i := range n {
			guard.body[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		before := slices.Clone(guard.storage)
		want := findUTF16Scalar(guard.body, needle)
		if got := FindUTF16(guard.body, needle); got != want {
			t.Errorf("FindUTF16() = %d, scalar = %d", got, want)
		}
		selection := detectSelectionInput()
		for _, candidate := range findUTF16FuzzVariants {
			if !candidate.supportedBy(selection) {
				continue
			}
			if got := candidate.value(guard.body, needle); got != want {
				t.Errorf("%s FindUTF16() = %d, scalar = %d", candidate.name, got, want)
			}
		}
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatal("FindUTF16 modified input or canaries")
		}
	})
}

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
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
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

// Hand-authored Go-only direct CountUTF8 benchmark registry scaffolding for
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b. It defines
// test-only variant slots and adds no product behavior.

type countUTF8DirectVariant struct {
	name string
	variant[func([]byte) int]
}

var countUTF8DirectVariants []countUTF8DirectVariant

func registerCountUTF8DirectVariant(candidate countUTF8DirectVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct CountUTF8 benchmark variant")
	}
	for _, registered := range countUTF8DirectVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct CountUTF8 benchmark variant " + candidate.name)
		}
	}
	countUTF8DirectVariants = append(countUTF8DirectVariants, candidate)
}

func TestRegisterCountUTF8DirectVariant(t *testing.T) {
	saved := countUTF8DirectVariants
	defer func() { countUTF8DirectVariants = saved }()
	registerCountUTF8DirectVariant(countUTF8DirectVariant{
		name: "test-scalar",
		variant: variant[func([]byte) int]{
			value:     countUTF8Scalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := countUTF8DirectVariants[len(countUTF8DirectVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, countUTF8Scalar) {
		t.Fatalf("registered direct variant = %q %p, want test-scalar %p", got.name, got.value, countUTF8Scalar)
	}
}

// Hand-authored Go-only direct CountUTF8 differential fuzz registry
// scaffolding for
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b. It defines test
// metadata only and adds no product behavior.

type countUTF8FuzzVariant struct {
	name string
	variant[func([]byte) int]
}

var countUTF8FuzzVariants []countUTF8FuzzVariant

func registerCountUTF8FuzzVariant(candidate countUTF8FuzzVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct CountUTF8 fuzz variant")
	}
	for _, registered := range countUTF8FuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct CountUTF8 fuzz variant " + candidate.name)
		}
	}
	countUTF8FuzzVariants = append(countUTF8FuzzVariants, candidate)
}

func TestRegisterCountUTF8FuzzVariant(t *testing.T) {
	saved := countUTF8FuzzVariants
	defer func() { countUTF8FuzzVariants = saved }()
	registerCountUTF8FuzzVariant(countUTF8FuzzVariant{
		name: "test-scalar",
		variant: variant[func([]byte) int]{
			value:     countUTF8Scalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := countUTF8FuzzVariants[len(countUTF8FuzzVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, countUTF8Scalar) {
		t.Fatalf("registered fuzz variant = %q %p, want test-scalar %p", got.name, got.value, countUTF8Scalar)
	}
}

// Go-only public-versus-scalar differential fuzz scaffold for the count_utf8
// port pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// fuzz/conversion.cpp and tests/count_utf8.cpp:11-84. The scalar function is
// the explicit oracle for the public entry point and every registered direct
// accelerated implementation.

func FuzzCountUTF8(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{},
		[]byte("köttbulle"),
		{'a', 0xc2, 0xa2, 0xe2, 0x82, 0xac, 0xf0, 0x90, 0x8d, 0x88},
		{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xff},
		bytes.Repeat([]byte{'a'}, 67),
		bytes.Repeat([]byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xff}, 1344),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		guard := newGuardedSlice(2, len(input), 3, byte(0xa5))
		copy(guard.body, input)
		before := bytes.Clone(guard.storage)
		want := countUTF8Scalar(guard.body)
		if got := CountUTF8(guard.body); got != want {
			t.Errorf("CountUTF8() = %d, scalar = %d", got, want)
		}
		selection := detectSelectionInput()
		for _, candidate := range countUTF8FuzzVariants {
			if !candidate.supportedBy(selection) {
				continue
			}
			if got := candidate.value(guard.body); got != want {
				t.Errorf("%s CountUTF8() = %d, scalar = %d", candidate.name, got, want)
			}
		}
		guard.requireCanariesIntact(t)
		if !bytes.Equal(guard.storage, before) {
			t.Fatal("CountUTF8 modified input or canaries")
		}
	})
}

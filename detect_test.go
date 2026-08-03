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

// Hand-authored Go-only direct DetectEncodings differential fuzz registry
// scaffolding for simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// fuzz/misc.cpp and src/fallback/implementation.cpp:8-32. It defines test
// metadata only and adds no product behavior.

type detectEncodingsFuzzVariant struct {
	name string
	variant[func([]byte) Encoding]
}

var detectEncodingsFuzzVariants []detectEncodingsFuzzVariant

func registerDetectEncodingsFuzzVariant(candidate detectEncodingsFuzzVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct DetectEncodings fuzz variant")
	}
	for _, registered := range detectEncodingsFuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct DetectEncodings fuzz variant " + candidate.name)
		}
	}
	detectEncodingsFuzzVariants = append(detectEncodingsFuzzVariants, candidate)
}

func TestRegisterDetectEncodingsFuzzVariant(t *testing.T) {
	saved := detectEncodingsFuzzVariants
	defer func() { detectEncodingsFuzzVariants = saved }()
	registerDetectEncodingsFuzzVariant(detectEncodingsFuzzVariant{
		name: "test-scalar",
		variant: variant[func([]byte) Encoding]{
			value:     detectEncodingsScalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := detectEncodingsFuzzVariants[len(detectEncodingsFuzzVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, detectEncodingsScalar) {
		t.Fatalf("registered fuzz variant = %q %p, want test-scalar %p", got.name, got.value, detectEncodingsScalar)
	}
}

// Go-only public/direct-versus-scalar differential fuzz scaffold for the
// detect port pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b: fuzz/misc.cpp and
// src/fallback/implementation.cpp:8-32. The scalar function is the explicit
// oracle for the public entry point and every registered direct accelerated
// implementation; the full returned Encoding bitset is compared, not just the
// autodetect priority winner.

func FuzzDetectEncodings(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{},
		[]byte("hello"),
		[]byte("hi"),
		{0xef, 0xbb, 0xbf},
		{0xff, 0xfe},
		{0xfe, 0xff},
		{0xff, 0xfe, 0x00, 0x00},
		{0x00, 0x00, 0xfe, 0xff},
		{'A', 0, 'B', 0},
		{'A', 0, 0, 0},
		{0x00, 0xd8, 0x00, 0xdc},
		{0x20, 0xd8, 0x00, 0x00},
		{0xff},
		{
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 223, 164, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
		},
		bytes.Repeat([]byte{'a'}, 67),
		bytes.Repeat([]byte{0xc2, 0xa2}, 64),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		guard := newGuardedSlice(2, len(input), 3, byte(0xa5))
		copy(guard.body, input)
		before := bytes.Clone(guard.storage)
		want := detectEncodingsScalar(guard.body)
		if got := DetectEncodings(guard.body); got != want {
			t.Errorf("DetectEncodings() = %d, scalar = %d", got, want)
		}
		selection := detectSelectionInput()
		for _, candidate := range detectEncodingsFuzzVariants {
			if !candidate.supportedBy(selection) {
				continue
			}
			if got := candidate.value(guard.body); got != want {
				t.Errorf("%s DetectEncodings() = %d, scalar = %d", candidate.name, got, want)
			}
		}
		guard.requireCanariesIntact(t)
		if !bytes.Equal(guard.storage, before) {
			t.Fatal("DetectEncodings modified input or canaries")
		}
	})
}

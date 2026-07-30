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

//go:build amd64 && goexperiment.simd

package simdutf

import "testing"

func requireDetectArchsimdAVX2(t *testing.T) {
	t.Helper()
	candidate := variant[func()]{
		value:     func() {},
		kind:      implementationArchsimd,
		required:  cpuAVX2,
		available: true,
	}
	if !candidate.supportedBy(detectSelectionInput()) {
		t.Skip("direct archsimd AVX2 implementation is unsupported")
	}
}

func TestDirectArchsimdDetectEncodingsAgainstScalar(t *testing.T) {
	requireDetectArchsimdAVX2(t)
	cases := []struct {
		name  string
		input []byte
	}{
		{name: "nil", input: nil},
		{name: "empty", input: []byte{}},
		{name: "utf8-only-odd", input: []byte("hello")},
		{name: "utf8-bom", input: []byte{0xef, 0xbb, 0xbf}},
		{name: "utf16le-bom", input: []byte{0xff, 0xfe}},
		{name: "utf16be-bom", input: []byte{0xfe, 0xff}},
		{name: "utf32le-bom", input: []byte{0xff, 0xfe, 0x00, 0x00}},
		{name: "utf32be-bom", input: []byte{0x00, 0x00, 0xfe, 0xff}},
		{name: "utf16le-even-ascii", input: []byte{'A', 0, 'B', 0}},
		{name: "utf32le-ascii-unit", input: []byte{'A', 0, 0, 0}},
		{name: "ascii-even", input: []byte("hi")},
		{name: "short-odd-invalid", input: []byte{0xff}},
		{name: "invalid-mix-issue516", input: []byte{0x20, 0xd8, 0x00, 0x00}},
		{name: "utf16le-only-surrogate-pair", input: []byte{0x00, 0xd8, 0x00, 0xdc}},
		{name: "issue519", input: []byte{
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 223, 164, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
			32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32, 32,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectEncodingsArchsimd(tc.input)
			want := detectEncodingsScalar(tc.input)
			if got != want {
				t.Fatalf("detectEncodingsArchsimd(%q) = %d, want scalar %d", tc.input, got, want)
			}
		})
	}
}

func TestArchsimdDetectEncodingsHelperExportsProvider(t *testing.T) {
	fn := archsimdDetectEncodings()
	if fn == nil {
		t.Fatal("archsimdDetectEncodings() = nil, want detectEncodingsArchsimd")
	}
	if !sameFunction(fn, detectEncodingsArchsimd) {
		t.Fatalf("archsimdDetectEncodings() = %p, want detectEncodingsArchsimd %p", fn, detectEncodingsArchsimd)
	}
}

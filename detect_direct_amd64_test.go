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

//go:build amd64

package simdutf

import "testing"

func requireDetectAMD64Variant(t *testing.T, feature cpuFeatures) {
	t.Helper()
	if detectAMD64Features()&feature != feature {
		t.Skip("required amd64 SIMD feature is unavailable")
	}
}

func detectEncodingsDirectCases() []struct {
	name  string
	input []byte
} {
	return []struct {
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
}

func TestDirectAMD64DetectEncodingsAgainstScalar(t *testing.T) {
	variants := []struct {
		name    string
		feature cpuFeatures
		fn      func([]byte) Encoding
	}{
		{name: "westmere", feature: cpuSSSE3, fn: detectEncodingsWestmere},
		{name: "haswell", feature: cpuAVX2, fn: detectEncodingsHaswell},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			requireDetectAMD64Variant(t, v.feature)
			for _, tc := range detectEncodingsDirectCases() {
				t.Run(tc.name, func(t *testing.T) {
					got := v.fn(tc.input)
					want := detectEncodingsScalar(tc.input)
					if got != want {
						t.Fatalf("%s(%q) = %d, want scalar %d", v.name, tc.input, got, want)
					}
				})
			}
		})
	}
}

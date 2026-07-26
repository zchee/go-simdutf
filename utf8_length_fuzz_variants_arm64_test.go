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

import "testing"

// Hand-authored Go-only direct fuzz registration for the arm64 UTF-8 length
// ports pinned to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f
// (tree eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// src/generic/utf8.h:8-17,72-86 and
// src/arm64/implementation.cpp:1121-1124,1178-1181,1292-1295.
func init() {
	registerUTF8LengthFuzzVariant(utf8LengthFuzzVariant{
		name: "neon",
		latin1: variant[func([]byte) int]{
			value: latin1LengthFromUTF8NEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
		utf16: variant[func([]byte) int]{
			value: utf16LengthFromUTF8NEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
		utf32: variant[func([]byte) int]{
			value: utf32LengthFromUTF8NEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
	})
}

func TestUTF8LengthNEONFuzzRegistration(t *testing.T) {
	for _, candidate := range utf8LengthFuzzVariants {
		if candidate.name != "neon" {
			continue
		}
		selection := selectionInput{features: cpuNEON}
		if !candidate.latin1.supportedBy(selection) ||
			!candidate.utf16.supportedBy(selection) ||
			!candidate.utf32.supportedBy(selection) {
			t.Fatal("neon differential-fuzz registration is not supported by cpuNEON")
		}
		if !sameFunction(candidate.latin1.value, latin1LengthFromUTF8NEON) ||
			!sameFunction(candidate.utf16.value, utf16LengthFromUTF8NEON) ||
			!sameFunction(candidate.utf32.value, utf32LengthFromUTF8NEON) {
			t.Fatal("neon differential-fuzz registration has unexpected functions")
		}
		return
	}
	t.Fatal("neon differential-fuzz registration not found")
}

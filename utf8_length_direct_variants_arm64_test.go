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

// Go-only direct benchmark registration for the arm64 UTF-8 length ports
// pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
// src/arm64/implementation.cpp:1121-1124,1178-1181,1292-1295. It changes no
// frozen benchmark name, corpus, setup, or product dispatch.
func init() {
	registerUTF8LengthDirectVariant(utf8LengthDirectVariant{
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

func TestUTF8LengthNEONDirectRegistration(t *testing.T) {
	for _, candidate := range utf8LengthDirectVariants {
		if candidate.name != "neon" {
			continue
		}
		selection := selectionInput{features: cpuNEON}
		if !candidate.latin1.supportedBy(selection) ||
			!candidate.utf16.supportedBy(selection) ||
			!candidate.utf32.supportedBy(selection) {
			t.Fatal("neon direct benchmark registration is not supported by cpuNEON")
		}
		if !sameFunction(candidate.latin1.value, latin1LengthFromUTF8NEON) ||
			!sameFunction(candidate.utf16.value, utf16LengthFromUTF8NEON) ||
			!sameFunction(candidate.utf32.value, utf32LengthFromUTF8NEON) {
			t.Fatal("neon direct benchmark registration has unexpected functions")
		}
		return
	}
	t.Fatal("neon direct benchmark registration not found")
}

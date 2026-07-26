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

// Hand-authored Go-only differential-fuzz registration for the pinned amd64
// UTF-8 length routes in
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// src/generic/utf8/utf16_length_from_utf8_bytemask.h, src/generic/utf8.h:8-20,
// and the Westmere/Haswell implementation routes.

func init() {
	registerUTF8LengthFuzzVariant(utf8LengthFuzzVariant{
		name:   "westmere",
		latin1: variant[func([]byte) int]{value: latin1LengthFromUTF8Westmere, kind: implementationWestmere, available: true},
		utf16:  variant[func([]byte) int]{value: utf16LengthFromUTF8Westmere, kind: implementationWestmere, available: true},
		utf32:  variant[func([]byte) int]{value: utf32LengthFromUTF8Westmere, kind: implementationWestmere, required: cpuPOPCNT, available: true},
	})
	registerUTF8LengthFuzzVariant(utf8LengthFuzzVariant{
		name:   "haswell",
		latin1: variant[func([]byte) int]{value: latin1LengthFromUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
		utf16:  variant[func([]byte) int]{value: utf16LengthFromUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
		utf32:  variant[func([]byte) int]{value: utf32LengthFromUTF8Haswell, kind: implementationHaswell, required: cpuAVX2 | cpuPOPCNT, available: true},
	})
}

func TestUTF8LengthAMD64FuzzRegistrations(t *testing.T) {
	seen := make(map[string]bool, 2)
	for _, candidate := range utf8LengthFuzzVariants {
		if candidate.name != "westmere" && candidate.name != "haswell" {
			continue
		}
		if seen[candidate.name] {
			t.Fatalf("duplicate %s fuzz registration", candidate.name)
		}
		seen[candidate.name] = true
		checkUTF8LengthAMD64Registration(t, candidate.name, candidate.latin1, candidate.utf16, candidate.utf32)
	}
	for _, name := range []string{"westmere", "haswell"} {
		if !seen[name] {
			t.Errorf("%s fuzz registration not found", name)
		}
	}
}

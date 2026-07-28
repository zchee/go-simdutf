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

// Go-only direct benchmark registration for the pinned amd64 UTF-8 length
// families. It changes no frozen benchmark name, corpus, setup, or product
// dispatch. Source authority is
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811): src/westmere/implementation.cpp
// and src/haswell/implementation.cpp length routes.

func init() {
	registerUTF8LengthDirectVariant(utf8LengthDirectVariant{
		name:   "westmere",
		latin1: variant[func([]byte) int]{value: latin1LengthFromUTF8Westmere, kind: implementationWestmere, available: true},
		utf16:  variant[func([]byte) int]{value: utf16LengthFromUTF8Westmere, kind: implementationWestmere, available: true},
		utf32:  variant[func([]byte) int]{value: utf32LengthFromUTF8Westmere, kind: implementationWestmere, required: cpuPOPCNT, available: true},
	})
	registerUTF8LengthDirectVariant(utf8LengthDirectVariant{
		name:   "haswell",
		latin1: variant[func([]byte) int]{value: latin1LengthFromUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
		utf16:  variant[func([]byte) int]{value: utf16LengthFromUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
		utf32:  variant[func([]byte) int]{value: utf32LengthFromUTF8Haswell, kind: implementationHaswell, required: cpuAVX2 | cpuPOPCNT, available: true},
	})
}

func TestUTF8LengthAMD64DirectRegistrations(t *testing.T) {
	seen := make(map[string]bool, 2)
	for _, candidate := range utf8LengthDirectVariants {
		if candidate.name != "westmere" && candidate.name != "haswell" {
			continue
		}
		if seen[candidate.name] {
			t.Fatalf("duplicate %s direct registration", candidate.name)
		}
		seen[candidate.name] = true
		checkUTF8LengthAMD64Registration(t, candidate.name, candidate.latin1, candidate.utf16, candidate.utf32)
	}
	for _, name := range []string{"westmere", "haswell"} {
		if !seen[name] {
			t.Errorf("%s direct registration not found", name)
		}
	}
}

func checkUTF8LengthAMD64Registration(
	t *testing.T,
	name string,
	latin1, utf16, utf32 variant[func([]byte) int],
) {
	t.Helper()
	var wantLatin1, wantUTF16, wantUTF32 func([]byte) int
	var wantKind implementationKind
	var latin1Required, utf16Required, utf32Required cpuFeatures
	switch name {
	case "westmere":
		wantLatin1 = latin1LengthFromUTF8Westmere
		wantUTF16 = utf16LengthFromUTF8Westmere
		wantUTF32 = utf32LengthFromUTF8Westmere
		wantKind = implementationWestmere
		utf32Required = cpuPOPCNT
	case "haswell":
		wantLatin1 = latin1LengthFromUTF8Haswell
		wantUTF16 = utf16LengthFromUTF8Haswell
		wantUTF32 = utf32LengthFromUTF8Haswell
		wantKind = implementationHaswell
		latin1Required = cpuAVX2
		utf16Required = cpuAVX2
		utf32Required = cpuAVX2 | cpuPOPCNT
	default:
		t.Fatalf("unexpected amd64 registration %q", name)
	}
	if !sameFunction(latin1.value, wantLatin1) ||
		!sameFunction(utf16.value, wantUTF16) ||
		!sameFunction(utf32.value, wantUTF32) {
		t.Errorf("%s registration has unexpected functions", name)
	}
	for operation, check := range map[string]struct {
		cell     variant[func([]byte) int]
		required cpuFeatures
	}{
		"latin1": {latin1, latin1Required},
		"utf16":  {utf16, utf16Required},
		"utf32":  {utf32, utf32Required},
	} {
		if check.cell.kind != wantKind || check.cell.required != check.required || !check.cell.available {
			t.Errorf("%s %s metadata = kind %d required %#x available %t, want kind %d required %#x available true",
				name, operation, check.cell.kind, check.cell.required, check.cell.available, wantKind, check.required)
		}
		if !check.cell.supportedBy(selectionInput{features: check.required}) {
			t.Errorf("%s %s is not supported with required features %#x", name, operation, check.required)
		}
		for feature := cpuFeatures(1); feature <= cpuNEON; feature <<= 1 {
			if check.required&feature == 0 {
				continue
			}
			missing := check.required &^ feature
			if check.cell.supportedBy(selectionInput{features: missing}) {
				t.Errorf("%s %s supported with required feature %#x missing", name, operation, feature)
			}
		}
	}
}

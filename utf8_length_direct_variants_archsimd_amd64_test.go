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

func init() {
	registerUTF8LengthDirectVariant(utf8LengthDirectVariant{
		name:   "archsimd",
		latin1: variant[func([]byte) int]{value: latin1LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
		utf16:  variant[func([]byte) int]{value: utf16LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
		utf32:  variant[func([]byte) int]{value: utf32LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2 | cpuPOPCNT, available: true},
	})
}

func TestUTF8LengthArchsimdDirectRegistration(t *testing.T) {
	candidate := findUTF8LengthArchsimdDirectVariant(t)
	checkUTF8LengthArchsimdRegistration(t, candidate.latin1, candidate.utf16, candidate.utf32)
}

func findUTF8LengthArchsimdDirectVariant(t *testing.T) utf8LengthDirectVariant {
	t.Helper()
	var found *utf8LengthDirectVariant
	for i := range utf8LengthDirectVariants {
		if utf8LengthDirectVariants[i].name != "archsimd" {
			continue
		}
		if found != nil {
			t.Fatal("duplicate archsimd direct registration")
		}
		found = &utf8LengthDirectVariants[i]
	}
	if found == nil {
		t.Fatal("archsimd direct registration not found")
	}
	return *found
}

func checkUTF8LengthArchsimdRegistration(
	t *testing.T,
	latin1, utf16, utf32 variant[func([]byte) int],
) {
	t.Helper()
	checks := []struct {
		name     string
		cell     variant[func([]byte) int]
		want     func([]byte) int
		required cpuFeatures
	}{
		{name: "latin1", cell: latin1, want: latin1LengthFromUTF8Archsimd, required: cpuAVX2},
		{name: "utf16", cell: utf16, want: utf16LengthFromUTF8Archsimd, required: cpuAVX2},
		{name: "utf32", cell: utf32, want: utf32LengthFromUTF8Archsimd, required: cpuAVX2 | cpuPOPCNT},
	}
	for _, check := range checks {
		if !sameFunction(check.cell.value, check.want) || check.cell.kind != implementationArchsimd ||
			check.cell.required != check.required || !check.cell.available {
			t.Errorf("%s metadata/function mismatch: kind %d required %#x available %t",
				check.name, check.cell.kind, check.cell.required, check.cell.available)
		}
		if !check.cell.supportedBy(selectionInput{features: check.required, archsimdAVX2: true}) {
			t.Errorf("%s unsupported with all required gates", check.name)
		}
		if check.cell.supportedBy(selectionInput{features: check.required}) {
			t.Errorf("%s supported without archsimd AVX2 gate", check.name)
		}
		for feature := cpuFeatures(1); feature <= cpuNEON; feature <<= 1 {
			if check.required&feature == 0 {
				continue
			}
			if check.cell.supportedBy(selectionInput{features: check.required &^ feature, archsimdAVX2: true}) {
				t.Errorf("%s supported with feature %#x missing", check.name, feature)
			}
		}
	}
}

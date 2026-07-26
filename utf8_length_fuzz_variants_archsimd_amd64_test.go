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
	registerUTF8LengthFuzzVariant(utf8LengthFuzzVariant{
		name:   "archsimd",
		latin1: variant[func([]byte) int]{value: latin1LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
		utf16:  variant[func([]byte) int]{value: utf16LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true},
		utf32:  variant[func([]byte) int]{value: utf32LengthFromUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2 | cpuPOPCNT, available: true},
	})
}

func TestUTF8LengthArchsimdFuzzRegistration(t *testing.T) {
	var found *utf8LengthFuzzVariant
	for i := range utf8LengthFuzzVariants {
		if utf8LengthFuzzVariants[i].name != "archsimd" {
			continue
		}
		if found != nil {
			t.Fatal("duplicate archsimd fuzz registration")
		}
		found = &utf8LengthFuzzVariants[i]
	}
	if found == nil {
		t.Fatal("archsimd fuzz registration not found")
	}
	checkUTF8LengthArchsimdRegistration(t, found.latin1, found.utf16, found.utf32)
}

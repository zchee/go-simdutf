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

// Go-only public-versus-scalar differential fuzz scaffold for the count_utf8
// port pinned to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
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

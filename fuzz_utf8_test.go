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

// Go-only public-versus-scalar differential fuzz scaffold. The pinned upstream
// fuzz target exercises both validation entry points at
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// fuzz/conversion.cpp:68-74. Scalar functions are the explicit oracle for the
// public entry points and every registered direct accelerated implementation.

func FuzzValidateUTF8(f *testing.F) {
	seeds := [][]byte{
		nil,
		{},
		[]byte("ASCII\x00suffix"),
		{0xc2, 0x80},
		{0xdf, 0xbf},
		{0xe0, 0xa0, 0x80},
		{0xef, 0xbf, 0xbf},
		{0xf0, 0x90, 0x80, 0x80},
		{0xf4, 0x8f, 0xbf, 0xbf},
		{0xef, 0xbf, 0xbd},
		{0xf8},
		{0xe1, 0x80},
		{0x80},
		{0xc0, 0x80},
		{0xf4, 0x90, 0x80, 0x80},
		{0xed, 0xa0, 0x80},
		{0x80, 0xff},
		{0xff, 0x80},
		{0xe1, 0x80, 0x41, 0xff},
		{0xf0, 0x90, 0x80},
		{0xf0, 0x90, 0x80, 0x41},
		append(bytes.Repeat([]byte{' '}, 64), 0xf2, 0x80, 0x80),
		append(bytes.Repeat([]byte{' '}, 63), 0xff),
		[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x80\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
		[]byte("\x0a\x04\x00\x00\xdb\xa1\xdd\xa1\xf1\xa0\xb6\x95\xe4\xb5\x89\xe7\x8f\x95\xe4\xa2\x83\xe7\x95\x89\xe7\x95\x91\xe7\x95\x89\x00\x01\x01\x1a\x20\x28\x00\x00\x60\x00\x00\x23\x00\xf1\xa0\xb6\x95\xe4\xb5\x89\xe7\x8f\x95\xe4\xa2\x83\xe7\x95\x89\xe7\x95\x91\xe7\x81\x00\x00\x01\x01\x1a\x20\x28\x00\x00\x60\x00\x00\x23\x00\x2f\x00\x00\x00\x00\x07\x04\x75\xc2\xa0\x34\x2f\x00\x00\x00\x00\x07\x04\x75\xc2\xa0\x33\x53\x2b"),
	}
	for _, length := range []int{31, 32, 33, 61, 62, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129} {
		valid := bytes.Repeat([]byte{'a'}, length)
		seeds = append(seeds, valid)
		invalid := bytes.Clone(valid)
		invalid[length-1] = 0xff
		seeds = append(seeds, invalid)
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		backing := make([]byte, len(input)+2)
		backing[0], backing[len(backing)-1] = 0xa5, 0x5a
		copy(backing[1:], input)
		before := bytes.Clone(backing)
		guarded := backing[1 : len(backing)-1]
		if got, want := ValidateUTF8WithErrors(guarded), validateUTF8WithErrorsScalar(guarded); got != want {
			t.Fatalf("ValidateUTF8WithErrors() = %+v, scalar = %+v", got, want)
		}
		if got, want := ValidateUTF8(guarded), validateUTF8Scalar(guarded); got != want {
			t.Fatalf("ValidateUTF8() = %t, scalar = %t", got, want)
		}
		for _, candidate := range utf8FuzzVariants {
			if !candidate.validate.supportedBy(detectSelectionInput()) || !candidate.withErrors.supportedBy(detectSelectionInput()) {
				continue
			}
			if got, want := candidate.withErrors.value(guarded), validateUTF8WithErrorsScalar(guarded); got != want {
				t.Fatalf("%s ValidateUTF8WithErrors() = %+v, scalar = %+v", candidate.name, got, want)
			}
			if got, want := candidate.validate.value(guarded), validateUTF8Scalar(guarded); got != want {
				t.Fatalf("%s ValidateUTF8() = %t, scalar = %t", candidate.name, got, want)
			}
		}
		if !bytes.Equal(backing, before) {
			t.Fatal("validation modified input or canaries")
		}
	})
}

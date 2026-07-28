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

//go:build amd64 && (darwin || linux)

package simdutf

import (
	"os"
	"strconv"
	"testing"
)

// Hand-authored Go-only guard-page coverage for the pinned amd64 UTF-8 length
// kernels in simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
// src/generic/utf8/utf16_length_from_utf8_bytemask.h and
// src/generic/utf8.h:8-20.

const utf8LengthAMD64PageGuardEnv = "SIMDUTF_UTF8_LENGTH_AMD64_GUARD"

func TestUTF8LengthAMD64GuardPageNoOverread(t *testing.T) {
	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 257}
	pageSize := os.Getpagesize()
	lengths = append(lengths, pageSize-1, pageSize)
	for _, length := range lengths {
		t.Run("length="+strconv.Itoa(length), func(t *testing.T) {
			runPageGuardSubprocess(t, "TestUTF8LengthAMD64GuardPageHelper", utf8LengthAMD64PageGuardEnv, strconv.Itoa(length))
		})
	}
}

func TestUTF8LengthAMD64GuardPageHelper(t *testing.T) {
	value := os.Getenv(utf8LengthAMD64PageGuardEnv)
	if value == "" {
		return
	}
	length, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("invalid guard length %q: %v", value, err)
	}
	withGuardPageBytes(t, length, func(input []byte) {
		for i := range input {
			input[i] = byte(i*37 + length)
		}
		if got, want := utf16LengthFromUTF8BlocksWestmere(input), utf16LengthFromUTF8Scalar(input[:length&^15]); got != want {
			t.Fatalf("Westmere raw UTF-16 = %d, scalar = %d", got, want)
		}
		if hasUTF8LengthAVX2() {
			if got, want := utf16LengthFromUTF8BlocksHaswell(input), utf16LengthFromUTF8Scalar(input[:length&^31]); got != want {
				t.Fatalf("Haswell raw UTF-16 = %d, scalar = %d", got, want)
			}
		}
		if hasUTF8LengthPOPCNT() {
			if got, want := utf32LengthFromUTF8BlocksWestmere(input), utf32LengthFromUTF8Scalar(input[:length&^63]); got != want {
				t.Fatalf("Westmere raw UTF-32 = %d, scalar = %d", got, want)
			}
		}
		if hasUTF8LengthAVX2() && hasUTF8LengthPOPCNT() {
			if got, want := utf32LengthFromUTF8BlocksHaswell(input), utf32LengthFromUTF8Scalar(input[:length&^63]); got != want {
				t.Fatalf("Haswell raw UTF-32 = %d, scalar = %d", got, want)
			}
		}
		checkUTF8LengthAMD64(t, input)
	})
}

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

//go:build arm64 && (darwin || linux)

package simdutf

import (
	"os"
	"strconv"
	"testing"
)

// Hand-authored Go-only guard-page coverage for the complete-block-only arm64
// UTF-16 length kernel pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/utf8.h:72-86 and
// src/arm64/implementation.cpp:1178-1181. The other two wrappers reuse the
// existing complete-block-safe count_utf8 NEON implementation.

const utf8LengthNEONPageGuardEnv = "SIMDUTF_UTF8_LENGTH_NEON_GUARD"

func TestUTF8LengthNEONGuardPageNoOverread(t *testing.T) {
	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 257}
	pageSize := os.Getpagesize()
	if pageSize-1 > lengths[len(lengths)-1] {
		lengths = append(lengths, pageSize-1)
	}
	if pageSize > lengths[len(lengths)-1] {
		lengths = append(lengths, pageSize)
	}
	for _, length := range lengths {
		t.Run("length="+strconv.Itoa(length), func(t *testing.T) {
			runPageGuardSubprocess(t, "TestUTF8LengthNEONGuardPageHelper", utf8LengthNEONPageGuardEnv, strconv.Itoa(length))
		})
	}
}

func TestUTF8LengthNEONGuardPageHelper(t *testing.T) {
	value := os.Getenv(utf8LengthNEONPageGuardEnv)
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
		complete := length &^ 63
		if got, want := utf16LengthFromUTF8BlocksNEON(input), utf16LengthFromUTF8Scalar(input[:complete]); got != want {
			t.Fatalf("raw block length = %d, scalar = %d", got, want)
		}
		checkUTF8LengthNEON(t, input)
	})
}

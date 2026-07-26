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

// Hand-authored Go-only guard-page coverage for the complete-block-only loads
// in the arm64 count port pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/utf8.h:8-17 and src/arm64/implementation.cpp:1113-1117.

const countUTF8NEONPageGuardEnv = "SIMDUTF_COUNT_UTF8_NEON_GUARD"

func TestCountUTF8NEONGuardPageNoOverread(t *testing.T) {
	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 61, 62, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129}
	largeLength := os.Getpagesize() - 1
	if largeLength <= lengths[len(lengths)-1] {
		t.Fatalf("page size %d does not provide a distinct valid large guard case", os.Getpagesize())
	}
	lengths = append(lengths, largeLength)
	for _, length := range lengths {
		t.Run("length="+strconv.Itoa(length), func(t *testing.T) {
			runPageGuardSubprocess(t, "TestCountUTF8NEONGuardPageHelper", countUTF8NEONPageGuardEnv, strconv.Itoa(length))
		})
	}
}

func TestCountUTF8NEONGuardPageHelper(t *testing.T) {
	value := os.Getenv(countUTF8NEONPageGuardEnv)
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
		if got, want := countUTF8BlocksNEON(input), countUTF8Scalar(input[:complete]); got != want {
			t.Fatalf("block count = %d, want %d", got, want)
		}
		checkCountUTF8NEON(t, input)
	})
}

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

// Hand-authored Go-only guard-page coverage for the complete-group-only loads
// in the Westmere and Haswell count_code_points_bytemask ports pinned to
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de.

const countUTF8AMD64PageGuardEnv = "SIMDUTF_COUNT_UTF8_AMD64_GUARD"

func TestCountUTF8AMD64GuardPageNoOverread(t *testing.T) {
	lengths := []int{0, 1, 31, 32, 33, 63, 64, 65, 127, 128, 129}
	pageSize := os.Getpagesize()
	for _, length := range []int{pageSize - 1, pageSize} {
		if length > lengths[len(lengths)-1] {
			lengths = append(lengths, length)
		}
	}
	for _, length := range lengths {
		t.Run("length="+strconv.Itoa(length), func(t *testing.T) {
			runPageGuardSubprocess(t, "TestCountUTF8AMD64GuardPageHelper", countUTF8AMD64PageGuardEnv, strconv.Itoa(length))
		})
	}
}

func TestCountUTF8AMD64GuardPageHelper(t *testing.T) {
	value := os.Getenv(countUTF8AMD64PageGuardEnv)
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
		if got, want := countUTF8BlocksWestmere(input), countUTF8Scalar(input[:length&^63]); got != want {
			t.Fatalf("Westmere raw count = %d, want %d", got, want)
		}
		if hasCountUTF8AVX2() {
			if got, want := countUTF8BlocksHaswell(input), countUTF8Scalar(input[:length&^127]); got != want {
				t.Fatalf("Haswell raw count = %d, want %d", got, want)
			}
		}
		checkCountUTF8AMD64(t, input)
	})
}

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

// Hand-authored Go-only deterministic no-overread coverage for the lookup4
// assembly port pinned to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f.
// It invokes direct test functions only and adds no product behavior.

const utf8NEONPageGuardEnv = "SIMDUTF_UTF8_NEON_GUARD"

func TestValidateUTF8NEONGuardPageNoOverread(t *testing.T) {
	for _, length := range []int{0, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129} {
		t.Run("length="+strconv.Itoa(length), func(t *testing.T) {
			runPageGuardSubprocess(t, "TestValidateUTF8NEONGuardPageHelper", utf8NEONPageGuardEnv, strconv.Itoa(length))
		})
	}
}

func TestValidateUTF8NEONGuardPageHelper(t *testing.T) {
	value := os.Getenv(utf8NEONPageGuardEnv)
	if value == "" {
		return
	}
	length, err := strconv.Atoi(value)
	if err != nil {
		t.Fatal(err)
	}
	withGuardPageBytes(t, length, func(input []byte) {
		for i := range input {
			input[i] = byte(i & 0x7f)
		}
		checkUTF8NEON(t, input)
	})
}

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
	"strings"
	"testing"
)

// Hand-authored Go-only deterministic no-overread coverage for the NEON port
// pinned to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/ascii_validation.h:6-45 and src/arm64/arm_validate_utf16.cpp:71-91.
// It invokes direct test functions only and adds no product behavior.

const asciiNEONPageGuardEnv = "SIMDUTF_NEON_GUARD"

func TestValidateASCIINEONGuardPageNoOverread(t *testing.T) {
	for _, kind := range []string{"byte", "utf16"} {
		lengths := []int{63, 64, 65, 127, 128, 129}
		if kind == "utf16" {
			lengths = []int{15, 16, 17, 31, 32, 33}
		}
		for _, length := range lengths {
			t.Run(kind+"/length="+strconv.Itoa(length), func(t *testing.T) {
				runPageGuardSubprocess(t, "TestValidateASCIINEONGuardPageHelper", asciiNEONPageGuardEnv,
					kind+","+strconv.Itoa(length))
			})
		}
	}
}

func TestValidateASCIINEONGuardPageHelper(t *testing.T) {
	guardCase := os.Getenv(asciiNEONPageGuardEnv)
	if guardCase == "" {
		return
	}
	parts := strings.Split(guardCase, ",")
	if len(parts) != 2 {
		t.Fatalf("invalid guard case %q", guardCase)
	}
	length, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("invalid guard length %q: %v", parts[1], err)
	}

	switch parts[0] {
	case "byte":
		withGuardPageBytes(t, length, func(input []byte) {
			for i := range input {
				input[i] = byte(i & 0x7f)
			}
			if !validateASCIINEON(input) {
				t.Fatal("valid guard-page byte input rejected")
			}
			if got, want := validateASCIIWithErrorsNEON(input), (Result{Error: Success, Count: len(input)}); got != want {
				t.Fatalf("result = %+v, want %+v", got, want)
			}
		})
	case "utf16":
		withGuardPageUint16s(t, length, func(input []uint16) {
			if !validateUTF16LEAsASCIINEON(input) || !validateUTF16BEAsASCIINEON(input) {
				t.Fatal("valid guard-page UTF-16 input rejected")
			}
		})
	default:
		t.Fatalf("invalid guard kind %q", parts[0])
	}
}

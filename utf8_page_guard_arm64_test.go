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

// Hand-authored Go-only deterministic no-overread coverage for the lookup4
// assembly port pinned to simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de.
// It invokes direct test functions only and adds no product behavior.

const utf8NEONPageGuardEnv = "SIMDUTF_UTF8_NEON_GUARD"

func TestValidateUTF8NEONGuardPageNoOverread(t *testing.T) {
	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 61, 62, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129}
	for _, kind := range []string{"ascii", "valid-tail", "truncated-tail"} {
		for _, length := range lengths {
			t.Run(kind+"/length="+strconv.Itoa(length), func(t *testing.T) {
				runPageGuardSubprocess(t, "TestValidateUTF8NEONGuardPageHelper", utf8NEONPageGuardEnv,
					kind+","+strconv.Itoa(length))
			})
		}
	}
}

func TestValidateUTF8NEONGuardPageHelper(t *testing.T) {
	guardCase := os.Getenv(utf8NEONPageGuardEnv)
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
	withGuardPageBytes(t, length, func(input []byte) {
		for i := range input {
			input[i] = 'a'
		}
		switch parts[0] {
		case "ascii":
		case "valid-tail":
			switch {
			case len(input) >= 4:
				copy(input[len(input)-4:], []byte{0xf0, 0x90, 0x80, 0x80})
			case len(input) >= 3:
				copy(input[len(input)-3:], []byte{0xe0, 0xa0, 0x80})
			case len(input) >= 2:
				copy(input[len(input)-2:], []byte{0xc2, 0x80})
			}
		case "truncated-tail":
			if len(input) != 0 {
				input[len(input)-1] = 0xf0
			}
		default:
			t.Fatalf("invalid guard kind %q", parts[0])
		}

		var remainder [64]byte
		copy(remainder[:], input[len(input)&^63:])
		validateUTF8Lookup4NEON(input, &remainder)
		checkUTF8NEON(t, input)
	})
}

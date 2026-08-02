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

//go:build amd64 && goexperiment.simd && (darwin || linux)

package simdutf

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

// Hand-authored Go-only deterministic no-overread coverage for the archsimd
// adaptation pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.
// It invokes direct test functions only and adds no product behavior.

const asciiArchsimdPageGuardEnv = "SIMDUTF_ARCHSIMD_GUARD"

func TestValidateASCIIArchsimdGuardPageNoOverread(t *testing.T) {
	requireASCIIArchsimdAVX2(t)
	for _, kind := range []string{"byte", "utf16"} {
		lengths := []int{31, 32, 33, 63, 64, 65}
		if kind == "utf16" {
			lengths = []int{15, 16, 17, 31, 32, 33}
		}
		for _, length := range lengths {
			t.Run(kind+"/length="+strconv.Itoa(length), func(t *testing.T) {
				runPageGuardSubprocess(t, "TestValidateASCIIArchsimdGuardPageHelper", asciiArchsimdPageGuardEnv,
					kind+","+strconv.Itoa(length))
			})
		}
	}
}

func TestValidateASCIIArchsimdGuardPageHelper(t *testing.T) {
	guardCase := os.Getenv(asciiArchsimdPageGuardEnv)
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
			if !validateASCIIArchsimd(input) {
				t.Fatal("valid guard-page byte input rejected")
			}
			if got, want := validateASCIIWithErrorsArchsimd(input), (Result{Error: Success, Count: len(input)}); got != want {
				t.Fatalf("result = %+v, want %+v", got, want)
			}
		})
	case "utf16":
		withGuardPageUint16s(t, length, func(input []uint16) {
			if !validateUTF16LEAsASCIIArchsimd(input) || !validateUTF16BEAsASCIIArchsimd(input) {
				t.Fatal("valid guard-page UTF-16 input rejected")
			}
		})
	default:
		t.Fatalf("invalid guard kind %q", parts[0])
	}
}

// Hand-authored Go-only physical guard-page coverage for the tagged CountUTF8
// adaptation pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/utf8.h:21-68 and src/haswell/implementation.cpp:1115-1119.

const countUTF8ArchsimdPageGuardEnv = "SIMDUTF_COUNT_UTF8_ARCHSIMD_GUARD"

func TestCountUTF8ArchsimdGuardPageNoOverread(t *testing.T) {
	requireCountUTF8ArchsimdAVX2(t)
	lengths := []int{0, 1, 31, 32, 33, 63, 64, 65, 95, 96, 97, 127, 128, 129, 255, 256, 257}
	pageSize := os.Getpagesize()
	lengths = append(lengths, pageSize-1, pageSize)
	for _, length := range lengths {
		t.Run("length="+strconv.Itoa(length), func(t *testing.T) {
			runPageGuardSubprocess(t, "TestCountUTF8ArchsimdGuardPageHelper", countUTF8ArchsimdPageGuardEnv, strconv.Itoa(length))
		})
	}
}

func TestCountUTF8ArchsimdGuardPageHelper(t *testing.T) {
	value := os.Getenv(countUTF8ArchsimdPageGuardEnv)
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
		checkCountUTF8Archsimd(t, input)
	})
}

// Hand-authored Go-only direct no-overread coverage for the lookup4 algorithm
// at simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216. It invokes
// tagged test functions only and adds no product behavior.

const utf8ArchsimdPageGuardEnv = "SIMDUTF_UTF8_ARCHSIMD_GUARD"

func TestValidateUTF8ArchsimdGuardPageNoOverread(t *testing.T) {
	requireUTF8ArchsimdAVX2(t)
	for _, length := range []int{0, 1, 31, 32, 33, 63, 64, 65, 95, 96, 97, 127, 128, 129} {
		runPageGuardSubprocess(t, "TestValidateUTF8ArchsimdGuardPageHelper", utf8ArchsimdPageGuardEnv, strconv.Itoa(length))
	}
}

func TestValidateUTF8ArchsimdGuardPageHelper(t *testing.T) {
	guardCase := os.Getenv(utf8ArchsimdPageGuardEnv)
	if guardCase == "" {
		t.Skip("guard-page subprocess helper")
	}
	length, err := strconv.Atoi(guardCase)
	if err != nil {
		t.Fatal(err)
	}
	requireUTF8ArchsimdAVX2(t)
	withGuardPageBytes(t, length, func(input []byte) {
		for i := range input {
			input[i] = 'a'
		}
		if !validateUTF8Archsimd(input) {
			t.Fatal("valid page-edge input rejected")
		}
		if got, want := validateUTF8WithErrorsArchsimd(input), (Result{Error: Success, Count: length}); got != want {
			t.Fatalf("with errors = %+v, want %+v", got, want)
		}
	})
}

const utf8LengthArchsimdPageGuardEnv = "SIMDUTF_UTF8_LENGTH_ARCHSIMD_GUARD"

func TestUTF8LengthArchsimdGuardPageNoOverread(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	lengths := []int{0, 1, 31, 32, 33, 63, 64, 65, 127, 128, 129}
	pageSize := os.Getpagesize()
	lengths = append(lengths, pageSize-1, pageSize)
	for _, length := range lengths {
		t.Run("length="+strconv.Itoa(length), func(t *testing.T) {
			runPageGuardSubprocess(t, "TestUTF8LengthArchsimdGuardPageHelper", utf8LengthArchsimdPageGuardEnv, strconv.Itoa(length))
		})
	}
}

func TestUTF8LengthArchsimdGuardPageHelper(t *testing.T) {
	value := os.Getenv(utf8LengthArchsimdPageGuardEnv)
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
		checkUTF8LengthArchsimd(t, input)
	})
}

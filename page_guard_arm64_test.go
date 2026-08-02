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
// pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
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

// Hand-authored Go-only guard-page coverage for the complete-block-only loads
// in the arm64 count port pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
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

// Hand-authored Go-only deterministic no-overread coverage for the lookup4
// assembly port pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b.
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

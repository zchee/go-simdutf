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
	"testing"
)

// Hand-authored Go-only physical guard-page coverage for the tagged CountUTF8
// adaptation pinned to simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
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

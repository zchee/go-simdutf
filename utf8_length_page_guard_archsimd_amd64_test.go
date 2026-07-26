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

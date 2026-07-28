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

//go:build amd64 && goexperiment.simd && unix

package simdutf

import (
	"os"
	"strconv"
	"testing"
)

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

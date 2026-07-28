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

//go:build (arm64 || (amd64 && goexperiment.simd)) && (darwin || linux)

package simdutf

import (
	"os"
	"testing"
	"unsafe"
)

// Hand-authored Go-only uint16 PROT_NONE mmap adapter for the port pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.
// Test-only unsafe is confined to an accessible mmap page; this adds no
// product behavior.

func withGuardPageUint16s(t *testing.T, length int, run func([]uint16)) {
	t.Helper()
	pageSize := os.Getpagesize()
	if length < 0 || length > pageSize/2 {
		t.Fatalf("invalid guarded uint16 length %d", length)
	}
	mapped := mapGuardPages(t, pageSize)
	defer unmapGuardPages(t, mapped)
	words := unsafe.Slice((*uint16)(unsafe.Pointer(&mapped[0])), pageSize/2)
	run(words[len(words)-length:])
}

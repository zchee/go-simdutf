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

//go:build (arm64 || amd64) && (darwin || linux)

package simdutf

import (
	"os"
	"syscall"
	"testing"
)

// Hand-authored Go-only PROT_NONE mmap scaffolding for the port pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.
// The mapping is test-only and adds no product behavior.

func withGuardPageBytes(t *testing.T, length int, run func([]byte)) {
	t.Helper()
	pageSize := os.Getpagesize()
	if length < 0 || length > pageSize {
		t.Fatalf("invalid guarded byte length %d", length)
	}
	mapped := mapGuardPages(t, pageSize)
	defer unmapGuardPages(t, mapped)
	run(mapped[pageSize-length : pageSize])
}

func mapGuardPages(t *testing.T, pageSize int) []byte {
	t.Helper()
	mapped, err := syscall.Mmap(-1, 0, 2*pageSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_PRIVATE|syscall.MAP_ANON)
	if err != nil {
		t.Fatalf("mmap guard pages: %v", err)
	}
	if err := syscall.Mprotect(mapped[pageSize:], syscall.PROT_NONE); err != nil {
		_ = syscall.Munmap(mapped)
		t.Fatalf("mprotect guard page: %v", err)
	}
	return mapped
}

func unmapGuardPages(t *testing.T, mapped []byte) {
	t.Helper()
	if err := syscall.Munmap(mapped); err != nil {
		t.Errorf("munmap guard pages: %v", err)
	}
}

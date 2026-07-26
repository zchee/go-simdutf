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

//go:build amd64 && (darwin || linux)

package simdutf

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

// Hand-authored Go-only deterministic no-overread coverage for the amd64
// lookup4 assembly ports pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f. It invokes direct
// test functions only and adds no product behavior.

const utf8AMD64PageGuardEnv = "SIMDUTF_UTF8_AMD64_GUARD"

func TestValidateUTF8AMD64GuardPageNoOverread(t *testing.T) {
	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 61, 62, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129}
	for _, kind := range []string{"ascii", "non-ascii", "valid-tail", "truncated-tail"} {
		for _, length := range lengths {
			t.Run(kind+"/length="+strconv.Itoa(length), func(t *testing.T) {
				runPageGuardSubprocess(t, "TestValidateUTF8AMD64GuardPageHelper", utf8AMD64PageGuardEnv, kind+","+strconv.Itoa(length))
			})
		}
	}
}

func TestValidateUTF8AMD64GuardPageHelper(t *testing.T) {
	guardCase := os.Getenv(utf8AMD64PageGuardEnv)
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
	withUTF8AMD64GuardPageBytes(t, length, func(input []byte) {
		for i := range input {
			input[i] = 'a'
		}
		switch parts[0] {
		case "ascii":
		case "non-ascii":
			for i := 0; i+2 <= len(input); i += 7 {
				input[i], input[i+1] = 0xc2, 0x80
			}
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

		for _, candidate := range utf8AMD64RawVariants() {
			if !candidate.supported {
				continue
			}
			candidate.prefix(input)
		}
		checkUTF8AMD64Variants(t, input)
	})
}

func withUTF8AMD64GuardPageBytes(t *testing.T, length int, run func([]byte)) {
	t.Helper()
	pageSize := os.Getpagesize()
	if length > pageSize {
		t.Fatalf("guarded input length %d exceeds page size %d", length, pageSize)
	}
	mapping, err := syscall.Mmap(-1, 0, pageSize*2, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := syscall.Munmap(mapping); err != nil {
			t.Errorf("munmap guard mapping: %v", err)
		}
	}()
	if _, _, errno := syscall.Syscall(syscall.SYS_MPROTECT,
		uintptr(unsafe.Pointer(&mapping[pageSize])), uintptr(pageSize), uintptr(syscall.PROT_NONE)); errno != 0 {
		t.Fatalf("mprotect guard page: %v", errno)
	}
	run(mapping[pageSize-length : pageSize])
}

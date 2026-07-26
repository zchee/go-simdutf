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

//go:build darwin || linux

package simdutf

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Hand-authored Go-only PROT_NONE no-overread scaffolding for the port pinned
// to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/ascii_validation.h:6-45 and src/generic/validate_utf16.h:128-158.
// Test-only unsafe is confined to an accessible mmap page; this adds no
// product behavior.

func runPageGuardSubprocess(t *testing.T, helper, envKey, value string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+helper+"$")
	prefix := envKey + "="
	command.Env = make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, prefix) {
			command.Env = append(command.Env, entry)
		}
	}
	command.Env = append(command.Env, prefix+value)
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("page-guard subprocess timed out: %v\n%s", ctx.Err(), output)
	}
	if err != nil {
		t.Fatalf("page-guard subprocess failed: %v\n%s", err, output)
	}
}

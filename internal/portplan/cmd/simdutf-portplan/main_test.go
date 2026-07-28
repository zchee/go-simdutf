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

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateDeterministicAndAtomic(t *testing.T) {
	root := repositoryRoot(t)
	temporary := t.TempDir()
	first := filepath.Join(temporary, "first")
	second := filepath.Join(temporary, "second")
	if err := generate(root, first); err != nil {
		t.Fatal(err)
	}
	if err := generate(root, second); err != nil {
		t.Fatal(err)
	}
	for _, name := range generatedNames {
		one, err := os.ReadFile(filepath.Join(first, name))
		if err != nil {
			t.Fatal(err)
		}
		two, err := os.ReadFile(filepath.Join(second, name))
		if err != nil || !bytes.Equal(one, two) {
			t.Fatalf("%s is not deterministic: %v", name, err)
		}
		checkedIn := filepath.Join(root, "docs/porting/simdutf-port-v1/generated", name)
		if reference, err := os.ReadFile(checkedIn); err == nil && !bytes.Equal(one, reference) {
			t.Fatalf("%s differs from checked-in generated artifact", name)
		} else if err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(first, "sentinel"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := publish(first, map[string][]byte{}); err == nil {
		t.Fatal("accepted incomplete artifact set")
	}
	if sentinel, err := os.ReadFile(filepath.Join(first, "sentinel")); err != nil || string(sentinel) != "keep" {
		t.Fatalf("failed publication altered prior output: %q %v", sentinel, err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	working, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(working, "go.mod")); err == nil {
			return working
		}
		parent := filepath.Dir(working)
		if parent == working {
			t.Fatal("repository root not found")
		}
		working = parent
	}
}

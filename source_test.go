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

package simdutf

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

const apacheLicenseHeader = `// Copyright 2026 The go-simdutf Authors.
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

`

// TestGoSourcesBeginWithApacheLicenseHeader is hand-authored Go-only license
// enforcement, not an upstream test vector.
func TestGoSourcesBeginWithApacheLicenseHeader(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		contents, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Errorf("read %s: %v", entry.Name(), err)
			continue
		}
		if !bytes.HasPrefix(contents, []byte(apacheLicenseHeader)) {
			t.Errorf("%s does not begin with the exact Apache license header from doc.go", entry.Name())
		}
	}
}

// TestAssemblySourcesBeginWithApacheLicenseHeader is hand-authored Go-only
// license enforcement, not an upstream test vector.
func TestAssemblySourcesBeginWithApacheLicenseHeader(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".s") {
			continue
		}
		contents, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Errorf("read %s: %v", entry.Name(), err)
			continue
		}
		if !bytes.HasPrefix(contents, []byte(apacheLicenseHeader)) {
			t.Errorf("%s does not begin with the exact Apache license header from doc.go", entry.Name())
		}
	}
}

// TestPhase1SourcesRecordPinnedProvenance is hand-authored Go-only provenance
// enforcement, not an upstream test vector.
func TestPhase1SourcesRecordPinnedProvenance(t *testing.T) {
	const upstreamSHA = "dec3aad192f47081110d9c766d4917bad243906f"
	files := []struct {
		name              string
		requiredFragments []string
	}{
		{"errors.go", []string{
			upstreamSHA,
			"include/simdutf/error.h",
		}},
		{"encoding.go", []string{
			upstreamSHA,
			"include/simdutf/encoding_types.h",
			"src/encoding_types.cpp",
		}},
		{"options.go", []string{
			upstreamSHA,
			"include/simdutf/implementation.h",
		}},
		{"cpu_amd64.go", []string{
			upstreamSHA,
			"src/simdutf/westmere.h",
			"src/simdutf/haswell.h",
			"independent implementation",
			"not translated or structurally copied from",
			"include/simdutf/internal/isadetection.h",
		}},
		{"cpu_amd64.s", []string{
			upstreamSHA,
			"src/simdutf/westmere.h",
			"src/simdutf/haswell.h",
			"independently written",
			"No policy or code from",
			"include/simdutf/internal/isadetection.h",
		}},
		{"dispatch.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_amd64.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_archsimd_amd64.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_archsimd_stub.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_arm64.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_generic.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
	}

	for _, file := range files {
		contents, err := os.ReadFile(file.name)
		if err != nil {
			t.Errorf("read %s: %v", file.name, err)
			continue
		}
		for _, fragment := range file.requiredFragments {
			if !bytes.Contains(contents, []byte(fragment)) {
				t.Errorf("%s does not record required provenance fragment %q", file.name, fragment)
			}
		}
	}
}
